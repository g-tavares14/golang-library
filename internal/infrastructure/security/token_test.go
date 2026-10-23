package security_test

import (
	"errors"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"go-lang/internal/infrastructure/security"
)

var testSecret = []byte("0123456789012345678901234567890123456789")

func TestTokenManagerRoundTrip(t *testing.T) {
	tokenManager := security.NewTokenManager(testSecret, time.Hour)

	token, err := tokenManager.Issue("user-1", "admin")
	if err != nil {
		t.Fatalf("Issue() error = %v", err)
	}

	claims, err := tokenManager.Parse(token)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	want := security.Claims{UserID: "user-1", Role: "admin"}
	if claims != want {
		t.Errorf("Parse() = %+v, want %+v", claims, want)
	}
}

func TestTokenManagerParseRejects(t *testing.T) {
	tokenManager := security.NewTokenManager(testSecret, time.Hour)

	expired, err := security.NewTokenManager(testSecret, -time.Minute).Issue("user-1", "user")
	if err != nil {
		t.Fatalf("Issue() error = %v", err)
	}

	otherSecret, err := security.NewTokenManager([]byte("uma-outra-chave-com-mais-de-32-bytes!!"), time.Hour).Issue("user-1", "user")
	if err != nil {
		t.Fatalf("Issue() error = %v", err)
	}

	// Token sem assinatura (alg=none), o ataque clássico contra JWT.
	unsigned, err := jwt.NewWithClaims(jwt.SigningMethodNone, jwt.MapClaims{
		"sub":  "user-1",
		"role": "admin",
		"exp":  time.Now().Add(time.Hour).Unix(),
	}).SignedString(jwt.UnsafeAllowNoneSignatureType)
	if err != nil {
		t.Fatalf("sign unsigned token: %v", err)
	}

	// Token assinado com a chave certa, mas sem expiração.
	withoutExpiry, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":  "user-1",
		"role": "admin",
	}).SignedString(testSecret)
	if err != nil {
		t.Fatalf("sign token without exp: %v", err)
	}

	tests := []struct {
		name  string
		token string
	}{
		{"vazio", ""},
		{"lixo", "isto.nao.e-jwt"},
		{"expirado", expired},
		{"outra chave", otherSecret},
		{"sem assinatura", unsigned},
		{"sem expiração", withoutExpiry},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := tokenManager.Parse(test.token)
			if !errors.Is(err, security.ErrInvalidToken) {
				t.Errorf("Parse() error = %v, want ErrInvalidToken", err)
			}
		})
	}
}
