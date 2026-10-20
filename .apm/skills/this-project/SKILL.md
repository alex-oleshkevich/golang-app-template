---
name: this-project
description: >-
  Structure and conventions of this repository: where files go, HTTP routes and
  error responses, CLI commands and flags, SQLite migrations and queries, tests,
  build, CI and deploy. Use it for any change to code, config or files in this
  repo, such as adding an endpoint, command, migration, test or package, and when
  reviewing a change, even if the task doesn't mention conventions.
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

Name a file after what it holds (`frontend.go`, not `app.go`). Add a package only
for a new concern. Comments say why, not what. Environment variables start with
`APP_`.

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
- Every error is RFC 9457 problem+json written through `respond`. A 5xx logs the
  error and sends none of its text to the client.

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

## Done

`just test`, `just lint` and `just ui-lint` pass, and generated code was
regenerated rather than edited. When reviewing, check the change against the
rules above and name the rule each problem breaks.
