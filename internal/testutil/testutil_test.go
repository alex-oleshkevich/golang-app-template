package testutil

import (
	"context"
	"net/http"
	"testing"
)

func TestDBAppliesConnectionSettings(t *testing.T) {
	conn := DB(t)
	var fk int
	var mode string
	ctx := context.Background()
	if err := conn.QueryRowContext(ctx, "PRAGMA foreign_keys").Scan(&fk); err != nil {
		t.Fatal(err)
	}
	if err := conn.QueryRowContext(ctx, "PRAGMA journal_mode").Scan(&mode); err != nil {
		t.Fatal(err)
	}
	if fk != 1 || mode != "wal" {
		t.Fatalf("foreign_keys=%d journal_mode=%s", fk, mode)
	}
}

func TestDBIsFreshPerTest(t *testing.T) {
	ctx := context.Background()
	if _, err := DB(t).ExecContext(ctx, "CREATE TABLE t (id INTEGER)"); err != nil {
		t.Fatal(err)
	}
	if _, err := DB(t).ExecContext(ctx, "CREATE TABLE t (id INTEGER)"); err != nil {
		t.Fatalf("second DB shares state with the first: %v", err)
	}
}

func TestGetSendsHeaders(t *testing.T) {
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Echo", r.Header.Get("X-In"))
	})
	if got := Get(h, "/", "X-In", "v").Header().Get("X-Echo"); got != "v" {
		t.Fatalf("header %q", got)
	}
}
