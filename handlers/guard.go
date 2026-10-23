package handlers

import (
	"context"
	"net/http"
	"strings"

	"go-lang/internal/domain"
	"go-lang/internal/infrastructure/security"
)

// claimsContextKey é a chave sob a qual o Guard guarda os dados do token no contexto da requisição.
type claimsContextKey struct{}

// Guard protege rotas exigindo um token JWT válido no cabeçalho Authorization.
type Guard struct {
	tokenManager *security.TokenManager
}

// NewGuard cria um Guard que valida tokens com tokenManager.
func NewGuard(tokenManager *security.TokenManager) *Guard {
	return &Guard{tokenManager: tokenManager}
}

// RequireLogin só deixa passar requisições com um token válido.
// O papel vem sempre do token assinado, nunca do corpo da requisição.
func (guard *Guard) RequireLogin(next http.HandlerFunc) http.HandlerFunc {
	return func(responseWriter http.ResponseWriter, request *http.Request) {
		claims, ok := guard.authenticate(request)
		if !ok {
			responseWriter.Header().Set("WWW-Authenticate", "Bearer")
			writeError(responseWriter, http.StatusUnauthorized, "missing or invalid token")
			return
		}

		ctx := context.WithValue(request.Context(), claimsContextKey{}, claims)
		next(responseWriter, request.WithContext(ctx))
	}
}

// RequireAdmin só deixa passar requisições com um token válido de um admin.
func (guard *Guard) RequireAdmin(next http.HandlerFunc) http.HandlerFunc {
	return guard.RequireLogin(func(responseWriter http.ResponseWriter, request *http.Request) {
		claims, _ := claimsFromContext(request.Context())
		if claims.Role != domain.RoleAdmin {
			writeError(responseWriter, http.StatusForbidden, "admin role required")
			return
		}

		next(responseWriter, request)
	})
}

// authenticate lê o token Bearer da requisição e devolve os dados dele.
func (guard *Guard) authenticate(request *http.Request) (security.Claims, bool) {
	scheme, token, found := strings.Cut(request.Header.Get("Authorization"), " ")
	if !found || !strings.EqualFold(scheme, "Bearer") {
		return security.Claims{}, false
	}

	claims, err := guard.tokenManager.Parse(strings.TrimSpace(token))
	if err != nil {
		return security.Claims{}, false
	}

	return claims, true
}

// claimsFromContext devolve os dados do token guardados pelo Guard.
func claimsFromContext(ctx context.Context) (security.Claims, bool) {
	claims, ok := ctx.Value(claimsContextKey{}).(security.Claims)
	return claims, ok
}
