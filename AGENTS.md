# AGENTS.md

## Layout

- `cake-backend/` — the only code in this repo; Go module `efournierrobert/cake-backend` (go 1.27.1). Run all `go` commands from `cake-backend/`, never the repo root (root is not a module; `go build ./...` from root fails).
- `cake-backend/cmd/main.go` — entrypoint: opens the DB, runs migrations, performs first-run admin setup, registers user, provider, model, space, and conversation handlers, and serves HTTP on `:8080` (port is hardcoded).
- `cake-backend/internal/repository/` — SQL layer via sqlx; one subpackage per domain (`users/` = user CRUD + credential lookup, `providers/` = provider CRUD, `models/` = model CRUD and catalog upserts, `roles/` = read access to `user_roles`, `spaces/` = space CRUD, `conversations/` = conversation CRUD, `messages/` = conversation messages), DB errors wrapped in `repo_errors`.
- `cake-backend/internal/services/` — business logic by domain. `users/` orchestrates user and role repositories and hashes passwords; `providers/` handles provider CRUD, connectivity tests, and per-provider/all-provider model refresh; `models/` maps persisted models to API DTOs and implements model list/update/delete logic; `spaces/` owns space operations and streaming chat; `conversations/` implements conversation and message retrieval/mutation. Services map repository failures to client-facing `handler_errors`.
- `cake-backend/internal/oai/` — minimal OpenAI-compatible HTTP client used to fetch provider model catalogs from `GET <base_url>/models`.
- `cake-backend/internal/handlers/` — HTTP handlers and API-facing DTOs. `handlers/users/`, `handlers/providers/`, `handlers/model/`, `handlers/spaces/`, and `handlers/conversations/` register endpoints; `handlers/models/` contains request/response DTOs; `handler_errors/` holds JSON `AppError` values (`code`, `message`, HTTP status).
- `cake-backend/internal/middleware/auth/` — authentication/authorization middleware. Protected endpoints read the `jwt-token` cookie (not an Authorization bearer header); administrator endpoints additionally require the JWT `role` claim to be `admin`.
- `cake-backend/migrations/` — goose SQL migrations.
- `cake-backend/doc/openapi.yaml` — OpenAPI reference for the implemented API. The Go route registrations, handlers, DTO JSON tags, middleware, and services are authoritative: update this document to match code when behavior changes; do not change working code merely to satisfy the spec.
- `docker/docker-compose.yaml` — only infra: MariaDB, db `cake`, user `cake-user`/`cake-user`, port `3306`.
- Root `.env` is gitignored; `.env.example` is the template.

## Setup & run

1. `docker compose -f docker/docker-compose.yaml up -d` (or any reachable MariaDB with the same DSN).
2. From `cake-backend/`: set `DATABASE_URL`, `JWT_SECRET`, and a hex-encoded AES key in `ENCRYPTION_SECRET` (16, 24, or 32 decoded bytes), then run `go run ./cmd`. The root `.env.example` shows the local values used by the IDE run configs.

## HTTP API

- The server uses Go's `http.ServeMux` method/path patterns. Routes are registered in the `New` functions under `internal/handlers/` and mounted directly at the server root.
- `POST /login` accepts JSON username/password, returns an empty `200` response, and sets a `jwt-token` cookie (`HttpOnly`, `Secure`, `SameSite=Strict`, `Path=/`, one-hour max age). `POST /logout` clears that cookie and is public. Protected routes authenticate from this cookie; bearer Authorization headers are not used.
- Admin-only areas include `/users` (except `/user`), all `/providers`, and `/models/all` plus model mutation. Spaces, conversations, chat, `/user`, and `/models` require a valid authenticated cookie. The middleware's authentication and role-denial responses are plain text; application errors from `handler_errors.WriteError` are JSON `{ "code", "message" }`.
- Chat is `POST /spaces/{uuid}/chat`, not `/chat`. Its body has optional `conversation_uuid` plus `model_uuid`, `message`, and `reasoning`; it returns a `text/event-stream` response. Event names currently include `response.start`, `response.output_text.delta`, `response.completed`, `response.failed`, `response.incomplete`, and `response.error`.
- Provider model refresh routes are `POST /providers/models/refresh` (all providers) and `POST /providers/{uuid}/models/refresh` (one provider). Model CRUD/list routes are under `/models`; there is no `/models/refresh` route.
- When changing HTTP behavior, update handlers/DTOs/tests and `doc/openapi.yaml` together, keeping the spec faithful to observed status codes, response bodies, cookie auth, and JSON field names. Do not treat OpenAPI as a contract to implement against when it disagrees with the code.

## Gotchas

- **Migrations run automatically at server startup.** `repository.NewDbConnection` calls `goose.Up(db.DB, "migrations")` with a *relative* path — the server must be started from `cake-backend/` or migrations are not found. There is no separate migrate command.
- The code targets the **goose v2 API** (`goose v2.7.0+incompatible`: `goose.SetDialect` / `goose.Up`). Do not upgrade to goose v3 without porting these calls.
- The module is **go 1.27.1**, where `uuid` is a standard-library package: code imports bare `"uuid"` (e.g. for `uuid.NewV4()`, `uuid.MustParse()`). It needs no entry in `go.mod`, and a bare `uuid` line must not be added — `go mod tidy` removes it. Do not "fix" the bare `uuid` import by switching to `github.com/google/uuid`.
- **MySQL driver placeholders:** use sqlx named params (`:name`) or `?`.
- The initial migration seeds `user_roles` (uuid `15afe83a-fd92-4d66-8f22-5a3bedbb53e1` = `user`, `5a18559d-9d20-4251-8a72-b36efbaff514` = `admin`) and `message_roles` via inline `INSERT` statements. Follow this pattern for seed data instead of seeding from app code.
- **Timestamps are microsecond-precision.** All migrations declare `DATETIME(6)` (an early draft of `20260929001102_providers_models.sql` shipped plain `DATETIME` and was later corrected — never downgrade a column to second precision). In tests, compare *stored* values (`After`, `UnixMicro`) directly, no sleeps needed for ordering; only assert strict inequality of two near-simultaneous timestamps after a short sleep, or it can flake.

## Verification

- Verify with `go build ./...` and `go vet ./...` from `cake-backend/`. Also run tests to make sure everything is working correctly.
- Tests cover repositories, user/provider/model/space/conversation services, handlers, and the OpenAI-compatible client. MariaDB-backed repository and service suites use testcontainers: `go test ./...` starts real MariaDB containers and runs migrations (Docker required); handler/service unit tests also use mocks or `httptest` servers.

## Conventions

- PR flow: numbered feature branches (e.g. `7-add-conversation-spaces`) merged into `main` on the Forgejo remote.
- New tables: add a timestamped goose file to `cake-backend/migrations/` with `-- +goose Up` / `-- +goose Down` sections.
