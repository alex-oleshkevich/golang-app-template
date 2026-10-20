# Build, run, ship

## Recipes

- `just dev`: dev loop on :8080
- `just ui-build`: build the UI into `internal/web/assets/app`
- `just build`: UI, then `bin/nisaba`
- `just deploy`: build the image and `uc deploy` it (needs `APP_DOMAIN`)
- `just install-skills`: install the agent skills from `apm.yml`
- `just lint`, `just ui-lint`, `just test`: what CI runs

## Dev loop

`just dev` starts air, which rebuilds Go on change, plus Vite for the UI.
`serve` proxies the UI to Vite at `APP_FRONTEND_DEV_SERVER`, so both live on
:8080 with hot reload. If either process dies, both stop.

## Build and image

- `just build` builds the UI, then `bin/nisaba` with the commit and build time
  linked in (`main.commit`, `main.buildTime`).
- `Dockerfile`: a node stage builds the UI, a Go stage builds the binary, and the
  final image is distroless and runs as nonroot. Data lives in `/data`. The
  default command is `serve --db-migrate` on 0.0.0.0:8080.

## Deploy and CI

- `compose.yml` is the Uncloud service spec. `x-ports` publishes `APP_DOMAIN` over
  HTTPS. `just deploy` runs `scripts/deploy.sh`, which calls `uc deploy`.
- CI (`.github/workflows/ci.yml`) runs Go checks (generated code up to date, vet,
  test, lint), the UI (Biome, build) and the image build. It deploys the default
  branch once the repository variable `UNCLOUD_CONNECT` is set.

## Pinned versions

Change every copy of a version together:

- Go: `go.mod`, `Dockerfile` (`golang:1.27.1-alpine3.24`)
- Node: `Dockerfile` (`node:24-alpine`), `ci.yml` `NODE_VERSION`
- pnpm: `Dockerfile`, `ci.yml` `PNPM_VERSION`
- golangci-lint: `ci.yml` `GOLANGCI_LINT_VERSION`; local installs should match
- uc: `ci.yml` `UC_VERSION`, same as the cluster's daemons
- air: `justfile` (`dev`)
- APM: `justfile` (`install-skills`)
- Biome: `ui/package.json`
