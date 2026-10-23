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
