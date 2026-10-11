package web

import (
	"encoding/json"
	"io/fs"
	"net/http"

	"nisaba/api"
	"nisaba/internal/web/assets"
	"nisaba/internal/web/respond"

	"github.com/getkin/kin-openapi/openapi3"
)

// specJSON publishes api.RawSpec, not api.GetSpecJSON, which rewrites operationIds into Go casing.
func specJSON() []byte {
	doc, err := (&openapi3.Loader{IsExternalRefsAllowed: false}).LoadFromData(api.RawSpec)
	if err != nil {
		panic("load openapi spec: " + err.Error())
	}
	body, err := doc.MarshalJSON()
	if err != nil {
		panic("render openapi spec: " + err.Error())
	}
	return body
}

func serveSpecJSON(body []byte) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		respond.To(w, r).JSON(json.RawMessage(body))
	}
}

// docsFiles serves internal/web/assets/docs under /api/docs: each UI's index.html and its vendored files.
func docsFiles() http.Handler {
	sub, err := fs.Sub(assets.FS, "docs")
	if err != nil {
		panic("embedded docs assets: " + err.Error())
	}
	return http.StripPrefix("/api/docs", http.FileServerFS(sub))
}

// immutable marks responses as cacheable forever; mount it only on paths that carry a version or content hash.
func immutable(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		next.ServeHTTP(w, r)
	})
}
