package web

import (
	"encoding/json"
	"maps"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"nisaba/internal/web/respond"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/go-chi/chi/v5"
	oapimiddleware "github.com/oapi-codegen/nethttp-middleware"
)

func serve(h http.Handler, path string, hdr ...string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, path, nil)
	for i := 0; i < len(hdr); i += 2 {
		req.Header.Set(hdr[i], hdr[i+1])
	}
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	return rr
}

func TestRouter(t *testing.T) {
	router := New(Deps{Config: ServerConfig{Docs: true}}).server.Handler

	tests := []struct {
		name, path, ctype, body string
		status                  int
	}{
		{"health", "/healthz", "application/json", "\"ok\"\n", http.StatusOK},
		{"unknown path", "/nope", "application/problem+json", `"status":404`, http.StatusNotFound},
		{"unknown api path", "/api/nope", "application/problem+json", `"status":404`, http.StatusNotFound},
		{"unknown v1 path", "/api/v1/nope", "application/problem+json", `"status":404`, http.StatusNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rr := serve(router, tt.path)
			ct := rr.Header().Get("Content-Type")
			if rr.Code != tt.status || ct != tt.ctype || !strings.Contains(rr.Body.String(), tt.body) {
				t.Fatalf("%d %q %q", rr.Code, ct, rr.Body.String())
			}
		})
	}
}

func TestSpecDocuments(t *testing.T) {
	router := New(Deps{Config: ServerConfig{Docs: true}}).server.Handler

	rr := serve(router, "/api/v1/openapi.json")
	if rr.Code != http.StatusOK || rr.Header().Get("Content-Type") != "application/json" {
		t.Fatalf("%d %q", rr.Code, rr.Header().Get("Content-Type"))
	}
	doc, err := openapi3.NewLoader().LoadFromData(rr.Body.Bytes())
	if err != nil || doc.Info.Title != "Nisaba" {
		t.Fatalf("spec json: %v", err)
	}

}

var assetURL = regexp.MustCompile(`(?:src|href)="([^"]+)"`)

func TestDocsPages(t *testing.T) {
	router := New(Deps{Config: ServerConfig{Docs: true}}).server.Handler

	rr := serve(router, "/api/docs")
	if rr.Code != http.StatusFound || rr.Header().Get("Location") != "/api/docs/scalar/" {
		t.Fatalf("redirect: %d %q", rr.Code, rr.Header().Get("Location"))
	}

	pages := []struct {
		path       string
		wantAssets int
		config     []string
	}{
		{"/api/docs/scalar/", 1, []string{"withDefaultFonts: false", "proxyUrl: ''", "telemetry: false"}},
		{"/api/docs/swagger/", 2, []string{"validatorUrl: null"}},
	}
	for _, p := range pages {
		t.Run(p.path, func(t *testing.T) {
			rr := serve(router, strings.TrimSuffix(p.path, "/"))
			if rr.Code != http.StatusMovedPermanently || rr.Header().Get("Location") == "" {
				t.Fatalf("no trailing-slash redirect: %d", rr.Code)
			}

			rr = serve(router, p.path)
			if rr.Code != http.StatusOK || !strings.HasPrefix(rr.Header().Get("Content-Type"), "text/html") ||
				rr.Header().Get("Cache-Control") != "" {
				t.Fatalf("%d %q %q", rr.Code, rr.Header().Get("Content-Type"), rr.Header().Get("Cache-Control"))
			}
			html := rr.Body.String()
			for _, want := range p.config {
				if !strings.Contains(html, want) {
					t.Errorf("page lacks %q", want)
				}
			}
			if strings.Contains(html, "http://") || strings.Contains(html, "https://") {
				t.Error("page references an absolute URL")
			}
			// Asset links are relative to the page; a stale version in index.html after `just vendor-docs` 404s here.
			urls := assetURL.FindAllStringSubmatch(html, -1)
			if len(urls) != p.wantAssets {
				t.Fatalf("assets %v", urls)
			}
			for _, m := range urls {
				ar := serve(router, p.path+m[1])
				ct := ar.Header().Get("Content-Type")
				if ar.Code != http.StatusOK || (!strings.Contains(ct, "javascript") && !strings.Contains(ct, "css")) {
					t.Errorf("%s: %d %q", m[1], ar.Code, ct)
				}
			}
		})
	}
}

func TestDocsAssetsGzipAndCache(t *testing.T) {
	router := New(Deps{Config: ServerConfig{Docs: true}}).server.Handler
	html := serve(router, "/api/docs/scalar/").Body.String()
	m := assetURL.FindStringSubmatch(html)
	if m == nil {
		t.Fatal("no asset in scalar page")
	}
	rr := serve(router, "/api/docs/scalar/"+m[1], "Accept-Encoding", "gzip")
	enc, cache := rr.Header().Get("Content-Encoding"), rr.Header().Get("Cache-Control")
	if rr.Code != http.StatusOK || enc != "gzip" || !strings.Contains(cache, "immutable") {
		t.Fatalf("%d %q %q", rr.Code, enc, cache)
	}
}

func TestDocsOffByDefault(t *testing.T) {
	router := New(Deps{}).server.Handler
	paths := []string{
		"/api/docs/scalar/", "/api/docs/swagger/", "/api/docs",
		"/api/v1/openapi.json",
	}
	for _, path := range paths {
		rr := serve(router, path)
		if rr.Code != http.StatusNotFound || rr.Header().Get("Content-Type") != "application/problem+json" {
			t.Errorf("%s: %d %q", path, rr.Code, rr.Header().Get("Content-Type"))
		}
	}
}

// The seed spec has no inputs to fail, so run the real handler against a spec with a required query param and body.
func TestValidationErrorHandler(t *testing.T) {
	spec, err := openapi3.NewLoader().LoadFromData([]byte(`
openapi: 3.0.3
info: {title: t, version: "1"}
servers: [{url: /api/v1}]
paths:
  /things:
    post:
      parameters:
        - {name: limit, in: query, schema: {type: integer, maximum: 10}}
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
              required: [name]
              properties:
                name: {type: string, minLength: 3}
      responses:
        "200": {description: ok}
`))
	if err != nil {
		t.Fatal(err)
	}
	r := chi.NewRouter()
	r.Route("/api/v1", func(r chi.Router) {
		r.Use(oapimiddleware.OapiRequestValidatorWithOptions(spec, &oapimiddleware.Options{
			SilenceServersWarning: true,
			ErrorHandlerWithOpts:  validationErrorHandler,
			Options:               openapi3filter.Options{MultiError: true},
		}))
		r.Post("/things", func(http.ResponseWriter, *http.Request) {})
	})

	tests := map[string]struct {
		url    string
		fields map[string]string
	}{
		"body": {"/api/v1/things", map[string]string{"name": "minimum string length is 3"}},
		"body and param": {"/api/v1/things?limit=99", map[string]string{
			"name":  "minimum string length is 3",
			"limit": "number must be at most 10",
		}},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, tc.url, strings.NewReader(`{"name":"x"}`))
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()
			r.ServeHTTP(rr, req)
			var p respond.ProblemDetails
			if err := json.Unmarshal(rr.Body.Bytes(), &p); err != nil {
				t.Fatal(err)
			}
			if rr.Code != http.StatusBadRequest || rr.Header().Get("Content-Type") != "application/problem+json" ||
				!maps.Equal(p.Errors, tc.fields) {
				t.Fatalf("%d %q %+v", rr.Code, rr.Header().Get("Content-Type"), p)
			}
		})
	}
}
