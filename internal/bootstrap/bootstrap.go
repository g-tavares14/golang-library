// Package bootstrap é a raiz de composição: liga a infraestrutura (Postgres, bcrypt, JWT) aos casos
// de uso e às rotas HTTP. O servidor real (cmd/api) e os testes ponta a ponta usam a mesma montagem.
package bootstrap

import (
	"database/sql"
	"net/http"
	"time"

	"golang.org/x/time/rate"

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

// loginLimiterIdleTTL é por quanto tempo o limitador do login lembra de um IP que parou de tentar.
const loginLimiterIdleTTL = 10 * time.Minute

// Config reúne os ajustes que variam entre o servidor real e os testes.
type Config struct {
	// JWTSecret assina os tokens de acesso.
	JWTSecret []byte
	// TokenTTL é por quanto tempo um token emitido no login continua válido.
	TokenTTL time.Duration
	// LoginRate é o ritmo em que cada IP recupera uma tentativa de login.
	LoginRate rate.Limit
	// LoginBurst é quantas tentativas de login seguidas um IP pode fazer. Precisa ser pelo menos 1.
	LoginBurst int
}

// New monta a aplicação sobre database, com os ajustes de config.
func New(database *sql.DB, config Config) *App {
	// Infraestrutura: acesso ao banco, hash de senha e tokens.
	bookRepository := postgres.NewBookRepository(database)
	userRepository := postgres.NewUserRepository(database)
	passwordHasher := security.NewPasswordHasher()
	tokenManager := security.NewTokenManager(config.JWTSecret, config.TokenTTL)

	// Casos de uso.
	bookService := application.NewBookService(bookRepository)
	userService := application.NewUserService(userRepository, passwordHasher)
	authService := application.NewAuthService(userRepository, passwordHasher, tokenManager)

	// Camada HTTP.
	bookHandler := rest.NewBookHandler(bookService)
	userHandler := rest.NewUserHandler(userService)
	authHandler := rest.NewAuthHandler(authService)
	guard := rest.NewGuard(tokenManager)
	loginLimiter := rest.NewRateLimiter(config.LoginRate, config.LoginBurst, loginLimiterIdleTTL)

	serveMux := http.NewServeMux()
	bookHandler.RegisterRoutes(serveMux, guard)
	userHandler.RegisterRoutes(serveMux, guard)
	authHandler.RegisterRoutes(serveMux, loginLimiter)

	return &App{Handler: serveMux, Users: userService}
}
