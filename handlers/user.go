package handlers

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"slices"

	"go-lang/repository"
)

// Papéis aceitos para um usuário.
const (
	roleUser  = "user"
	roleAdmin = "admin"
)

var allowedRoles = []string{roleUser, roleAdmin}

// UserHandler atende as rotas HTTP de usuários.
type UserHandler struct {
	userRepository *repository.UserRepository
}

// NewUserHandler cria um UserHandler que usa userRepository.
func NewUserHandler(userRepository *repository.UserRepository) *UserHandler {
	return &UserHandler{userRepository: userRepository}
}

// RegisterRoutes liga cada rota (método + caminho) ao seu handler.
func (userHandler *UserHandler) RegisterRoutes(serveMux *http.ServeMux) {
	serveMux.HandleFunc("POST /users", userHandler.Create)
	serveMux.HandleFunc("GET /users", userHandler.List)
	serveMux.HandleFunc("PUT /users/{id}", userHandler.Update)
	serveMux.HandleFunc("DELETE /users/{id}", userHandler.Delete)
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

	if user.Role == "" {
		user.Role = roleUser
	}
	if !slices.Contains(allowedRoles, user.Role) {
		writeError(responseWriter, http.StatusBadRequest, "invalid role")
		return
	}

	err = userHandler.userRepository.Create(&user)
	if err != nil {
		if errors.Is(err, repository.ErrEmailAlreadyExists) {
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

	var responses []userResponse
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

	if user.Role != roleAdmin {
		writeError(responseWriter, http.StatusForbidden, "only admin can update users")
		return
	}

	user.ID = request.PathValue("id")

	err = userHandler.userRepository.Update(&user)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			writeError(responseWriter, http.StatusNotFound, "user not found")
			return
		}
		if errors.Is(err, repository.ErrInvalidUserID) {
			writeError(responseWriter, http.StatusBadRequest, "invalid user id")
			return
		}
		if errors.Is(err, repository.ErrEmailAlreadyExists) {
			writeError(responseWriter, http.StatusConflict, "email already exists")
			return
		}
		log.Println("update user:", err)
		writeError(responseWriter, http.StatusInternalServerError, "internal server error")
		return
	}

	writeJSON(responseWriter, http.StatusOK, newUserResponse(&user))
}

// Delete trata DELETE /users/{id}.
func (userHandler *UserHandler) Delete(responseWriter http.ResponseWriter, request *http.Request) {
	user, err := userHandler.userRepository.Delete(request.PathValue("id"))
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			writeError(responseWriter, http.StatusNotFound, "user not found")
			return
		}
		if errors.Is(err, repository.ErrInvalidUserID) {
			writeError(responseWriter, http.StatusBadRequest, "invalid user id")
			return
		}
		log.Println("delete user:", err)
		writeError(responseWriter, http.StatusInternalServerError, "internal server error")
		return
	}

	writeJSON(responseWriter, http.StatusOK, newUserResponse(user))
}
