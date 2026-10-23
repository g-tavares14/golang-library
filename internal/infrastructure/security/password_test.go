package security_test

import (
	"errors"
	"strings"
	"testing"

	"go-lang/internal/infrastructure/security"
)

func TestHashPasswordAndCheckPassword(t *testing.T) {
	hash, err := security.HashPassword("s3nha-forte")
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}

	if hash == "s3nha-forte" {
		t.Fatal("HashPassword() devolveu a senha em texto puro")
	}

	if !security.CheckPassword(hash, "s3nha-forte") {
		t.Error("CheckPassword() rejeitou a senha correta")
	}

	if security.CheckPassword(hash, "outra-senha") {
		t.Error("CheckPassword() aceitou uma senha errada")
	}
}

func TestHashPasswordUsesRandomSalt(t *testing.T) {
	first, err := security.HashPassword("mesma-senha")
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}

	second, err := security.HashPassword("mesma-senha")
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}

	if first == second {
		t.Error("duas chamadas com a mesma senha geraram o mesmo hash")
	}
}

func TestHashPasswordTooLong(t *testing.T) {
	_, err := security.HashPassword(strings.Repeat("a", 73))
	if !errors.Is(err, security.ErrPasswordTooLong) {
		t.Errorf("HashPassword() error = %v, want ErrPasswordTooLong", err)
	}
}

func TestCheckPasswordInvalidHash(t *testing.T) {
	if security.CheckPassword("isto-nao-e-um-hash", "qualquer") {
		t.Error("CheckPassword() aceitou um hash inválido")
	}
}
