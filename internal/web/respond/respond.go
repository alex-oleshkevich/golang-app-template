package respond

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5/middleware"
)

type Response struct {
	w      http.ResponseWriter
	r      *http.Request
	status int
}

func To(w http.ResponseWriter, r *http.Request) Response {
	return Response{w: w, r: r, status: http.StatusOK}
}

func (res Response) Header(key, value string) Response {
	res.w.Header().Set(key, value)
	return res
}

func (res Response) Status(status int) Response {
	res.status = status
	return res
}

func (res Response) Cookie(c *http.Cookie) Response {
	http.SetCookie(res.w, c)
	return res
}

func (res Response) JSON(data any) {
	// Encode before WriteHeader, so a failure can still become a clean 500.
	body, err := json.Marshal(data)
	if err != nil {
		// An unencodable value is a programming error, not something the handler can recover from.
		res.log(slog.LevelError, "encode json response", err)
		res.w.Header().Del("Content-Type")
		res.InternalServerError()
		return
	}
	res.write("application/json", append(body, '\n'))
}

func (res Response) Text(body string) {
	res.write("text/plain; charset=utf-8", []byte(body))
}

func (res Response) HTML(body string) {
	res.write("text/html; charset=utf-8", []byte(body))
}

func (res Response) Markdown(body string) {
	res.write("text/markdown; charset=utf-8", []byte(body))
}

func (res Response) NoContent() {
	res.w.WriteHeader(http.StatusNoContent)
}

func (res Response) Redirect(url string) {
	status := res.status
	if status < 300 || status > 399 {
		status = http.StatusFound
	}
	http.Redirect(res.w, res.r, url, status)
}

func (res Response) write(contentType string, body []byte) {
	if res.w.Header().Get("Content-Type") == "" {
		res.w.Header().Set("Content-Type", contentType)
	}
	res.w.WriteHeader(res.status)
	if _, err := res.w.Write(body); err != nil {
		// Almost always a client that went away; nothing a handler could do about it.
		res.log(slog.LevelDebug, "write response", err)
	}
}

func (res Response) log(level slog.Level, msg string, err error) {
	ctx := res.r.Context()
	slog.Log(ctx, level, msg,
		"err", err,
		"method", res.r.Method,
		"path", res.r.URL.Path,
		"request_id", middleware.GetReqID(ctx),
	)
}
