package application

import (
	"go-lang/internal/domain"
)

// adminName é o nome do admin criado por EnsureAdmin.
const adminName = "Admin"

// UserService reúne os casos de uso de usuários.
type UserService struct {
	users  domain.UserRepository
	hasher PasswordHasher
}

// NewUserService cria um UserService que grava em users e gera hashes com hasher.
func NewUserService(users domain.UserRepository, hasher PasswordHasher) *UserService {
	return &UserService{users: users, hasher: hasher}
}

// Register cadastra um usuário comum. O papel é sempre domain.RoleUser e a senha é gravada como hash.
func (userService *UserService) Register(name, email, password string) (*domain.User, error) {
	user, err := domain.NewUser(name, email, password)
	if err != nil {
		return nil, err
	}

	user.Password, err = userService.hasher.Hash(password)
	if err != nil {
		return nil, err
	}

	if err := userService.users.Create(user); err != nil {
		return nil, err
	}

	return user, nil
}

// List devolve todos os usuários.
func (userService *UserService) List() ([]domain.User, error) {
	return userService.users.List()
}

// Replace substitui nome, e-mail e senha do usuário. O papel nunca muda por aqui.
func (userService *UserService) Replace(userID, name, email, password string) (*domain.User, error) {
	user := &domain.User{ID: userID, Name: name, Email: email, Password: password}
	if err := user.Validate(); err != nil {
		return nil, err
	}

	var err error
	user.Password, err = userService.hasher.Hash(password)
	if err != nil {
		return nil, err
	}

	if err := userService.users.Update(user); err != nil {
		return nil, err
	}

	return user, nil
}

// Patch altera só os campos enviados. A senha, quando enviada, é gravada como hash.
func (userService *UserService) Patch(userID string, patch domain.UserPatch) (*domain.User, error) {
	if err := patch.Validate(); err != nil {
		return nil, err
	}

	if patch.Password != nil {
		passwordHash, err := userService.hasher.Hash(*patch.Password)
		if err != nil {
			return nil, err
		}
		patch.Password = &passwordHash
	}

	return userService.users.Patch(userID, patch)
}

// ChangeRole define o papel de targetID em nome de actorID, respeitando as regras de domain.ValidateRoleChange.
func (userService *UserService) ChangeRole(actorID, targetID, role string) (*domain.User, error) {
	if err := domain.ValidateRoleChange(actorID, targetID, role); err != nil {
		return nil, err
	}

	return userService.users.UpdateRole(targetID, role)
}

// Delete apaga o usuário e devolve o registro removido.
func (userService *UserService) Delete(userID string) (*domain.User, error) {
	return userService.users.Delete(userID)
}

// EnsureAdmin cria o admin inicial se o e-mail ainda não existir e informa se criou.
// É idempotente: um usuário já existente com esse e-mail não é alterado.
func (userService *UserService) EnsureAdmin(email, password string) (bool, error) {
	passwordHash, err := userService.hasher.Hash(password)
	if err != nil {
		return false, err
	}

	admin := &domain.User{Name: adminName, Email: email, Password: passwordHash, Role: domain.RoleAdmin}

	return userService.users.CreateIfEmailMissing(admin)
}
