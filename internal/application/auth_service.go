package application

import (
	"errors"

	"go-lang/internal/domain"
)

// AuthService reúne o caso de uso de login.
type AuthService struct {
	users  domain.UserRepository
	hasher PasswordHasher
	tokens TokenIssuer
}

// NewAuthService cria um AuthService que confere credenciais em users e emite tokens com tokens.
func NewAuthService(users domain.UserRepository, hasher PasswordHasher, tokens TokenIssuer) *AuthService {
	return &AuthService{users: users, hasher: hasher, tokens: tokens}
}

// Login confere e-mail e senha e devolve um token de acesso.
// E-mail inexistente e senha errada dão o mesmo erro, domain.ErrInvalidCredentials, no mesmo tempo.
func (authService *AuthService) Login(email, password string) (string, error) {
	if email == "" || password == "" {
		return "", domain.ErrLoginFieldsEmpty
	}

	user, err := authService.users.FindByEmail(email)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			authService.hasher.VerifyAgainstDummy(password)
			return "", domain.ErrInvalidCredentials
		}
		return "", err
	}

	if !authService.hasher.Verify(user.Password, password) {
		return "", domain.ErrInvalidCredentials
	}

	return authService.tokens.Issue(user.ID, user.Role)
}
