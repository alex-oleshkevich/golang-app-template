# HTTP API

## Recipes

- `just generate`: regenerate `api/api.gen.go` after editing `api/openapi.yaml`
- `just dev`: API and UI on :8080, both reloading
- `just cmd serve --docs`: serve once, with the spec and docs pages at `/api/docs/`
- `just vendor-docs`: re-download the Swagger UI and Scalar files listed in
  `internal/web/assets/docs/VERSIONS`
- `just test`: router tests included

## Add an endpoint

1. Add the path and an `operationId` to `api/openapi.yaml`. Point every error
   response at `#/components/responses/Error`, so clients get problem details.
2. Run `just generate`.
3. Implement the new method on `handlers.API` in `internal/web/handlers/`, one
   file per resource (`notes.go`). `internal.go` keeps health and not-found.
4. Return the generated response type for each status. Returning a Go error
   means "unexpected": `responseErrorHandler` logs it and answers 500.
5. Add cases to `internal/web/router_test.go`, which builds the real router with
   `New(Deps{...}).server.Handler`.

Handlers get their dependencies from fields on `handlers.API`, filled from
`web.Deps` in `internal/web/routes.go`. No package globals.

## What happens to a request

Middleware order (`internal/web/server.go`): client IP from `X-Forwarded-For`,
request ID, request log (also recovers panics), request ID into the log line,
25s handler timeout, HEAD served by GET routes, compression. The server's write
timeout is 30s, longer than the handler timeout, so a slow handler still gets its
timeout response out.

- Unknown `/api/*` paths answer a problem 404 (`handlers.NotFound`). The frontend
  catch-all must never answer an API path.
- A request that fails the spec gets 400 "Validation error" with `errors`
  mapping each field to its reason (`validationErrorHandler`).
- Routes outside the spec (`/healthz`, docs, frontend) are added by hand in
  `routes.go`. Everything else comes from the spec.

## respond

Hand-written handlers start with `res := respond.To(w, r)`.

- Chain `Status`, `Header`, `Cookie`, then finish with `JSON`, `Text`, `HTML`,
  `Markdown`, `NoContent` or `Redirect`.
- Errors: `BadRequest`, `NotAuthenticated`, `NotAuthorized`, `NotFound`,
  `Conflict`, `ValidationError`, `RateLimitExceeded`, `InternalServerError`, or
  `Error(ProblemDetails{...})` for another status. Options: `WithDetail`,
  `WithErrors`, `WithTitle`, `WithInstance`. `instance` defaults to the path.
- `ParseJSON(&dst)` and `ParseForm(&dst)` reject bodies over 1 MiB (change with
  `MaxBytes`). `ParseJSON` also rejects unknown fields and trailing data. If
  `dst` has `Validate() map[string]string`, a non-empty result becomes a
  validation problem. On false the error is already written, so just return.

```go
func createNote(w http.ResponseWriter, r *http.Request) {
	res := respond.To(w, r)
	var in noteInput
	if !res.ParseJSON(&in) {
		return
	}
	res.Status(http.StatusCreated).JSON(note)
}
```

## Errors

Two layers, and only one of them knows HTTP:

- **Domain errors** come from code below the handlers. They are sentinel or
  typed errors in that package's `errors.go`, with no status, so CLI commands can
  return and check them too.
- **HTTP errors** exist only in handlers. A spec handler returns the generated
  `<Operation><Status>ApplicationProblemPlusJSONResponse`; a hand-written route
  calls `respond`. The handler turns a domain error into the response that fits.

```go
// notes/errors.go
var ErrArchived = errors.New("note is archived")

// handlers/notes.go
err := notes.Archive(ctx, a.DB, req.Id)
switch {
case errors.Is(err, notes.ErrArchived):
	return api.ArchiveNote409ApplicationProblemPlusJSONResponse{
		Status: http.StatusConflict,
		Title:  "Conflict",
		Detail: "The note is already archived.",
	}, nil
case err != nil:
	return nil, fmt.Errorf("archive note %d: %w", req.Id, err)
}
return api.ArchiveNote204Response{}, nil
```

Add a domain error only when a caller has to tell it apart from other failures.
Any other error is wrapped with `fmt.Errorf("...: %w", err)` and returned; it
becomes a logged 500.

- 4xx `detail` may tell the client what it did wrong. A 5xx never carries
  `err.Error()`; that text is for the log.
- Every error a route can return is listed in its `responses` in
  `api/openapi.yaml`, pointing at `#/components/responses/Error`.

## Never

- `http.Error`, `w.Write` of a hand-built error body, or any error that isn't a
  generated problem response or `respond`.
- `respond` imported below `internal/web`.
- `err.Error()` in a 5xx response.
- A second validation in the handler of what the spec already checks.
- An error status a route returns but `openapi.yaml` doesn't list.
- A domain error nothing checks for.
