// Package static contains HTTP handlers that serve static assets for the application.
package static

import (
	"embed"
	"net/http"

	"github.com/marcosalvi-01/gowatch/logging"

	"github.com/go-chi/chi/v5"
)

var log = logging.Get("static")

type Handlers struct{}

func NewHandlers() *Handlers {
	return &Handlers{}
}

//go:embed static/*
var staticFiles embed.FS

func (h *Handlers) RegisterRoutes(r chi.Router) {
	log.Debug("registering static file routes")
	r.Handle("/*", http.FileServer(http.FS(staticFiles)))
}
