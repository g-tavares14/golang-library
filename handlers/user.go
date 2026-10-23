package handlers

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"go-lang/internal/domain"
	"go-lang/internal/infrastructure/security"
)

// UserStore é o acesso a usuários de que o UserHandler precisa. O *postgres.UserRepository a satisfaz.
type UserStore interface {
	Create(user *domain.User) error
	List() ([]domain.User, error)
	Update(user *domain.User) error
	Patch(userID string, patch domain.UserPatch) (*domain.User, error)
	UpdateRole(userID, role string) (*domain.User, error)
	Delete(userID string) (*domain.User, error)
}

// UserHandler atende as rotas HTTP de usuários.
type UserHandler struct {
	userRepository UserStore
}

// NewUserHandler cria um UserHandler que usa userRepository.
func NewUserHandler(userRepository UserStore) *UserHandler {
	return &UserHandler{userRepository: userRepository}
}

// RegisterRoutes liga cada rota (método + caminho) ao seu handler.
// O cadastro é público; listar exige login; alterar (PUT e PATCH), apagar e mudar o papel exigem admin.
func (userHandler *UserHandler) RegisterRoutes(serveMux *http.ServeMux, guard *Guard) {
	serveMux.HandleFunc("POST /users", userHandler.Create)
	serveMux.HandleFunc("GET /users", guard.RequireLogin(userHandler.List))
	serveMux.HandleFunc("PUT /users/{id}", guard.RequireAdmin(userHandler.Update))
	serveMux.HandleFunc("PATCH /users/{id}", guard.RequireAdmin(userHandler.Patch))
	serveMux.HandleFunc("DELETE /users/{id}", guard.RequireAdmin(userHandler.Delete))
	serveMux.HandleFunc("PUT /users/{id}/role", guard.RequireAdmin(userHandler.UpdateRole))
}

// Create trata POST /users.
func (userHandler *UserHandler) Create(responseWriter http.ResponseWriter, request *http.Request) {
	var input userInput

	err := json.NewDecoder(request.Body).Decode(&input)
	if err != nil {
		writeError(responseWriter, http.StatusBadRequest, err.Error())
		return
	}

	user := input.toUser()

	if user.Name == "" || user.Email == "" || user.Password == "" {
		writeError(responseWriter, http.StatusBadRequest, "user name, email or password is empty")
		return
	}

	if err := validateCredentials(user.Email, user.Password); err != nil {
		writeError(responseWriter, http.StatusBadRequest, err.Error())
		return
	}

	// O cadastro é público: o papel é decidido pelo servidor, nunca pelo cliente.
	user.Role = domain.RoleUser

	passwordHash, err := security.HashPassword(user.Password)
	if err != nil {
		if errors.Is(err, security.ErrPasswordTooLong) {
			writeError(responseWriter, http.StatusBadRequest, "password is too long")
			return
		}
		log.Println("hash password:", err)
		writeError(responseWriter, http.StatusInternalServerError, "internal server error")
		return
	}
	user.Password = passwordHash

	err = userHandler.userRepository.Create(&user)
	if err != nil {
		if errors.Is(err, domain.ErrEmailAlreadyExists) {
			writeError(responseWriter, http.StatusConflict, "email already exists")
			return
		}
		log.Println("create user:", err)
		writeError(responseWriter, http.StatusInternalServerError, "internal server error")
		return
	}

	writeJSON(responseWriter, http.StatusCreated, newUserResponse(&user))
}

// List trata GET /users.
func (userHandler *UserHandler) List(responseWriter http.ResponseWriter, _ *http.Request) {
	users, err := userHandler.userRepository.List()
	if err != nil {
		log.Println("list users:", err)
		writeError(responseWriter, http.StatusInternalServerError, "internal server error")
		return
	}

	responses := make([]userResponse, 0, len(users))
	for i := range users {
		responses = append(responses, newUserResponse(&users[i]))
	}

	writeJSON(responseWriter, http.StatusOK, responses)
}

// Update trata PUT /users/{id}.
func (userHandler *UserHandler) Update(responseWriter http.ResponseWriter, request *http.Request) {
	var input userInput

	err := json.NewDecoder(request.Body).Decode(&input)
	if err != nil {
		writeError(responseWriter, http.StatusBadRequest, err.Error())
		return
	}

	user := input.toUser()

	if user.Name == "" || user.Email == "" || user.Password == "" {
		writeError(responseWriter, http.StatusBadRequest, "user name, email or password is empty")
		return
	}

	if err := validateCredentials(user.Email, user.Password); err != nil {
		writeError(responseWriter, http.StatusBadRequest, err.Error())
		return
	}

	user.ID = request.PathValue("id")

	passwordHash, err := security.HashPassword(user.Password)
	if err != nil {
		if errors.Is(err, security.ErrPasswordTooLong) {
			writeError(responseWriter, http.StatusBadRequest, "password is too long")
			return
		}
		log.Println("hash password:", err)
		writeError(responseWriter, http.StatusInternalServerError, "internal server error")
		return
	}
	user.Password = passwordHash

	err = userHandler.userRepository.Update(&user)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			writeError(responseWriter, http.StatusNotFound, "user not found")
			return
		}
		if errors.Is(err, domain.ErrInvalidUserID) {
			writeError(responseWriter, http.StatusBadRequest, "invalid user id")
			return
		}
		if errors.Is(err, domain.ErrEmailAlreadyExists) {
			writeError(responseWriter, http.StatusConflict, "email already exists")
			return
		}
		log.Println("update user:", err)
		writeError(responseWriter, http.StatusInternalServerError, "internal server error")
		return
	}

	writeJSON(responseWriter, http.StatusOK, newUserResponse(&user))
}

