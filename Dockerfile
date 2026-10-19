# syntax=docker/dockerfile:1

FROM node:24-alpine AS ui

WORKDIR /src/ui

RUN npm install -g pnpm@11.10.0

COPY ui/package.json ui/pnpm-lock.yaml ui/pnpm-workspace.yaml ./
RUN --mount=type=cache,target=/root/.local/share/pnpm/store pnpm install --frozen-lockfile

COPY ui/ ./
# vite.config.ts writes to ../internal/web/assets/app, the directory the Go binary embeds.
RUN pnpm build

FROM golang:1.27.1-alpine3.24 AS build

WORKDIR /src

RUN apk add --no-cache ca-certificates && mkdir /data

COPY go.mod go.sum ./
RUN go mod download

COPY . .
COPY --from=ui /src/internal/web/assets/app ./internal/web/assets/app

# .git is not in the build context, so the commit comes in as a build arg (see compose.yml).
ARG COMMIT=unknown
RUN CGO_ENABLED=0 GOOS=linux go build \
    -trimpath \
    -ldflags "-s -w -X main.commit=${COMMIT} -X main.buildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
    -o /out/nisaba ./cmd/nisaba

FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=build /out/nisaba /nisaba
# A new named volume copies this dir's ownership, so the nonroot user can write the database.
COPY --from=build --chown=nonroot:nonroot /data /data

USER nonroot:nonroot

ENV NISABA_DB=/data/nisaba.db
EXPOSE 8080

ENTRYPOINT ["/nisaba"]
CMD ["serve", "--address", "0.0.0.0", "--port", "8080", "--db-migrate"]
