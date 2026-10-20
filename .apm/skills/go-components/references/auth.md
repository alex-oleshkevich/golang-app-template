# Auth: who is calling

Authentication answers one question, who is calling, in a middleware that runs on
every route. It never decides what the caller may do: that is authorization
(`authz.md`), from the caller's role in a team (`teams.md`). Handlers only read
the result.

## Identity

```go
package auth

type Identity struct {
	UserID int64
	Method string  // the security scheme that proved it: "session" or "token"
	TeamID int64   // filled by the teams middleware; 0 when the caller has no team
	Role   Role    // the caller's role in TeamID
	Scopes []Scope // RoleScopes[Role]; empty without a team
}

func FromContext(ctx context.Context) (*Identity, bool) // false means anonymous
```

`Method` uses the same names as the `securitySchemes` in `api/openapi.yaml`, so
the authorization step can compare them directly.

## Authenticators

```go
// Authenticate returns (nil, nil) when its credential is absent, so the next one
// can try, and an error when the credential is present but invalid.
type Authenticator interface {
	Authenticate(r *http.Request) (*Identity, error)
}

func Middleware(chain ...Authenticator) func(http.Handler) http.Handler
```

The middleware runs the chain in order; the first identity wins. An error stops
the chain with a 401 problem and a `WWW-Authenticate` header. No identity means
anonymous, which is allowed through: routes that need a caller say so in the spec.

- **session**: `github.com/alexedwards/scs/v2` with a cookie named
  `__Host-session` (`HttpOnly`, `Secure`, `SameSite=Lax`, path `/`). For the
  browser UI on the same origin. The store is `scs/sqlite3store` on our
  connection; it only uses `database/sql`, but check it with a test against
  modernc's driver (it passes times to `julianday()`), and write a 3-method store
  of our own if it fails.
- **token**: `Authorization: Bearer <token>`. 32 random bytes, base64url, shown to
  the user once. The `tokens` table stores only `sha256(token)`, plus `user_id`,
  `name`, `expires_at` and `last_used_at`. For the CLI, scripts and other
  services. Revoking is deleting the row.

Our own tokens are opaque, never JWTs: a JWT can't be revoked before it expires.
Tokens from an external identity provider are checked with go-oidc's
`IDTokenVerifier`, which fetches the provider's keys.

## Logging in

- **Password**: `github.com/alexedwards/argon2id` with its default parameters.
  When the email is unknown, still compare against a fixed dummy hash, so the
  response time doesn't reveal which accounts exist. Both failures answer the
  same 401.
- **External provider** ("sign in with Google"): `github.com/coreos/go-oidc/v3`
  with `golang.org/x/oauth2`. Use the authorization-code flow with PKCE and a
  `state` value stored in the session. Match accounts on the provider's `sub`,
  never on email alone.
- After a successful login and on logout, call `sessions.RenewToken(ctx)` so a
  session token planted before login can't be reused.
- Rate-limit login per account and per IP (`ratelimit.md`), failing closed.

## Cross-site requests

Cookie sessions need CSRF protection. Wrap the router with
`http.NewCrossOriginProtection().Handler(router)` (Go 1.25+). It rejects unsafe cross-origin
browser requests using `Sec-Fetch-Site`, so there are no CSRF tokens to manage.
Bearer tokens aren't sent by browsers on their own, so they need nothing extra.

## Rules

- An authenticator returns `(nil, nil)` only when its credential is absent. A
  present but invalid credential is an error; returning nil would let the
  request through as anonymous.
- Handlers never build an `Identity` or read cookies and headers for auth. They
  call `auth.FromContext`.
- Never log tokens, session ids, passwords or password hashes.

## Tests

- `testutil.Login(t, db, userID)` returns a header carrying a fresh token for
  that user; tests send it with `testutil.Get`.
- A valid token reaches the handler; a garbage token gets 401, not anonymous;
  no credential on a protected route gets 401.
- A cross-origin `POST` with a session cookie gets 403 from
  `CrossOriginProtection`.
