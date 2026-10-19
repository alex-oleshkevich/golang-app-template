package web

import (
	"net/http"
	"strings"
	"testing"
	"testing/fstest"
)

func TestServeFrontend(t *testing.T) {
	frontend := serveFrontend(fstest.MapFS{
		"index.html":    {Data: []byte("<!doctype html>app")},
		"vite.svg":      {Data: []byte("<svg/>")},
		"assets/app.js": {Data: []byte("console.log(1)")},
	})

	tests := []struct {
		name, path, body, cache string
		status                  int
	}{
		{"root", "/", "app", "no-cache", http.StatusOK},
		{"client route", "/settings/profile", "app", "no-cache", http.StatusOK},
		{"asset dir is a route, not a listing", "/assets/", "app", "no-cache", http.StatusOK},
		{"hashed asset", "/assets/app.js", "console.log", "public, max-age=31536000, immutable", http.StatusOK},
		{"public file", "/vite.svg", "<svg/>", "", http.StatusOK},
		{"missing asset", "/assets/gone.js", `"status":404`, "", http.StatusNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rr := serve(frontend, tt.path)
			cache := rr.Header().Get("Cache-Control")
			if rr.Code != tt.status || cache != tt.cache || !strings.Contains(rr.Body.String(), tt.body) {
				t.Fatalf("%d %q %q", rr.Code, cache, rr.Body.String())
			}
		})
	}
}

func TestServeFrontendNotBuilt(t *testing.T) {
	rr := serve(serveFrontend(fstest.MapFS{".gitkeep": {}}), "/")
	if rr.Code != http.StatusNotFound || rr.Header().Get("Content-Type") != "application/problem+json" {
		t.Fatalf("%d %q", rr.Code, rr.Header().Get("Content-Type"))
	}
}
