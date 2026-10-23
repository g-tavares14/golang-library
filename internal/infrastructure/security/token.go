package security

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// ErrInvalidToken é devolvido quando o token está malformado, expirado ou com assinatura inválida.
var ErrInvalidToken = errors.New("invalid token")

// Claims são os dados do usuário guardados dentro do token.
type Claims struct {
	UserID string
	Role   string
}

// tokenClaims é o payload JWT: o id do usuário vai em sub e o papel em role.
type tokenClaims struct {
	Role string `json:"role"`
	jwt.RegisteredClaims
}

// TokenManager emite e valida tokens JWT assinados com HMAC-SHA256.
type TokenManager struct {
	secret []byte
	ttl    time.Duration
}

// NewTokenManager cria um TokenManager que assina com secret e emite tokens válidos por ttl.
func NewTokenManager(secret []byte, ttl time.Duration) *TokenManager {
	return &TokenManager{secret: secret, ttl: ttl}
}

// Issue gera um token para o usuário com o papel informado.
func (tokenManager *TokenManager) Issue(userID, role string) (string, error) {
	now := time.Now()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, tokenClaims{
		Role: role,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(tokenManager.ttl)),
		},
	})

	signed, err := token.SignedString(tokenManager.secret)
	if err != nil {
		return "", fmt.Errorf("sign token: %w", err)
	}
	return signed, nil
}

// Parse valida a assinatura e a expiração do token e devolve os dados do usuário.
// Qualquer falha de validação é devolvida como ErrInvalidToken.
func (tokenManager *TokenManager) Parse(signedToken string) (Claims, error) {
	var parsed tokenClaims

	_, err := jwt.ParseWithClaims(
		signedToken,
		&parsed,
		func(*jwt.Token) (any, error) { return tokenManager.secret, nil },
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithExpirationRequired(),
	)
	if err != nil || parsed.Subject == "" {
		return Claims{}, ErrInvalidToken
	}

	return Claims{UserID: parsed.Subject, Role: parsed.Role}, nil
}
