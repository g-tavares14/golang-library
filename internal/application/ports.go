// Package application reúne os casos de uso: orquestra as regras do domain usando as
// interfaces abaixo. Não conhece HTTP, SQL, bcrypt nem JWT; quem implementa essas
// interfaces fica em infrastructure.
package application

// PasswordHasher gera e confere hashes de senha.
type PasswordHasher interface {
	// Hash devolve o hash da senha ou domain.ErrPasswordTooLong quando ela passa do limite do algoritmo.
	Hash(password string) (string, error)
	// Verify informa se a senha corresponde ao hash.
	Verify(hash, password string) bool
	// VerifyAgainstDummy gasta o mesmo tempo de uma conferência de verdade, sem dono, para o login
	// de um e-mail inexistente não ser mais rápido que o de uma senha errada.
	VerifyAgainstDummy(password string)
}

// TokenIssuer emite o token de acesso de um usuário.
type TokenIssuer interface {
	Issue(userID, role string) (string, error)
}
