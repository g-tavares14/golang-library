package handlers_test

import (
	"net/http"
	"strings"
	"testing"
)

func TestBookLifecycle(t *testing.T) {
	app := newTestApp(t)

	created := app.do("POST", "/books", app.adminToken,
		`{"title":"A Linguagem Go","author":"Donovan","description":"Clássico"}`)
	assertStatus(t, created, http.StatusCreated)

	id, _ := decode(t, created)["id"].(string)
	if id == "" {
		t.Fatal("o livro criado não tem id")
	}

	listed := app.do("GET", "/books", app.memberToken, "")
	assertStatus(t, listed, http.StatusOK)
	if got := len(decodeList(t, listed)); got != 1 {
		t.Fatalf("len(books) = %d, want 1", got)
	}

	updated := app.do("PUT", "/books/"+id, app.adminToken, `{"title":"Go em Ação","author":"Kennedy"}`)
	assertStatus(t, updated, http.StatusOK)
	if got := decode(t, updated)["title"]; got != "Go em Ação" {
		t.Errorf("title = %v, want Go em Ação", got)
	}

	deleted := app.do("DELETE", "/books/"+id, app.adminToken, "")
	assertStatus(t, deleted, http.StatusOK)
	assertStatus(t, app.do("DELETE", "/books/"+id, app.adminToken, ""), http.StatusNotFound)
}

func TestCreateBookValidation(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{"sem título", `{"author":"Donovan"}`},
		{"sem o campo author", `{"title":"A Linguagem Go"}`},
		{"JSON inválido", `{`},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			app := newTestApp(t)
			assertStatus(t, app.do("POST", "/books", app.adminToken, test.body), http.StatusBadRequest)
		})
	}
}

func TestListBooksEmptyIsArray(t *testing.T) {
	app := newTestApp(t)

	recorder := app.do("GET", "/books", app.memberToken, "")
	assertStatus(t, recorder, http.StatusOK)

	if got := strings.TrimSpace(recorder.Body.String()); got != "[]" {
		t.Errorf("corpo = %s, want []", got)
	}
}
