// Package e2e testa a API de ponta a ponta: sobe as rotas reais (a mesma montagem do cmd/api)
// sobre um Postgres de verdade e conversa com elas por HTTP.
package e2e_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"go-lang/internal/bootstrap"
	"go-lang/internal/testutil"
)

const (
	testSecret   = "0123456789012345678901234567890123456789"
	adminPass    = "Admin-senha-1"
	userPass     = "Senha-longa-1"
	newUserPass  = "Nova-senha-2"
	bcryptPrefix = "$2a$"
)

// client envia requisições JSON ao servidor de teste.
type client struct {
	t       *testing.T
	baseURL string
}

// do envia a requisição e devolve o status e o corpo JSON já decodificado (objeto, lista ou nil).
func (c *client) do(method, path, token string, body any) (int, any) {
	c.t.Helper()

	var payload bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&payload).Encode(body); err != nil {
			c.t.Fatalf("encode body: %v", err)
		}
	}

	request, err := http.NewRequest(method, c.baseURL+path, &payload)
	if err != nil {
		c.t.Fatalf("new request: %v", err)
	}
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		c.t.Fatalf("%s %s: %v", method, path, err)
	}
	defer func() { _ = response.Body.Close() }()

	var decoded any
	_ = json.NewDecoder(response.Body).Decode(&decoded)

	return response.StatusCode, decoded
}

// expect confere o status e devolve o corpo como objeto (vazio se não for objeto).
func (c *client) expect(want int, method, path, token string, body any) map[string]any {
	c.t.Helper()

	status, payload := c.do(method, path, token, body)
	if status != want {
		c.t.Fatalf("%s %s = %d, want %d (corpo: %v)", method, path, status, want, payload)
	}

	object, _ := payload.(map[string]any)
	return object
}

