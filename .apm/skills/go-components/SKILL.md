---
name: go-components
description: >-
  Interfaces, library choices and wiring for infrastructure components in Go
  services: cache and counters, file storage and uploads (local disk, S3,
  signed URLs). Use it whenever a task adds, configures, replaces or tests one
  of them, or reaches for Redis, Valkey, memoization, S3, MinIO, R2 or file
  uploads, even if the user only says "cache this" or "let users attach a file".
---

# Go components

Each component follows the same shape, so a service can swap a backend by
changing one URL and tests never need the real backend.

## Shape of every component

- One small interface in its own package (`internal/cache`), `ctx` first, with
  package errors such as `ErrMiss` for the expected failures.
- Two implementations: one for production, one in memory for tests and local
  dev. Both run one shared test suite in `<pkg>test` (`cachetest.Run(t, newFn)`),
  so they can't drift apart.
- Chosen by a URL flag with an `APP_` env source (`--cache-url`,
  `APP_CACHE_URL=memory://`). `serve` builds it with `Open(ctx, url)` and passes
  it in through `web.Deps`. Never a package global.
- Closed when the command ends (`defer c.Close()`).
- Typed helpers are generic package functions over the interface, not methods,
  so every implementation stays minimal.

## Components

- Cache: `${CLAUDE_SKILL_DIR}/references/cache.md`. Read it before touching
  `internal/cache` or adding caching anywhere.
- Files: `${CLAUDE_SKILL_DIR}/references/files.md`. Read it before touching
  `internal/files`, uploads, downloads or signed URLs.
