// Package api contains HTTP handlers that serve JSON API endpoints.
// These handlers are responsible for processing API requests, validating
// input, calling appropriate services, and returning JSON responses.
package api

import (
	"context"
	"encoding/json"
	"io"
	"net/http"

	"github.com/marcosalvi-01/gowatch/db"
	"github.com/marcosalvi-01/gowatch/internal/models"
	"github.com/marcosalvi-01/gowatch/internal/services"
	"github.com/marcosalvi-01/gowatch/logging"

	"github.com/go-chi/chi/v5"
)

var log = logging.Get("api")

type Handlers struct {
	db             db.DB
	watchedService *services.WatchedService
	listService    *services.ListService
}

func NewHandlers(db db.DB, watchedService *services.WatchedService, listService *services.ListService) *Handlers {
	return &Handlers{
		db:             db,
		watchedService: watchedService,
		listService:    listService,
	}
}

func (h *Handlers) RegisterRoutes(r chi.Router) {
	r.Get("/health", h.healthCheck)
	r.Get("/export/all", h.exportAll)
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

func (h *Handlers) exportAll(w http.ResponseWriter, r *http.Request) {
	log.Debug("exporting all data")

	watchedExport, err := h.watchedService.ExportWatched(r.Context())
	if err != nil {
		log.Error("failed to export watched movies", "error", err)
		http.Error(w, "Failed to export watched movies due to an internal error.", http.StatusInternalServerError)
		return
	}

	listsExport, err := h.listService.ExportLists(r.Context())
	if err != nil {
		log.Error("failed to export lists", "error", err)
		http.Error(w, "Failed to export lists due to an internal error.", http.StatusInternalServerError)
		return
	}

	export := models.ImportAllData{
		Watched: watchedExport,
		Lists:   listsExport,
	}

	log.Info("successfully exported all data")
	jsonResponse(w, http.StatusOK, export)
}

func (h *Handlers) importWatched(w http.ResponseWriter, r *http.Request) {
	log.Debug("starting import")

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		log.Error("failed to read request body", "error", err)
		http.Error(w, "failed to read request body", http.StatusBadRequest)
		return
	}

	// First try to decode as combined format
	var allData models.ImportAllData
	if err := json.Unmarshal(bodyBytes, &allData); err != nil {
		// If that fails, try legacy watched-only format
		var watchedData models.ImportWatchedMoviesLog
		if err := json.Unmarshal(bodyBytes, &watchedData); err != nil {
			log.Error("failed to decode JSON payload", "error", err)
			http.Error(w, "failed to decode request payload", http.StatusBadRequest)
			return
		}
		allData.Watched = watchedData
	}

	totalMovies := len(allData.Watched)
	totalLists := len(allData.Lists)
	for _, importMovie := range allData.Watched {
		totalMovies += len(importMovie.Movies)
	}
	for _, list := range allData.Lists {
		totalMovies += len(list.Movies)
	}
	log.Info("import request received", "totalDays", len(allData.Watched), "totalLists", totalLists, "totalMovies", totalMovies)

	ctx := context.WithoutCancel(r.Context())

	go func() {
		log.Info("import job started")
		if err := h.watchedService.ImportAll(ctx, allData); err != nil {
			log.Error("import job failed", "error", err)
			return
		}
		log.Info("import job finished successfully")
	}()

	jsonResponse(w, http.StatusAccepted, "import started")
}

func (h *Handlers) healthCheck(w http.ResponseWriter, r *http.Request) {
	log.Debug("checking health")

	if err := h.db.Health(); err != nil {
		log.Error("health check failed", "error", err)
		http.Error(w, "unhealthy", http.StatusServiceUnavailable)
		return
	}

	jsonResponse(w, http.StatusOK, map[string]string{"status": "healthy"})
}
