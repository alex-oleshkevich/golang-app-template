---
name: this-project
description: >-
  Use this skill whenever you add, change or move code in this repository: an
  endpoint, handler, error response, CLI command or flag, migration, query, test,
  package, or a question of which file something belongs in. Use it even when the
  user only says "add X" or "fix Y", and when reviewing a change. It holds the
  layout, the HTTP, error, CLI and database rules, and the mistakes that silently
  break builds, the API and deploys here.
---

# This project

One Go binary: a JSON API, a CLI and the React UI embedded into it. Follow what
the code already does. If this file and the code disagree, trust the code and
fix this file.

## Layout

- `cmd/nisaba/`: `main` only. Global log flags and the list of commands.
- `internal/commands/<name>/cmd.go`: one package per CLI command.
- `internal/web/`: server, middleware, routes, error mapping. `handlers/`
  implements the API, `respond/` writes responses and problem+json.
- `internal/db/`: connection, embedded migrations, sqlc queries.
- `internal/logging/`, `internal/testutil/`: shared by everything above.
- `api/openapi.yaml`: the API contract. `api/api.gen.go` is generated from it.
- `ui/`: frontend sources. Builds into `internal/web/assets/app`, which the
  binary embeds.
- `scripts/`: shell helpers behind the `justfile`. Run things with `just`.

A name means one thing everywhere:

- `handlers/<resource>.go`: the API methods for one resource
- `internal/db/queries/<table>.sql`: every query on one table
- `internal/commands/<name>/cmd.go`: `NewCommand()` and its actions
- `errors.go` in a package: that package's sentinel and typed errors

Start a resource with only the files it needs and add more when a second caller
or a second concern appears. Never create `util`, `utils`, `helpers`, `common`
or `base` packages or files; code two packages share moves up to the parent
package that owns that job. Comments say why, not what. Environment variables
start with `APP_`.

## Read the area file before you change that area

- Endpoints, handlers, error responses: `${CLAUDE_SKILL_DIR}/references/http-api.md`
- CLI commands and flags: `${CLAUDE_SKILL_DIR}/references/cli.md`
- Schema, migrations, queries: `${CLAUDE_SKILL_DIR}/references/database.md`
- Tests: `${CLAUDE_SKILL_DIR}/references/testing.md`
- Dev loop, Docker, CI, deploy, tool versions: `${CLAUDE_SKILL_DIR}/references/delivery.md`

## Rules

HTTP
- Spec first: add the operation to `api/openapi.yaml`, run `just generate`, then
  implement the method on `handlers.API`. Never edit `api/api.gen.go`.
- The OpenAPI middleware already validates requests. Handlers don't repeat it.
- Every error is RFC 9457 problem+json. Spec handlers return the generated
  error response type; hand-written routes write through `respond`. A 5xx logs
  the error and sends none of its text to the client.
- Only handlers know HTTP. Code below them returns plain Go errors; the handler
  maps them to a response with `errors.Is` or `errors.As`.

CLI
- Declare flags inline in the command that reads them, even if another command
  has the same flag. Add `Sources: cli.EnvVars("APP_...")` when deploy sets it.
- Commands return errors. Only `main` prints them and picks the exit code.

Database
- Open SQLite only through `db.Open`. It sets WAL, foreign keys and the busy
  timeout, and makes write transactions take the lock up front.
- Change the schema only with a new migration (`just cmd db new <name>`). An
  applied migration is never edited.
- SQL lives in `internal/db/queries/*.sql` and is compiled with `just sqlc`.

Tests
- Use `internal/testutil`. Database tests run on a real migrated SQLite from
  `testutil.DB`, not on mocks.

Delivery
- Pinned tool versions live in several files. Change every copy together; the
  list is in `references/delivery.md`.

## Logging

Log with `log/slog`: a fixed lowercase message plus key-value pairs, never a
formatted string, so the fields stay searchable.

```go
slog.InfoContext(ctx, "note archived", "note_id", note.ID)
```

- Pass the request context (`slog.ErrorContext(r.Context(), ...)`). The request
  log line already carries `request_id`; a line you write yourself needs
  `"request_id", middleware.GetReqID(ctx)`, as `respond` does.
- Log ids, never secrets, tokens or personal data.
- Log an error once, where it is handled. A handler that returns an error is
  logged by `responseErrorHandler`; don't log it again before returning.

## Gotchas

These fail silently, only on a fresh clone, or only in CI:

- **Vite empties `internal/web/assets/app/`.** The `.gitkeep` there is copied back
  from `ui/public/.gitkeep`. Lose it and `go build` fails on a fresh clone with
  `pattern all:app: no matching files found`.
- **A spec change without `just generate`** builds locally on the old code and
  fails CI, which checks that `api/api.gen.go` is up to date.
- **`ParseJSON` rejects unknown fields.** A client that sends one extra field gets
  400. That is intended; decode by hand only for payloads you don't control.
- **golangci-lint older than 2.15 can't read Go 1.27 packages** and reports bogus
  type errors. Check `golangci-lint version` before trusting a lint failure.
- **Vite must listen on `127.0.0.1`.** `localhost` can bind IPv6 only, and the
  proxy on :8080 then can't reach it. `vite.config.ts` reads the address from
  `APP_FRONTEND_DEV_SERVER`.
- **SQLite WAL needs a writable directory**, not just a writable file: it creates
  `-wal` and `-shm` beside the database.
- **Editing this skill changes `apm.lock.yaml`.** Run `just install-skills` and
  commit the lock with the change, or `apm audit` reports drift.

## Done

`just test`, `just lint` and `just ui-lint` pass, and generated code was
regenerated rather than edited. When reviewing, check the change against the
rules above and name the rule each problem breaks.
