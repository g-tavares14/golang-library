package repository

import (
	"database/sql"
	"errors"
	"log"

	"github.com/lib/pq"

	"go-lang/internal/domain"
)

// translateBookError converte erros do Postgres em erros do domínio.
// Qualquer outro erro é devolvido sem alteração.
func translateBookError(err error) error {
	if pgError, ok := errors.AsType[*pq.Error](err); ok && pgError.Code == pgInvalidTextRepresent {
		return domain.ErrInvalidBookID
	}
	return err
}

// BookRepository acessa a tabela books.
type BookRepository struct {
	db *sql.DB
}

// NewBookRepository cria um BookRepository que usa a conexão db.
func NewBookRepository(db *sql.DB) *BookRepository {
	return &BookRepository{db: db}
}

// Create insere o livro e preenche ID e CreatedAt com os valores gerados pelo banco.
func (bookRepository *BookRepository) Create(book *domain.Book) error {
	return bookRepository.db.QueryRow(
		`INSERT INTO books (title, description, author)
		 VALUES ($1, $2, $3)
		 RETURNING id, created_at`,
		book.Title, book.Description, book.Author,
	).Scan(&book.ID, &book.CreatedAt)
}

// List devolve todos os livros, do mais novo para o mais antigo.
func (bookRepository *BookRepository) List() ([]domain.Book, error) {
	rows, err := bookRepository.db.Query(
		`SELECT id, title, description, author, created_at
		 FROM books
		 ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			log.Println("close rows:", err)
		}
	}()

	books := []domain.Book{}
	for rows.Next() {
		var book domain.Book
		if err := rows.Scan(&book.ID, &book.Title, &book.Description, &book.Author, &book.CreatedAt); err != nil {
			return nil, err
		}
		books = append(books, book)
	}

	return books, rows.Err()
}

// Update altera o livro e preenche CreatedAt com o valor que está no banco.
// Devolve domain.ErrBookNotFound quando nenhum livro tem o id informado.
func (bookRepository *BookRepository) Update(book *domain.Book) error {
	err := bookRepository.db.QueryRow(
		`UPDATE books
		 SET title = $1, description = $2, author = $3
		 WHERE id = $4
		 RETURNING created_at`,
		book.Title, book.Description, book.Author, book.ID,
	).Scan(&book.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.ErrBookNotFound
	}
	return translateBookError(err)
}

// Delete apaga o livro e devolve o registro removido.
// Devolve domain.ErrBookNotFound quando nenhum livro tem o id informado.
func (bookRepository *BookRepository) Delete(bookID string) (*domain.Book, error) {
	return deleteReturning(
		bookRepository.db,
		`DELETE FROM books
		 WHERE id = $1
		 RETURNING id, title, description, author, created_at`,
		bookID,
		func(row *sql.Row, book *domain.Book) error {
			return row.Scan(&book.ID, &book.Title, &book.Description, &book.Author, &book.CreatedAt)
		},
		domain.ErrBookNotFound,
		translateBookError,
	)
}
