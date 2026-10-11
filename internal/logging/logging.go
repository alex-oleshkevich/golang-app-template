package logging

import (
	"fmt"
	"io"
	"log/slog"

	"github.com/lmittmann/tint"
)

type LogFormat string

const (
	LogText LogFormat = "text"
	LogJSON LogFormat = "json"
)

func ParseFormat(s string) (LogFormat, error) {
	switch f := LogFormat(s); f {
	case LogText, LogJSON:
		return f, nil
	}
	return "", fmt.Errorf("log format %q is not one of text, json", s)
}

func New(format LogFormat, level slog.Level, w io.Writer) *slog.Logger {
	if format == LogText {
		return slog.New(tint.NewTextHandler(w, &tint.Options{Level: level}))
	}
	return slog.New(slog.NewJSONHandler(w, &slog.HandlerOptions{Level: level}))
}
