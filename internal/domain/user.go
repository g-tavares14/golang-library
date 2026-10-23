package domain

import (
	"net/mail"
	"time"
)

const (
	// maxEmailLength acompanha o tamanho da coluna users.email (VARCHAR(100)).
	maxEmailLength = 100
	// minPasswordLength é o tamanho mínimo aceito para uma senha, em bytes.
	minPasswordLength = 8
)

// User é uma pessoa que usa a API.
type User struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Password  string    `json:"password"`
	CreatedAt time.Time `json:"created_at"`
	Role      string    `json:"role"`
}

// UserPatch reúne os campos de um usuário que podem ser alterados parcialmente.
// Um campo nil significa "manter o valor atual".
type UserPatch struct {
	Name     *string
	Email    *string
	Password *string
}

// UserRepository é o armazenamento de usuários de que as regras da aplicação precisam.
type UserRepository interface {
	// Create grava o usuário e preenche ID e CreatedAt.
	Create(user *User) error
	// CreateIfEmailMissing grava o usuário, a menos que o e-mail já exista, e informa se gravou.
	CreateIfEmailMissing(user *User) (bool, error)
	// List devolve todos os usuários.
	List() ([]User, error)
	// FindByEmail devolve o usuário com o e-mail informado ou ErrUserNotFound.
	FindByEmail(email string) (*User, error)
	// Update altera nome, e-mail e senha, nunca o papel.
	Update(user *User) error
	// Patch altera só os campos não nulos de patch.
	Patch(userID string, patch UserPatch) (*User, error)
	// UpdateRole altera apenas o papel do usuário.
	UpdateRole(userID, role string) (*User, error)
	// Delete apaga o usuário e devolve o registro removido.
	Delete(userID string) (*User, error)
}

// NewUser cria um usuário a partir de um cadastro público: o papel é sempre RoleUser,
// qualquer que seja a vontade do cliente. A senha ainda está em texto puro e já validada;
// quem grava precisa trocá-la pelo hash antes.
func NewUser(name, email, password string) (*User, error) {
	user := &User{Name: name, Email: email, Password: password, Role: RoleUser}
	if err := user.Validate(); err != nil {
		return nil, err
	}

	return user, nil
}

// Validate confere se nome, e-mail e senha (em texto puro) estão preenchidos e são aceitáveis.
func (user *User) Validate() error {
	if user.Name == "" || user.Email == "" || user.Password == "" {
		return ErrUserFieldsEmpty
	}

	if err := ValidateEmail(user.Email); err != nil {
		return err
	}

	return ValidatePassword(user.Password)
}

// Validate confere se a atualização parcial traz ao menos um campo e se cada campo enviado é aceitável.
func (patch UserPatch) Validate() error {
	if patch.Name == nil && patch.Email == nil && patch.Password == nil {
		return ErrNoFieldsToUpdate
	}

	if (patch.Name != nil && *patch.Name == "") ||
		(patch.Email != nil && *patch.Email == "") ||
		(patch.Password != nil && *patch.Password == "") {
		return ErrUserFieldsEmpty
	}

	if patch.Email != nil {
		if err := ValidateEmail(*patch.Email); err != nil {
			return err
		}
	}

	if patch.Password != nil {
		return ValidatePassword(*patch.Password)
	}

	return nil
}

// ValidateEmail confere se email é um endereço simples, como "ana@example.com".
// mail.ParseAddress também aceita formas como "Ana <ana@example.com>"; comparar o
// resultado com a entrada rejeita esses casos.
func ValidateEmail(email string) error {
	if len(email) > maxEmailLength {
		return ErrInvalidEmail
	}

	address, err := mail.ParseAddress(email)
	if err != nil || address.Address != email {
		return ErrInvalidEmail
	}

	return nil
}

// ValidatePassword confere se password atende à política de senhas da API: no mínimo
// 8 caracteres, com letra maiúscula, letra minúscula e número.
// O limite de 72 bytes do bcrypt é tratado ao gerar o hash; aqui entram as regras de qualidade.
func ValidatePassword(password string) error {
	var hasUpperCase, hasLowerCase, hasNumber bool

	for _, char := range password {
		switch {
		case char >= 'A' && char <= 'Z':
			hasUpperCase = true
		case char >= 'a' && char <= 'z':
			hasLowerCase = true
		case char >= '0' && char <= '9':
			hasNumber = true
		}
	}

	if len(password) < minPasswordLength || !hasUpperCase || !hasLowerCase || !hasNumber {
		return ErrWeakPassword
	}

	return nil
}
