// Package testutil reúne apoio aos testes de integração que precisam de um Postgres de verdade.
package testutil

import (
	"database/sql"
	"os"
	"testing"

	_ "github.com/lib/pq" // driver do PostgreSQL para o database/sql
)

// OpenTestDB conecta ao Postgres indicado em TEST_DATABASE_URL, que precisa ter as migrations aplicadas.
// Localmente, sem a variável, o teste é pulado. Na CI (CI=true) ela é obrigatória: faltar é erro,
// para um nome errado de variável não transformar todos os testes de banco em SKIP com a CI verde.
//
// Exemplo local:
//
//	TEST_DATABASE_URL="host=localhost port=5432 user=postgres password=... dbname=golang_test sslmode=disable" go test ./...
func OpenTestDB(t *testing.T) *sql.DB {
	t.Helper()

	connectionString := os.Getenv("TEST_DATABASE_URL")
	if connectionString == "" {
		if os.Getenv("CI") != "" {
			t.Fatal("TEST_DATABASE_URL não definida na CI; os testes de banco não podem ser pulados")
		}
		t.Skip("TEST_DATABASE_URL não definida; pulando teste de integração com o Postgres")
	}

	database, err := sql.Open("postgres", connectionString)
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })

	if err := database.Ping(); err != nil {
		t.Fatalf("ping database: %v", err)
	}

	return database
}
