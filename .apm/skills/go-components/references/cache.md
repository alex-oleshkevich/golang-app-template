# Cache

A cache keeps the result of slow work for a while. It is never the source of
truth: any value may disappear at any time, and the code must still be correct
when it does.

## Interface

```go
package cache

var ErrMiss = errors.New("cache: miss")

type Cache interface {
	Get(ctx context.Context, key string) ([]byte, error) // ErrMiss when absent or expired
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
	// Incr adds delta atomically and returns the new value. A missing key counts
	// as 0. ttl applies only when the key is created.
	Incr(ctx context.Context, key string, delta int64, ttl time.Duration) (int64, error)
	Close() error
}
```

## Helpers

Values are JSON-encoded.

- `GetAs[T](ctx, c, key) (T, error)`: decode, `ErrMiss` when absent.
- `SetAs[T](ctx, c, key, v, ttl) error`
- `GetOrDefault[T](ctx, c, key, def) (T, error)`: the cached value or `def`.
  Never stores `def`.
- `GetOrSet[T](ctx, c, key, ttl, fn) (T, error)`: the cached value, or run `fn`,
  store its result and return it. Concurrent misses on one key share one `fn`
  call (`golang.org/x/sync/singleflight`). If the cache itself fails, log the
  error and still return `fn`'s result: a broken cache slows requests down, it
  doesn't fail them. An error from `fn` is returned and nothing is stored.
- `Decr(ctx, c, key, delta, ttl)`: `c.Incr(ctx, key, -delta, ttl)`.

## Implementations

- `memory://?size=10000` (default): `github.com/maypok86/otter/v2`. Bounded by
  entry count, per-entry TTL with `SetExpiresAfter`, `Incr` through `Compute` so
  it is atomic. Right for one replica.
- `redis://host:6379/0`: `github.com/redis/go-redis/v9`, which works with Redis
  and Valkey. `Incr` sends `INCRBY` and `EXPIRE key ttl NX` in one pipeline, so
  the TTL is set once and a busy counter still expires. Use it once there is more
  than one replica.

Not used: ristretto (its admission policy can drop a `Set` silently), gocache
(a wrapper over wrappers), valkey-go (faster, far less used; switch only if Redis
is the bottleneck).

## Rules

- Keys are `<app>:v<N>:<what>:<id>`, e.g. `nisaba:v1:user:42`. Bump `N` when the
  encoded shape changes; old entries then expire on their own.
- Every `Set` has a TTL. Pick it from how stale the data may be, not from how
  expensive it is.
- Delete the key after the write that makes it stale, after the transaction
  commits.
- Counters may be evicted. Use them for approximate numbers (views, attempts),
  never for stock, balances or anything that must not reset.
- Don't cache errors or per-user data under a shared key.

## Tests

- Code that uses a cache gets `memory://` in tests.
- `cachetest.Run(t, newCache)` runs against both implementations and covers:
  miss, set and get, TTL expiry, delete, `Incr` creating at 0 and keeping its
  first TTL, `GetAs[int64]` reading a counter, and `GetOrSet` calling `fn` once
  under concurrent misses. The Redis run is skipped unless `APP_TEST_REDIS_URL`
  is set.
