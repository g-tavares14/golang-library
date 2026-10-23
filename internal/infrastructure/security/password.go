// Package security reúne o hash de senhas e a emissão e validação de tokens.
package security

import (
	"errors"

	"golang.org/x/crypto/bcrypt"

	"go-lang/internal/domain"
)

// fakePasswordHash é um hash bcrypt sem dono, usado só para gastar o mesmo tempo de CPU.
const fakePasswordHash = "$2a$10$7Rv8Qo8w6QzuQ6Tb7l4C7.yuMcjlO6JdP2HFmZRmY24rJ0dRIa7OG"

// PasswordHasher gera e confere hashes de senha com bcrypt.
type PasswordHasher struct{}

// NewPasswordHasher cria um PasswordHasher.
func NewPasswordHasher() *PasswordHasher {
	return &PasswordHasher{}
}

// Hash gera o hash bcrypt da senha, pronto para gravar na coluna password.
// Senhas acima de 72 bytes, o limite do bcrypt, devolvem domain.ErrPasswordTooLong.
func (*PasswordHasher) Hash(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		if errors.Is(err, bcrypt.ErrPasswordTooLong) {
			return "", domain.ErrPasswordTooLong
		}
		return "", err
	}
	return string(hash), nil
}

// Verify informa se a senha em texto puro corresponde ao hash bcrypt.
func (*PasswordHasher) Verify(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

// VerifyAgainstDummy faz uma comparação bcrypt que sempre falha. Serve para o login de um
// e-mail inexistente demorar o mesmo que o de uma senha errada, sem revelar quais e-mails existem.
func (hasher *PasswordHasher) VerifyAgainstDummy(password string) {
	hasher.Verify(fakePasswordHash, password)
}
