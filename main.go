package main

import (
	"log"
	"net/http"

	"go-lang/config"
	"go-lang/handlers"
	"go-lang/repository"
)

func main() {
	database := config.SetupDB()
	defer func() {
		if err := database.Close(); err != nil {
			log.Println("close database:", err)
		}
	}()

	bookRepository := repository.NewBookRepository(database)
	userRepository := repository.NewUserRepository(database)
	bookHandler := handlers.NewBookHandler(bookRepository)
	userHandler := handlers.NewUserHandler(userRepository)

	serveMux := http.NewServeMux()
	bookHandler.RegisterRoutes(serveMux)
	userHandler.RegisterRoutes(serveMux)

	address := ":8080"
	log.Println("server listening on", address)
	log.Fatal(http.ListenAndServe(address, serveMux))
}
