package serve

import (
	"context"
	"io"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"
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
	port := freePort(t)
	ctx, cancel := context.WithCancel(context.Background())
	errc := make(chan error, 1)
	go func() {
		errc <- NewCommand().Run(ctx, []string{"serve", "--port", port})
	}()

	body := waitBody(t, "http://127.0.0.1:"+port+"/healthz")
	if body != "\"ok\"\n" {
		t.Fatalf("body %q", body)
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
	port := freePort(t)
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

func waitBody(t *testing.T, url string) string {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	var err error
	for time.Now().Before(deadline) {
		var resp *http.Response
		resp, err = http.Get(url)
		if err == nil {
			defer resp.Body.Close()
			body, readErr := io.ReadAll(resp.Body)
			if readErr != nil {
				t.Fatal(readErr)
			}
			if resp.StatusCode != http.StatusOK {
				t.Fatalf("status %d", resp.StatusCode)
			}
			return string(body)
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("GET %s: %v", url, err)
	return ""
}

func freePort(t *testing.T) string {
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
