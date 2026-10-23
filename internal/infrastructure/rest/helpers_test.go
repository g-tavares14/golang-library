package rest_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"go-lang/internal/application"
	"go-lang/internal/domain"
	"go-lang/internal/infrastructure/rest"
	"go-lang/internal/infrastructure/security"
)

var testSecret = []byte("0123456789012345678901234567890123456789")

// fakeUserStore é um UserStore e UserFinder em memória.
type fakeUserStore struct {
	users  map[string]*domain.User
	nextID int
}

func newFakeUserStore() *fakeUserStore {
	return &fakeUserStore{users: map[string]*domain.User{}}
}

// add insere um usuário já pronto (com o hash da senha) e devolve o id.
func (store *fakeUserStore) add(name, email, passwordHash, role string) string {
	store.nextID++
	id := strconv.Itoa(store.nextID)
	store.users[id] = &domain.User{
		ID: id, Name: name, Email: email, Password: passwordHash, Role: role, CreatedAt: time.Now(),
	}
	return id
}

func (store *fakeUserStore) emailTaken(email, exceptID string) bool {
	for id, user := range store.users {
		if user.Email == email && id != exceptID {
			return true
		}
	}
	return false
}

func (store *fakeUserStore) Create(user *domain.User) error {
	if store.emailTaken(user.Email, "") {
		return domain.ErrEmailAlreadyExists
	}
	user.ID = store.add(user.Name, user.Email, user.Password, user.Role)
	user.CreatedAt = store.users[user.ID].CreatedAt
	return nil
}

func (store *fakeUserStore) CreateIfEmailMissing(user *domain.User) (bool, error) {
	if store.emailTaken(user.Email, "") {
		return false, nil
	}
	user.ID = store.add(user.Name, user.Email, user.Password, user.Role)
	return true, nil
}

func (store *fakeUserStore) List() ([]domain.User, error) {
	users := []domain.User{}
	for _, user := range store.users {
		users = append(users, *user)
	}
	return users, nil
}

func (store *fakeUserStore) FindByEmail(email string) (*domain.User, error) {
	for _, user := range store.users {
		if user.Email == email {
			found := *user
			return &found, nil
		}
	}
	return nil, domain.ErrUserNotFound
}

func (store *fakeUserStore) Update(user *domain.User) error {
	stored, ok := store.users[user.ID]
	if !ok {
		return domain.ErrUserNotFound
	}
	if store.emailTaken(user.Email, user.ID) {
		return domain.ErrEmailAlreadyExists
	}
	stored.Name, stored.Email, stored.Password = user.Name, user.Email, user.Password
	user.CreatedAt, user.Role = stored.CreatedAt, stored.Role
	return nil
}

func (store *fakeUserStore) Patch(userID string, patch domain.UserPatch) (*domain.User, error) {
	stored, ok := store.users[userID]
	if !ok {
		return nil, domain.ErrUserNotFound
	}
	if patch.Email != nil && store.emailTaken(*patch.Email, userID) {
		return nil, domain.ErrEmailAlreadyExists
	}
	if patch.Name != nil {
		stored.Name = *patch.Name
	}
	if patch.Email != nil {
		stored.Email = *patch.Email
	}
	if patch.Password != nil {
		stored.Password = *patch.Password
	}
	updated := *stored
	return &updated, nil
}

func (store *fakeUserStore) UpdateRole(userID, role string) (*domain.User, error) {
	stored, ok := store.users[userID]
	if !ok {
		return nil, domain.ErrUserNotFound
	}
	stored.Role = role
	updated := *stored
	return &updated, nil
}

func (store *fakeUserStore) Delete(userID string) (*domain.User, error) {
	stored, ok := store.users[userID]
	if !ok {
		return nil, domain.ErrUserNotFound
	}
	delete(store.users, userID)
	return stored, nil
}

// fakeBookStore é um BookStore em memória.
type fakeBookStore struct {
	books  map[string]*domain.Book
	nextID int
}

func newFakeBookStore() *fakeBookStore {
	return &fakeBookStore{books: map[string]*domain.Book{}}
}

func (store *fakeBookStore) Create(book *domain.Book) error {
	store.nextID++
	book.ID = strconv.Itoa(store.nextID)
	book.CreatedAt = time.Now()
	stored := *book
	store.books[book.ID] = &stored
	return nil
}

