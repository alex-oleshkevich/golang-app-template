package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"nisaba/internal/web/respond"
)

func TestHealth(t *testing.T) {
	rr := httptest.NewRecorder()
	Health(rr, httptest.NewRequest(http.MethodGet, "/healthz", nil))

	if rr.Code != http.StatusOK {
		t.Fatalf("status %d", rr.Code)
	}
	if rr.Body.String() != "\"ok\"\n" {
		t.Fatalf("body %q", rr.Body.String())
	}
}

func TestNotFoundBody(t *testing.T) {
	rr := httptest.NewRecorder()
	NotFound(rr, httptest.NewRequest(http.MethodGet, "/missing", nil))

	var body respond.ProblemDetails
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Status != http.StatusNotFound || body.Title != "Not found" {
		t.Fatalf("%+v", body)
	}
}
