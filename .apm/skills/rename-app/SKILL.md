---
name: rename-app
description: >-
  Rename the application everywhere in this repository: Go module path, binary
  and cmd/ directory, CLI name, API title, Docker image, compose service and
  volume, UI package, docs and skills. Use it whenever the user wants to rename
  the app or project, give the template its real name, change the module path
  (e.g. to github.com/owner/repo), or says "this app is called X now".
---

# Rename the app

The app's name is spread over code, build, deploy and docs. Rename it with
search-and-replace driven by `git grep`, never from a remembered file list, and
finish with zero hits for the old name.

## Inputs

Detect the current names, then ask for whatever the user didn't give:

- **Old slug**: `basename cmd/*/` (the one directory in `cmd/`).
- **Old display name**: `info.title` in `api/openapi.yaml`.
- **New slug**: must match `^[a-z][a-z0-9-]*$`. It names the binary, the
  `cmd/` directory, the Docker image, the compose project and volume.
- **New display name**: defaults to the slug with its first letter capitalised.
- **Module path**: defaults to the slug. Use a full path such as
  `github.com/owner/repo` only when the user asks for one.
- **Already deployed?** Ask. Renaming the compose service and volume on a
  running cluster creates a new service with an **empty** volume, and the old
  one keeps running with the data.

Stop if the working tree is dirty (`git status --porcelain`), so the rename is
one reviewable diff.

## Steps

Run from the repository root. `OLD`/`NEW` are slugs, `Old`/`New` display names,
`MOD` the module path.

1. Move the entry point: `git mv cmd/OLD cmd/NEW`.
2. Replace both spellings in every tracked file:

   ```bash
   git grep -lz -e OLD -e Old | xargs -0 sed -i -e 's/Old/New/g' -e 's/OLD/NEW/g'
   ```

   This covers `go.mod`, every import, `.golangci.yml`, `justfile`,
   `.air.toml`, `Dockerfile`, `compose.yml`, `ci.yml`, `.gitignore`,
   `api/openapi.yaml`, the docs HTML titles, `ui/package.json`, `AGENTS.md`
   and the skills.
3. Only if `MOD` differs from `NEW`, switch the module path:

   ```bash
   go mod edit -module MOD
   git grep -lz '"NEW/' -- '*.go' | xargs -0 sed -i 's|"NEW/|"MOD/|g'
   ```

   Then rewrite the `^NEW/` patterns in `.golangci.yml` to `MOD`. They are
   regexes, so escape the dots: `^github\.com/owner/repo/.*`.
4. If the app is deployed, put the old volume back under the new service so it
   keeps its data: in `compose.yml` keep the volume key `OLD-data` (both under
   the service and under `volumes:`), with a comment saying why. After the
   first deploy on the new name, the old service still runs; tell the user to
   check the new one, then `uc service rm OLD`.
5. Regenerate: `just generate` (the OpenAPI spec is embedded in
   `api/api.gen.go`, so the title change lands there) and `just install-skills`
   (skill files changed, so `apm.lock.yaml` must be rewritten by the tool, never
   by hand).
6. Local files git doesn't track: rename `OLD.db*` to `NEW.db*` if present, and
   delete `bin/` and `tmp/` builds of the old binary.

## Check

- `git grep -i -e OLD` prints nothing, except a deliberate `OLD-data` volume.
- `go build ./...`, `just test`, `just lint`, `just ui-lint` and `just build`
  pass; `bin/NEW --version` runs.
- `docker build .` succeeds, since the Dockerfile builds `./cmd/NEW`.

Tell the user what is outside the repository and still has the old name: the
GitHub repository and its remote, the `APP_DOMAIN` DNS record, and CI variables.
