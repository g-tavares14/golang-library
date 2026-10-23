package repository_test

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	_ "github.com/lib/pq" // driver do PostgreSQL para o database/sql

	"go-lang/models"
	"go-lang/repository"
)

// openTestDB conecta ao Postgres indicado em TEST_DATABASE_URL, que precisa ter as migrations aplicadas.
// Sem a variável, o teste é pulado. Exemplo:
//
//	TEST_DATABASE_URL="host=localhost port=5432 user=postgres password=... dbname=golang sslmode=disable" go test ./repository/
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
func newTestUser(t *testing.T, database *sql.DB, role string) *models.User {
	t.Helper()

	user := &models.User{Name: "Teste", Email: uniqueEmail("user"), Password: "hash", Role: role}
	if err := repository.NewUserRepository(database).Create(user); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	t.Cleanup(func() { _, _ = database.Exec(`DELETE FROM users WHERE id = $1`, user.ID) })

	return user
}

func TestUserCreateAndFindByEmail(t *testing.T) {
	database := openTestDB(t)
	userRepository := repository.NewUserRepository(database)

	created := newTestUser(t, database, models.RoleUser)
	if created.ID == "" || created.CreatedAt.IsZero() {
		t.Fatalf("Create() não preencheu ID e CreatedAt: %+v", created)
	}

	found, err := userRepository.FindByEmail(created.Email)
	if err != nil {
		t.Fatalf("FindByEmail() error = %v", err)
	}
	if found.ID != created.ID || found.Password != "hash" || found.Role != models.RoleUser {
		t.Errorf("FindByEmail() = %+v, want o usuário criado", found)
	}

	_, err = userRepository.FindByEmail(uniqueEmail("ninguem"))
	if !errors.Is(err, repository.ErrUserNotFound) {
		t.Errorf("FindByEmail() de e-mail inexistente: error = %v, want ErrUserNotFound", err)
	}
}

func TestUserCreateDuplicateEmail(t *testing.T) {
	database := openTestDB(t)
	existing := newTestUser(t, database, models.RoleUser)

	duplicate := &models.User{Name: "Outro", Email: existing.Email, Password: "hash", Role: models.RoleUser}
	err := repository.NewUserRepository(database).Create(duplicate)
	if !errors.Is(err, repository.ErrEmailAlreadyExists) {
		t.Errorf("Create() error = %v, want ErrEmailAlreadyExists", err)
	}
}

func TestUserCreateIfEmailMissing(t *testing.T) {
	database := openTestDB(t)
	userRepository := repository.NewUserRepository(database)

	admin := &models.User{Name: "Admin", Email: uniqueEmail("seed"), Password: "hash", Role: models.RoleAdmin}
	t.Cleanup(func() { _, _ = database.Exec(`DELETE FROM users WHERE email = $1`, admin.Email) })

	created, err := userRepository.CreateIfEmailMissing(admin)
	if err != nil || !created {
		t.Fatalf("primeira chamada: created = %v, error = %v, want true, nil", created, err)
	}

	again := &models.User{Name: "Outro nome", Email: admin.Email, Password: "outro-hash", Role: models.RoleUser}
	created, err = userRepository.CreateIfEmailMissing(again)
	if err != nil || created {
		t.Fatalf("segunda chamada: created = %v, error = %v, want false, nil", created, err)
	}

	found, err := userRepository.FindByEmail(admin.Email)
	if err != nil {
		t.Fatalf("FindByEmail() error = %v", err)
	}
	if found.Role != models.RoleAdmin || found.Password != "hash" {
		t.Errorf("o registro existente foi alterado: %+v", found)
	}
}

func TestUserUpdateKeepsRole(t *testing.T) {
	database := openTestDB(t)
	userRepository := repository.NewUserRepository(database)
	created := newTestUser(t, database, models.RoleAdmin)

	// Role fica vazio de propósito: o Update não pode tocar nele.
	update := &models.User{ID: created.ID, Name: "Novo nome", Email: created.Email, Password: "novo-hash"}
	if err := userRepository.Update(update); err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if update.Role != models.RoleAdmin {
		t.Errorf("Update() devolveu role = %q, want %q", update.Role, models.RoleAdmin)
	}

	found, err := userRepository.FindByEmail(created.Email)
	if err != nil {
		t.Fatalf("FindByEmail() error = %v", err)
	}
	if found.Role != models.RoleAdmin || found.Name != "Novo nome" || found.Password != "novo-hash" {
		t.Errorf("registro após Update = %+v", found)
	}
}

