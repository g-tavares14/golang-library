package domain

import "errors"

var (
	// ErrBookNotFound é devolvido quando nenhum livro tem o id pedido.
	ErrBookNotFound = errors.New("book not found")
	// ErrInvalidBookID é devolvido quando o id informado não é um UUID válido.
	ErrInvalidBookID = errors.New("invalid book id")
	// ErrUserNotFound é devolvido quando nenhum usuário tem o id pedido.
	ErrUserNotFound = errors.New("user not found")
	// ErrEmailAlreadyExists é devolvido quando o e-mail informado já está em uso.
	ErrEmailAlreadyExists = errors.New("email already exists")
	// ErrInvalidUserID é devolvido quando o id informado não é um UUID válido.
	ErrInvalidUserID = errors.New("invalid user id")
)
