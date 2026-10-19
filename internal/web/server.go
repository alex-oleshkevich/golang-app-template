package web

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/httplog/v3"
)

type ServerConfig struct {
	ListenAddr string
	ListenPort string
	Docs       bool
	// FrontendDevServer, when set, is proxied instead of serving the embedded build.
	FrontendDevServer *url.URL
}

const (
	handlerTimeout = 25 * time.Second
	writeTimeout   = 30 * time.Second
)

type Deps struct {
	Config ServerConfig
}

type Server struct {
	server http.Server
}

func New(deps Deps) *Server {
	router := chi.NewRouter()
	// Rightmost X-Forwarded-For entry: right for exactly one proxy hop (the Uncloud Caddy ingress).
	router.Use(middleware.ClientIPFromXFF())
	router.Use(middleware.RequestID)
	router.Use(httplog.RequestLogger(slog.Default(), &httplog.Options{
		RecoverPanics: true,
		// The container healthcheck hits this every 10s; only failures are worth a line.
		Skip: func(r *http.Request, status int) bool {
			return r.URL.Path == "/healthz" && status == http.StatusOK
		},
	}))
	router.Use(logRequestID)
	router.Use(middleware.Timeout(handlerTimeout))
	// HEAD falls through to the GET route, so `curl -I` and link checkers see the page instead of a 405.
	router.Use(middleware.GetHead)
	router.Use(middleware.Compress(5))

	installRoutes(deps, router)

	return &Server{
		server: http.Server{
			Addr:              net.JoinHostPort(deps.Config.ListenAddr, deps.Config.ListenPort),
			Handler:           router,
			ReadHeaderTimeout: 5 * time.Second,
			ReadTimeout:       15 * time.Second,
			WriteTimeout:      writeTimeout,
			IdleTimeout:       60 * time.Second,
		},
	}
}

func (s *Server) Run(ctx context.Context) error {
	slog.InfoContext(ctx, "listening for requests", "addr", s.server.Addr)

	errc := make(chan error, 1)
	go func() {
		errc <- s.server.ListenAndServe()
	}()

	select {
	case err := <-errc:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return s.server.Shutdown(shutdownCtx)
	}
}
