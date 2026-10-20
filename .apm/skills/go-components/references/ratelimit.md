# Rate limiting

Three different jobs, three tools:

- Incoming HTTP per client IP: `github.com/go-chi/httprate` middleware.
- Limits on things that aren't IPs (login attempts per account, emails per
  user): `ratelimit.Limiter` on `cache.Cache`.
- Staying inside a third-party API's quota: `golang.org/x/time/rate`.

## Incoming HTTP: httprate

```go
// routes.go, next to the group it protects
clientIP := func(r *http.Request) (string, error) {
	return httprate.CanonicalizeIP(middleware.GetClientIP(r.Context())), nil
}
r.Use(httprate.LimitBy(300, time.Minute, clientIP,
	httprate.WithLimitHandler(func(w http.ResponseWriter, r *http.Request) {
		respond.To(w, r).RateLimitExceeded()
	}),
))
```

- Key on `middleware.GetClientIP`, which `server.go` fills from
  `X-Forwarded-For` (`ClientIPFromXFF`). Never `RemoteAddr` or the deprecated
  `KeyByIP`: behind Caddy that's the proxy, so every client shares one bucket.
- `CanonicalizeIP` reduces IPv6 to its /64, so a client can't rotate addresses
  inside its own block to dodge the limit.
- Always set `WithLimitHandler`. The default answers plain text; ours is
  problem+json. httprate has already set `Retry-After` (the full window length)
  and `X-RateLimit-*`.
- One generous limit on the whole API, stricter ones on expensive or sensitive
  routes. `httprate.JoinKeys(clientIP, httprate.KeyByEndpoint)` limits per IP and
  route together. Limits are constants next to the route group, not config.
- Keep httprate's default in-memory counter. Each replica then counts on its own,
  so the effective limit is per replica; size it with that in mind.

Don't give httprate a network-backed counter (`WithLimitCounter` over Redis):
each limiter holds one process-wide mutex across two counter calls, so every
request through it waits on two sequential round trips. At 1 ms per round trip
that caps the limiter near 500 requests per second per replica. Shared limits
belong to `ratelimit.Limiter`, used only where volume is low.

## App limits: ratelimit.Limiter

```go
package ratelimit

type Limiter interface {
	// Allow counts one hit on key and reports whether it is within the limit.
	Allow(ctx context.Context, key string) (ok bool, retryAfter time.Duration, err error)
}

// New counts hits in fixed windows with c.Incr. Keys become
// "<prefix>:<key>:<window start unix>" and expire after one window.
func New(c cache.Cache, prefix string, limit int, window time.Duration) Limiter
```

- Use it for what an IP limit can't see: login attempts per account (attackers
  rotate IPs), password-reset or invite emails per user, exports per user.
- It shares counts across replicas whenever the cache is Redis, and is
  per-process on `memory://`.
- Fixed windows allow up to twice the limit across a window boundary. That is
  fine for abuse protection; don't use it for billing or quotas.
- On a cache error, `Allow` returns the error. The caller decides: login and
  other brute-force targets fail closed (answer 429 or 503), everything else
  logs and lets the request through.
- In a handler, answer `respond.To(w, r).Header("Retry-After", secs).RateLimitExceeded()`
  with `secs` from `retryAfter`, rounded up.

## Outgoing calls: x/time/rate

One `rate.NewLimiter(rate.Limit(perSecond), burst)` per external client, built
with the client and kept on it. Call `limiter.Wait(ctx)` before each request, so
a cancelled request stops waiting. This is in-process only; set it a little
under the provider's quota divided by the replica count.

## Gotchas

- **httprate's default error handler leaks and double-writes.** On a counter error
  it answers `err.Error()` with status 428, and `RespondOnLimit` then also runs
  the limit handler. The in-memory counter never errors, which is one more reason
  to keep it.
- **Tests share limiter state** if a router is built once for several tests.
  Build a fresh router per test, or use limits high enough not to trip.

## Tests

- Middleware: build the router, send `limit+1` requests with the same
  `X-Forwarded-For`, expect 429 with `Retry-After` and a problem+json body on the
  last one; a different client IP still gets 200.
- `ratelimit.Limiter`: on `memory://`, `limit` calls allowed, the next refused
  with `retryAfter` > 0, a new window allowed again (inject the clock).
