# TODO

Pendências do projeto, na ordem sugerida. Marque com `[x]` quando terminar.

## 🔴 1. Segurança (a refatoração que ficou para depois)

- [x] **Autenticação e autorização.** Hoje o `PUT /users/{id}` decide "é admin?" pelo `role` que vem no **corpo** da requisição ([handlers/user.go](handlers/user.go), `Update`). Qualquer pessoa passa mandando `"role": "admin"`. Precisa de login (por exemplo JWT) e um middleware que leia o papel do token, e não do JSON.
- [x] **Papel decidido pelo servidor no cadastro.** `POST /users` aceita `"role": "admin"` de qualquer cliente (mass assignment). No cadastro público, force `user` e crie uma rota protegida para promover a admin.
- [x] **Hash da senha.** `Create` e `Update` gravam a senha em texto puro. Usar `golang.org/x/crypto/bcrypt` antes de salvar (a coluna `password` é `VARCHAR(255)`, cabe o hash).
- [x] **Não devolver `password` no JSON.** `Create` e `Update` respondem com `"password": "..."`. Criar uma struct de resposta (`UserResponse` sem a senha) e, de preferência, uma struct de entrada separada.
- [ ] **Validar o formato do e-mail** e exigir uma senha mínima (hoje só checa se está vazio).

## 🟡 2. API e código

- [x] **Decidir `null` ou `[]`** nos `List` de livros e usuários. Decidido: devolvem `[]` quando vazios.
- [x] **`PUT` de usuário exige `password`.** Para só trocar o nome, o cliente precisa reenviar a senha. Decidido: o `PUT` segue completo e o `PATCH /users/{id}` altera só os campos enviados.
- [x] **Encerramento do servidor.** O `defer database.Close()` do [main.go](main.go) nunca roda, porque o `log.Fatal` final chama `os.Exit`. Usar `http.Server` com `Shutdown` e sinal de parada.
- [ ] **Reduzir duplicação (opcional).** Ideias que não foram aplicadas: `decodeJSON[T]` nos handlers, `EmptyIfNil[T]` nos `List`, ou um `Repository[T]` genérico. Só vale a pena com mais tabelas.
- [ ] **Reavaliar `GET /books/{id}` e `GET /users/{id}`** (busca individual), que foram removidos de propósito.

## 🟢 3. Projeto e ferramentas

- [x] **Testes.** Não existe nenhum `*_test.go`. Feito: handlers com `httptest` e `auth` rodam sempre; os testes de `repository` só rodam com `TEST_DATABASE_URL` definida (Postgres com as migrations aplicadas).
- [x] **README** com: subir o banco (`docker compose up -d db`), rodar migrations (`docker compose run --rm migrate up`), copiar o `.env.example`, rodar (`go run .`) e a lista de endpoints.
- [ ] **CI no GitHub Actions** rodando `go build ./...`, `golangci-lint fmt --diff ./...` e `golangci-lint run ./...` a cada push.
- [ ] **Arquivo `.http`** no repositório documentando as rotas (criar, listar, atualizar, apagar) para testar pelo GoLand.
- [ ] **Colocar `~/go/bin` no `PATH`** do terminal, para o comando `golangci-lint` funcionar sem o caminho completo (hoje ele está em `/Users/gtavares/go/bin/golangci-lint`).
- [ ] **GoLand:** se o aviso `Unable to resolve table 'users'` aparecer no `.sql`, fazer **Refresh** na janela Database (o cache do IDE ficou antes da migration 3).

## ✅ Já feito

- [x] Postgres no Docker Compose e conexão via `.env`
- [x] CRUD de livros e usuários em camadas (handlers, repository, models)
- [x] Migrations do Goose (`00001` a `00003`, `created_at` alinhado nas duas tabelas)
- [x] Erros em JSON (`{"error": "..."}`) e códigos corretos (`400`, `404`, `409`)
- [x] Padronização de nomes, imports e comentários, com `.golangci.yml` (0 problemas)
- [x] Atalho `⇧⌘L` para rodar o `golangci-lint fmt`
- [x] Repositório Git e push para `g-tavares14/golang-library` (privado)
