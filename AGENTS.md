# AGENTS.md

## Layout

- `cake-backend/` — the only code in this repo; Go module `efournierrobert/cake-backend` (go 1.27.1). Run all `go` commands from `cake-backend/`, never the repo root (root is not a module; `go build ./...` from root fails).
- `cake-backend/cmd/main.go` — entrypoint: opens the DB, runs migrations, serves an (currently empty) HTTP mux on `:8080` (port is hardcoded).
- `cake-backend/internal/repository/` — SQL layer via sqlx; one subpackage per domain (`users/` = repository + models, `roles/` = read access to `user_roles`), DB errors wrapped in `repo_errors`.
- `cake-backend/internal/services/users/` — business logic for the Users endpoints: orchestrates the users and roles repositories, hashes passwords with bcrypt, maps `repo_errors` to client-facing `handler_errors`, exposes methods per OpenAPI Users operation. Not wired into any HTTP handler yet (no handler is registered on the mux in `cmd/main.go`).
- `cake-backend/internal/handlers/` — API-facing types: `handlers/users/` holds the request/response DTOs matching the OpenAPI schemas; `handler_errors/` holds the `AppError` values (code, client-facing message, HTTP status) that handlers return to clients.
- `cake-backend/migrations/` — goose SQL migrations.
- `cake-backend/doc/openapi.yaml` — the API contract (v1.0.0, self-declared provisional). Almost no endpoints are implemented yet; treat this doc as the spec to build to and keep it in sync with code.
- `docker/docker-compose.yaml` — only infra: MariaDB, db `cake`, user `cake-user`/`cake-user`, port `3306`.
- Root `.env` is gitignored; `.env.example` is the template.

## Setup & run

1. `docker compose -f docker/docker-compose.yaml up -d` (or any reachable MariaDB with the same DSN).
2. From `cake-backend/`: `go run ./cmd` with `DATABASE_URL="cake-user:cake-user@tcp(127.0.0.1:3306)/cake?parseTime=true"` set (the IDE run configs use exactly these env values).

## Gotchas

- **Migrations run automatically at server startup.** `repository.NewDbConnection` calls `goose.Up(db.DB, "migrations")` with a *relative* path — the server must be started from `cake-backend/` or migrations are not found. There is no separate migrate command.
- The code targets the **goose v2 API** (`goose v2.7.0+incompatible`: `goose.SetDialect` / `goose.Up`). Do not upgrade to goose v3 without porting these calls.
- **MySQL driver placeholders:** use sqlx named params (`:name`) or `?`.
- The initial migration seeds `user_roles` (uuid `15afe83a-fd92-4d66-8f22-5a3bedbb53e1` = `user`, `5a18559d-9d20-4251-8a72-b36efbaff514` = `admin`) and `message_roles` via inline `INSERT` statements. Follow this pattern for seed data instead of seeding from app code.

## Verification

- Verify with `go build ./...` and `go vet ./...` from `cake-backend/`. Also run tests to make sure everything is working correctly.
- Tests currently live only in `internal/repository/users/` and use testcontainers: `go test ./...` spins up a real MariaDB container (Docker required) and runs migrations against it.

## Conventions

- PR flow: numbered feature branches (e.g. `7-add-conversation-spaces`) merged into `main` on the Forgejo remote.
- New tables: add a timestamped goose file to `cake-backend/migrations/` with `-- +goose Up` / `-- +goose Down` sections.
