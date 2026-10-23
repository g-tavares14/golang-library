package models

import "time"

// User representa uma linha da tabela users.
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
