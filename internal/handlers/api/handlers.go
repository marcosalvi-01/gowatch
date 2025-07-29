// Package api contains HTTP handlers that serve JSON API endpoints.
// These handlers are responsible for processing API requests, validating
// input, calling appropriate services, and returning JSON responses.
package api

import (
	"context"
	"encoding/json"
	"gowatch/internal/models"
	"gowatch/internal/services"
	"gowatch/logging"
	"net/http"

	"github.com/go-chi/chi/v5"
)

var log = logging.Get("api")

type Handlers struct {
	movieService *services.MovieService
}

func NewHandlers(movieService *services.MovieService) *Handlers {
	return &Handlers{
		movieService: movieService,
	}
}

func (h *Handlers) RegisterRoutes(r chi.Router) {
	r.Route("/movies", func(r chi.Router) {
		r.Post("/import", h.importWatched)
		r.Get("/export", h.exportWatched)
	})
}

func (h *Handlers) exportWatched(w http.ResponseWriter, r *http.Request) {
	export, err := h.movieService.ExportWatched(r.Context())
	if err != nil {
		log.Error("TODO", "error", err)
		http.Error(w, "TODO", http.StatusInternalServerError)
		return
	}

	jsonResponse(w, http.StatusOK, export)
}

func (h *Handlers) importWatched(w http.ResponseWriter, r *http.Request) {
	var payload models.WatchedMoviesLog
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		log.Error("failed to marshal JSON response", "error", err)
		http.Error(w, "failed to encode response as JSON", http.StatusInternalServerError)
		return
	}

	ctx := context.WithoutCancel(r.Context())

	go func() {
		log.Info("import job started")
		if err := h.movieService.ImportWatched(ctx, payload); err != nil {
			log.Error("import job failed", "error", err)
			return
		}
		log.Info("import job finished successfully")
	}()

	jsonResponse(w, http.StatusAccepted, "import started")
}
