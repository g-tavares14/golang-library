package domain

// Papéis aceitos para um usuário.
const (
	RoleUser  = "user"
	RoleAdmin = "admin"
)

// IsValidRole informa se role é um dos papéis aceitos.
func IsValidRole(role string) bool {
	return role == RoleUser || role == RoleAdmin
}

// ValidateRoleChange confere se actorID pode definir o papel role para targetID.
// O papel precisa existir, e ninguém muda o próprio papel, para o sistema não ficar sem admin.
func ValidateRoleChange(actorID, targetID, role string) error {
	if !IsValidRole(role) {
		return ErrInvalidRole
	}

	if actorID == targetID {
		return ErrOwnRoleChange
	}

	return nil
}
