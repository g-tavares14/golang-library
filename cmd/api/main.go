package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go-lang/internal/application"
	"go-lang/internal/infrastructure/config"
	"go-lang/internal/infrastructure/postgres"
	"go-lang/internal/infrastructure/rest"
	"go-lang/internal/infrastructure/security"
)

const (
	// shutdownTimeout é o tempo máximo para as requisições em andamento terminarem no encerramento.
	shutdownTimeout = 10 * time.Second
	// tokenTTL é por quanto tempo um token emitido no login continua válido.
	tokenTTL = time.Hour
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

// run sobe o servidor e só retorna depois do encerramento, para que os defers rodem.
func run() error {
	database := config.SetupDB()
	defer func() {
		if err := database.Close(); err != nil {
			log.Println("close database:", err)
		}
	}()

	// Infraestrutura: acesso ao banco, hash de senha e tokens.
	bookRepository := postgres.NewBookRepository(database)
	userRepository := postgres.NewUserRepository(database)
	passwordHasher := security.NewPasswordHasher()
	tokenManager := security.NewTokenManager(config.JWTSecret(), tokenTTL)

	// Casos de uso.
	bookService := application.NewBookService(bookRepository)
	userService := application.NewUserService(userRepository, passwordHasher)
	authService := application.NewAuthService(userRepository, passwordHasher, tokenManager)

	if err := seedAdmin(userService); err != nil {
		return err
	}

	// Camada HTTP.
	bookHandler := rest.NewBookHandler(bookService)
	userHandler := rest.NewUserHandler(userService)
	authHandler := rest.NewAuthHandler(authService)
	guard := rest.NewGuard(tokenManager)

	serveMux := http.NewServeMux()
	bookHandler.RegisterRoutes(serveMux, guard)
	userHandler.RegisterRoutes(serveMux, guard)
	authHandler.RegisterRoutes(serveMux)

	server := &http.Server{
		Addr:              ":8080",
		Handler:           serveMux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	// stop cancela o contexto quando o processo recebe Ctrl+C (SIGINT) ou SIGTERM.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	serverErrors := make(chan error, 1)
	go func() {
		log.Println("server listening on", server.Addr)
		serverErrors <- server.ListenAndServe()
	}()

	select {
	case err := <-serverErrors:
		return err
	case <-ctx.Done():
		log.Println("shutting down server")
	}

	shutdownContext, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if err := server.Shutdown(shutdownContext); err != nil {
		return err
	}

	if err := <-serverErrors; !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	return nil
}
