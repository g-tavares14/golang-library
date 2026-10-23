package domain

import "errors"

// Erros de existência e de identificação de registros.
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

// Erros de regras de negócio violadas pela entrada.
var (
	// ErrBookFieldsEmpty é devolvido quando o livro não tem título ou escritor.
	ErrBookFieldsEmpty = errors.New("book title or author is empty")
	// ErrUserFieldsEmpty é devolvido quando falta nome, e-mail ou senha do usuário.
	ErrUserFieldsEmpty = errors.New("user name, email or password is empty")
	// ErrNoFieldsToUpdate é devolvido quando uma atualização parcial não traz nenhum campo.
	ErrNoFieldsToUpdate = errors.New("no fields to update")
	// ErrInvalidEmail é devolvido quando o e-mail não é um endereço simples válido.
	ErrInvalidEmail = errors.New("invalid email")
	// ErrWeakPassword é devolvido quando a senha não atende à política de senhas.
	ErrWeakPassword = errors.New("password must have at least 8 characters, with upper case, lower case and a number")
	// ErrPasswordTooLong é devolvido quando a senha passa do limite que o algoritmo de hash aceita.
	ErrPasswordTooLong = errors.New("password is too long")
	// ErrInvalidRole é devolvido quando o papel informado não existe.
	ErrInvalidRole = errors.New("invalid role")
	// ErrOwnRoleChange é devolvido quando um usuário tenta mudar o próprio papel.
	ErrOwnRoleChange = errors.New("cannot change your own role")
	// ErrLoginFieldsEmpty é devolvido quando o login não traz e-mail ou senha.
	ErrLoginFieldsEmpty = errors.New("email or password is empty")
	// ErrInvalidCredentials é devolvido quando o e-mail ou a senha do login não conferem.
	// A mesma mensagem vale para os dois casos, para não revelar quais e-mails existem.
	ErrInvalidCredentials = errors.New("invalid credentials")
	// ErrUserNameTooLong é devolvido quando o nome do usuário é muito longo.
	// O limite é definido pela coluna de nome do usuário no banco de dados (VARCHAR 100).
	ErrUserNameTooLong = errors.New("user name is too long")
)
