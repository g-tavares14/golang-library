package handlers_test

import (
	"net/http"
	"strings"
	"testing"

	"go-lang/auth"
	"go-lang/models"
)

func TestCreateUserForcesRoleUser(t *testing.T) {
	app := newTestApp(t)

	// Mass assignment: o cliente tenta se cadastrar como admin.
	recorder := app.do("POST", "/users", "",
		`{"name":"Ana","email":"ana@example.com","password":"senha-longa-1","role":"admin"}`)
	assertStatus(t, recorder, http.StatusCreated)

	body := decode(t, recorder)
	if body["role"] != models.RoleUser {
		t.Errorf("role na resposta = %v, want %q", body["role"], models.RoleUser)
	}

	created, err := app.users.FindByEmail("ana@example.com")
	if err != nil {
		t.Fatalf("usuário não foi gravado: %v", err)
	}
	if created.Role != models.RoleUser {
		t.Errorf("role gravada = %q, want %q", created.Role, models.RoleUser)
	}
}

func TestCreateUserStoresHashAndHidesPassword(t *testing.T) {
	app := newTestApp(t)

	recorder := app.do("POST", "/users", "",
		`{"name":"Ana","email":"ana@example.com","password":"senha-longa-1"}`)
	assertStatus(t, recorder, http.StatusCreated)

	if _, leaked := decode(t, recorder)["password"]; leaked {
		t.Error("a resposta não pode conter password")
	}
	if strings.Contains(recorder.Body.String(), "senha-longa-1") {
		t.Error("a resposta contém a senha em texto puro")
	}

	created, err := app.users.FindByEmail("ana@example.com")
	if err != nil {
		t.Fatalf("usuário não foi gravado: %v", err)
	}
	if created.Password == "senha-longa-1" {
		t.Fatal("a senha foi gravada em texto puro")
	}
	if !auth.CheckPassword(created.Password, "senha-longa-1") {
		t.Error("o hash gravado não corresponde à senha enviada")
	}
}

func TestCreateUserErrors(t *testing.T) {
	tests := []struct {
		name string
		body string
		want int
	}{
		{"sem nome", `{"email":"ana@example.com","password":"senha-longa-1"}`, http.StatusBadRequest},
		{"sem e-mail", `{"name":"Ana","password":"senha-longa-1"}`, http.StatusBadRequest},
		{"sem senha", `{"name":"Ana","email":"ana@example.com"}`, http.StatusBadRequest},
		{"JSON inválido", `{`, http.StatusBadRequest},
		{"senha acima de 72 bytes", `{"name":"Ana","email":"ana@example.com","password":"` +
			strings.Repeat("a", 73) + `"}`, http.StatusBadRequest},
		{"e-mail repetido", `{"name":"Outra","email":"maria@example.com","password":"senha-longa-1"}`, http.StatusConflict},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			app := newTestApp(t)
			assertStatus(t, app.do("POST", "/users", "", test.body), test.want)
		})
	}
}

func TestListUsersHidesPassword(t *testing.T) {
	app := newTestApp(t)

	recorder := app.do("GET", "/users", app.memberToken, "")
	assertStatus(t, recorder, http.StatusOK)

	users := decodeList(t, recorder)
	if len(users) != 2 {
		t.Fatalf("len(users) = %d, want 2", len(users))
	}
	for _, item := range users {
		if _, leaked := item.(map[string]any)["password"]; leaked {
			t.Error("a listagem não pode conter password")
		}
	}
}

func TestListUsersEmptyIsArray(t *testing.T) {
	app := newTestApp(t)
	app.users.users = nil // sem nenhum usuário no store

	recorder := app.do("GET", "/users", app.adminToken, "")
	assertStatus(t, recorder, http.StatusOK)

	if got := strings.TrimSpace(recorder.Body.String()); got != "[]" {
		t.Errorf("corpo = %s, want []", got)
	}
}

func TestUpdateUserKeepsRole(t *testing.T) {
	app := newTestApp(t)

	// Mesmo que o corpo traga "role", o PUT não pode mexer no papel.
	recorder := app.do("PUT", "/users/"+app.memberID, app.adminToken,
		`{"name":"Maria Silva","email":"maria@example.com","password":"nova-senha-1","role":"admin"}`)
	assertStatus(t, recorder, http.StatusOK)

	body := decode(t, recorder)
	if body["role"] != models.RoleUser {
		t.Errorf("role na resposta = %v, want %q", body["role"], models.RoleUser)
	}
	if _, leaked := body["password"]; leaked {
		t.Error("a resposta não pode conter password")
	}

	stored := app.users.users[app.memberID]
	if stored.Role != models.RoleUser {
		t.Errorf("role gravada = %q, want %q", stored.Role, models.RoleUser)
	}
	if !auth.CheckPassword(stored.Password, "nova-senha-1") {
		t.Error("a nova senha não foi gravada como hash")
	}
}

