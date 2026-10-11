package respond

import "net/http"

type ProblemDetails struct {
	Title    string            `json:"title"`
	Detail   string            `json:"detail"`
	Status   int               `json:"status"`
	Instance string            `json:"instance"`
	Errors   map[string]string `json:"errors"`
}

type ErrorOption = func(*ProblemDetails)

func (res Response) Error(details ProblemDetails, opts ...ErrorOption) {
	for _, opt := range opts {
		opt(&details)
	}
	if details.Instance == "" {
		details.Instance = res.r.URL.Path
	}
	res.
		Status(details.Status).
		Header("Content-Type", "application/problem+json").
		JSON(details)
}

func (res Response) BadRequest(opts ...ErrorOption) {
	res.Error(ProblemDetails{
		Status: http.StatusBadRequest,
		Title:  "Bad request",
	}, opts...)
}

func (res Response) NotAuthenticated(opts ...ErrorOption) {
	res.Error(ProblemDetails{
		Status: http.StatusUnauthorized,
		Title:  "Not authenticated",
	}, opts...)
}

func (res Response) NotAuthorized(opts ...ErrorOption) {
	res.Error(ProblemDetails{
		Status: http.StatusForbidden,
		Title:  "Not authorized",
	}, opts...)
}

func (res Response) NotFound(opts ...ErrorOption) {
	res.Error(ProblemDetails{
		Status: http.StatusNotFound,
		Title:  "Not found",
	}, opts...)
}

func (res Response) Conflict(opts ...ErrorOption) {
	res.Error(ProblemDetails{
		Status: http.StatusConflict,
		Title:  "Conflict",
	}, opts...)
}

func (res Response) ValidationError(opts ...ErrorOption) {
	res.Error(ProblemDetails{
		Status: http.StatusBadRequest,
		Title:  "Validation error",
	}, opts...)
}

func (res Response) RateLimitExceeded(opts ...ErrorOption) {
	res.Error(ProblemDetails{
		Status: http.StatusTooManyRequests,
		Title:  "Rate limit exceeded",
	}, opts...)
}

func (res Response) InternalServerError(opts ...ErrorOption) {
	res.Error(ProblemDetails{
		Status: http.StatusInternalServerError,
		Title:  "Internal server error",
	}, opts...)
}

func WithDetail(detail string) ErrorOption {
	return func(d *ProblemDetails) {
		d.Detail = detail
	}
}

func WithErrors(errors map[string]string) ErrorOption {
	return func(d *ProblemDetails) {
		d.Errors = errors
	}
}

func WithInstance(instance string) ErrorOption {
	return func(d *ProblemDetails) {
		d.Instance = instance
	}
}

func WithTitle(title string) ErrorOption {
	return func(d *ProblemDetails) {
		d.Title = title
	}
}
