package rest

import (
	"errors"
	"log"
	"net/http"

	"go-lang/internal/domain"
)

// statusByError associa cada erro de domínio ao status HTTP que o representa.
var statusByError = []struct {
	err    error
	status int
}{
	{domain.ErrBookFieldsEmpty, http.StatusBadRequest},
	{domain.ErrUserFieldsEmpty, http.StatusBadRequest},
	{domain.ErrNoFieldsToUpdate, http.StatusBadRequest},
	{domain.ErrInvalidEmail, http.StatusBadRequest},
	{domain.ErrWeakPassword, http.StatusBadRequest},
	{domain.ErrPasswordTooLong, http.StatusBadRequest},
	{domain.ErrInvalidRole, http.StatusBadRequest},
	{domain.ErrLoginFieldsEmpty, http.StatusBadRequest},
	{domain.ErrInvalidUserID, http.StatusBadRequest},
	{domain.ErrInvalidBookID, http.StatusBadRequest},
	{domain.ErrInvalidCredentials, http.StatusUnauthorized},
	{domain.ErrOwnRoleChange, http.StatusForbidden},
	{domain.ErrUserNotFound, http.StatusNotFound},
	{domain.ErrBookNotFound, http.StatusNotFound},
	{domain.ErrEmailAlreadyExists, http.StatusConflict},
	{domain.ErrUserNameTooLong, http.StatusBadRequest},
}

// writeServiceError traduz o erro de um caso de uso em resposta HTTP.
// Erros de domínio viram o status da tabela acima, com a própria mensagem; qualquer outro é
// registrado no log (com operation) e vira 500 sem detalhes para o cliente.
func writeServiceError(responseWriter http.ResponseWriter, operation string, err error) {
	for _, mapping := range statusByError {
		if errors.Is(err, mapping.err) {
			writeError(responseWriter, mapping.status, mapping.err.Error())
			return
		}
	}

	log.Println(operation+":", err)
	writeError(responseWriter, http.StatusInternalServerError, "internal server error")
}
