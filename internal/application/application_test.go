package application_test

import (
	"errors"
	"testing"

	"go-lang/internal/application"
	"go-lang/internal/domain"
)

// fakeHasher troca a senha por "hash:<senha>", o que dispensa o bcrypt nos testes.
type fakeHasher struct{ dummyCalls int }

func (*fakeHasher) Hash(password string) (string, error) { return "hash:" + password, nil }

func (*fakeHasher) Verify(hash, password string) bool { return hash == "hash:"+password }

func (hasher *fakeHasher) VerifyAgainstDummy(string) { hasher.dummyCalls++ }

// fakeTokens emite tokens legíveis, como "token:<id>:<papel>".
type fakeTokens struct{}

func (fakeTokens) Issue(userID, role string) (string, error) {
	return "token:" + userID + ":" + role, nil
}

// fakeUsers é um domain.UserRepository mínimo: guarda usuários por e-mail.
// Os métodos que o teste não usa vêm da interface embutida e dariam panic se fossem chamados.
type fakeUsers struct {
	domain.UserRepository
	byEmail map[string]*domain.User
}

func newFakeUsers() *fakeUsers { return &fakeUsers{byEmail: map[string]*domain.User{}} }

func (store *fakeUsers) Create(user *domain.User) error {
	if _, exists := store.byEmail[user.Email]; exists {
		return domain.ErrEmailAlreadyExists
	}
	user.ID = "id-" + user.Email
	store.byEmail[user.Email] = user
	return nil
}

func (store *fakeUsers) CreateIfEmailMissing(user *domain.User) (bool, error) {
	if _, exists := store.byEmail[user.Email]; exists {
		return false, nil
	}
	user.ID = "id-" + user.Email
	store.byEmail[user.Email] = user
	return true, nil
}

func (store *fakeUsers) FindByEmail(email string) (*domain.User, error) {
	user, exists := store.byEmail[email]
	if !exists {
		return nil, domain.ErrUserNotFound
	}
	return user, nil
}

func TestRegisterHashesPasswordAndForcesRoleUser(t *testing.T) {
	store := newFakeUsers()
	service := application.NewUserService(store, &fakeHasher{})

	user, err := service.Register("Ana", "ana@example.com", "Senha-longa-1")
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	if user.Role != domain.RoleUser {
		t.Errorf("Role = %q, want %q", user.Role, domain.RoleUser)
	}
	if got := store.byEmail["ana@example.com"].Password; got != "hash:Senha-longa-1" {
		t.Errorf("senha gravada = %q, want o hash", got)
	}
}

func TestRegisterDoesNotTouchStoreWhenInvalid(t *testing.T) {
	store := newFakeUsers()
	service := application.NewUserService(store, &fakeHasher{})

	_, err := service.Register("Ana", "ana@example.com", "fraca")
	if !errors.Is(err, domain.ErrWeakPassword) {
		t.Fatalf("Register() error = %v, want ErrWeakPassword", err)
	}
	if len(store.byEmail) != 0 {
		t.Error("um cadastro inválido chegou ao repositório")
	}
}

func TestEnsureAdminIsIdempotent(t *testing.T) {
	store := newFakeUsers()
	service := application.NewUserService(store, &fakeHasher{})

	created, err := service.EnsureAdmin("admin@example.com", "qualquer-senha")
	if err != nil || !created {
		t.Fatalf("primeira chamada: created = %v, error = %v, want true, nil", created, err)
	}

	admin := store.byEmail["admin@example.com"]
	if admin.Role != domain.RoleAdmin || admin.Password != "hash:qualquer-senha" {
		t.Errorf("admin gravado = %+v", admin)
	}

	created, err = service.EnsureAdmin("admin@example.com", "outra-senha")
	if err != nil || created {
		t.Fatalf("segunda chamada: created = %v, error = %v, want false, nil", created, err)
	}
	if store.byEmail["admin@example.com"].Password != "hash:qualquer-senha" {
		t.Error("a segunda chamada alterou o admin existente")
	}
}

func TestChangeRoleRejectsBeforeReachingStore(t *testing.T) {
	// O fakeUsers não implementa UpdateRole: se a regra não barrasse antes, o teste daria panic.
	service := application.NewUserService(newFakeUsers(), &fakeHasher{})

	if _, err := service.ChangeRole("1", "1", domain.RoleUser); !errors.Is(err, domain.ErrOwnRoleChange) {
		t.Errorf("ChangeRole() no próprio usuário = %v, want ErrOwnRoleChange", err)
	}
	if _, err := service.ChangeRole("1", "2", "root"); !errors.Is(err, domain.ErrInvalidRole) {
		t.Errorf("ChangeRole() com papel inválido = %v, want ErrInvalidRole", err)
	}
}

func TestLogin(t *testing.T) {
	store := newFakeUsers()
	hasher := &fakeHasher{}
	if _, err := application.NewUserService(store, hasher).Register("Ana", "ana@example.com", "Senha-longa-1"); err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	service := application.NewAuthService(store, hasher, fakeTokens{})

	token, err := service.Login("ana@example.com", "Senha-longa-1")
	if err != nil || token != "token:id-ana@example.com:user" {
		t.Errorf("Login() = %q, %v", token, err)
	}

	if _, err := service.Login("ana@example.com", "errada"); !errors.Is(err, domain.ErrInvalidCredentials) {
		t.Errorf("senha errada: error = %v, want ErrInvalidCredentials", err)
	}
	if hasher.dummyCalls != 0 {
		t.Error("a comparação falsa só vale para e-mail inexistente")
	}

	if _, err := service.Login("ninguem@example.com", "qualquer"); !errors.Is(err, domain.ErrInvalidCredentials) {
		t.Errorf("e-mail inexistente: error = %v, want ErrInvalidCredentials", err)
	}
	if hasher.dummyCalls != 1 {
		t.Errorf("dummyCalls = %d, want 1: o e-mail inexistente precisa gastar a comparação falsa", hasher.dummyCalls)
	}

	if _, err := service.Login("", ""); !errors.Is(err, domain.ErrLoginFieldsEmpty) {
		t.Errorf("campos vazios: error = %v, want ErrLoginFieldsEmpty", err)
	}
}
