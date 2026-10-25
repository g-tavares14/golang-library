// Package bootstrap é a raiz de composição: liga a infraestrutura (Postgres, bcrypt, JWT) aos casos
// de uso e às rotas HTTP. O servidor real (cmd/api) e os testes ponta a ponta usam a mesma montagem.
package bootstrap

import (
	"database/sql"
	"net/http"
	"time"

	"go-lang/internal/application"
	"go-lang/internal/infrastructure/postgres"
	"go-lang/internal/infrastructure/rest"
	"go-lang/internal/infrastructure/security"
)

// App reúne o que o chamador precisa do sistema montado.
type App struct {
	// Handler atende todas as rotas HTTP da API.
	Handler http.Handler
	// Users dá acesso aos casos de uso de usuário, por exemplo para criar o admin inicial.
	Users *application.UserService
}

// New monta a aplicação sobre database. Os tokens são assinados com jwtSecret e valem por tokenTTL.
func New(database *sql.DB, jwtSecret []byte, tokenTTL time.Duration) *App {
	// Infraestrutura: acesso ao banco, hash de senha e tokens.
	bookRepository := postgres.NewBookRepository(database)
	userRepository := postgres.NewUserRepository(database)
	passwordHasher := security.NewPasswordHasher()
	tokenManager := security.NewTokenManager(jwtSecret, tokenTTL)

	// Casos de uso.
	bookService := application.NewBookService(bookRepository)
	userService := application.NewUserService(userRepository, passwordHasher)
	authService := application.NewAuthService(userRepository, passwordHasher, tokenManager)

	// Camada HTTP.
	bookHandler := rest.NewBookHandler(bookService)
	userHandler := rest.NewUserHandler(userService)
	authHandler := rest.NewAuthHandler(authService)
	guard := rest.NewGuard(tokenManager)

	serveMux := http.NewServeMux()
	bookHandler.RegisterRoutes(serveMux, guard)
	userHandler.RegisterRoutes(serveMux, guard)
	authHandler.RegisterRoutes(serveMux)

	return &App{Handler: serveMux, Users: userService}
}
