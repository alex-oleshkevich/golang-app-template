package logging

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
)

func TestNewLoggerFormat(t *testing.T) {
	var text bytes.Buffer
	New(LogText, slog.LevelInfo, &text).Info("hello")
	if strings.Contains(text.String(), `"msg"`) || !strings.Contains(text.String(), "hello") {
		t.Fatalf("text log %q", text.String())
	}

	var js bytes.Buffer
	New(LogJSON, slog.LevelInfo, &js).Info("hello")
	if !strings.Contains(js.String(), `"msg":"hello"`) {
		t.Fatalf("json log %q", js.String())
	}
}

func TestNewLoggerLevel(t *testing.T) {
	for _, format := range []LogFormat{LogText, LogJSON} {
		var buf bytes.Buffer
		log := New(format, slog.LevelWarn, &buf)
		log.Info("dropped")
		log.Warn("kept")
		if strings.Contains(buf.String(), "dropped") || !strings.Contains(buf.String(), "kept") {
			t.Fatalf("%s log %q", format, buf.String())
		}
	}
}

func TestParseFormat(t *testing.T) {
	if _, err := ParseFormat("text"); err != nil {
		t.Fatal(err)
	}
	if _, err := ParseFormat("yaml"); err == nil {
		t.Fatal("expected error")
	}
}
