package domain_test

import (
	"errors"
	"strings"
	"testing"

	"go-lang/internal/domain"
)

func TestNewUserAlwaysHasRoleUser(t *testing.T) {
	user, err := domain.NewUser("Ana", "ana@example.com", "Senha-longa-1")
	if err != nil {
		t.Fatalf("NewUser() error = %v", err)
	}

	if user.Role != domain.RoleUser {
		t.Errorf("Role = %q, want %q", user.Role, domain.RoleUser)
	}
}

func TestValidateEmail(t *testing.T) {
	tests := []struct {
		name  string
		email string
		want  error
	}{
		{"simples", "ana@example.com", nil},
		{"com subdomínio e tag", "ana+tag@mail.example.com", nil},
		{"sem arroba", "ana.example.com", domain.ErrInvalidEmail},
		{"sem domínio", "ana@", domain.ErrInvalidEmail},
		{"sem usuário", "@example.com", domain.ErrInvalidEmail},
		{"com nome de exibição", "Ana <ana@example.com>", domain.ErrInvalidEmail},
		{"com espaço no fim", "ana@example.com ", domain.ErrInvalidEmail},
		{"no limite de 100", strings.Repeat("a", 88) + "@example.com", nil},
		{"acima de 100", strings.Repeat("a", 89) + "@example.com", domain.ErrInvalidEmail},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := domain.ValidateEmail(test.email); !errors.Is(err, test.want) {
				t.Errorf("ValidateEmail(%q) = %v, want %v", test.email, err, test.want)
			}
		})
	}
}

func TestValidatePassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
		want     error
	}{
		{"aceitável", "Senha-longa-1", nil},
		{"no limite de 8", "Senha-a1", nil},
		{"7 caracteres", "Senh-a1", domain.ErrWeakPassword},
		{"sem maiúscula", "senha-longa-1", domain.ErrWeakPassword},
		{"sem minúscula", "SENHA-LONGA-1", domain.ErrWeakPassword},
		{"sem número", "Senha-longa-x", domain.ErrWeakPassword},
		{"vazia", "", domain.ErrWeakPassword},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := domain.ValidatePassword(test.password); !errors.Is(err, test.want) {
				t.Errorf("ValidatePassword(%q) = %v, want %v", test.password, err, test.want)
			}
		})
	}
}

func TestUserValidateRequiresAllFields(t *testing.T) {
	tests := []struct {
		name string
		user domain.User
	}{
		{"sem nome", domain.User{Email: "ana@example.com", Password: "Senha-longa-1"}},
		{"sem e-mail", domain.User{Name: "Ana", Password: "Senha-longa-1"}},
		{"sem senha", domain.User{Name: "Ana", Email: "ana@example.com"}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := test.user.Validate(); !errors.Is(err, domain.ErrUserFieldsEmpty) {
				t.Errorf("Validate() = %v, want ErrUserFieldsEmpty", err)
			}
		})
	}
}

func TestUserValidateNameLength(t *testing.T) {
	tests := []struct {
		name     string
		userName string
		want     error
	}{
		{"100 caracteres", strings.Repeat("a", 100), nil},
		{"101 caracteres", strings.Repeat("a", 101), domain.ErrUserNameTooLong},
		// 100 caracteres acentuados ocupam 200 bytes: só passa se o limite contar caracteres.
		{"100 caracteres acentuados", strings.Repeat("é", 100), nil},
		{"101 caracteres acentuados", strings.Repeat("é", 101), domain.ErrUserNameTooLong},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			user := domain.User{Name: test.userName, Email: "ana@example.com", Password: "Senha-longa-1"}
			if err := user.Validate(); !errors.Is(err, test.want) {
				t.Errorf("Validate() = %v, want %v", err, test.want)
			}
		})
	}
}

func TestUserPatchValidate(t *testing.T) {
	empty, name, badEmail, weak := "", "Ana", "nao-e-email", "fraca"
	maxName, longName := strings.Repeat("a", 100), strings.Repeat("a", 101)
	accentedMaxName := strings.Repeat("é", 100)

	tests := []struct {
		name  string
		patch domain.UserPatch
		want  error
	}{
		{"nenhum campo", domain.UserPatch{}, domain.ErrNoFieldsToUpdate},
		{"nome vazio", domain.UserPatch{Name: &empty}, domain.ErrUserFieldsEmpty},
		{"só o nome", domain.UserPatch{Name: &name}, nil},
		{"e-mail inválido", domain.UserPatch{Email: &badEmail}, domain.ErrInvalidEmail},
		{"senha fraca", domain.UserPatch{Password: &weak}, domain.ErrWeakPassword},
		{"nome com 100 caracteres", domain.UserPatch{Name: &maxName}, nil},
		{"nome com 101 caracteres", domain.UserPatch{Name: &longName}, domain.ErrUserNameTooLong},
		{"nome com 100 caracteres acentuados", domain.UserPatch{Name: &accentedMaxName}, nil},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := test.patch.Validate(); !errors.Is(err, test.want) {
				t.Errorf("Validate() = %v, want %v", err, test.want)
			}
		})
	}
}

func TestValidateRoleChange(t *testing.T) {
	tests := []struct {
		name                string
		actor, target, role string
		want                error
	}{
		{"promover outro", "1", "2", domain.RoleAdmin, nil},
		{"rebaixar outro", "1", "2", domain.RoleUser, nil},
		{"papel inexistente", "1", "2", "root", domain.ErrInvalidRole},
		{"mudar o próprio papel", "1", "1", domain.RoleUser, domain.ErrOwnRoleChange},
		{"papel inválido vence o próprio papel", "1", "1", "root", domain.ErrInvalidRole},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := domain.ValidateRoleChange(test.actor, test.target, test.role)
			if !errors.Is(err, test.want) {
				t.Errorf("ValidateRoleChange() = %v, want %v", err, test.want)
			}
		})
	}
}

func TestBookValidate(t *testing.T) {
	tests := []struct {
		name string
		book domain.Book
		want error
	}{
		{"completo", domain.Book{Title: "Go", Author: "Pike"}, nil},
		{"sem título", domain.Book{Author: "Pike"}, domain.ErrBookFieldsEmpty},
		{"sem escritor", domain.Book{Title: "Go"}, domain.ErrBookFieldsEmpty},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := test.book.Validate(); !errors.Is(err, test.want) {
				t.Errorf("Validate() = %v, want %v", err, test.want)
			}
		})
	}
}
