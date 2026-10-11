package respond

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func get() (*httptest.ResponseRecorder, Response) {
	rr := httptest.NewRecorder()
	return rr, To(rr, httptest.NewRequest(http.MethodGet, "/things/1", nil))
}

func problem(t *testing.T, rr *httptest.ResponseRecorder) ProblemDetails {
	t.Helper()
	if ct := rr.Result().Header.Get("Content-Type"); ct != "application/problem+json" {
		t.Fatalf("content-type %q", ct)
	}
	var body ProblemDetails
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	return body
}

func TestJSON(t *testing.T) {
	rr, res := get()
	res.JSON(map[string]string{"a": "b"})
	if rr.Code != http.StatusOK {
		t.Fatalf("status %d", rr.Code)
	}
	if rr.Body.String() != "{\"a\":\"b\"}\n" {
		t.Fatalf("body %q", rr.Body.String())
	}
	if ct := rr.Result().Header.Get("Content-Type"); ct != "application/json" {
		t.Fatalf("content-type %q", ct)
	}
}

func TestHeaderAndStatusInAnyOrder(t *testing.T) {
	rr, res := get()
	res.Status(http.StatusCreated).Header("X-Id", "7").Text("hi")
	got := rr.Result()
	if got.StatusCode != http.StatusCreated || got.Header.Get("X-Id") != "7" {
		t.Fatalf("status %d headers %v", got.StatusCode, got.Header)
	}
	if ct := got.Header.Get("Content-Type"); ct != "text/plain; charset=utf-8" {
		t.Fatalf("content-type %q", ct)
	}
}

func TestJSONEncodeErrorBecomes500(t *testing.T) {
	rr, res := get()
	res.Status(http.StatusCreated).JSON(make(chan int))
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("status %d", rr.Code)
	}
	problem(t, rr)
}

func TestNoContentRedirectCookie(t *testing.T) {
	rr, res := get()
	res.NoContent()
	if rr.Code != http.StatusNoContent || rr.Body.Len() != 0 {
		t.Fatalf("status %d body %q", rr.Code, rr.Body.String())
	}

	rr, res = get()
	res.Cookie(&http.Cookie{Name: "s", Value: "1"}).Redirect("/next")
	if rr.Code != http.StatusFound || rr.Header().Get("Location") != "/next" {
		t.Fatalf("status %d location %q", rr.Code, rr.Header().Get("Location"))
	}
	if rr.Header().Get("Set-Cookie") != "s=1" {
		t.Fatalf("cookie %q", rr.Header().Get("Set-Cookie"))
	}

	rr, res = get()
	res.Status(http.StatusSeeOther).Redirect("/next")
	if rr.Code != http.StatusSeeOther {
		t.Fatalf("status %d", rr.Code)
	}
}

func TestProblemBodies(t *testing.T) {
	cases := []struct {
		name   string
		call   func(Response)
		status int
		title  string
	}{
		{"bad request", func(r Response) { r.BadRequest() }, http.StatusBadRequest, "Bad request"},
		{"validation", func(r Response) { r.ValidationError() }, http.StatusBadRequest, "Validation error"},
		{"not found", func(r Response) { r.NotFound() }, http.StatusNotFound, "Not found"},
		{"conflict", func(r Response) { r.Conflict() }, http.StatusConflict, "Conflict"},
		{"unauthenticated", func(r Response) { r.NotAuthenticated() }, http.StatusUnauthorized, "Not authenticated"},
		{"unauthorized", func(r Response) { r.NotAuthorized() }, http.StatusForbidden, "Not authorized"},
		{"rate limit", func(r Response) { r.RateLimitExceeded() }, http.StatusTooManyRequests, "Rate limit exceeded"},
		{"internal", func(r Response) { r.InternalServerError() }, http.StatusInternalServerError, "Internal server error"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rr, res := get()
			tc.call(res)
			body := problem(t, rr)
			if body.Status != tc.status || body.Title != tc.title || body.Instance != "/things/1" {
				t.Fatalf("%+v", body)
			}
		})
	}
}

type noteInput struct {
	Title string   `json:"title" form:"title"`
	Stars int      `json:"stars" form:"stars"`
	Pin   bool     `json:"pin" form:"pin"`
	Tags  []string `json:"tags" form:"tag"`
}

func (n noteInput) Validate() map[string]string {
	if n.Title == "" {
		return map[string]string{"title": "is required"}
	}
	return nil
}

func post(contentType, body string) (*httptest.ResponseRecorder, Response) {
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/notes", strings.NewReader(body))
	req.Header.Set("Content-Type", contentType)
	return rr, To(rr, req)
}

