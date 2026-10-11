package web

import (
	"cmp"
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	oapimiddleware "github.com/oapi-codegen/nethttp-middleware"

	"nisaba/internal/web/respond"
)

func validationErrorHandler(_ context.Context, err error, w http.ResponseWriter, r *http.Request, opts oapimiddleware.ErrorHandlerOpts) {
	res := respond.To(w, r)
	if opts.MatchedRoute == nil {
		res.NotFound()
		return
	}

	if fields := schemaErrors(err); len(fields) > 0 {
		res.ValidationError(respond.WithErrors(fields))
		return
	}

	res.Error(respond.ProblemDetails{
		Status: opts.StatusCode,
		Title:  http.StatusText(opts.StatusCode),
	}, respond.WithDetail(firstLine(err)))
}

// schemaErrors flattens request validation failures into field -> reason.
func schemaErrors(err error) map[string]string {
	fields := map[string]string{}
	collectSchemaErrors(err, "", fields)
	return fields
}

// param is the enclosing parameter's name; MultiError nests inside RequestError, so it is passed down.
func collectSchemaErrors(err error, param string, fields map[string]string) {
	// Walks the tree itself: errors.As would jump past RequestError and lose the parameter name.
	switch e := err.(type) { //nolint:errorlint // see above
	case openapi3.MultiError:
		for _, inner := range e {
			collectSchemaErrors(inner, param, fields)
		}
	case *openapi3filter.RequestError:
		if e.Parameter != nil {
			param = e.Parameter.Name
		}
		collectSchemaErrors(e.Err, param, fields)
	default:
		var schemaErr *openapi3.SchemaError
		if !errors.As(err, &schemaErr) {
			return
		}
		key := strings.Join(schemaErr.JSONPointer(), ".")
		if key == "" {
			key = cmp.Or(param, "body")
		}
		fields[key] = schemaErr.Reason
	}
}

func requestErrorHandler(w http.ResponseWriter, r *http.Request, _ error) {
	respond.To(w, r).BadRequest()
}

func responseErrorHandler(w http.ResponseWriter, r *http.Request, err error) {
	slog.ErrorContext(r.Context(), "handler failed", "err", err, "method", r.Method, "path", r.URL.Path)
	respond.To(w, r).InternalServerError()
}

func routerErrorHandler(w http.ResponseWriter, r *http.Request, err error) {
	respond.To(w, r).BadRequest(respond.WithDetail(firstLine(err)))
}

func firstLine(err error) string {
	line, _, _ := strings.Cut(err.Error(), "\n")
	return line
}
