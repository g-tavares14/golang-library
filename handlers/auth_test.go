package handlers_test

import (
	"net/http"
	"testing"

	"go-lang/internal/domain"
)

func TestLoginSuccess(t *testing.T) {
	app := newTestApp(t)

	recorder := app.do("POST", "/login", "", `{"email":"admin@example.com","password":"senha-do-teste"}`)
	assertStatus(t, recorder, http.StatusOK)

	token, _ := decode(t, recorder)["token"].(string)
	claims, err := app.tokenManager.Parse(token)
	if err != nil {
		t.Fatalf("token devolvido é inválido: %v", err)
	}

	if claims.UserID != app.adminID || claims.Role != domain.RoleAdmin {
		t.Errorf("claims = %+v, want UserID=%s Role=%s", claims, app.adminID, domain.RoleAdmin)
	}
}

func TestLoginFailures(t *testing.T) {
	tests := []struct {
		name string
		body string
		want int
	}{
		{"senha errada", `{"email":"admin@example.com","password":"errada"}`, http.StatusUnauthorized},
		{"e-mail inexistente", `{"email":"ninguem@example.com","password":"senha-do-teste"}`, http.StatusUnauthorized},
		{"sem senha", `{"email":"admin@example.com"}`, http.StatusBadRequest},
		{"sem e-mail", `{"password":"senha-do-teste"}`, http.StatusBadRequest},
		{"JSON inválido", `{`, http.StatusBadRequest},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			app := newTestApp(t)

			recorder := app.do("POST", "/login", "", test.body)
			assertStatus(t, recorder, test.want)

			if _, hasToken := decode(t, recorder)["token"]; hasToken {
				t.Error("a resposta de falha não pode conter token")
			}
		})
	}
}

func TestLoginErrorDoesNotRevealWhichEmailsExist(t *testing.T) {
	app := newTestApp(t)

	wrongPassword := app.do("POST", "/login", "", `{"email":"admin@example.com","password":"errada"}`)
	unknownEmail := app.do("POST", "/login", "", `{"email":"ninguem@example.com","password":"errada"}`)

	if wrongPassword.Body.String() != unknownEmail.Body.String() {
		t.Errorf("mensagens diferentes revelam quais e-mails existem:\n%q\n%q",
			wrongPassword.Body.String(), unknownEmail.Body.String())
	}
}