func TestUserUpdateErrors(t *testing.T) {
	database := openTestDB(t)
	userRepository := repository.NewUserRepository(database)

	missing := &models.User{ID: "00000000-0000-0000-0000-000000000000", Name: "X", Email: uniqueEmail("x"), Password: "h"}
	if err := userRepository.Update(missing); !errors.Is(err, repository.ErrUserNotFound) {
		t.Errorf("Update() de id inexistente: error = %v, want ErrUserNotFound", err)
	}

	invalid := &models.User{ID: "nao-e-uuid", Name: "X", Email: uniqueEmail("x"), Password: "h"}
	if err := userRepository.Update(invalid); !errors.Is(err, repository.ErrInvalidUserID) {
		t.Errorf("Update() de id inválido: error = %v, want ErrInvalidUserID", err)
	}

	first := newTestUser(t, database, models.RoleUser)
	second := newTestUser(t, database, models.RoleUser)
	clash := &models.User{ID: second.ID, Name: "X", Email: first.Email, Password: "h"}
	if err := userRepository.Update(clash); !errors.Is(err, repository.ErrEmailAlreadyExists) {
		t.Errorf("Update() com e-mail de outro: error = %v, want ErrEmailAlreadyExists", err)
	}
}

func TestUserPatchChangesOnlySentFields(t *testing.T) {
	database := openTestDB(t)
	userRepository := repository.NewUserRepository(database)
	created := newTestUser(t, database, models.RoleUser)

	newName := "Só o nome"
	patched, err := userRepository.Patch(created.ID, models.UserPatch{Name: &newName})
	if err != nil {
		t.Fatalf("Patch() error = %v", err)
	}
	if patched.Name != newName || patched.Email != created.Email || patched.Role != models.RoleUser {
		t.Errorf("Patch() = %+v", patched)
	}

	found, err := userRepository.FindByEmail(created.Email)
	if err != nil {
		t.Fatalf("FindByEmail() error = %v", err)
	}
	if found.Password != "hash" {
		t.Errorf("a senha mudou, mas o Patch não a enviou: %q", found.Password)
	}

	_, err = userRepository.Patch("00000000-0000-0000-0000-000000000000", models.UserPatch{Name: &newName})
	if !errors.Is(err, repository.ErrUserNotFound) {
		t.Errorf("Patch() de id inexistente: error = %v, want ErrUserNotFound", err)
	}
}

func TestUserUpdateRole(t *testing.T) {
	database := openTestDB(t)
	userRepository := repository.NewUserRepository(database)
	created := newTestUser(t, database, models.RoleUser)

	promoted, err := userRepository.UpdateRole(created.ID, models.RoleAdmin)
	if err != nil {
		t.Fatalf("UpdateRole() error = %v", err)
	}
	if promoted.Role != models.RoleAdmin || promoted.Email != created.Email {
		t.Errorf("UpdateRole() = %+v", promoted)
	}

	_, err = userRepository.UpdateRole("00000000-0000-0000-0000-000000000000", models.RoleAdmin)
	if !errors.Is(err, repository.ErrUserNotFound) {
		t.Errorf("UpdateRole() de id inexistente: error = %v, want ErrUserNotFound", err)
	}

	_, err = userRepository.UpdateRole("nao-e-uuid", models.RoleAdmin)
	if !errors.Is(err, repository.ErrInvalidUserID) {
		t.Errorf("UpdateRole() de id inválido: error = %v, want ErrInvalidUserID", err)
	}
}

func TestUserListAndDelete(t *testing.T) {
	database := openTestDB(t)
	userRepository := repository.NewUserRepository(database)
	created := newTestUser(t, database, models.RoleUser)

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
	if _, err := userRepository.Delete(created.ID); !errors.Is(err, repository.ErrUserNotFound) {
		t.Errorf("Delete() repetido: error = %v, want ErrUserNotFound", err)
	}
	if _, err := userRepository.Delete("nao-e-uuid"); !errors.Is(err, repository.ErrInvalidUserID) {
		t.Errorf("Delete() de id inválido: error = %v, want ErrInvalidUserID", err)
	}
}

func TestBookLifecycle(t *testing.T) {
	database := openTestDB(t)
	bookRepository := repository.NewBookRepository(database)

	description := "Clássico"
	book := &models.Book{Title: "A Linguagem Go", Author: "Donovan", Description: &description}
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
	var found *models.Book
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
	if _, err := bookRepository.Delete(book.ID); !errors.Is(err, repository.ErrBookNotFound) {
		t.Errorf("Delete() repetido: error = %v, want ErrBookNotFound", err)
	}
	if _, err := bookRepository.Delete("nao-e-uuid"); !errors.Is(err, repository.ErrInvalidBookID) {
		t.Errorf("Delete() de id inválido: error = %v, want ErrInvalidBookID", err)
	}
	if err := bookRepository.Update(&models.Book{ID: "nao-e-uuid", Title: "x", Author: "y"}); !errors.Is(err, repository.ErrInvalidBookID) {
		t.Errorf("Update() de id inválido: error = %v, want ErrInvalidBookID", err)
	}
}
