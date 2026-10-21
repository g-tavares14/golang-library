package repository

import (
	"database/sql"
	"errors"
	"log"

	"github.com/lib/pq"

	"go-lang/models"
)

var (
	// ErrUserNotFound é devolvido quando nenhum usuário tem o id pedido.
	ErrUserNotFound = errors.New("user not found")
	// ErrEmailAlreadyExists é devolvido quando o e-mail informado já está em uso.
	ErrEmailAlreadyExists = errors.New("email already exists")
	// ErrInvalidUserID é devolvido quando o id informado não é um UUID válido.
	ErrInvalidUserID = errors.New("invalid user id")
)

// Códigos de erro do Postgres que o repository traduz para erros do domínio.
const (
	pgUniqueViolation      = "23505"
	pgInvalidTextRepresent = "22P02"
)

// translateUserError converte erros do Postgres em erros do domínio.
// Qualquer outro erro é devolvido sem alteração.
func translateUserError(err error) error {
	if pgError, ok := errors.AsType[*pq.Error](err); ok {
		switch pgError.Code {
		case pgUniqueViolation:
			return ErrEmailAlreadyExists
		case pgInvalidTextRepresent:
			return ErrInvalidUserID
		}
	}
	return err
}

// UserRepository acessa a tabela users.
type UserRepository struct {
	db *sql.DB
}

// NewUserRepository cria um UserRepository que usa a conexão db.
func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

// Create insere o usuário e preenche ID e CreatedAt com os valores gerados pelo banco.
func (userRepository *UserRepository) Create(user *models.User) error {
	err := userRepository.db.QueryRow(
		`INSERT INTO users (name, email, password, role)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id, created_at`,
		user.Name, user.Email, user.Password, user.Role,
	).Scan(&user.ID, &user.CreatedAt)
	return translateUserError(err)
}

// List devolve todos os usuários, do mais novo para o mais antigo.
func (userRepository *UserRepository) List() ([]models.User, error) {
	rows, err := userRepository.db.Query(
		`SELECT id, name, email, created_at, role
		 FROM users
		 ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			log.Println("close rows:", err)
		}
	}()

	var users []models.User
	for rows.Next() {
		var user models.User
		if err := rows.Scan(&user.ID, &user.Name, &user.Email, &user.CreatedAt, &user.Role); err != nil {
			return nil, err
		}
		users = append(users, user)
	}

	return users, rows.Err()
}

// Update altera o usuário e preenche CreatedAt com o valor que está no banco.
// Devolve ErrUserNotFound quando nenhum usuário tem o id informado.
func (userRepository *UserRepository) Update(user *models.User) error {
	err := userRepository.db.QueryRow(
		`UPDATE users
		 SET name = $1, email = $2, password = $3, role = $4
		 WHERE id = $5
		 RETURNING created_at`,
		user.Name, user.Email, user.Password, user.Role, user.ID,
	).Scan(&user.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrUserNotFound
	}
	return translateUserError(err)
}

// Delete apaga o usuário e devolve o registro removido.
// Devolve ErrUserNotFound quando nenhum usuário tem o id informado.
func (userRepository *UserRepository) Delete(userID string) (*models.User, error) {
	return deleteReturning(
		userRepository.db,
		`DELETE FROM users
		 WHERE id = $1
		 RETURNING id, name, email, created_at, role`,
		userID,
		func(row *sql.Row, user *models.User) error {
			return row.Scan(&user.ID, &user.Name, &user.Email, &user.CreatedAt, &user.Role)
		},
		ErrUserNotFound,
		translateUserError,
	)
}
