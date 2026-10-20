# CLI

Built on urfave/cli v3. `internal/commands/serve/cmd.go` and
`internal/commands/db/cmd.go` are the models to copy.

## Recipes

- `just cmd <command> [flags]`: run the CLI through `go run`, e.g.
  `just cmd serve --port 9090`
- `just cmd --help`, `just cmd <command> --help`: check names, usage and flags
- `just build`: build `bin/nisaba` with the commit stamped into `--version`
- `just test`, `just lint`

## Add a command

1. Create `internal/commands/<name>/cmd.go` with `func NewCommand() *cli.Command`.
   Actions are unexported funcs such as `run(ctx context.Context, c *cli.Command) error`.
2. Register it in `cmd/nisaba/main.go` under `Commands`, in alphabetical order.
3. Test it in `cmd_test.go` next to it (see `serve/cmd_test.go`).

Subcommands go in a table of `{Name, Usage, ArgsUsage, Action}`, as in `db/cmd.go`.

## Flags

- Inline in the command. `serve` and `db` both declare `--db` on purpose: each
  command reads on its own, with no shared flags package.
- Validate with a `Validator` func in the same file. Its message says what is
  allowed: `port "x" is outside 1-65535`.
- A flag deploy may need to set gets `Sources: cli.EnvVars("APP_<NAME>")`. The flag
  beats the env var, which beats the default.

## Output, errors, signals

- User-facing output goes to `c.Root().Writer`, so tests can capture it. Logs go
  through the default `slog` logger, which `main` sets up from `--log-format` and
  `--log-level` (stderr).
- Return errors. `main` prints `error: <msg>` and exits 1, or with the code of a
  `cli.ExitCoder`, or 130 after Ctrl-C.
- To reuse another command, run it: `serve --db-migrate` runs
  `dbcmd.NewCommand().Run(ctx, []string{"db", "up", "--db", path})`. Don't copy
  its logic.
- `ctx` is cancelled on SIGINT and SIGTERM. Long-running work stops on
  `ctx.Done()`, as `web.Server.Run` does.
