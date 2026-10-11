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
	router.Use(middleware.Recoverer)
	router.Use(middleware.Compress(5))
	router.Use(middleware.RequestID)

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
