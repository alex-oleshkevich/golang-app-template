package web

import (
	"io/fs"
	"net/http"
	"path"
	"strings"

	"nisaba/internal/web/assets"
	"nisaba/internal/web/handlers"
)

func frontendFS() fs.FS {
	sub, err := fs.Sub(assets.FS, "app")
	if err != nil {
		panic("embedded app assets: " + err.Error())
	}
	return sub
}

func serveFrontend(fsys fs.FS) http.HandlerFunc {
	files := http.FileServerFS(fsys)
	return func(w http.ResponseWriter, r *http.Request) {
		name := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
		if info, err := fs.Stat(fsys, name); err == nil && !info.IsDir() {
			// Vite content-hashes everything under assets/; other files keep their name across deploys.
			if strings.HasPrefix(name, "assets/") {
				immutable(files).ServeHTTP(w, r)
				return
			}
			files.ServeHTTP(w, r)
			return
		}
		if path.Ext(name) != "" {
			handlers.NotFound(w, r)
			return
		}
		if _, err := fs.Stat(fsys, "index.html"); err != nil {
			// The frontend has not been built into this binary.
			handlers.NotFound(w, r)
			return
		}
		// Revalidate every time: index.html names the hashed assets of the current deploy.
		w.Header().Set("Cache-Control", "no-cache")
		http.ServeFileFS(w, r, fsys, "index.html")
	}
}
