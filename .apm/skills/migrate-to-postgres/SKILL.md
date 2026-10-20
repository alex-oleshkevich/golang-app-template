---
name: migrate-to-postgres
description: >-
  Switch this app's database from SQLite to PostgreSQL: driver, db.Open,
  goose dialect, sqlc engine, migration and query SQL, test databases, the
  local dev database, CI, Docker and the Uncloud deploy. Use it whenever the
  user wants Postgres instead of SQLite, asks to "use a real database", needs
  several app replicas sharing one database, or mentions pgx or DATABASE_URL.
---

# Migrate to PostgreSQL

The switch keeps `*sql.DB` everywhere: pgx through its `database/sql` driver,
so goose, sqlc's generated code, `web.Deps` and every caller stay as they are.
Only the connection, the SQL dialect, and how tests and deploys get a database
change.

`<app>` below is the app slug, `basename cmd/*/`. `<tag>` is the PostgreSQL image
tag: look up the current major on postgresql.org and pin an exact tag such as
`18.1-alpine`, never `latest`.

## Before you start

Ask whether a deployed SQLite database holds data that must be kept. If it
does, stop and agree a copy plan with the user first (for example: deploy
Postgres, `db up` to create the schema, then `pgloader` with `data only`). The
steps below assume a fresh database.

## Steps

1. **Driver.** `go get github.com/jackc/pgx/v5@latest`, then drop the
   `modernc.org/sqlite` import and run `go mod tidy`.
2. **`internal/db/open.go`.** Replace the SQLite pragmas; WAL, busy timeout and
   `_txlock=immediate` have no Postgres counterpart and are not needed.

   ```go
   import _ "github.com/jackc/pgx/v5/stdlib"

   // Open connects to PostgreSQL at dsn, a URL like postgres://user:pass@host:5432/name.
   func Open(ctx context.Context, dsn string) (*sql.DB, error) {
   	db, err := sql.Open("pgx", dsn)
   	if err != nil {
   		return nil, fmt.Errorf("open postgres: %w", err)
   	}
   	// database/sql opens connections without limit; the server refuses them past max_connections.
   	db.SetMaxOpenConns(10)
   	if err := db.PingContext(ctx); err != nil {
   		_ = db.Close()
   		return nil, fmt.Errorf("open postgres: %w", err)
   	}
   	return db, nil
   }
   ```

   Never put `dsn` in an error or a log line: it carries the password.
3. **Migrator.** In `internal/db/migrate.go` use `goose.DialectPostgres`.
4. **Migrations.** Rewrite the existing files in `internal/db/migrations` in
   place, using the dialect list below. This is the one time editing an applied
   migration is right: no Postgres database has applied them yet.
5. **Queries.** Set `engine: postgresql` in `sqlc.yaml`, convert
   `internal/db/queries/*.sql`, run `just sqlc` and fix the callers whose
   generated types changed.
6. **Flags.** In `internal/commands/db/cmd.go` and
   `internal/commands/serve/cmd.go`, keep `--db` and `APP_DB`; change the usage
   to `PostgreSQL URL` and the default to
   `postgres://postgres:postgres@127.0.0.1:5432/<app>?sslmode=disable`, the
   database `just pg` starts.
7. **Local database.** Add to the `justfile`:

   ```just
   [doc("Start PostgreSQL for dev and tests on 127.0.0.1:5432, user and password postgres.")]
   pg:
       docker start <app>-pg >/dev/null 2>&1 || docker run -d --name <app>-pg -p 127.0.0.1:5432:5432 -e POSTGRES_PASSWORD=postgres -e POSTGRES_DB=<app> postgres:<tag>
   ```
8. **Tests.** In `internal/testutil/testutil.go`, add `DBURL` and open `DB`
   through it; the rest of `DB` stays.

   ```go
   // DBURL creates an empty database for this test and returns its URL. APP_TEST_DB
   // points at the server; the database is dropped when the test ends.
   func DBURL(t testing.TB) string {
   	t.Helper()
   	admin := cmp.Or(os.Getenv("APP_TEST_DB"), "postgres://postgres:postgres@127.0.0.1:5432/postgres?sslmode=disable")
   	ctx := context.Background()
   	conn, err := db.Open(ctx, admin)
   	if err != nil {
   		t.Fatalf("%v; start one with: just pg", err)
   	}
   	name := "test_" + strings.ToLower(rand.Text()) // crypto/rand
   	if _, err := conn.ExecContext(ctx, "CREATE DATABASE "+name); err != nil {
   		t.Fatal(err)
   	}
   	t.Cleanup(func() {
   		_, _ = conn.ExecContext(context.Background(), "DROP DATABASE "+name+" WITH (FORCE)")
   		_ = conn.Close()
   	})
   	u, err := url.Parse(admin)
   	if err != nil {
   		t.Fatal(err)
   	}
   	u.Path = "/" + name
   	return u.String()
   }
   ```

   Every test that runs `serve` or `db` now needs a database: pass
   `"--db", testutil.DBURL(t)`. Find them with
   `git grep -n -e '"serve"' -e '"db"' -- '*_test.go'`; one in
   `cmd/<app>/main_test.go` passes no `--db` at all today.
