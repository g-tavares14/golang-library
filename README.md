# go-lang

API REST em Go com Postgres para livros e usuários, com login por JWT.

## Como rodar

Pré-requisitos: Go 1.27+ e Docker.

```bash
# 1. Variáveis de ambiente (edite JWT_SECRET, ADMIN_PASSWORD e DB_PASSWORD)
cp .env.example .env

# 2. Banco de dados (o Compose fica em deployment/ e lê o .env da raiz)
docker compose -f deployment/docker-compose.yml --env-file .env up -d db

# 3. Migrations
docker compose -f deployment/docker-compose.yml --env-file .env run --rm migrate up

# 4. Servidor em http://localhost:8080 (rode da raiz: o .env é lido do diretório atual)
go run ./cmd/api
```

### Variáveis do `.env`

| Variável | Para que serve |
| --- | --- |
| `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME` | Conexão com o Postgres (o Compose usa os mesmos valores) |
| `JWT_SECRET` | Chave que assina os tokens, com no mínimo 32 caracteres. Gere uma com `openssl rand -base64 48` |
| `ADMIN_EMAIL`, `ADMIN_PASSWORD` | Admin inicial, criado na subida do servidor se o e-mail ainda não existir. Deixe `ADMIN_EMAIL` vazio para não criar |

## Autenticação

1. `POST /login` com e-mail e senha devolve `{"token": "..."}` (válido por 1 hora).
2. Envie o token nas demais rotas: `Authorization: Bearer <token>`.

O `POST /login` tem limite de tentativas por IP, contra força bruta: 5 seguidas, e depois uma nova a cada 12 segundos. Acima disso a resposta é `429` com o cabeçalho `Retry-After` (em segundos). Toda tentativa conta, certa ou errada, e os cabeçalhos `X-Forwarded-For` e `X-Real-IP` são ignorados porque qualquer cliente pode forjá-los.

O papel (`user` ou `admin`) vem do token assinado, nunca do corpo da requisição. O cadastro público sempre cria `user`; só um admin promove alguém, em `PUT /users/{id}/role`.

## Endpoints

| Método | Rota | Acesso | Descrição |
| --- | --- | --- | --- |
| `POST` | `/login` | público | Troca e-mail e senha por um token |
| `POST` | `/users` | público | Cadastra um usuário (papel `user`) |
| `GET` | `/users` | login | Lista usuários |
| `PUT` | `/users/{id}` | admin | Substitui nome, e-mail e senha (todos obrigatórios) |
| `PATCH` | `/users/{id}` | admin | Altera só os campos enviados |
| `PUT` | `/users/{id}/role` | admin | Define o papel: `{"role": "admin"}` ou `{"role": "user"}` (não vale para o próprio admin) |
| `DELETE` | `/users/{id}` | admin | Apaga o usuário |
| `GET` | `/books` | login | Lista livros |
| `POST` | `/books` | admin | Cria um livro |
| `PUT` | `/books/{id}` | admin | Altera um livro |
| `DELETE` | `/books/{id}` | admin | Apaga um livro |

Regras de cadastro: o e-mail precisa ser um endereço simples (até 100 caracteres) e a senha precisa ter no mínimo 8 caracteres, com letra maiúscula, minúscula e número.

Os erros vêm como `{"error": "..."}`. Exemplos prontos de cada requisição estão em [requests.http](requests.http), para rodar pelo GoLand.

## Estrutura do projeto

```text
cmd/api/                       ponto de entrada: lê o .env, cria o admin inicial e sobe o servidor
internal/
  bootstrap/                   raiz de composição: liga infraestrutura, casos de uso e rotas
  domain/                      entidades, regras de negócio, erros e interfaces dos repositórios
  application/                 casos de uso (services) que orquestram as regras do domain
  infrastructure/
    rest/                      camada HTTP: handlers, guard (JWT) e DTOs
    postgres/                  repositórios com database/sql e tradução dos erros do Postgres
    security/                  bcrypt e tokens JWT
    config/                    leitura do .env e conexão com o banco
  testutil/                    apoio aos testes de integração (conexão com o banco de teste)
test/e2e/                      testes ponta a ponta: as rotas reais sobre um Postgres, por HTTP
deployment/                    docker-compose.yml (Postgres e migrations)
migrations/                    migrations do Goose
```

A dependência aponta sempre para dentro: `infrastructure` conhece `application` e `domain`, a `application` conhece só o `domain`, e o `domain` não importa nada do projeto. Regras como "o cadastro público sempre cria `user`" ou "ninguém muda o próprio papel" ficam no `domain`; o HTTP só traduz erros de domínio em status (`rest/errors.go`).

## Desenvolvimento

```bash
go build ./...
golangci-lint fmt --diff ./...   # formatação
golangci-lint run ./...          # lint
go test ./...                    # domain, application, rest e security rodam sempre
```

Os testes de `internal/infrastructure/postgres` e `test/e2e` precisam de um Postgres com as migrations aplicadas e ficam pulados sem `TEST_DATABASE_URL`. Use um banco de testes, não o de desenvolvimento:

```bash
# 1. cria o banco de testes e aplica as migrations (com o Compose no ar)
docker compose -f deployment/docker-compose.yml --env-file .env exec db psql -U postgres -d golang -c "CREATE DATABASE golang_test"
go run github.com/pressly/goose/v3/cmd/goose@v3.29.0 -dir migrations postgres "host=localhost port=5432 user=postgres password=... dbname=golang_test sslmode=disable" up

# 2. roda tudo, inclusive os testes de banco
TEST_DATABASE_URL="host=localhost port=5432 user=postgres password=... dbname=golang_test sslmode=disable" go test ./...
```

Os testes criam dados com e-mails e títulos únicos e apagam o que criaram, mas nunca aponte `TEST_DATABASE_URL` para um banco com dados que importam.

## CI

O workflow `.github/workflows/ci.yml` roda a cada push: build, formatação, lint, as migrations num Postgres 17 descartável e `go test ./...` com `TEST_DATABASE_URL` apontando para ele. Na CI (`CI=true`), um teste de banco sem `TEST_DATABASE_URL` falha em vez de ser pulado, para os testes de banco não "passarem" sem rodar.
