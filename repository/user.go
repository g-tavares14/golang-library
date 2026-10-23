package repository

import (
	"database/sql"
	"errors"
	"log"

	"github.com/lib/pq"

	"go-lang/internal/domain"
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
			return domain.ErrEmailAlreadyExists
		case pgInvalidTextRepresent:
			return domain.ErrInvalidUserID
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
func (userRepository *UserRepository) Create(user *domain.User) error {
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
func (userRepository *UserRepository) CreateIfEmailMissing(user *domain.User) (bool, error) {
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
func (userRepository *UserRepository) List() ([]domain.User, error) {
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

	users := []domain.User{}
	for rows.Next() {
		var user domain.User
		if err := rows.Scan(&user.ID, &user.Name, &user.Email, &user.CreatedAt, &user.Role); err != nil {
			return nil, err
		}
		users = append(users, user)
	}

	return users, rows.Err()
}

// FindByEmail devolve o usuário com o e-mail informado, incluindo o hash da senha.
// Devolve domain.ErrUserNotFound quando nenhum usuário usa esse e-mail.
func (userRepository *UserRepository) FindByEmail(email string) (*domain.User, error) {
	var user domain.User

	err := userRepository.db.QueryRow(
		`SELECT id, name, email, password, created_at, role
		 FROM users
		 WHERE email = $1`,
		email,
	).Scan(&user.ID, &user.Name, &user.Email, &user.Password, &user.CreatedAt, &user.Role)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}

	return &user, nil
}

// Update altera nome, e-mail e senha do usuário e preenche CreatedAt e Role com os valores do banco.
// O papel nunca é alterado aqui; para isso existe UpdateRole.
// Devolve domain.ErrUserNotFound quando nenhum usuário tem o id informado.
func (userRepository *UserRepository) Update(user *domain.User) error {
	err := userRepository.db.QueryRow(
		`UPDATE users
		 SET name = $1, email = $2, password = $3
		 WHERE id = $4
		 RETURNING created_at, role`,
		user.Name, user.Email, user.Password, user.ID,
	).Scan(&user.CreatedAt, &user.Role)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.ErrUserNotFound
	}
	return translateUserError(err)
}

// Patch altera só os campos não nulos de patch e devolve o registro atualizado, sem a senha.
// Devolve domain.ErrUserNotFound quando nenhum usuário tem o id informado.
func (userRepository *UserRepository) Patch(userID string, patch domain.UserPatch) (*domain.User, error) {
	var user domain.User

	err := userRepository.db.QueryRow(
		`UPDATE users
		 SET name = COALESCE($1, name),
		     email = COALESCE($2, email),
		     password = COALESCE($3, password)
		 WHERE id = $4
		 RETURNING id, name, email, created_at, role`,
		patch.Name, patch.Email, patch.Password, userID,
	).Scan(&user.ID, &user.Name, &user.Email, &user.CreatedAt, &user.Role)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrUserNotFound
	}
	if err != nil {
		return nil, translateUserError(err)
	}

	return &user, nil
}

// UpdateRole altera o papel do usuário e devolve o registro atualizado, sem a senha.
// Devolve domain.ErrUserNotFound quando nenhum usuário tem o id informado.
func (userRepository *UserRepository) UpdateRole(userID, role string) (*domain.User, error) {
	var user domain.User

	err := userRepository.db.QueryRow(
		`UPDATE users
		 SET role = $1
		 WHERE id = $2
		 RETURNING id, name, email, created_at, role`,
		role, userID,
	).Scan(&user.ID, &user.Name, &user.Email, &user.CreatedAt, &user.Role)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrUserNotFound
	}
	if err != nil {
		return nil, translateUserError(err)
	}

	return &user, nil
}

// Delete apaga o usuário e devolve o registro removido.
// Devolve domain.ErrUserNotFound quando nenhum usuário tem o id informado.
func (userRepository *UserRepository) Delete(userID string) (*domain.User, error) {
	return deleteReturning(
		userRepository.db,
		`DELETE FROM users
		 WHERE id = $1
		 RETURNING id, name, email, created_at, role`,
		userID,
		func(row *sql.Row, user *domain.User) error {
			return row.Scan(&user.ID, &user.Name, &user.Email, &user.CreatedAt, &user.Role)
		},
		domain.ErrUserNotFound,
		translateUserError,
	)
}