9. **CI.** Give the `go` job in `.github/workflows/ci.yml` a database; the
   `DBURL` default already points at it.

   ```yaml
       services:
         postgres:
           image: postgres:<tag>
           env:
             POSTGRES_PASSWORD: postgres
           ports: ["5432:5432"]
           options: >-
             --health-cmd pg_isready --health-interval 2s --health-timeout 5s --health-retries 15
   ```
10. **Image.** In the `Dockerfile`, remove `mkdir /data`, the `COPY ... /data
    /data` line with its comment, and `ENV APP_DB=...`. The image no longer
    keeps state.
11. **Deploy.** In `compose.yml`, drop the app's `/data` volume, its WAL
    comment and the `<app>-data` volume, and add:

    ```yaml
      <app>:
        environment:
          APP_DB: postgres://<app>:${APP_DB_PASSWORD:?set APP_DB_PASSWORD}@postgres:5432/<app>?sslmode=disable

      postgres:
        image: postgres:<tag>
        restart: unless-stopped
        environment:
          POSTGRES_USER: <app>
          POSTGRES_DB: <app>
          POSTGRES_PASSWORD: ${APP_DB_PASSWORD:?set APP_DB_PASSWORD}
        volumes:
          - <app>-pg:/var/lib/postgresql
        healthcheck:
          test: ["CMD", "pg_isready", "-U", "<app>"]
          interval: 5s
          timeout: 3s
          retries: 10

    volumes:
      <app>-pg:
    ```

    - The service name `postgres` is its hostname on the Uncloud mesh, and the
      mesh is WireGuard, so `sslmode=disable` is safe there.
    - Images from 18 on keep data under `/var/lib/postgresql/<major>/`, so mount
      `/var/lib/postgresql`. Before 18 the mount is `/var/lib/postgresql/data`.
    - A volume lives on one machine. On a cluster of more than one, pin the
      service with `x-machines`.
    - The password goes into a URL, so make it URL-safe:
      `openssl rand -hex 24`. Require it in `scripts/deploy.sh`
      (`: "${APP_DB_PASSWORD:?set APP_DB_PASSWORD}"`), pass
      `APP_DB_PASSWORD: ${{ secrets.APP_DB_PASSWORD }}` in the CI deploy step,
      and add it to the comment there that lists the deploy secrets.
    - The app may start before Postgres is ready. `serve --db-migrate` then
      fails and `restart: unless-stopped` retries it; that is expected.
12. **Docs and skills.** Remove every SQLite statement:
    `.apm/skills/this-project/SKILL.md` (the Database rules, the WAL gotcha),
    its `references/database.md`, `testing.md` (`DB(t)`, `just pg`) and
    `delivery.md` (`/data`, and add `<tag>` to the pinned versions: `justfile`,
    `ci.yml`, `compose.yml`), `AGENTS.md`, and the `<app>.db*` line in
    `.gitignore`. Then `just install-skills` to rewrite `apm.lock.yaml`.

## SQLite to PostgreSQL

- `INTEGER PRIMARY KEY [AUTOINCREMENT]` → `bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY`
- `INTEGER` → `bigint`. SQLite integers are 64-bit; sqlc maps Postgres
  `integer` to `int32`, which would change Go types.
- Dates in `TEXT` or `DATETIME` → `timestamptz`; `CURRENT_TIMESTAMP` and
  `datetime('now')` → `now()`
- `0`/`1` flags → `boolean`; `BLOB` → `bytea`; JSON in `TEXT` → `jsonb`, and
  `json_extract(c, '$.k')` → `c->>'k'`
- `?` and `?1` parameters → `$1`, or `sqlc.arg(name)` for readable names
- `INSERT OR IGNORE` → `ON CONFLICT DO NOTHING`; `INSERT OR REPLACE` →
  `ON CONFLICT (key) DO UPDATE SET ...`
- `LIKE` ignores ASCII case in SQLite and not in Postgres → `ILIKE` where the
  query relied on that
- `strftime` → `to_char` or `date_trunc`
- `CREATE INDEX CONCURRENTLY` needs `-- +goose NO TRANSACTION` in its file
- A duplicate key is `*pgconn.PgError` with `Code == "23505"`; check it with
  `errors.AsType[*pgconn.PgError](err)` where the handler maps it to 409

## Check

- `just pg`, then `just test`, `just lint`, `just ui-lint` and `just build` pass.
- `just cmd db up`, `just cmd db history` and `just cmd db down base` work
  against the local database.
- `docker build .` succeeds.
- `git grep -n -i -e sqlite -e modernc -e WAL` finds nothing that still
  describes the app.
