package application

import "go-lang/internal/domain"

// BookService reúne os casos de uso de livros.
type BookService struct {
	books domain.BookRepository
}

// NewBookService cria um BookService que grava em books.
func NewBookService(books domain.BookRepository) *BookService {
	return &BookService{books: books}
}

// Create valida e grava o livro, preenchendo ID e CreatedAt.
func (bookService *BookService) Create(book *domain.Book) error {
	if err := book.Validate(); err != nil {
		return err
	}

	return bookService.books.Create(book)
}

// List devolve todos os livros.
func (bookService *BookService) List() ([]domain.Book, error) {
	return bookService.books.List()
}

// Update valida e altera o livro, preenchendo CreatedAt.
func (bookService *BookService) Update(book *domain.Book) error {
	if err := book.Validate(); err != nil {
		return err
	}

	return bookService.books.Update(book)
}

// Delete apaga o livro e devolve o registro removido.
func (bookService *BookService) Delete(bookID string) (*domain.Book, error) {
	return bookService.books.Delete(bookID)
}
