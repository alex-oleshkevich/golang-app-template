# Tests

Tests live in the same package as the code, in `_test.go` files. Use a table
when there are several cases.

## Recipes

- `just test`: all Go tests
- `just testc`: the same, plus per-function coverage
- `just lint`: golangci-lint; `just lint --fix` applies what it can
- `just ui-lint`: Biome on the frontend; `just ui-lint --write` fixes formatting

## internal/testutil

- `Get(h, path, "Header", "value", ...)` sends one GET to a handler and returns
  the recorder.
- `DB(t)` gives a fresh SQLite file in `t.TempDir()`, opened with `db.Open` and
  fully migrated, closed when the test ends.
- `FreePort(t)` and `WaitGet(t, url)` start and wait for a real server.

Add a helper here only once two packages need it. Lint treats this package like
test code.

## Patterns

- HTTP: build the real router (`New(Deps{...}).server.Handler`, see
  `internal/web/router_test.go`) and assert the status, the `Content-Type` and
  the problem body, not only the status.
- Real server: pick `FreePort`, run the command in a goroutine, `WaitGet` it.
  See `internal/commands/serve/cmd_test.go` and `cmd/nisaba/main_test.go`.
- Database: `testutil.DB(t)`. Each call is a new database, so tests can't leak
  rows into each other.
