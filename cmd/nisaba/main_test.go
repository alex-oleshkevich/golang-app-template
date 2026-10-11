package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"syscall"
	"testing"
	"time"
)

func TestLogFormatJSON(t *testing.T) {
	if code := run([]string{"nisaba", "--log-format", "json", "health", "--port", "1"}); code == 0 {
		t.Fatal("expected non-zero exit")
	}
	if got := fmt.Sprintf("%T", slog.Default().Handler()); got != "*slog.JSONHandler" {
		t.Fatalf("handler %s", got)
	}
}

func TestLogFlagsOnSubcommand(t *testing.T) {
	if code := run([]string{"nisaba", "health", "--log-format", "text", "--log-level", "warn", "--port", "1"}); code == 0 {
		t.Fatal("expected non-zero exit")
	}
	if got := fmt.Sprintf("%T", slog.Default().Handler()); got != "*tint.handler" {
		t.Fatalf("handler %s", got)
	}
	if slog.Default().Enabled(context.Background(), slog.LevelInfo) {
		t.Fatal("info should be disabled at warn level")
	}
}

func TestDefaultLogsAreText(t *testing.T) {
	if code := run([]string{"nisaba", "health", "--port", "1"}); code == 0 {
		t.Fatal("expected non-zero exit")
	}
	if got := fmt.Sprintf("%T", slog.Default().Handler()); got != "*tint.handler" {
		t.Fatalf("handler %s", got)
	}
}

func TestRejectsBadLogFlags(t *testing.T) {
	for _, args := range [][]string{
		{"nisaba", "--log-format", "yaml", "health", "--port", "1"},
		{"nisaba", "--log-level", "loud", "health", "--port", "1"},
	} {
		if code := run(args); code == 0 {
			t.Fatalf("%v: expected non-zero exit", args)
		}
	}
}

func TestHelpLeavesLoggerAlone(t *testing.T) {
	slog.SetDefault(slog.New(slog.DiscardHandler))
	if code := run([]string{"nisaba", "serve", "--help"}); code != 0 {
		t.Fatalf("exit %d", code)
	}
	if got := fmt.Sprintf("%T", slog.Default().Handler()); got != "slog.discardHandler" {
		t.Fatalf("handler %s", got)
	}
}

func TestSigtermShutdownExitsZero(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	_, port, _ := net.SplitHostPort(ln.Addr().String())
	ln.Close()

	codec := make(chan int, 1)
	go func() {
		codec <- run([]string{"nisaba", "serve", "--port", port})
	}()

	// Once the server answers, run's signal handler is installed, so SIGTERM won't kill the test binary.
	deadline := time.Now().Add(2 * time.Second)
	for {
		resp, err := http.Get("http://127.0.0.1:" + port + "/healthz")
		if err == nil {
			resp.Body.Close()
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("server never came up: %v", err)
		}
		time.Sleep(10 * time.Millisecond)
	}

	if err := syscall.Kill(syscall.Getpid(), syscall.SIGTERM); err != nil {
		t.Fatal(err)
	}
	select {
	case code := <-codec:
		if code != 0 {
			t.Fatalf("exit %d", code)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("shutdown timed out")
	}
}
