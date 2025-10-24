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
	watchedService *services.WatchedService
}

func NewHandlers(watchedService *services.WatchedService) *Handlers {
	return &Handlers{
		watchedService: watchedService,
	}
}

func (h *Handlers) RegisterRoutes(r chi.Router) {
	r.Route("/movies", func(r chi.Router) {
		r.Post("/import", h.importWatched)
		r.Get("/export", h.exportWatched)
	})
}

func (h *Handlers) exportWatched(w http.ResponseWriter, r *http.Request) {
	log.Debug("exporting watched movies")

	export, err := h.watchedService.ExportWatched(r.Context())
	if err != nil {
		log.Error("failed to export watched movies", "error", err)
		http.Error(w, "Failed to export watched movies due to an internal error.", http.StatusInternalServerError)
		return
	}

	log.Info("successfully exported watched movies")
	jsonResponse(w, http.StatusOK, export)
}

func (h *Handlers) importWatched(w http.ResponseWriter, r *http.Request) {
	log.Debug("starting watched movies import")

	var payload models.ImportWatchedMoviesLog
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		log.Error("failed to decode JSON payload", "error", err)
		http.Error(w, "failed to decode request payload", http.StatusBadRequest)
		return
	}

	totalMovies := 0
	for _, importMovie := range payload {
		totalMovies += len(importMovie.Movies)
	}
	log.Info("import request received", "totalDays", len(payload), "totalMovies", totalMovies)

	ctx := context.WithoutCancel(r.Context())

	go func() {
		log.Info("import job started")
		if err := h.watchedService.ImportWatched(ctx, payload); err != nil {
			log.Error("import job failed", "error", err)
			return
		}
		log.Info("import job finished successfully")
	}()

	jsonResponse(w, http.StatusAccepted, "import started")
}
