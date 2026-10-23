package main

import (
	"errors"
	"fmt"
	"log"
	"os"

	"go-lang/auth"
	"go-lang/internal/domain"
	"go-lang/internal/infrastructure/postgres"
)

// seedAdmin cria o admin inicial a partir de ADMIN_EMAIL e ADMIN_PASSWORD.
// É idempotente: se o e-mail já existe, não altera nada. Sem ADMIN_EMAIL, não faz nada.
func seedAdmin(userRepository *postgres.UserRepository) error {
	email := os.Getenv("ADMIN_EMAIL")
	if email == "" {
		log.Println("ADMIN_EMAIL is not set, skipping admin seed")
		return nil
	}

	password := os.Getenv("ADMIN_PASSWORD")
	if password == "" {
		return errors.New("ADMIN_PASSWORD is required when ADMIN_EMAIL is set")
	}

	passwordHash, err := auth.HashPassword(password)
	if err != nil {
		return fmt.Errorf("hash admin password: %w", err)
	}

	admin := domain.User{Name: "Admin", Email: email, Password: passwordHash, Role: domain.RoleAdmin}

	created, err := userRepository.CreateIfEmailMissing(&admin)
	if err != nil {
		return fmt.Errorf("create admin: %w", err)
	}

	if created {
		log.Println("admin user created:", email)
	}

	return nil
}
