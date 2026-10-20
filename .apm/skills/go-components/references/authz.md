# Authz: what a caller may do

A **scope** is one permission, named `resource:action` (`notes:write`). A **role**
is a named set of scopes. Each operation in `api/openapi.yaml` lists the scopes it
needs, and one function checks them for every request. Handlers never check
permissions themselves.

## Declaring it in the spec

```yaml
components:
  securitySchemes:
    session: {type: apiKey, in: cookie, name: __Host-session}
    token: {type: http, scheme: bearer}

paths:
  /notes:
    post:
      security: # either login method, each needing notes:write
        - session: ["notes:write"]
        - token: ["notes:write"]
  /signup:
    post:
      security: [] # public, said explicitly
```

Every operation has a `security` key. `security: []` marks it public. An
operation without the key would silently be public, so a test rejects it.

## Scopes and roles in code

```go
package auth

type Scope string

const (
	NotesRead  Scope = "notes:read"
	NotesWrite Scope = "notes:write"
	TeamManage Scope = "team:manage"
)

var AllScopes = []Scope{NotesRead, NotesWrite, TeamManage}

type Role string

const (
	RoleOwner  Role = "owner"
	RoleAdmin  Role = "admin"
	RoleMember Role = "member"
)

var RoleScopes = map[Role][]Scope{
	RoleOwner:  AllScopes,
	RoleAdmin:  AllScopes,
	RoleMember: {NotesRead, NotesWrite},
}
```

- Each scope string in the spec has a constant here, and each constant appears
  in the spec. A test parses `api.RawSpec` and fails on any difference.
- Every role has a `RoleScopes` entry, checked by the same test.
- Scopes come only from the role. Never from a token claim, a session value or a
  column on the user.

## The check

The request validator in `routes.go` gets
`openapi3filter.Options{AuthenticationFunc: authorize}`. kin-openapi calls it once
per security alternative of the operation, with the request's context:

```go
func authorize(ctx context.Context, in *openapi3filter.AuthenticationInput) error {
	id, ok := auth.FromContext(ctx)
	if !ok {
		return auth.ErrNotAuthenticated
	}
	if id.Method != in.SecuritySchemeName {
		return auth.ErrWrongMethod
	}
	if missing := id.Missing(in.Scopes); len(missing) > 0 {
		return &auth.ScopeError{Missing: missing}
	}
	return nil
}
```

The operation passes when any alternative passes. Otherwise the middleware hands
`validationErrorHandler` a `*openapi3filter.SecurityRequirementsError` with
status 401. That type has no `Unwrap`, so the handler walks its `Errors` field
(inside an `openapi3.MultiError` when several checks fail):

- any `*auth.ScopeError` from an alternative the caller logged in with: 403
  "Not authorized", `errors` naming the missing scopes;
- otherwise: 401 "Not authenticated" with `WWW-Authenticate`.

## Changing them

- **New permission**: add the constant and `AllScopes` entry, add it to the roles
  that need it, list it on the operations in the spec.
- **New role**: a constant and a `RoleScopes` entry. Roles are stored as text, so
  no migration, but validate on write with `ParseRole`.
- **Renaming or removing a role** changes stored values: migrate
  `memberships.role` first.
- **Changing what a role can do** is an edit to `RoleScopes`; it applies on the
  next request.

## Never

- A permission check in a handler (`if id.Role == RoleAdmin`).
- An operation without a `security` key.
- A scope string written inline in Go instead of a constant.

## Tests

- A member with the role under test calls the route: allowed roles get 2xx, the
  others 403 with the missing scope in `errors`.
- Anonymous gets 401. A session-only operation called with a token gets 401.
- The spec test: every operation has `security`, every spec scope has a constant
  and back, every role has scopes.
