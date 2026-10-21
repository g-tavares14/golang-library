// Package config reúne a configuração e a conexão com o banco de dados.
package config

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq" // driver do PostgreSQL para o database/sql
)

// SetupDB lê o .env, conecta ao Postgres e confirma a conexão com Ping.
func SetupDB() *sql.DB {
	err := godotenv.Load(".env")
	if err != nil {
		log.Fatalf("load .env file: %v", err)
	}

	databaseHost := os.Getenv("DB_HOST")
	databasePort := os.Getenv("DB_PORT")
	databaseUser := os.Getenv("DB_USER")
	databasePassword := os.Getenv("DB_PASSWORD")
	databaseName := os.Getenv("DB_NAME")

	connectionString := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		databaseHost, databasePort, databaseUser, databasePassword, databaseName,
	)

	database, err := sql.Open("postgres", connectionString)
	if err != nil {
		log.Fatalf("open database: %v", err)
	}

	if err := database.Ping(); err != nil {
		log.Fatalf("ping database: %v", err)
	}

	log.Println("database connection established")

	return database
}
