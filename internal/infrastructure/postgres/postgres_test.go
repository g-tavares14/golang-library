package postgres_test

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	_ "github.com/lib/pq" // driver do PostgreSQL para o database/sql

	"go-lang/internal/domain"
	"go-lang/internal/infrastructure/postgres"
)

// openTestDB conecta ao Postgres indicado em TEST_DATABASE_URL, que precisa ter as migrations aplicadas.
// Sem a variável, o teste é pulado. Exemplo:
//
//	TEST_DATABASE_URL="host=localhost port=5432 user=postgres password=... dbname=golang sslmode=disable" go test ./internal/infrastructure/postgres/
func openTestDB(t *testing.T) *sql.DB {
	t.Helper()

	connectionString := os.Getenv("TEST_DATABASE_URL")
	if connectionString == "" {
		t.Skip("TEST_DATABASE_URL não definida; pulando teste de integração com o Postgres")
	}

	database, err := sql.Open("postgres", connectionString)
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })

	if err := database.Ping(); err != nil {
		t.Fatalf("ping database: %v", err)
	}

	return database
}

// uniqueEmail gera um e-mail que não colide com outros testes nem com execuções anteriores.
func uniqueEmail(prefix string) string {
	return fmt.Sprintf("%s-%d@test.example.com", prefix, time.Now().UnixNano())
}

// newTestUser cria um usuário no banco e o apaga ao final do teste.
func newTestUser(t *testing.T, database *sql.DB, role string) *domain.User {
	t.Helper()

	user := &domain.User{Name: "Teste", Email: uniqueEmail("user"), Password: "hash", Role: role}
	if err := postgres.NewUserRepository(database).Create(user); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	t.Cleanup(func() { _, _ = database.Exec(`DELETE FROM users WHERE id = $1`, user.ID) })

	return user
}

func TestUserCreateAndFindByEmail(t *testing.T) {
	database := openTestDB(t)
	userRepository := postgres.NewUserRepository(database)

	created := newTestUser(t, database, domain.RoleUser)
	if created.ID == "" || created.CreatedAt.IsZero() {
		t.Fatalf("Create() não preencheu ID e CreatedAt: %+v", created)
	}

	found, err := userRepository.FindByEmail(created.Email)
	if err != nil {
		t.Fatalf("FindByEmail() error = %v", err)
	}
	if found.ID != created.ID || found.Password != "hash" || found.Role != domain.RoleUser {
		t.Errorf("FindByEmail() = %+v, want o usuário criado", found)
	}

	_, err = userRepository.FindByEmail(uniqueEmail("ninguem"))
	if !errors.Is(err, domain.ErrUserNotFound) {
		t.Errorf("FindByEmail() de e-mail inexistente: error = %v, want ErrUserNotFound", err)
	}
}

func TestUserCreateDuplicateEmail(t *testing.T) {
	database := openTestDB(t)
	existing := newTestUser(t, database, domain.RoleUser)

	duplicate := &domain.User{Name: "Outro", Email: existing.Email, Password: "hash", Role: domain.RoleUser}
	err := postgres.NewUserRepository(database).Create(duplicate)
	if !errors.Is(err, domain.ErrEmailAlreadyExists) {
		t.Errorf("Create() error = %v, want ErrEmailAlreadyExists", err)
	}
}

func TestUserCreateIfEmailMissing(t *testing.T) {
	database := openTestDB(t)
	userRepository := postgres.NewUserRepository(database)

	admin := &domain.User{Name: "Admin", Email: uniqueEmail("seed"), Password: "hash", Role: domain.RoleAdmin}
	t.Cleanup(func() { _, _ = database.Exec(`DELETE FROM users WHERE email = $1`, admin.Email) })

	created, err := userRepository.CreateIfEmailMissing(admin)
	if err != nil || !created {
		t.Fatalf("primeira chamada: created = %v, error = %v, want true, nil", created, err)
	}

	again := &domain.User{Name: "Outro nome", Email: admin.Email, Password: "outro-hash", Role: domain.RoleUser}
	created, err = userRepository.CreateIfEmailMissing(again)
	if err != nil || created {
		t.Fatalf("segunda chamada: created = %v, error = %v, want false, nil", created, err)
	}

	found, err := userRepository.FindByEmail(admin.Email)
	if err != nil {
		t.Fatalf("FindByEmail() error = %v", err)
	}
	if found.Role != domain.RoleAdmin || found.Password != "hash" {
		t.Errorf("o registro existente foi alterado: %+v", found)
	}
}

