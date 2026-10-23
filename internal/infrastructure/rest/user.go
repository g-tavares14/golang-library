package rest

import (
	"encoding/json"
	"net/http"

	"go-lang/internal/application"
)

// UserHandler atende as rotas HTTP de usuários.
type UserHandler struct {
	userService *application.UserService
}

// NewUserHandler cria um UserHandler que usa userService.
func NewUserHandler(userService *application.UserService) *UserHandler {
	return &UserHandler{userService: userService}
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

	user, err := userHandler.userService.Register(input.Name, input.Email, input.Password)
	if err != nil {
		writeServiceError(responseWriter, "create user", err)
		return
	}

	writeJSON(responseWriter, http.StatusCreated, newUserResponse(user))
}

// List trata GET /users.
func (userHandler *UserHandler) List(responseWriter http.ResponseWriter, _ *http.Request) {
	users, err := userHandler.userService.List()
	if err != nil {
		writeServiceError(responseWriter, "list users", err)
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

	user, err := userHandler.userService.Replace(request.PathValue("id"), input.Name, input.Email, input.Password)
	if err != nil {
		writeServiceError(responseWriter, "update user", err)
		return
	}

	writeJSON(responseWriter, http.StatusOK, newUserResponse(user))
}

// Patch trata PATCH /users/{id}: altera só os campos enviados, então dá para trocar o nome sem reenviar a senha.
func (userHandler *UserHandler) Patch(responseWriter http.ResponseWriter, request *http.Request) {
	var input userPatchInput

	err := json.NewDecoder(request.Body).Decode(&input)
	if err != nil {
		writeError(responseWriter, http.StatusBadRequest, err.Error())
		return
	}

	user, err := userHandler.userService.Patch(request.PathValue("id"), input.toPatch())
	if err != nil {
		writeServiceError(responseWriter, "patch user", err)
		return
	}

	writeJSON(responseWriter, http.StatusOK, newUserResponse(user))
}

// UpdateRole trata PUT /users/{id}/role, a única forma de promover ou rebaixar um usuário.
func (userHandler *UserHandler) UpdateRole(responseWriter http.ResponseWriter, request *http.Request) {
	var input roleInput

	err := json.NewDecoder(request.Body).Decode(&input)
	if err != nil {
		writeError(responseWriter, http.StatusBadRequest, err.Error())
		return
	}

	claims, _ := claimsFromContext(request.Context())

	user, err := userHandler.userService.ChangeRole(claims.UserID, request.PathValue("id"), input.Role)
	if err != nil {
		writeServiceError(responseWriter, "update user role", err)
		return
	}

	writeJSON(responseWriter, http.StatusOK, newUserResponse(user))
}

// Delete trata DELETE /users/{id}.
func (userHandler *UserHandler) Delete(responseWriter http.ResponseWriter, request *http.Request) {
	user, err := userHandler.userService.Delete(request.PathValue("id"))
	if err != nil {
		writeServiceError(responseWriter, "delete user", err)
		return
	}

	writeJSON(responseWriter, http.StatusOK, newUserResponse(user))
}
