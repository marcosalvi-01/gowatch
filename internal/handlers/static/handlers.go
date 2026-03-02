// Package static contains HTTP handlers that serve static assets for the application.
package static

import (
	"embed"
	"net/http"

	"github.com/marcosalvi-01/gowatch/logging"

	"github.com/go-chi/chi/v5"
)

var log = logging.Get("static")

const (
	manifestPath        = "static/manifest.webmanifest"
	manifestContentType = "application/manifest+json"
	noCacheHeaderValue  = "no-cache"
)

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

func (h *Handlers) Manifest(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", manifestContentType)
	w.Header().Set("Cache-Control", noCacheHeaderValue)
	h.serveEmbeddedFile(w, r, manifestPath)
}

func (h *Handlers) serveEmbeddedFile(w http.ResponseWriter, r *http.Request, path string) {
	content, err := staticFiles.ReadFile(path)
	if err != nil {
		log.Error("failed to read embedded file", "path", path, "error", err)
		http.NotFound(w, r)
		return
	}

	_, err = w.Write(content)
	if err != nil {
		log.Error("failed to write embedded file response", "path", path, "error", err)
	}
}
