package handlers_test

import (
	"net/http"
	"strings"
	"testing"
)

func TestCreateUserRejectsInvalidEmail(t *testing.T) {
	tests := []struct {
		name  string
		email string
	}{
		{"sem arroba", "ana.example.com"},
		{"sem domínio", "ana@"},
		{"sem usuário", "@example.com"},
		{"com nome de exibição", "Ana <ana@example.com>"},
		{"com espaço no fim", "ana@example.com "},
		{"acima de 100 caracteres", strings.Repeat("a", 95) + "@example.com"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			app := newTestApp(t)

			body := `{"name":"Ana","email":"` + test.email + `","password":"Senha-longa-1"}`
			assertStatus(t, app.do("POST", "/users", "", body), http.StatusBadRequest)

			if _, err := app.users.FindByEmail(test.email); err == nil {
				t.Error("o usuário com e-mail inválido foi gravado")
			}
		})
	}
}

func TestUpdateAndPatchUserRejectInvalidEmail(t *testing.T) {
	app := newTestApp(t)
	path := "/users/" + app.memberID

	put := app.do("PUT", path, app.adminToken, `{"name":"Maria","email":"nao-e-email","password":"Senha-longa-1"}`)
	assertStatus(t, put, http.StatusBadRequest)

	patch := app.do("PATCH", path, app.adminToken, `{"email":"nao-e-email"}`)
	assertStatus(t, patch, http.StatusBadRequest)

	if app.users.users[app.memberID].Email != "maria@example.com" {
		t.Error("o e-mail foi alterado apesar de inválido")
	}
}

func TestCreateUserRejectsWeakPassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
		want     int
	}{
		{"curta demais", "Ab1", http.StatusBadRequest},
		{"sem maiúscula", "senha-longa-1", http.StatusBadRequest},
		{"sem minúscula", "SENHA-LONGA-1", http.StatusBadRequest},
		{"sem número", "Senha-longa-x", http.StatusBadRequest},
		{"no limite de 8 caracteres", "Senha-a1", http.StatusCreated},
		{"aceitável", "Senha-longa-1", http.StatusCreated},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			app := newTestApp(t)

			body := `{"name":"Ana","email":"ana@example.com","password":"` + test.password + `"}`
			assertStatus(t, app.do("POST", "/users", "", body), test.want)
		})
	}
}

func TestUpdateAndPatchUserRejectWeakPassword(t *testing.T) {
	app := newTestApp(t)
	path := "/users/" + app.memberID
	before := app.users.users[app.memberID].Password

	put := app.do("PUT", path, app.adminToken, `{"name":"Maria","email":"maria@example.com","password":"fraca"}`)
	assertStatus(t, put, http.StatusBadRequest)

	patch := app.do("PATCH", path, app.adminToken, `{"password":"fraca"}`)
	assertStatus(t, patch, http.StatusBadRequest)

	if app.users.users[app.memberID].Password != before {
		t.Error("a senha foi alterada apesar de fraca")
	}
}
