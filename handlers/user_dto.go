package handlers

import (
	"time"

	"go-lang/models"
)

// userInput é o corpo JSON aceito ao criar ou atualizar um usuário.
// É separado de models.User para que o cliente só controle os campos permitidos:
// o papel (role) nunca vem do corpo da requisição.
type userInput struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

// toUser converte a entrada para o modelo usado pelo repository.
func (input userInput) toUser() models.User {
	return models.User{
		Name:     input.Name,
		Email:    input.Email,
		Password: input.Password,
	}
}

// userPatchInput é o corpo JSON aceito em PATCH /users/{id}. Campos ausentes ficam nil e não mudam.
type userPatchInput struct {
	Name     *string `json:"name"`
	Email    *string `json:"email"`
	Password *string `json:"password"`
}

// toPatch converte a entrada para o modelo usado pelo repository.
func (input userPatchInput) toPatch() models.UserPatch {
	return models.UserPatch{
		Name:     input.Name,
		Email:    input.Email,
		Password: input.Password,
	}
}

// roleInput é o corpo JSON aceito em PUT /users/{id}/role.
type roleInput struct {
	Role string `json:"role"`
}

// userResponse é o corpo JSON devolvido ao cliente. Nunca inclui a senha.
type userResponse struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
}

// newUserResponse converte o modelo para a resposta pública, sem a senha.
func newUserResponse(user *models.User) userResponse {
	return userResponse{
		ID:        user.ID,
		Name:      user.Name,
		Email:     user.Email,
		Role:      user.Role,
		CreatedAt: user.CreatedAt,
	}
}
