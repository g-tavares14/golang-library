package domain

import "time"

// Book é um livro do acervo.
type Book struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Description *string   `json:"description"`
	Author      string    `json:"author"`
	CreatedAt   time.Time `json:"created_at"`
}

// BookRepository é o armazenamento de livros de que as regras da aplicação precisam.
type BookRepository interface {
	// Create grava o livro e preenche ID e CreatedAt.
	Create(book *Book) error
	// List devolve todos os livros.
	List() ([]Book, error)
	// Update altera o livro e preenche CreatedAt.
	Update(book *Book) error
	// Delete apaga o livro e devolve o registro removido.
	Delete(bookID string) (*Book, error)
}

// Validate confere se o livro tem título e escritor.
func (book *Book) Validate() error {
	if book.Title == "" || book.Author == "" {
		return ErrBookFieldsEmpty
	}

	return nil
}