func TestAPIEndToEnd(t *testing.T) {
	database := testutil.OpenTestDB(t)

	// Tudo que o teste cria usa este prefixo, para a limpeza apagar só o que é dele.
	prefix := fmt.Sprintf("e2e-%d", time.Now().UnixNano())
	adminEmail := prefix + "-admin@test.example.com"
	userEmail := prefix + "-user@test.example.com"

	// A limpeza é por prefixo, e não por id, para também apagar o que um passo que falhou deixou para trás.
	t.Cleanup(func() {
		_, _ = database.Exec(`DELETE FROM users WHERE email LIKE $1`, prefix+"%")
		_, _ = database.Exec(`DELETE FROM books WHERE title LIKE $1`, prefix+"%")
	})

	app := bootstrap.New(database, []byte(testSecret), time.Hour)
	server := httptest.NewServer(app.Handler)
	t.Cleanup(server.Close)

	// O admin nasce pelo mesmo caminho do seed do servidor real.
	if created, err := app.Users.EnsureAdmin(adminEmail, adminPass); err != nil || !created {
		t.Fatalf("EnsureAdmin() = %v, %v, want true, nil", created, err)
	}

	var adminToken, userToken, userID string

	t.Run("rotas protegidas exigem token", func(t *testing.T) {
		c := &client{t: t, baseURL: server.URL}
		c.expect(http.StatusUnauthorized, "GET", "/books", "", nil)
		c.expect(http.StatusUnauthorized, "GET", "/users", "", nil)
	})

	t.Run("login", func(t *testing.T) {
		c := &client{t: t, baseURL: server.URL}
		c.expect(http.StatusUnauthorized, "POST", "/login", "", map[string]string{"email": adminEmail, "password": "errada"})
		c.expect(http.StatusUnauthorized, "POST", "/login", "", map[string]string{"email": "ninguem@test.example.com", "password": adminPass})

		body := c.expect(http.StatusOK, "POST", "/login", "", map[string]string{"email": adminEmail, "password": adminPass})
		adminToken, _ = body["token"].(string)
		if adminToken == "" {
			t.Fatal("o login do admin não devolveu token")
		}
	})

	t.Run("cadastro público", func(t *testing.T) {
		c := &client{t: t, baseURL: server.URL}
		c.expect(http.StatusBadRequest, "POST", "/users", "", map[string]string{"name": "E2E", "email": userEmail, "password": "fraca"})
		c.expect(http.StatusBadRequest, "POST", "/users", "", map[string]string{"name": "E2E", "email": "Nome <x@y.com>", "password": userPass})
		c.expect(http.StatusBadRequest, "POST", "/users", "", map[string]string{"name": strings.Repeat("a", 101), "email": userEmail, "password": userPass})

		// O cliente tenta se cadastrar como admin: o servidor ignora o papel enviado.
		body := c.expect(http.StatusCreated, "POST", "/users", "",
			map[string]string{"name": "E2E", "email": userEmail, "password": userPass, "role": "admin"})
		if body["role"] != "user" {
			t.Errorf("role = %v, want user", body["role"])
		}
		if _, leaked := body["password"]; leaked {
			t.Error("a resposta do cadastro não pode conter password")
		}
		userID, _ = body["id"].(string)

		c.expect(http.StatusConflict, "POST", "/users", "", map[string]string{"name": "E2E", "email": userEmail, "password": userPass})

		// A senha é gravada como hash bcrypt e o papel no banco é user.
		var storedPassword, storedRole string
		err := database.QueryRow(`SELECT password, role FROM users WHERE email = $1`, userEmail).Scan(&storedPassword, &storedRole)
		if err != nil {
			t.Fatalf("select user: %v", err)
		}
		if !strings.HasPrefix(storedPassword, bcryptPrefix) || storedRole != "user" {
			t.Errorf("no banco: password começa com bcrypt = %v, role = %q", strings.HasPrefix(storedPassword, bcryptPrefix), storedRole)
		}
	})

	t.Run("usuário comum", func(t *testing.T) {
		c := &client{t: t, baseURL: server.URL}
		body := c.expect(http.StatusOK, "POST", "/login", "", map[string]string{"email": userEmail, "password": userPass})
		userToken, _ = body["token"].(string)

		c.expect(http.StatusOK, "GET", "/books", userToken, nil)

		status, payload := c.do("GET", "/users", userToken, nil)
		users, _ := payload.([]any)
		if status != http.StatusOK || len(users) == 0 {
			t.Fatalf("GET /users = %d, %v", status, payload)
		}
		for _, item := range users {
			if _, leaked := item.(map[string]any)["password"]; leaked {
				t.Fatal("a listagem não pode conter password")
			}
		}

		c.expect(http.StatusForbidden, "POST", "/books", userToken, map[string]string{"title": prefix + " negado", "author": "A"})
		c.expect(http.StatusForbidden, "PUT", "/users/"+userID, userToken,
			map[string]string{"name": "x", "email": userEmail, "password": userPass, "role": "admin"})
		c.expect(http.StatusForbidden, "PUT", "/users/"+userID+"/role", userToken, map[string]string{"role": "admin"})
	})

	t.Run("admin gerencia livros", func(t *testing.T) {
		c := &client{t: t, baseURL: server.URL}
		title := prefix + " livro"

		created := c.expect(http.StatusCreated, "POST", "/books", adminToken, map[string]string{"title": title, "author": "Autora"})
		id, _ := created["id"].(string)

		updated := c.expect(http.StatusOK, "PUT", "/books/"+id, adminToken, map[string]string{"title": title + " editado", "author": "Autora"})
		if updated["title"] != title+" editado" {
			t.Errorf("title = %v", updated["title"])
		}

		c.expect(http.StatusBadRequest, "PUT", "/books/nao-uuid", adminToken, map[string]string{"title": "t", "author": "a"})
		c.expect(http.StatusBadRequest, "POST", "/books", adminToken, map[string]string{"title": strings.Repeat("a", 256), "author": "Autora"})
		c.expect(http.StatusOK, "DELETE", "/books/"+id, adminToken, nil)
		c.expect(http.StatusNotFound, "DELETE", "/books/"+id, adminToken, nil)
	})

	t.Run("admin gerencia usuários", func(t *testing.T) {
		c := &client{t: t, baseURL: server.URL}
		path := "/users/" + userID

		renamed := c.expect(http.StatusOK, "PATCH", path, adminToken, map[string]string{"name": "E2E Renomeado"})
		if renamed["name"] != "E2E Renomeado" {
			t.Errorf("name = %v", renamed["name"])
		}
		// PATCH sem senha não troca a senha: a antiga continua valendo.
		c.expect(http.StatusOK, "POST", "/login", "", map[string]string{"email": userEmail, "password": userPass})

		c.expect(http.StatusBadRequest, "PATCH", path, adminToken, map[string]string{"email": "invalido"})
		c.expect(http.StatusBadRequest, "PATCH", path, adminToken, map[string]string{})

		c.expect(http.StatusOK, "PATCH", path, adminToken, map[string]string{"password": newUserPass})
		c.expect(http.StatusUnauthorized, "POST", "/login", "", map[string]string{"email": userEmail, "password": userPass})

		// O PUT completo não mexe no papel.
		replaced := c.expect(http.StatusOK, "PUT", path, adminToken,
			map[string]string{"name": "E2E PUT", "email": userEmail, "password": newUserPass, "role": "admin"})
		if replaced["role"] != "user" {
			t.Errorf("role após PUT = %v, want user", replaced["role"])
		}
	})

	t.Run("promoção de papel", func(t *testing.T) {
		c := &client{t: t, baseURL: server.URL}

		promoted := c.expect(http.StatusOK, "PUT", "/users/"+userID+"/role", adminToken, map[string]string{"role": "admin"})
		if promoted["role"] != "admin" {
			t.Errorf("role = %v, want admin", promoted["role"])
		}
		c.expect(http.StatusBadRequest, "PUT", "/users/"+userID+"/role", adminToken, map[string]string{"role": "root"})

		// O novo login já traz o papel de admin.
		body := c.expect(http.StatusOK, "POST", "/login", "", map[string]string{"email": userEmail, "password": newUserPass})
		promotedToken, _ := body["token"].(string)
		c.expect(http.StatusCreated, "POST", "/books", promotedToken, map[string]string{"title": prefix + " promovido", "author": "A"})

		// Ninguém muda o próprio papel, nem o admin.
		var adminID string
		if err := database.QueryRow(`SELECT id FROM users WHERE email = $1`, adminEmail).Scan(&adminID); err != nil {
			t.Fatalf("select admin: %v", err)
		}
		c.expect(http.StatusForbidden, "PUT", "/users/"+adminID+"/role", adminToken, map[string]string{"role": "user"})
		c.expect(http.StatusBadRequest, "PUT", "/users/nao-uuid/role", adminToken, map[string]string{"role": "user"})
	})

	t.Run("admin apaga o usuário", func(t *testing.T) {
		c := &client{t: t, baseURL: server.URL}
		c.expect(http.StatusOK, "DELETE", "/users/"+userID, adminToken, nil)
		c.expect(http.StatusNotFound, "DELETE", "/users/"+userID, adminToken, nil)
	})
}
