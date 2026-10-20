// Package testutil holds helpers shared by tests across packages.
package testutil

import (
	"context"
	"database/sql"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"nisaba/internal/db"

	"github.com/pressly/goose/v3"
)

// FreePort returns a port that was free on 127.0.0.1 a moment ago.
func FreePort(t testing.TB) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	_, port, err := net.SplitHostPort(ln.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	return port
}

// Get serves one GET request to h; hdr is header name, value pairs.
func Get(h http.Handler, path string, hdr ...string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, path, nil)
	for i := 0; i+1 < len(hdr); i += 2 {
		req.Header.Set(hdr[i], hdr[i+1])
	}
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	return rr
}

// WaitGet polls url until a server answers, for tests that start a real listener.
func WaitGet(t testing.TB, url string) (status int, body string) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for {
		resp, err := http.Get(url)
		if err == nil {
			defer resp.Body.Close()
			b, readErr := io.ReadAll(resp.Body)
			if readErr != nil {
				t.Fatal(readErr)
			}
			return resp.StatusCode, string(b)
		}
		if time.Now().After(deadline) {
			t.Fatalf("GET %s: %v", url, err)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// DB opens a fresh database with every migration applied, through the same Open as the app.
func DB(t testing.TB) *sql.DB {
	t.Helper()
	ctx := context.Background()
	conn, err := db.Open(ctx, filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })

	migrator, err := db.Migrator(conn)
	if errors.Is(err, goose.ErrNoMigrations) {
		return conn
	}
	if err != nil {
		t.Fatal(err)
	}
	if _, err := migrator.Up(ctx); err != nil {
		t.Fatal(err)
	}
	return conn
}
