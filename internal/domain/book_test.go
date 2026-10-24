package domain_test

import (
	"errors"
	"strings"
	"testing"

	"go-lang/internal/domain"
)

func TestBookValidate(t *testing.T) {
	tests := []struct {
		name string
		book domain.Book
		want error
	}{
		{"completo", domain.Book{Title: "Go", Author: "Pike"}, nil},
		{"sem título", domain.Book{Author: "Pike"}, domain.ErrBookFieldsEmpty},
		{"sem escritor", domain.Book{Title: "Go"}, domain.ErrBookFieldsEmpty},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := test.book.Validate(); !errors.Is(err, test.want) {
				t.Errorf("Validate() = %v, want %v", err, test.want)
			}
		})
	}
}

func TestBookValidateFieldLength(t *testing.T) {
	// Cada campo é testado sozinho: um único campo acima do limite já estoura a coluna do banco.
	at := func(length int) string { return strings.Repeat("a", length) }
	accented := strings.Repeat("é", 255) // 255 caracteres, mas 510 bytes
	over := at(256)
	exact := at(255)

	tests := []struct {
		name string
		book domain.Book
		want error
	}{
		{"título com 255", domain.Book{Title: exact, Author: "Pike"}, nil},
		{"título com 256", domain.Book{Title: over, Author: "Pike"}, domain.ErrBookFieldTooLong},
		{"escritor com 255", domain.Book{Title: "Go", Author: exact}, nil},
		{"escritor com 256", domain.Book{Title: "Go", Author: over}, domain.ErrBookFieldTooLong},
		{"descrição com 255", domain.Book{Title: "Go", Author: "Pike", Description: &exact}, nil},
		{"descrição com 256", domain.Book{Title: "Go", Author: "Pike", Description: &over}, domain.ErrBookFieldTooLong},
		{"descrição ausente", domain.Book{Title: "Go", Author: "Pike", Description: nil}, nil},
		// Só passam se o limite contar caracteres, como a coluna VARCHAR(255) do Postgres.
		{"título com 255 acentuados", domain.Book{Title: accented, Author: "Pike"}, nil},
		{"escritor com 255 acentuados", domain.Book{Title: "Go", Author: accented}, nil},
		{"descrição com 255 acentuados", domain.Book{Title: "Go", Author: "Pike", Description: &accented}, nil},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := test.book.Validate(); !errors.Is(err, test.want) {
				t.Errorf("Validate() = %v, want %v", err, test.want)
			}
		})
	}
}
