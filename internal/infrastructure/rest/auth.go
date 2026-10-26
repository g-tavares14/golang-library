package rest

import (
	"net/http"

	"go-lang/internal/application"
)

// AuthHandler atende a rota de login.
type AuthHandler struct {
	authService *application.AuthService
}

// NewAuthHandler cria um AuthHandler que faz o login com authService.
func NewAuthHandler(authService *application.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

// RegisterRoutes liga cada rota (método + caminho) ao seu handler.
// O login passa por loginLimiter, que barra tentativas em excesso de um mesmo IP (força bruta).
func (authHandler *AuthHandler) RegisterRoutes(serveMux *http.ServeMux, loginLimiter *RateLimiter) {
	serveMux.HandleFunc("POST /login", loginLimiter.Limit(authHandler.Login))
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

	if !decodeJSON(responseWriter, request, &input) {
		return
	}

	token, err := authHandler.authService.Login(input.Email, input.Password)
	if err != nil {
		writeServiceError(responseWriter, "login", err)
		return
	}

	writeJSON(responseWriter, http.StatusOK, loginResponse{Token: token})
}