// Patch trata PATCH /users/{id}: altera só os campos enviados, então dá para trocar o nome sem reenviar a senha.
func (userHandler *UserHandler) Patch(responseWriter http.ResponseWriter, request *http.Request) {
	var input userPatchInput

	err := json.NewDecoder(request.Body).Decode(&input)
	if err != nil {
		writeError(responseWriter, http.StatusBadRequest, err.Error())
		return
	}

	patch := input.toPatch()

	if patch.Name == nil && patch.Email == nil && patch.Password == nil {
		writeError(responseWriter, http.StatusBadRequest, "no fields to update")
		return
	}

	if (patch.Name != nil && *patch.Name == "") ||
		(patch.Email != nil && *patch.Email == "") ||
		(patch.Password != nil && *patch.Password == "") {
		writeError(responseWriter, http.StatusBadRequest, "user name, email or password is empty")
		return
	}

	if patch.Email != nil {
		if err := validateEmail(*patch.Email); err != nil {
			writeError(responseWriter, http.StatusBadRequest, err.Error())
			return
		}
	}

	if patch.Password != nil {
		if err := validatePassword(*patch.Password); err != nil {
			writeError(responseWriter, http.StatusBadRequest, err.Error())
			return
		}

		passwordHash, err := security.HashPassword(*patch.Password)
		if err != nil {
			if errors.Is(err, security.ErrPasswordTooLong) {
				writeError(responseWriter, http.StatusBadRequest, "password is too long")
				return
			}
			log.Println("hash password:", err)
			writeError(responseWriter, http.StatusInternalServerError, "internal server error")
			return
		}
		patch.Password = &passwordHash
	}

	user, err := userHandler.userRepository.Patch(request.PathValue("id"), patch)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			writeError(responseWriter, http.StatusNotFound, "user not found")
			return
		}
		if errors.Is(err, domain.ErrInvalidUserID) {
			writeError(responseWriter, http.StatusBadRequest, "invalid user id")
			return
		}
		if errors.Is(err, domain.ErrEmailAlreadyExists) {
			writeError(responseWriter, http.StatusConflict, "email already exists")
			return
		}
		log.Println("patch user:", err)
		writeError(responseWriter, http.StatusInternalServerError, "internal server error")
		return
	}

	writeJSON(responseWriter, http.StatusOK, newUserResponse(user))
}

// Delete trata DELETE /users/{id}.
func (userHandler *UserHandler) Delete(responseWriter http.ResponseWriter, request *http.Request) {
	user, err := userHandler.userRepository.Delete(request.PathValue("id"))
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			writeError(responseWriter, http.StatusNotFound, "user not found")
			return
		}
		if errors.Is(err, domain.ErrInvalidUserID) {
			writeError(responseWriter, http.StatusBadRequest, "invalid user id")
			return
		}
		log.Println("delete user:", err)
		writeError(responseWriter, http.StatusInternalServerError, "internal server error")
		return
	}

	writeJSON(responseWriter, http.StatusOK, newUserResponse(user))
}

// UpdateRole trata PUT /users/{id}/role, a única forma de promover ou rebaixar um usuário.
// Um admin não pode mudar o próprio papel, para o sistema não ficar sem admin.
func (userHandler *UserHandler) UpdateRole(responseWriter http.ResponseWriter, request *http.Request) {
	var input roleInput

	err := json.NewDecoder(request.Body).Decode(&input)
	if err != nil {
		writeError(responseWriter, http.StatusBadRequest, err.Error())
		return
	}

	if !domain.IsValidRole(input.Role) {
		writeError(responseWriter, http.StatusBadRequest, "invalid role")
		return
	}

	userID := request.PathValue("id")

	claims, _ := claimsFromContext(request.Context())
	if claims.UserID == userID {
		writeError(responseWriter, http.StatusForbidden, "cannot change your own role")
		return
	}

	user, err := userHandler.userRepository.UpdateRole(userID, input.Role)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			writeError(responseWriter, http.StatusNotFound, "user not found")
			return
		}
		if errors.Is(err, domain.ErrInvalidUserID) {
			writeError(responseWriter, http.StatusBadRequest, "invalid user id")
			return
		}
		log.Println("update user role:", err)
		writeError(responseWriter, http.StatusInternalServerError, "internal server error")
		return
	}

	writeJSON(responseWriter, http.StatusOK, newUserResponse(user))
}