func (store *fakeBookStore) List() ([]domain.Book, error) {
	books := []domain.Book{}
	for _, book := range store.books {
		books = append(books, *book)
	}
	return books, nil
}

func (store *fakeBookStore) Update(book *domain.Book) error {
	stored, ok := store.books[book.ID]
	if !ok {
		return domain.ErrBookNotFound
	}
	stored.Title, stored.Description, stored.Author = book.Title, book.Description, book.Author
	book.CreatedAt = stored.CreatedAt
	return nil
}

func (store *fakeBookStore) Delete(bookID string) (*domain.Book, error) {
	stored, ok := store.books[bookID]
	if !ok {
		return nil, domain.ErrBookNotFound
	}
	delete(store.books, bookID)
	return stored, nil
}

// testApp reúne o servidor de teste e os dados prontos para as requisições.
type testApp struct {
	mux          *http.ServeMux
	users        *fakeUserStore
	books        *fakeBookStore
	tokenManager *security.TokenManager
	adminID      string
	memberID     string // usuário comum
	adminToken   string
	memberToken  string
}

// newTestApp monta as rotas reais com stores em memória, um admin e um usuário comum.
func newTestApp(t *testing.T) *testApp {
	t.Helper()

	app := &testApp{
		mux:          http.NewServeMux(),
		users:        newFakeUserStore(),
		books:        newFakeBookStore(),
		tokenManager: security.NewTokenManager(testSecret, time.Hour),
	}

	passwordHasher := security.NewPasswordHasher()

	passwordHash, err := passwordHasher.Hash("senha-do-teste")
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}

	app.adminID = app.users.add("Admin", "admin@example.com", passwordHash, domain.RoleAdmin)
	app.memberID = app.users.add("Maria", "maria@example.com", passwordHash, domain.RoleUser)
	app.adminToken = app.issueToken(t, app.adminID, domain.RoleAdmin)
	app.memberToken = app.issueToken(t, app.memberID, domain.RoleUser)

	guard := rest.NewGuard(app.tokenManager)
	rest.NewBookHandler(application.NewBookService(app.books)).RegisterRoutes(app.mux, guard)
	rest.NewUserHandler(application.NewUserService(app.users, passwordHasher)).RegisterRoutes(app.mux, guard)
	rest.NewAuthHandler(application.NewAuthService(app.users, passwordHasher, app.tokenManager)).RegisterRoutes(app.mux)

	return app
}

func (app *testApp) issueToken(t *testing.T, userID, role string) string {
	t.Helper()

	token, err := app.tokenManager.Issue(userID, role)
	if err != nil {
		t.Fatalf("Issue() error = %v", err)
	}
	return token
}

// do envia uma requisição às rotas; token vazio significa sem cabeçalho Authorization.
func (app *testApp) do(method, path, token, body string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, path, strings.NewReader(body))
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}

	recorder := httptest.NewRecorder()
	app.mux.ServeHTTP(recorder, request)
	return recorder
}

// decode lê o corpo JSON da resposta em um map genérico.
func decode(t *testing.T, recorder *httptest.ResponseRecorder) map[string]any {
	t.Helper()

	var body map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("resposta não é um objeto JSON: %v\ncorpo: %s", err, recorder.Body.String())
	}
	return body
}

// decodeList lê o corpo JSON da resposta em uma lista genérica.
func decodeList(t *testing.T, recorder *httptest.ResponseRecorder) []any {
	t.Helper()

	var body []any
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("resposta não é uma lista JSON: %v\ncorpo: %s", err, recorder.Body.String())
	}
	return body
}

func assertStatus(t *testing.T, recorder *httptest.ResponseRecorder, want int) {
	t.Helper()

	if recorder.Code != want {
		t.Fatalf("status = %d, want %d\ncorpo: %s", recorder.Code, want, recorder.Body.String())
	}
}

// httpRecord executa uma requisição já montada nas rotas do app.
func httpRecord(app *testApp, request *http.Request) *httptest.ResponseRecorder {
	recorder := httptest.NewRecorder()
	app.mux.ServeHTTP(recorder, request)
	return recorder
}
