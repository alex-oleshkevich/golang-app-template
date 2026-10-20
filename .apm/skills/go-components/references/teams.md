# Teams: one app, many tenants

Every request acts inside one **team**. A user belongs to teams through
memberships, and each membership has a **role**; the role gives the scopes
(`authz.md`). Team-owned data is only ever read through the caller's team.

## Tables

```sql
CREATE TABLE teams (
	id         INTEGER PRIMARY KEY,
	name       TEXT NOT NULL,
	created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);

CREATE TABLE memberships (
	team_id    INTEGER NOT NULL REFERENCES teams (id) ON DELETE CASCADE,
	user_id    INTEGER NOT NULL REFERENCES users (id) ON DELETE CASCADE,
	role       TEXT NOT NULL, -- auth.Role, validated in Go with ParseRole
	created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
	PRIMARY KEY (team_id, user_id)
);
CREATE INDEX memberships_user ON memberships (user_id);
```

`role` is plain text with no `CHECK`: SQLite can't change a `CHECK` without
rebuilding the table, and the Go type already guards writes.

## Which team a request uses

A middleware after `auth.Middleware`, before the request validator, resolves the
caller's membership and fills `TeamID`, `Role` and `Scopes` on the identity:

1. the team in the `X-Team-Id` header;
2. else the team in the `team` cookie;
3. else the caller's oldest membership.

A header or cookie naming a team the caller isn't in gives **no team**. It never
falls back to another one, or a typo would act on the wrong team's data. No team
means no scopes, so every scoped operation answers 403.

Clients with several teams open (a tab each) send `X-Team-Id` on every request.
`PUT /api/v1/teams/current` stores the default in the cookie.

## Team-owned data

- Every team-owned table has
  `team_id INTEGER NOT NULL REFERENCES teams (id) ON DELETE CASCADE` and an index
  that starts with `team_id`.
- Every sqlc query on such a table filters on it:
  `-- name: GetNote :one` / `SELECT * FROM notes WHERE team_id = ? AND id = ?`.
  Handlers pass `id.TeamID` from the identity, never a team id from the request
  body or the path.
- A row of another team is therefore just not found: answer 404, not 403, so
  ids of other teams' data can't be probed.
- A new team is created by `teams.Create`, which inserts the team and the
  caller's `owner` membership in one transaction.
- A team always keeps at least one owner. Removing or demoting the last one is a
  409.

## Never

- A query on a team-owned table without `team_id = ?`.
- A team id taken from the request body, path or query for authorization.
- Falling back to another team when the requested one isn't the caller's.

## Tests

- `testutil` creates a user, a team and a membership with the role under test;
  every test that touches team data does this.
- Two teams, one row each: listing and fetching as a member of team A never
  returns team B's row, and fetching B's id answers 404.
- `X-Team-Id` naming a team the caller isn't in gives 403 on a scoped operation,
  never data from the caller's own team.
