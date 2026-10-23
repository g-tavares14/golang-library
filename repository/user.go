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

// CreateIfEmailMissing insere o usuário, a menos que o e-mail já esteja em uso.
// Devolve true quando criou e false quando o e-mail já existia, sem alterar o registro existente.
func (userRepository *UserRepository) CreateIfEmailMissing(user *models.User) (bool, error) {
	err := userRepository.db.QueryRow(
		`INSERT INTO users (name, email, password, role)
		 VALUES ($1, $2, $3, $4)
		 ON CONFLICT (email) DO NOTHING
		 RETURNING id, created_at`,
		user.Name, user.Email, user.Password, user.Role,
	).Scan(&user.ID, &user.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	return true, nil
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

// FindByEmail devolve o usuário com o e-mail informado, incluindo o hash da senha.
// Devolve ErrUserNotFound quando nenhum usuário usa esse e-mail.
func (userRepository *UserRepository) FindByEmail(email string) (*models.User, error) {
	var user models.User

	err := userRepository.db.QueryRow(
		`SELECT id, name, email, password, created_at, role
		 FROM users
		 WHERE email = $1`,
		email,
	).Scan(&user.ID, &user.Name, &user.Email, &user.Password, &user.CreatedAt, &user.Role)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}

	return &user, nil
}

// Update altera nome, e-mail e senha do usuário e preenche CreatedAt e Role com os valores do banco.
// O papel nunca é alterado aqui; para isso existe UpdateRole.
// Devolve ErrUserNotFound quando nenhum usuário tem o id informado.
func (userRepository *UserRepository) Update(user *models.User) error {
	err := userRepository.db.QueryRow(
		`UPDATE users
		 SET name = $1, email = $2, password = $3
		 WHERE id = $4
		 RETURNING created_at, role`,
		user.Name, user.Email, user.Password, user.ID,
	).Scan(&user.CreatedAt, &user.Role)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrUserNotFound
	}
	return translateUserError(err)
}

// UpdateRole altera o papel do usuário e devolve o registro atualizado, sem a senha.
// Devolve ErrUserNotFound quando nenhum usuário tem o id informado.
func (userRepository *UserRepository) UpdateRole(userID, role string) (*models.User, error) {
	var user models.User

	err := userRepository.db.QueryRow(
		`UPDATE users
		 SET role = $1
		 WHERE id = $2
		 RETURNING id, name, email, created_at, role`,
		role, userID,
	).Scan(&user.ID, &user.Name, &user.Email, &user.CreatedAt, &user.Role)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, translateUserError(err)
	}

	return &user, nil
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
