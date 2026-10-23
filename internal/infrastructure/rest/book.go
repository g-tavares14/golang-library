package rest

import (
	"encoding/json"
	"net/http"

	"go-lang/internal/application"
	"go-lang/internal/domain"
)

// BookHandler atende as rotas HTTP de livros.
type BookHandler struct {
	bookService *application.BookService
}

// NewBookHandler cria um BookHandler que usa bookService.
func NewBookHandler(bookService *application.BookService) *BookHandler {
	return &BookHandler{bookService: bookService}
}

// RegisterRoutes liga cada rota (método + caminho) ao seu handler.
// Listar exige login; criar, alterar e apagar exigem admin.
func (bookHandler *BookHandler) RegisterRoutes(serveMux *http.ServeMux, guard *Guard) {
	serveMux.HandleFunc("POST /books", guard.RequireAdmin(bookHandler.Create))
	serveMux.HandleFunc("GET /books", guard.RequireLogin(bookHandler.List))
	serveMux.HandleFunc("PUT /books/{id}", guard.RequireAdmin(bookHandler.Update))
	serveMux.HandleFunc("DELETE /books/{id}", guard.RequireAdmin(bookHandler.Delete))
}

// Create trata POST /books.
func (bookHandler *BookHandler) Create(responseWriter http.ResponseWriter, request *http.Request) {
	var book domain.Book

	err := json.NewDecoder(request.Body).Decode(&book)
	if err != nil {
		writeError(responseWriter, http.StatusBadRequest, err.Error())
		return
	}

	err = bookHandler.bookService.Create(&book)
	if err != nil {
		writeServiceError(responseWriter, "create book", err)
		return
	}

	writeJSON(responseWriter, http.StatusCreated, book)
}

// List trata GET /books.
func (bookHandler *BookHandler) List(responseWriter http.ResponseWriter, _ *http.Request) {
	books, err := bookHandler.bookService.List()
	if err != nil {
		writeServiceError(responseWriter, "list books", err)
		return
	}

	writeJSON(responseWriter, http.StatusOK, books)
}

// Update trata PUT /books/{id}.
func (bookHandler *BookHandler) Update(responseWriter http.ResponseWriter, request *http.Request) {
	var book domain.Book

	err := json.NewDecoder(request.Body).Decode(&book)
	if err != nil {
		writeError(responseWriter, http.StatusBadRequest, err.Error())
		return
	}

	book.ID = request.PathValue("id")

	err = bookHandler.bookService.Update(&book)
	if err != nil {
		writeServiceError(responseWriter, "update book", err)
		return
	}

	writeJSON(responseWriter, http.StatusOK, book)
}

// Delete trata DELETE /books/{id}.
func (bookHandler *BookHandler) Delete(responseWriter http.ResponseWriter, request *http.Request) {
	book, err := bookHandler.bookService.Delete(request.PathValue("id"))
	if err != nil {
		writeServiceError(responseWriter, "delete book", err)
		return
	}

	writeJSON(responseWriter, http.StatusOK, book)
}
