package handlers

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"go-lang/auth"
	"go-lang/internal/domain"
)

// UserFinder é a busca de usuário por e-mail de que o AuthHandler precisa. O *repository.UserRepository a satisfaz.
type UserFinder interface {
	FindByEmail(email string) (*domain.User, error)
}

// AuthHandler atende a rota de login.
type AuthHandler struct {
	userRepository UserFinder
	tokenManager   *auth.TokenManager
}

// NewAuthHandler cria um AuthHandler que confere as credenciais em userRepository e emite tokens com tokenManager.
func NewAuthHandler(userRepository UserFinder, tokenManager *auth.TokenManager) *AuthHandler {
	return &AuthHandler{userRepository: userRepository, tokenManager: tokenManager}
}

// RegisterRoutes liga cada rota (método + caminho) ao seu handler.
func (authHandler *AuthHandler) RegisterRoutes(serveMux *http.ServeMux) {
	serveMux.HandleFunc("POST /login", authHandler.Login)
}

// loginInput é o corpo JSON aceito em POST /login.
type loginInput struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// loginResponse é o corpo JSON devolvido em um login bem-sucedido.
type loginResponse struct {
	Token string `json:"token"`
}

// Login trata POST /login.
func (authHandler *AuthHandler) Login(responseWriter http.ResponseWriter, request *http.Request) {
	var input loginInput

	err := json.NewDecoder(request.Body).Decode(&input)
	if err != nil {
		writeError(responseWriter, http.StatusBadRequest, err.Error())
		return
	}

	if input.Email == "" || input.Password == "" {
		writeError(responseWriter, http.StatusBadRequest, "email or password is empty")
		return
	}

	user, err := authHandler.userRepository.FindByEmail(input.Email)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			auth.SpendPasswordCheck(input.Password)
			writeError(responseWriter, http.StatusUnauthorized, "invalid credentials")
			return
		}
		log.Println("find user by email:", err)
		writeError(responseWriter, http.StatusInternalServerError, "internal server error")
		return
	}

	if !auth.CheckPassword(user.Password, input.Password) {
		writeError(responseWriter, http.StatusUnauthorized, "invalid credentials")
		return
	}

	token, err := authHandler.tokenManager.Issue(user.ID, user.Role)
	if err != nil {
		log.Println("issue token:", err)
		writeError(responseWriter, http.StatusInternalServerError, "internal server error")
		return
	}

	writeJSON(responseWriter, http.StatusOK, loginResponse{Token: token})
}
