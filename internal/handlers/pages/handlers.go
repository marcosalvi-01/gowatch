// Package pages contains HTTP handlers that serve page contents in HTML.
package pages

import (
	"gowatch/internal/services"
	"gowatch/internal/ui"
	"gowatch/logging"
	"net/http"
	"strconv"

	"github.com/a-h/templ"
	"github.com/go-chi/chi/v5"
)

var log = logging.Get("pages")

type Handlers struct {
	tmdbService    *services.MovieService
	watchedService *services.WatchedService
}

func NewHandlers(tmdbService *services.MovieService, watchedService *services.WatchedService) *Handlers {
	return &Handlers{
		tmdbService:    tmdbService,
		watchedService: watchedService,
	}
}

func (h *Handlers) RegisterRoutes(r chi.Router) {
	log.Info("registering page routes")

	r.Handle("/", templ.Handler(ui.HomePage()))
	r.Handle("/stats", templ.Handler(ui.StatsPage()))
	r.Get("/search", h.SearchPage)
	r.Get("/watched", h.WatchedPage)
	r.Get("/movie/{id}", h.MoviePage)

	log.Debug("registered routes", "routes", []string{"/", "/stats", "/search", "/watched", "/movie/{id}"})
}

func (h *Handlers) WatchedPage(w http.ResponseWriter, r *http.Request) {
	log.Debug("handling watched page request", "method", r.Method, "url", r.URL.Path)

	movies, err := h.watchedService.GetAllWatchedMoviesInDay(r.Context())
	if err != nil {
		log.Error("failed to retrieve watched movies", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	log.Info("successfully retrieved watched movies", "count", len(movies))
	templ.Handler(ui.WatchedPage(movies)).ServeHTTP(w, r)
}

func (h *Handlers) MoviePage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	paramID := chi.URLParam(r, "id")

	id, err := strconv.ParseInt(paramID, 10, 64)
	if err != nil {
		http.Error(w, "Invalid movie ID", http.StatusBadRequest)
		return
	}

	movie, err := h.tmdbService.GetMovieDetails(ctx, id)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	rec, err := h.watchedService.GetWatchedMovieRecordsByID(ctx, id)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	templ.Handler(ui.MoviePage(*movie, rec)).ServeHTTP(w, r)
}

func (h *Handlers) SearchPage(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")

	results, err := h.tmdbService.SearchMovies(query)
	if err != nil {
		log.Error("failed to search for movie", "query", query, "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	templ.Handler(ui.SearchPage(query, results)).ServeHTTP(w, r)
}
