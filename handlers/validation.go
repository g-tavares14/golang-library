package handlers

import (
	"errors"
	"net/mail"
)

// maxEmailLength acompanha o tamanho da coluna users.email (VARCHAR(100)).
const maxEmailLength = 100

var errInvalidEmail = errors.New("invalid email")

// validateEmail confere se email é um endereço simples, como "ana@example.com".
// mail.ParseAddress também aceita formas como "Ana <ana@example.com>"; comparar o
// resultado com a entrada rejeita esses casos.
func validateEmail(email string) error {
	if len(email) > maxEmailLength {
		return errInvalidEmail
	}

	address, err := mail.ParseAddress(email)
	if err != nil || address.Address != email {
		return errInvalidEmail
	}

	return nil
}

// validateCredentials valida e-mail e senha juntos, na ordem em que o cliente os preenche.
func validateCredentials(email, password string) error {
	if err := validateEmail(email); err != nil {
		return err
	}

	return validatePassword(password)
}

// minPasswordLength é o tamanho mínimo aceito para uma senha, em bytes.
const minPasswordLength = 8

var errWeakPassword = errors.New("password must have at least 8 characters, with upper case, lower case and a number")

// validatePassword confere se password atende à política de senhas da API: no mínimo
// 8 caracteres, com letra maiúscula, letra minúscula e número.
// O limite de 72 bytes do bcrypt já é tratado ao gerar o hash; aqui entram as regras de qualidade.
func validatePassword(password string) error {
	var hasUpperCase, hasLowerCase, hasNumber bool

	for _, char := range password {
		switch {
		case char >= 'A' && char <= 'Z':
			hasUpperCase = true
		case char >= 'a' && char <= 'z':
			hasLowerCase = true
		case char >= '0' && char <= '9':
			hasNumber = true
		}
	}

	if len(password) < minPasswordLength || !hasUpperCase || !hasLowerCase || !hasNumber {
		return errWeakPassword
	}

	return nil
}
