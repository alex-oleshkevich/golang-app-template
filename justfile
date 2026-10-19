# Nisaba development and deploy tasks.

[doc("Serve API and frontend on :8080, both reloading on change. The argument is Vite's address.")]
dev $NISABA_FRONTEND_DEV_SERVER="http://127.0.0.1:5173":
    command -v air >/dev/null || go install github.com/air-verse/air@v1.67.4
    air

[doc("Build the frontend in ui/ into internal/web/assets/app, which the binary embeds.")]
ui-build:
    pnpm --dir ui build
    # vite empties the output dir; the placeholder keeps go:embed valid on a fresh clone.
    touch internal/web/assets/app/.gitkeep

[doc("Build the frontend, then the nisaba binary into bin/nisaba.")]
build: ui-build
    CGO_ENABLED=0 go build -trimpath -ldflags "-s -w -X main.commit=$(git rev-parse --short=7 HEAD) -X main.buildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ)" -o bin/nisaba ./cmd/nisaba

[doc("Build the image and deploy it to the Uncloud cluster. Requires NISABA_DOMAIN.")]
deploy *args:
    ./scripts/deploy.sh {{args}}

[doc("Regenerate api/api.gen.go from api/openapi.yaml.")]
generate:
    go tool oapi-codegen --config api/cfg.yaml api/openapi.yaml

[doc("Regenerate internal/db query code from internal/db/migrations and internal/db/queries.")]
sqlc:
    sqlc generate

[doc("Download the Swagger UI and Scalar files pinned in internal/web/assets/docs/VERSIONS.")]
vendor-docs:
    ./scripts/vendor-docs.sh

[doc("Run golangci-lint. Extra words are flags: just lint --fix")]
lint *args:
    golangci-lint run {{args}} ./...

[doc("Lint and format-check the frontend with Biome. Extra words are flags: just ui-lint --write")]
ui-lint *args:
    pnpm --dir ui exec biome check {{args}}

[doc("Run the test suite.")]
test:
    go test ./...

[doc("Run the test suite and print per-function coverage.")]
testc:
    go test -coverprofile=coverage.out ./...
    go tool cover -func=coverage.out

[doc("Run nisaba through go run. Extra words are arguments: just cmd serve --port 9090")]
cmd *args:
    go run ./cmd/nisaba {{args}}
