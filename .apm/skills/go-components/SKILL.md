---
name: go-components
description: >-
  Go service components: cache, file uploads and S3, rate limiting, login,
  sessions, API tokens, OIDC, roles, scopes, permissions, teams and
  multi-tenancy. Use it whenever a task adds, changes or tests one of them, even
  if the user only says "cache this", "let users attach a file" or "only admins
  can do this". Holds the interfaces, library picks, wiring and the traps of each
  (Redis, signed URLs, 429s, brute-force limits, protecting routes and
  team-owned data).
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
- Rate limiting: `${CLAUDE_SKILL_DIR}/references/ratelimit.md`. Read it before
  adding a limit to a route, an action or an outgoing client.
- Authentication: `${CLAUDE_SKILL_DIR}/references/auth.md`. Login methods,
  sessions, tokens, the `Identity` handlers read.
- Authorization: `${CLAUDE_SKILL_DIR}/references/authz.md`. Scopes and roles,
  declared per operation in `openapi.yaml` and checked in one place.
- Teams: `${CLAUDE_SKILL_DIR}/references/teams.md`. Memberships, the current
  team, and the `team_id` rule for team-owned data.

Read all three auth files before any auth change: they depend on each other.
Handlers never check identity or permissions themselves.
