// Package auth reúne o hash de senhas, a emissão de tokens e a verificação de papéis.
package auth

import "golang.org/x/crypto/bcrypt"

// ErrPasswordTooLong é devolvido quando a senha passa de 72 bytes, o limite do bcrypt.
var ErrPasswordTooLong = bcrypt.ErrPasswordTooLong

// HashPassword gera o hash bcrypt da senha, pronto para gravar na coluna password.
func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// CheckPassword informa se a senha em texto puro corresponde ao hash bcrypt.
func CheckPassword(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

// fakePasswordHash é um hash bcrypt sem dono, usado só para gastar o mesmo tempo de CPU.
const fakePasswordHash = "$2a$10$7Rv8Qo8w6QzuQ6Tb7l4C7.yuMcjlO6JdP2HFmZRmY24rJ0dRIa7OG"

// SpendPasswordCheck faz uma comparação bcrypt que sempre falha. Serve para o login de um
// e-mail inexistente demorar o mesmo que o de uma senha errada, sem revelar quais e-mails existem.
func SpendPasswordCheck(password string) {
	CheckPassword(fakePasswordHash, password)
}
