package rest_test

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

func TestBookFieldLengthIsRejectedWith400(t *testing.T) {
	tooLong := strings.Repeat("a", 256)

	tests := []struct {
		name string
		body string
	}{
		{"título", `{"title":"` + tooLong + `","author":"Donovan"}`},
		{"escritor", `{"title":"A Linguagem Go","author":"` + tooLong + `"}`},
		{"descrição", `{"title":"A Linguagem Go","author":"Donovan","description":"` + tooLong + `"}`},
	}

	for _, test := range tests {
		t.Run("POST com "+test.name+" longo", func(t *testing.T) {
			app := newTestApp(t)

			recorder := app.do("POST", "/books", app.adminToken, test.body)
			assertStatus(t, recorder, http.StatusBadRequest)

			if got := decode(t, recorder)["error"]; got != "book field is too long" {
				t.Errorf("error = %v, want %q", got, "book field is too long")
			}
		})
	}

	t.Run("PUT com escritor longo", func(t *testing.T) {
		app := newTestApp(t)

		created := app.do("POST", "/books", app.adminToken, `{"title":"A Linguagem Go","author":"Donovan"}`)
		assertStatus(t, created, http.StatusCreated)
		id, _ := decode(t, created)["id"].(string)

		recorder := app.do("PUT", "/books/"+id, app.adminToken, `{"title":"A Linguagem Go","author":"`+tooLong+`"}`)
		assertStatus(t, recorder, http.StatusBadRequest)

		if got := decode(t, recorder)["error"]; got != "book field is too long" {
			t.Errorf("error = %v, want %q", got, "book field is too long")
		}
	})

	t.Run("campos no limite e acentuados são aceitos", func(t *testing.T) {
		app := newTestApp(t)

		accented := strings.Repeat("é", 255) // 510 bytes, 255 caracteres
		body := `{"title":"` + accented + `","author":"` + strings.Repeat("a", 255) + `","description":"` + accented + `"}`
		assertStatus(t, app.do("POST", "/books", app.adminToken, body), http.StatusCreated)
	})
}

func TestListBooksEmptyIsArray(t *testing.T) {
	app := newTestApp(t)

	recorder := app.do("GET", "/books", app.memberToken, "")
	assertStatus(t, recorder, http.StatusOK)

	if got := strings.TrimSpace(recorder.Body.String()); got != "[]" {
		t.Errorf("corpo = %s, want []", got)
	}
}
