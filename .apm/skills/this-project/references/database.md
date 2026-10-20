# Database

SQLite through modernc.org/sqlite (pure Go, no CGO).

## Recipes

- `just cmd db new <name>`: create a migration file
- `just cmd db up [head|N]`, `just cmd db down [base|-N]`: apply or roll back
- `just cmd db history`: list migrations and when they ran
- `just sqlc`: regenerate query code after editing `queries/` or a migration
- `just test`: database tests run on migrated SQLite

All take `--db <file>` after the subcommand; the default is `nisaba.db` or
`APP_DB`.

## Connection

`db.Open(ctx, path)` in `internal/db/open.go` sets `busy_timeout(5000)`,
`foreign_keys(1)`, `journal_mode(WAL)`, `synchronous(NORMAL)` and
`_txlock=immediate`. The last one makes a write transaction take the write lock
when it begins, so it waits on `busy_timeout` instead of failing with
`SQLITE_BUSY` halfway through.

WAL keeps `-wal` and `-shm` files next to the database, so the whole directory
must be writable. In the container that directory is the `/data` volume.

## Migrations

- `just cmd db new <name>` creates a goose SQL file in `internal/db/migrations`
  with `-- +goose Up` and `-- +goose Down` sections. Migrations are embedded in
  the binary.
- `db up [head|N]`, `db down [base|-N]`, `db history`.
- `serve --db-migrate` (or `APP_DB_MIGRATE=true`) runs `db up` before serving.
  The Docker image starts this way.
- An applied migration is history. Fix mistakes with a new migration.

## Queries

Write `internal/db/queries/<table>.sql` with sqlc annotations
(`-- name: GetNote :one`) and run `just sqlc`. sqlc reads the schema from the
migrations and writes Go into `internal/db` (package `db`).

## Getting the database to handlers

`serve` opens the connection and builds `web.Deps`. Nothing reads the database
yet. The first handler that needs it adds the field to `web.Deps` and
`handlers.API`, and `serve` passes its connection in. Never use a package-level
connection.
