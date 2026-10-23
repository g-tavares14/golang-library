package rest_test

import (
	"net/http"
	"testing"
	"time"

	"go-lang/internal/infrastructure/security"
)

func TestRoutePermissions(t *testing.T) {
	// Status esperado de cada rota para: sem token, usuário comum e admin.
	// Para o admin usamos ids inexistentes de propósito: 404 prova que ele passou do guard.
	tests := []struct {
		name         string
		method, path string
		body         string
		anonymous    int
		member       int
		admin        int
	}{
		{"listar usuários", "GET", "/users", "", http.StatusUnauthorized, http.StatusOK, http.StatusOK},
		{"listar livros", "GET", "/books", "", http.StatusUnauthorized, http.StatusOK, http.StatusOK},
		{"criar livro", "POST", "/books", `{"title":"Go","author":"Pike"}`,
			http.StatusUnauthorized, http.StatusForbidden, http.StatusCreated},
		{"alterar livro", "PUT", "/books/999", `{"title":"Go","author":"Pike"}`,
			http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound},
		{"apagar livro", "DELETE", "/books/999", "", http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound},
		{"PUT usuário", "PUT", "/users/999", `{"name":"X","email":"x@example.com","password":"Senha-longa-1"}`,
			http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound},
		{"PATCH usuário", "PATCH", "/users/999", `{"name":"X"}`,
			http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound},
		{"apagar usuário", "DELETE", "/users/999", "", http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound},
		{"mudar papel", "PUT", "/users/999/role", `{"role":"admin"}`,
			http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			profiles := []struct {
				name  string
				token func(app *testApp) string
				want  int
			}{
				{"sem token", func(*testApp) string { return "" }, test.anonymous},
				{"usuário", func(app *testApp) string { return app.memberToken }, test.member},
				{"admin", func(app *testApp) string { return app.adminToken }, test.admin},
			}

			for _, profile := range profiles {
				app := newTestApp(t)

				recorder := app.do(test.method, test.path, profile.token(app), test.body)
				if recorder.Code != profile.want {
					t.Errorf("%s: status = %d, want %d\ncorpo: %s",
						profile.name, recorder.Code, profile.want, recorder.Body.String())
				}
			}
		})
	}
}

func TestGuardRejectsBadTokens(t *testing.T) {
	app := newTestApp(t)

	expired, err := security.NewTokenManager(testSecret, -time.Minute).Issue(app.adminID, "admin")
	if err != nil {
		t.Fatalf("Issue() error = %v", err)
	}

	tests := []struct {
		name   string
		header string
	}{
		{"cabeçalho vazio", ""},
		{"sem esquema", app.adminToken},
		{"esquema errado", "Basic " + app.adminToken},
		{"token lixo", "Bearer lixo"},
		{"token expirado", "Bearer " + expired},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request, _ := http.NewRequest(http.MethodGet, "/books", nil)
			if test.header != "" {
				request.Header.Set("Authorization", test.header)
			}

			recorder := httpRecord(app, request)
			assertStatus(t, recorder, http.StatusUnauthorized)

			if got := recorder.Header().Get("WWW-Authenticate"); got != "Bearer" {
				t.Errorf("WWW-Authenticate = %q, want %q", got, "Bearer")
			}
		})
	}
}

func TestGuardIgnoresRoleInBody(t *testing.T) {
	app := newTestApp(t)

	// O corpo diz "admin", mas o token é de usuário comum: quem manda é o token.
	recorder := app.do("PUT", "/users/"+app.memberID, app.memberToken,
		`{"name":"Maria","email":"maria@example.com","password":"Senha-longa-1","role":"admin"}`)
	assertStatus(t, recorder, http.StatusForbidden)
}
