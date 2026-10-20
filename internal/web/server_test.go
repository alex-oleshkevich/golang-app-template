package web

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"nisaba/internal/testutil"
)

func TestRunServesHealthzUntilCancel(t *testing.T) {
	port := testutil.FreePort(t)
	ctx, cancel := context.WithCancel(context.Background())
	srv := New(Deps{Config: ServerConfig{ListenAddr: "127.0.0.1", ListenPort: port}})

	errc := make(chan error, 1)
	go func() {
		errc <- srv.Run(ctx)
	}()

	status, body := testutil.WaitGet(t, "http://127.0.0.1:"+port+"/healthz")
	if status != http.StatusOK || body != "\"ok\"\n" {
		t.Fatalf("%d %q", status, body)
	}

	cancel()
	select {
	case err := <-errc:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("shutdown timed out")
	}
}

func TestRouterRecoversPanic(t *testing.T) {
	// Before New: the request logger keeps the default logger it was built with.
	var logs bytes.Buffer
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&logs, nil)))
	t.Cleanup(func() { slog.SetDefault(prev) })

	srv := New(Deps{Config: ServerConfig{ListenAddr: "127.0.0.1", ListenPort: "8080"}})
	router, ok := srv.server.Handler.(*chi.Mux)
	if !ok {
		t.Fatal("handler is not a *chi.Mux")
	}
	router.Get("/panic", func(http.ResponseWriter, *http.Request) {
		panic("boom")
	})

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/panic", nil))
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("status %d", rr.Code)
	}

	var rec map[string]any
	if err := json.Unmarshal(logs.Bytes(), &rec); err != nil {
		t.Fatalf("want one JSON request log line, got %q: %v", logs.String(), err)
	}
	if rec["level"] != "ERROR" || rec["http.response.status_code"] != float64(500) ||
		rec["error.message"] != "panic: boom" || rec["request_id"] == "" {
		t.Fatalf("request log line %v", rec)
	}
}

func TestHandlersGetADeadline(t *testing.T) {
	router, ok := New(Deps{}).server.Handler.(*chi.Mux)
	if !ok {
		t.Fatal("handler is not a *chi.Mux")
	}
	var left time.Duration
	router.Get("/deadline", func(_ http.ResponseWriter, r *http.Request) {
		if deadline, ok := r.Context().Deadline(); ok {
			left = time.Until(deadline)
		}
	})
	router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/deadline", nil))
	if left <= 0 || left > handlerTimeout || handlerTimeout >= writeTimeout {
		t.Fatalf("deadline in %s, handler timeout %s, write timeout %s", left, handlerTimeout, writeTimeout)
	}
}
