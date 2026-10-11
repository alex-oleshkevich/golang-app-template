package health

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHealthOK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/healthz" {
			t.Errorf("path %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	err := NewCommand().Run(context.Background(), []string{"health", "--port", portOf(srv)})
	if err != nil {
		t.Fatal(err)
	}
}

func TestHealthStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	t.Cleanup(srv.Close)

	err := NewCommand().Run(context.Background(), []string{"health", "--port", portOf(srv)})
	if err == nil || !strings.Contains(err.Error(), "503") {
		t.Fatalf("got %v", err)
	}
}

func TestHealthUnreachable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	port := portOf(srv)
	srv.Close()

	err := NewCommand().Run(context.Background(), []string{"health", "--port", port})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestHealthRedirectIsUnhealthy(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "http://example.com/healthz", http.StatusFound)
	}))
	t.Cleanup(srv.Close)

	err := NewCommand().Run(context.Background(), []string{"health", "--port", portOf(srv)})
	if err == nil || !strings.Contains(err.Error(), "302") {
		t.Fatalf("got %v", err)
	}
}

func TestHealthRejectsPort(t *testing.T) {
	err := NewCommand().Run(context.Background(), []string{"health", "--port", "0"})
	if err == nil || !strings.Contains(err.Error(), "outside 1-65535") {
		t.Fatalf("got %v", err)
	}
}

func portOf(srv *httptest.Server) string {
	_, port, err := net.SplitHostPort(srv.Listener.Addr().String())
	if err != nil {
		panic(err)
	}
	return port
}
