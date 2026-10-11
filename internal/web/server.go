package web

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/httplog/v3"
)

type ServerConfig struct {
	ListenAddr string
	ListenPort string
	Docs       bool
}

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
	router.Use(middleware.Compress(5))

	installRoutes(deps, router)

	return &Server{
		server: http.Server{
			Addr:              net.JoinHostPort(deps.Config.ListenAddr, deps.Config.ListenPort),
			Handler:           router,
			ReadHeaderTimeout: 5 * time.Second,
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
