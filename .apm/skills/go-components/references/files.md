# Files

Uploads and generated files (avatars, attachments, exports) on local disk or
S3-compatible storage, behind a small interface of our own. `gocloud.dev/blob`
does the work underneath: one implementation covers every backend, chosen by URL.

## Interface

```go
package files

var (
	ErrNotFound    = errors.New("files: not found")
	ErrUnsupported = errors.New("files: not supported by this backend")
)

type Info struct {
	Key         string
	Size        int64
	ContentType string
	ModTime     time.Time
}

type Store interface {
	Put(ctx context.Context, key string, r io.Reader, contentType string) (Info, error)
	Open(ctx context.Context, key string) (io.ReadCloser, Info, error) // ErrNotFound
	Stat(ctx context.Context, key string) (Info, error)                // ErrNotFound
	Delete(ctx context.Context, key string) error                      // a missing key is not an error
	UploadURL(ctx context.Context, key string, ttl time.Duration) (string, error)   // ErrUnsupported
	DownloadURL(ctx context.Context, key string, ttl time.Duration) (string, error) // ErrUnsupported
	Close() error
}
```

The interface exists for two reasons: gocloud has one main maintainer, so the
rest of the code shouldn't import it; and it is where local and S3 get the same
upload checks.

## Backends

`Open(ctx, url)` calls `blob.OpenBucket` and wraps the bucket. Import only the
drivers you use (`_ "gocloud.dev/blob/fileblob"`), so the AWS SDK only enters the
binary with `s3blob`.

- `file:///data/files?create_dir=1&no_tmp_dir=1` (default). Writes beside each
  file a `.attrs` sidecar holding the content type.
- `s3://bucket?region=eu-central-1`. S3-compatible stores add
  `endpoint=https://...` and, for MinIO, `use_path_style=true`; Cloudflare R2
  uses `region=auto`. Credentials come from the standard `AWS_ACCESS_KEY_ID` and
  `AWS_SECRET_ACCESS_KEY`, set as deploy secrets.
- `mem://` for tests. It has no signed URLs.

## Signed URLs

They differ by backend, and the store hides that behind `UploadURL` and
`DownloadURL`:

- s3: native presigned `GET` and `PUT`. A presigned `PUT` can't require a
  content type (gocloud returns `Unimplemented`) or limit the size, so every
  direct upload is checked after the fact (below). Signed `DELETE` isn't
  supported.
- file: needs `base_url=/files&secret_key_path=/run/secrets/files_key` in the URL.
  gocloud signs; we serve. The store's HTTP handler checks the URL with
  `URLSignerHMAC.KeyFromURL` (signature and expiry), checks that the request's
  method equals the signed `method` query value, then streams the file or
  accepts the `PUT`.
- mem: both return `ErrUnsupported`. Test signed URLs on `file://` in
  `t.TempDir()`.

## Uploads

Through our server (small files, simplest):

1. Cap the body with `http.MaxBytesReader`.
2. Read the first 512 bytes (`bufio.Reader.Peek`) and get the real type with
   `http.DetectContentType`. Never trust the client's `Content-Type`.
3. Reject types outside the feature's allowlist.
4. `Put` the stream with that type. The store sets
   `DisableContentTypeDetection` and passes the type explicitly.

Direct to S3 (large files):

1. The server generates the key and returns `UploadURL(ctx, key, 15*time.Minute)`.
2. The client `PUT`s the bytes there.
3. The client calls a "complete" endpoint. The server runs `Stat` for the size
   and reads the first 512 bytes for the type; if either fails the feature's
   limits, it `Delete`s the object and answers 400.

## Rules

- Keys are generated: `<feature>/<uuidv7>.<ext>`. Never the user's filename. The
  original name, size, type and owner live in the feature's table; the store
  holds bytes only.
- Downloads set `Content-Disposition: attachment` and the stored type. User files
  are never served inline as HTML from the app's own domain.
- Delete the row first, the bytes after the commit. A crash in between leaves an
  orphaned file, which a cleanup job removes; never a row pointing at nothing.
- `Put` must close the gocloud writer and return its error: the file isn't
  written until `Close` succeeds. Cancel `ctx` to abort a write.
- Map `gcerrors.Code(err) == gcerrors.NotFound` to `ErrNotFound` inside the store.
  Nothing outside it inspects gocloud errors.

## Gotchas

- **The container's `/tmp` is a 16 MB tmpfs** (`compose.yml`). fileblob writes to
  `os.TempDir()` first, so larger uploads fail. `no_tmp_dir=1` puts the temp file
  beside the final one on the `/data` volume.
- **`List` may lag behind writes** on S3-compatible stores. Don't use it to check
  that an upload landed; use `Stat`.

## Tests

- Code that stores files gets `mem://`.
- `filestest.Run(t, newStore)` runs on `mem://` and on `file://` in `t.TempDir()`:
  put and open, stat, delete of a missing key, `ErrNotFound`, signed upload and
  download through the handler on `file://`, `ErrUnsupported` on `mem://`. The S3
  run is skipped unless `APP_TEST_FILES_URL` is set.