func TestUserUpdateKeepsRole(t *testing.T) {
	database := openTestDB(t)
	userRepository := postgres.NewUserRepository(database)
	created := newTestUser(t, database, domain.RoleAdmin)

	// Role fica vazio de propósito: o Update não pode tocar nele.
	update := &domain.User{ID: created.ID, Name: "Novo nome", Email: created.Email, Password: "novo-hash"}
	if err := userRepository.Update(update); err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if update.Role != domain.RoleAdmin {
		t.Errorf("Update() devolveu role = %q, want %q", update.Role, domain.RoleAdmin)
	}

	found, err := userRepository.FindByEmail(created.Email)
	if err != nil {
		t.Fatalf("FindByEmail() error = %v", err)
	}
	if found.Role != domain.RoleAdmin || found.Name != "Novo nome" || found.Password != "novo-hash" {
		t.Errorf("registro após Update = %+v", found)
	}
}

func TestUserUpdateErrors(t *testing.T) {
	database := openTestDB(t)
	userRepository := postgres.NewUserRepository(database)

	missing := &domain.User{ID: "00000000-0000-0000-0000-000000000000", Name: "X", Email: uniqueEmail("x"), Password: "h"}
	if err := userRepository.Update(missing); !errors.Is(err, domain.ErrUserNotFound) {
		t.Errorf("Update() de id inexistente: error = %v, want ErrUserNotFound", err)
	}

	invalid := &domain.User{ID: "nao-e-uuid", Name: "X", Email: uniqueEmail("x"), Password: "h"}
	if err := userRepository.Update(invalid); !errors.Is(err, domain.ErrInvalidUserID) {
		t.Errorf("Update() de id inválido: error = %v, want ErrInvalidUserID", err)
	}

	first := newTestUser(t, database, domain.RoleUser)
	second := newTestUser(t, database, domain.RoleUser)
	clash := &domain.User{ID: second.ID, Name: "X", Email: first.Email, Password: "h"}
	if err := userRepository.Update(clash); !errors.Is(err, domain.ErrEmailAlreadyExists) {
		t.Errorf("Update() com e-mail de outro: error = %v, want ErrEmailAlreadyExists", err)
	}
}

func TestUserPatchChangesOnlySentFields(t *testing.T) {
	database := openTestDB(t)
	userRepository := postgres.NewUserRepository(database)
	created := newTestUser(t, database, domain.RoleUser)

	newName := "Só o nome"
	patched, err := userRepository.Patch(created.ID, domain.UserPatch{Name: &newName})
	if err != nil {
		t.Fatalf("Patch() error = %v", err)
	}
	if patched.Name != newName || patched.Email != created.Email || patched.Role != domain.RoleUser {
		t.Errorf("Patch() = %+v", patched)
	}

	found, err := userRepository.FindByEmail(created.Email)
	if err != nil {
		t.Fatalf("FindByEmail() error = %v", err)
	}
	if found.Password != "hash" {
		t.Errorf("a senha mudou, mas o Patch não a enviou: %q", found.Password)
	}

	_, err = userRepository.Patch("00000000-0000-0000-0000-000000000000", domain.UserPatch{Name: &newName})
	if !errors.Is(err, domain.ErrUserNotFound) {
		t.Errorf("Patch() de id inexistente: error = %v, want ErrUserNotFound", err)
	}
}

