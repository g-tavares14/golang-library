package rest_test

import (
	"net/http"
	"strings"
	"testing"
)

// maxBodyBytes espelha o limite de rest.maxBodyBytes (1 MiB), que não é exportado.
const maxBodyBytes = 1 << 20

// bodyWithSize monta um JSON válido com exatamente size bytes. O campo "padding" é ignorado pelos
// handlers, então o tamanho do corpo muda sem tocar nos campos validados (título, escritor, nome...).
func bodyWithSize(t *testing.T, size int) string {
	t.Helper()

	const prefix = `{"title":"Go","author":"Pike","padding":"`
	const suffix = `"}`

	padding := size - len(prefix) - len(suffix)
	if padding < 0 {
		t.Fatalf("size %d é menor que o corpo mínimo", size)
	}

	body := prefix + strings.Repeat("a", padding) + suffix
	if len(body) != size {
		t.Fatalf("len(body) = %d, want %d", len(body), size)
	}
	return body
}

func TestBodyAtTheLimitIsAccepted(t *testing.T) {
	app := newTestApp(t)

	recorder := app.do("POST", "/books", app.adminToken, bodyWithSize(t, maxBodyBytes))
	assertStatus(t, recorder, http.StatusCreated)
}

func TestBodyOneByteOverTheLimitIsRejected(t *testing.T) {
	app := newTestApp(t)

	recorder := app.do("POST", "/books", app.adminToken, bodyWithSize(t, maxBodyBytes+1))
	assertStatus(t, recorder, http.StatusRequestEntityTooLarge)

	if got := decode(t, recorder)["error"]; got != "request body too large" {
		t.Errorf("error = %v, want %q", got, "request body too large")
	}
}

// Todas as rotas que leem JSON passam por decodeJSON, então todas devem aplicar o limite.
func TestEveryJSONRouteRejectsOversizedBody(t *testing.T) {
	routes := []struct {
		name   string
		method string
		path   func(app *testApp) string
		token  func(app *testApp) string
	}{
		{"POST /login", "POST", func(*testApp) string { return "/login" }, func(*testApp) string { return "" }},
		{"POST /users", "POST", func(*testApp) string { return "/users" }, func(*testApp) string { return "" }},
		{"PUT /users/{id}", "PUT", func(app *testApp) string { return "/users/" + app.memberID },
			func(app *testApp) string { return app.adminToken }},
		{"PATCH /users/{id}", "PATCH", func(app *testApp) string { return "/users/" + app.memberID },
			func(app *testApp) string { return app.adminToken }},
		{"PUT /users/{id}/role", "PUT", func(app *testApp) string { return "/users/" + app.memberID + "/role" },
			func(app *testApp) string { return app.adminToken }},
		{"POST /books", "POST", func(*testApp) string { return "/books" }, func(app *testApp) string { return app.adminToken }},
		{"PUT /books/{id}", "PUT", func(*testApp) string { return "/books/1" }, func(app *testApp) string { return app.adminToken }},
	}

	for _, route := range routes {
		t.Run(route.name, func(t *testing.T) {
			app := newTestApp(t)

			recorder := app.do(route.method, route.path(app), route.token(app), bodyWithSize(t, 2*maxBodyBytes))
			assertStatus(t, recorder, http.StatusRequestEntityTooLarge)
		})
	}
}

func TestMalformedJSONIsABadRequest(t *testing.T) {
	app := newTestApp(t)

	recorder := app.do("POST", "/books", app.adminToken, `{`)
	assertStatus(t, recorder, http.StatusBadRequest)

	if got := decode(t, recorder)["error"]; got != "invalid request body" {
		t.Errorf("error = %v, want %q", got, "invalid request body")
	}
}
