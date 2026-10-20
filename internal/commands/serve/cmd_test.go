package serve

import (
	"context"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"nisaba/internal/testutil"
)

func TestValidatePort(t *testing.T) {
	ok := []string{"1", "8080", "65535"}
	bad := []string{"", "0", "65536", "abc", "-1"}
	for _, port := range ok {
		if err := validatePort(port); err != nil {
			t.Errorf("port %q: %v", port, err)
		}
	}
	for _, port := range bad {
		if err := validatePort(port); err == nil {
			t.Errorf("port %q: expected error", port)
		}
	}
}

func TestServeListensOnLoopbackByDefault(t *testing.T) {
	port := testutil.FreePort(t)
	ctx, cancel := context.WithCancel(context.Background())
	errc := make(chan error, 1)
	go func() {
		errc <- NewCommand().Run(ctx, []string{
			"serve", "--port", port, "--db", filepath.Join(t.TempDir(), "t.db"), "--db-migrate",
		})
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

func TestServeRejectsEmptyAddress(t *testing.T) {
	port := testutil.FreePort(t)
	err := NewCommand().Run(context.Background(), []string{"serve", "--address", "", "--port", port})
	if err == nil || !strings.Contains(err.Error(), "address must not be empty") {
		t.Fatalf("got %v", err)
	}
}

func TestServeRejectsPortFlag(t *testing.T) {
	err := NewCommand().Run(context.Background(), []string{"serve", "--port", "0"})
	if err == nil || !strings.Contains(err.Error(), "outside 1-65535") {
		t.Fatalf("got %v", err)
	}
}
