package web

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/httplog/v3"
)

// logRequestID adds chi's request ID to the request's log line, which httplog does not log on its own.
func logRequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		httplog.SetAttrs(r.Context(), slog.String("request_id", middleware.GetReqID(r.Context())))
		next.ServeHTTP(w, r)
	})
}

// immutable marks responses as cacheable forever; mount it only on paths that carry a version or content hash.
func immutable(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		next.ServeHTTP(w, r)
	})
}
