package main

import (
	"errors"
	"fmt"
	"log"
	"os"

	"go-lang/internal/application"
)

// seedAdmin cria o admin inicial a partir de ADMIN_EMAIL e ADMIN_PASSWORD.
// É idempotente: se o e-mail já existe, não altera nada. Sem ADMIN_EMAIL, não faz nada.
func seedAdmin(userService *application.UserService) error {
	email := os.Getenv("ADMIN_EMAIL")
	if email == "" {
		log.Println("ADMIN_EMAIL is not set, skipping admin seed")
		return nil
	}

	password := os.Getenv("ADMIN_PASSWORD")
	if password == "" {
		return errors.New("ADMIN_PASSWORD is required when ADMIN_EMAIL is set")
	}

	created, err := userService.EnsureAdmin(email, password)
	if err != nil {
		return fmt.Errorf("ensure admin: %w", err)
	}

	if created {
		log.Println("admin user created:", email)
	}

	return nil
}
