package handlers

import (
	"net/http"

	"nisaba/api"
	"nisaba/internal/web/respond"
)

type API struct{}

var _ api.StrictServerInterface = API{}

func Health(w http.ResponseWriter, r *http.Request) {
	respond.To(w, r).JSON("ok")
}

func NotFound(w http.ResponseWriter, r *http.Request) {
	respond.To(w, r).NotFound()
}