func TestUserUpdateRole(t *testing.T) {
	database := openTestDB(t)
	userRepository := postgres.NewUserRepository(database)
	created := newTestUser(t, database, domain.RoleUser)

	promoted, err := userRepository.UpdateRole(created.ID, domain.RoleAdmin)
	if err != nil {
		t.Fatalf("UpdateRole() error = %v", err)
	}
	if promoted.Role != domain.RoleAdmin || promoted.Email != created.Email {
		t.Errorf("UpdateRole() = %+v", promoted)
	}

	_, err = userRepository.UpdateRole("00000000-0000-0000-0000-000000000000", domain.RoleAdmin)
	if !errors.Is(err, domain.ErrUserNotFound) {
		t.Errorf("UpdateRole() de id inexistente: error = %v, want ErrUserNotFound", err)
	}

	_, err = userRepository.UpdateRole("nao-e-uuid", domain.RoleAdmin)
	if !errors.Is(err, domain.ErrInvalidUserID) {
		t.Errorf("UpdateRole() de id inválido: error = %v, want ErrInvalidUserID", err)
	}
}

func TestUserListAndDelete(t *testing.T) {
	database := openTestDB(t)
	userRepository := postgres.NewUserRepository(database)
	created := newTestUser(t, database, domain.RoleUser)

	users, err := userRepository.List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if users == nil {
		t.Error("List() devolveu nil; o esperado é uma lista (possivelmente vazia)")
	}

	listed := false
	for _, user := range users {
		if user.ID == created.ID {
			listed = true
		}
	}
	if !listed {
		t.Error("List() não devolveu o usuário criado")
	}

	deleted, err := userRepository.Delete(created.ID)
	if err != nil || deleted.ID != created.ID {
		t.Fatalf("Delete() = %+v, %v", deleted, err)
	}
	if _, err := userRepository.Delete(created.ID); !errors.Is(err, domain.ErrUserNotFound) {
		t.Errorf("Delete() repetido: error = %v, want ErrUserNotFound", err)
	}
	if _, err := userRepository.Delete("nao-e-uuid"); !errors.Is(err, domain.ErrInvalidUserID) {
		t.Errorf("Delete() de id inválido: error = %v, want ErrInvalidUserID", err)
	}
}

func TestBookLifecycle(t *testing.T) {
	database := openTestDB(t)
	bookRepository := postgres.NewBookRepository(database)

	description := "Clássico"
	book := &domain.Book{Title: "A Linguagem Go", Author: "Donovan", Description: &description}
	if err := bookRepository.Create(book); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	t.Cleanup(func() { _, _ = database.Exec(`DELETE FROM books WHERE id = $1`, book.ID) })

	if book.ID == "" || book.CreatedAt.IsZero() {
		t.Fatalf("Create() não preencheu ID e CreatedAt: %+v", book)
	}

	book.Title = "Go em Ação"
	if err := bookRepository.Update(book); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	books, err := bookRepository.List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	var found *domain.Book
	for i := range books {
		if books[i].ID == book.ID {
			found = &books[i]
		}
	}
	if found == nil || found.Title != "Go em Ação" {
		t.Fatalf("List() não devolveu o livro atualizado: %+v", found)
	}

	deleted, err := bookRepository.Delete(book.ID)
	if err != nil || deleted.ID != book.ID {
		t.Fatalf("Delete() = %+v, %v", deleted, err)
	}
	if _, err := bookRepository.Delete(book.ID); !errors.Is(err, domain.ErrBookNotFound) {
		t.Errorf("Delete() repetido: error = %v, want ErrBookNotFound", err)
	}
	if _, err := bookRepository.Delete("nao-e-uuid"); !errors.Is(err, domain.ErrInvalidBookID) {
		t.Errorf("Delete() de id inválido: error = %v, want ErrInvalidBookID", err)
	}
	if err := bookRepository.Update(&domain.Book{ID: "nao-e-uuid", Title: "x", Author: "y"}); !errors.Is(err, domain.ErrInvalidBookID) {
		t.Errorf("Update() de id inválido: error = %v, want ErrInvalidBookID", err)
	}
}