func TestPatchUser(t *testing.T) {
	t.Run("troca só o nome e mantém a senha", func(t *testing.T) {
		app := newTestApp(t)
		before := app.users.users[app.memberID].Password

		recorder := app.do("PATCH", "/users/"+app.memberID, app.adminToken, `{"name":"Maria Silva"}`)
		assertStatus(t, recorder, http.StatusOK)

		if got := decode(t, recorder)["name"]; got != "Maria Silva" {
			t.Errorf("name = %v, want Maria Silva", got)
		}

		stored := app.users.users[app.memberID]
		if stored.Password != before {
			t.Error("a senha mudou, mas o PATCH não a enviou")
		}
		if stored.Email != "maria@example.com" {
			t.Errorf("email mudou para %q", stored.Email)
		}
	})

	t.Run("senha enviada é gravada como hash", func(t *testing.T) {
		app := newTestApp(t)

		recorder := app.do("PATCH", "/users/"+app.memberID, app.adminToken, `{"password":"outra-senha-1"}`)
		assertStatus(t, recorder, http.StatusOK)

		stored := app.users.users[app.memberID]
		if stored.Password == "outra-senha-1" || !auth.CheckPassword(stored.Password, "outra-senha-1") {
			t.Error("a senha não foi gravada como hash bcrypt")
		}
	})

	tests := []struct {
		name string
		path string
		body string
		want int
	}{
		{"corpo vazio", "/users/1", `{}`, http.StatusBadRequest},
		{"nome vazio", "/users/1", `{"name":""}`, http.StatusBadRequest},
		{"senha vazia", "/users/1", `{"password":""}`, http.StatusBadRequest},
		{"JSON inválido", "/users/1", `{`, http.StatusBadRequest},
		{"e-mail de outro usuário", "/users/2", `{"email":"admin@example.com"}`, http.StatusConflict},
		{"usuário inexistente", "/users/999", `{"name":"X"}`, http.StatusNotFound},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			app := newTestApp(t)
			assertStatus(t, app.do("PATCH", test.path, app.adminToken, test.body), test.want)
		})
	}
}

func TestUpdateUserRole(t *testing.T) {
	t.Run("admin promove um usuário", func(t *testing.T) {
		app := newTestApp(t)

		recorder := app.do("PUT", "/users/"+app.memberID+"/role", app.adminToken, `{"role":"admin"}`)
		assertStatus(t, recorder, http.StatusOK)

		if got := decode(t, recorder)["role"]; got != models.RoleAdmin {
			t.Errorf("role = %v, want %q", got, models.RoleAdmin)
		}
		if app.users.users[app.memberID].Role != models.RoleAdmin {
			t.Error("o papel não foi gravado")
		}
	})

	t.Run("admin não muda o próprio papel", func(t *testing.T) {
		app := newTestApp(t)

		recorder := app.do("PUT", "/users/"+app.adminID+"/role", app.adminToken, `{"role":"user"}`)
		assertStatus(t, recorder, http.StatusForbidden)

		if app.users.users[app.adminID].Role != models.RoleAdmin {
			t.Error("o admin foi rebaixado")
		}
	})

	tests := []struct {
		name string
		path string
		body string
		want int
	}{
		{"papel inválido", "/users/2/role", `{"role":"root"}`, http.StatusBadRequest},
		{"papel ausente", "/users/2/role", `{}`, http.StatusBadRequest},
		{"JSON inválido", "/users/2/role", `{`, http.StatusBadRequest},
		{"usuário inexistente", "/users/999/role", `{"role":"admin"}`, http.StatusNotFound},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			app := newTestApp(t)
			assertStatus(t, app.do("PUT", test.path, app.adminToken, test.body), test.want)
		})
	}
}

func TestDeleteUser(t *testing.T) {
	app := newTestApp(t)

	recorder := app.do("DELETE", "/users/"+app.memberID, app.adminToken, "")
	assertStatus(t, recorder, http.StatusOK)

	if _, leaked := decode(t, recorder)["password"]; leaked {
		t.Error("a resposta não pode conter password")
	}
	if _, exists := app.users.users[app.memberID]; exists {
		t.Error("o usuário não foi apagado")
	}

	assertStatus(t, app.do("DELETE", "/users/"+app.memberID, app.adminToken, ""), http.StatusNotFound)
}
