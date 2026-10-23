package models

// Papéis aceitos para um usuário.
const (
	RoleUser  = "user"
	RoleAdmin = "admin"
)

// IsValidRole informa se role é um dos papéis aceitos.
func IsValidRole(role string) bool {
	return role == RoleUser || role == RoleAdmin
}
