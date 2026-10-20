# Agent instructions

One Go binary: a JSON API, a CLI and a React UI embedded into it, deployed to an
Uncloud cluster.

## Skills

**Load the `this-project` skill before adding or changing code.** It holds the
layout, the HTTP, error, CLI and database rules, the `just` recipes, and the
mistakes that break builds and deploys here. Run `just install-skills` once after
cloning; the skills are listed in `apm.yml`, and the project's own live in
`.apm/skills/`.

## Repository

```
cmd/nisaba/           main: global flags, list of commands
internal/commands/    one package per CLI command
internal/web/         server, routes, handlers, responses
internal/db/          SQLite connection, migrations, queries
api/                  openapi.yaml and the code generated from it
ui/                   React sources, built into internal/web/assets/app
scripts/              shell helpers the justfile and CI call
.github/workflows/    CI and deploy
```

## Stack

Read versions from `go.mod`, `ui/package.json` and the `env` block of
`.github/workflows/ci.yml`, never from memory. Several are newer than most
documentation you may recall: Go 1.27, urfave/cli v3, chi v5, oapi-codegen strict
server, goose v3, Vite 8, React 19, Tailwind 4, Biome 2.

## Commands

`just --list` shows every recipe. Before finishing: `just test`, `just lint`,
`just ui-lint`.
