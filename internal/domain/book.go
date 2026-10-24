package domain

import (
	"time"
	"unicode/utf8"
)

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

// maxBookFieldLength é o tamanho máximo aceito para qualquer campo do livro, em caracteres UTF-8.
const maxBookFieldLength = 255

// Validate confere se o livro tem título e escritor e se título, escritor e descrição
// cabem nas colunas do banco.
func (book *Book) Validate() error {
	if book.Title == "" || book.Author == "" {
		return ErrBookFieldsEmpty
	}

	if utf8.RuneCountInString(book.Title) > maxBookFieldLength || utf8.RuneCountInString(book.Author) > maxBookFieldLength {
		return ErrBookFieldTooLong
	}

	if book.Description != nil && utf8.RuneCountInString(*book.Description) > maxBookFieldLength {
		return ErrBookFieldTooLong
	}

	return nil
}