func TestParseJSON(t *testing.T) {
	rr, res := post("application/json", `{"title":"a","stars":3,"tags":["x"]}`)
	var in noteInput
	if !res.ParseJSON(&in) {
		t.Fatalf("rejected: %s", rr.Body.String())
	}
	if in.Title != "a" || in.Stars != 3 || len(in.Tags) != 1 {
		t.Fatalf("%+v", in)
	}
}

func TestParseJSONRejects(t *testing.T) {
	cases := []struct {
		name, body string
		status     int
		check      func(ProblemDetails) bool
	}{
		{"empty", ``, http.StatusBadRequest, func(p ProblemDetails) bool { return p.Detail == "" }},
		{"malformed", `{"title":`, http.StatusBadRequest, func(p ProblemDetails) bool { return p.Detail == "" }},
		{"unknown field", `{"title":"a","x":1}`, http.StatusBadRequest, func(p ProblemDetails) bool { return p.Detail == "" }},
		{"two values", `{"title":"a"} {}`, http.StatusBadRequest, func(p ProblemDetails) bool { return strings.Contains(p.Detail, "single JSON value") }},
		{"wrong type", `{"title":"a","stars":"many"}`, http.StatusBadRequest, func(p ProblemDetails) bool { return p.Detail == "" && p.Errors == nil }},
		{"invalid", `{"stars":1}`, http.StatusBadRequest, func(p ProblemDetails) bool { return p.Errors["title"] == "is required" }},
		{"too large", `{"title":"` + strings.Repeat("a", defaultMaxBodyBytes) + `"}`, http.StatusBadRequest, func(p ProblemDetails) bool { return p.Detail == "" }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rr, res := post("application/json", tc.body)
			var in noteInput
			if res.ParseJSON(&in) {
				t.Fatal("accepted")
			}
			if rr.Code != tc.status {
				t.Fatalf("status %d", rr.Code)
			}
			if p := problem(t, rr); !tc.check(p) {
				t.Fatalf("%+v", p)
			}
		})
	}
}

func TestParseForm(t *testing.T) {
	form := url.Values{"title": {"a"}, "stars": {"4"}, "pin": {"on"}, "tag": {"x", "y"}}
	rr, res := post("application/x-www-form-urlencoded", form.Encode())
	var in noteInput
	if !res.ParseForm(&in) {
		t.Fatalf("rejected: %s", rr.Body.String())
	}
	if in.Title != "a" || in.Stars != 4 || !in.Pin || len(in.Tags) != 2 {
		t.Fatalf("%+v", in)
	}
}

func TestParseFormBlankAndUncheckedInputs(t *testing.T) {
	rr, res := post("application/x-www-form-urlencoded", "title=a&stars=&pin=")
	var in noteInput
	if !res.ParseForm(&in) {
		t.Fatalf("rejected: %s", rr.Body.String())
	}
	if in.Stars != 0 || in.Pin {
		t.Fatalf("%+v", in)
	}
}

func TestParseFormRejects(t *testing.T) {
	rr, res := post("application/x-www-form-urlencoded", "title=a&stars=lots")
	var in noteInput
	if res.ParseForm(&in) {
		t.Fatal("accepted")
	}
	if p := problem(t, rr); rr.Code != http.StatusBadRequest || p.Detail != "" || p.Errors != nil {
		t.Fatalf("status %d %+v", rr.Code, p)
	}

	rr, res = post("application/x-www-form-urlencoded", "stars=2")
	var fresh noteInput
	if res.ParseForm(&fresh) {
		t.Fatal("accepted")
	}
	if p := problem(t, rr); p.Errors["title"] != "is required" {
		t.Fatalf("%+v", p)
	}
}

func TestParseJSONOptions(t *testing.T) {
	rr, res := post("application/json", `{"title":"a long title"}`)
	var in noteInput
	if res.ParseJSON(&in, MaxBytes(10)) {
		t.Fatal("accepted over limit")
	}
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status %d", rr.Code)
	}

	rr, res = post("application/json", `{"title":"a","extra":1}`)
	in = noteInput{}
	if !res.ParseJSON(&in, AllowUnknownFields()) {
		t.Fatalf("rejected: %s", rr.Body.String())
	}
}

func TestParseFormOptions(t *testing.T) {
	rr, res := post("application/x-www-form-urlencoded", "title=a+long+title")
	if res.ParseForm(&noteInput{}, MaxBytes(10)) {
		t.Fatal("accepted over limit")
	}
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status %d", rr.Code)
	}
}
