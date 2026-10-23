package handlers

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"go-lang/internal/domain"
)

// BookStore é o acesso a livros de que o BookHandler precisa. O *repository.BookRepository a satisfaz.
type BookStore interface {
	Create(book *domain.Book) error
	List() ([]domain.Book, error)
	Update(book *domain.Book) error
	Delete(bookID string) (*domain.Book, error)
}

// BookHandler atende as rotas HTTP de livros.
type BookHandler struct {
	bookRepository BookStore
}

// NewBookHandler cria um BookHandler que usa bookRepository.
func NewBookHandler(bookRepository BookStore) *BookHandler {
	return &BookHandler{bookRepository: bookRepository}
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

	if book.Title == "" || book.Author == "" {
		writeError(responseWriter, http.StatusBadRequest, "book title or author is empty")
		return
	}

	err = bookHandler.bookRepository.Create(&book)
	if err != nil {
		log.Println("create book:", err)
		writeError(responseWriter, http.StatusInternalServerError, "internal server error")
		return
	}

	writeJSON(responseWriter, http.StatusCreated, book)
}

// List trata GET /books.
func (bookHandler *BookHandler) List(responseWriter http.ResponseWriter, _ *http.Request) {
	books, err := bookHandler.bookRepository.List()
	if err != nil {
		log.Println("list books:", err)
		writeError(responseWriter, http.StatusInternalServerError, "internal server error")
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

	if book.Title == "" || book.Author == "" {
		writeError(responseWriter, http.StatusBadRequest, "book title or author is empty")
		return
	}

	book.ID = request.PathValue("id")

	err = bookHandler.bookRepository.Update(&book)
	if err != nil {
		if errors.Is(err, domain.ErrBookNotFound) {
			writeError(responseWriter, http.StatusNotFound, "book not found")
			return
		}
		if errors.Is(err, domain.ErrInvalidBookID) {
			writeError(responseWriter, http.StatusBadRequest, "invalid book id")
			return
		}
		log.Println("update book:", err)
		writeError(responseWriter, http.StatusInternalServerError, "internal server error")
		return
	}

	writeJSON(responseWriter, http.StatusOK, book)
}

// Delete trata DELETE /books/{id}.
func (bookHandler *BookHandler) Delete(responseWriter http.ResponseWriter, request *http.Request) {
	book, err := bookHandler.bookRepository.Delete(request.PathValue("id"))
	if err != nil {
		if errors.Is(err, domain.ErrBookNotFound) {
			writeError(responseWriter, http.StatusNotFound, "book not found")
			return
		}
		if errors.Is(err, domain.ErrInvalidBookID) {
			writeError(responseWriter, http.StatusBadRequest, "invalid book id")
			return
		}
		log.Println("delete book:", err)
		writeError(responseWriter, http.StatusInternalServerError, "internal server error")
		return
	}

	writeJSON(responseWriter, http.StatusOK, book)
}
