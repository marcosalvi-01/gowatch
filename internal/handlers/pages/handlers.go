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
	tmdbService    *services.TMDBService
	watchedService *services.WatchedService
}

func NewHandlers(tmdbService *services.TMDBService, watchedService *services.WatchedService) *Handlers {
	log.Info("initializing page handlers")
	return &Handlers{
		tmdbService:    tmdbService,
		watchedService: watchedService,
	}
}

func (h *Handlers) RegisterRoutes(r chi.Router) {
	log.Info("registering page routes")
	r.Handle("/", templ.Handler(ui.HomePage()))
	r.Handle("/stats", templ.Handler(ui.StatsPage()))
	r.Handle("/search", templ.Handler(ui.SearchPage("Enter movie title to search...")))
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
	paramID := chi.URLParam(r, "id")
	log.Debug("handling movie page request", "method", r.Method, "url", r.URL.Path, "movieID", paramID)

	id, err := strconv.ParseInt(paramID, 10, 64)
	if err != nil {
		log.Error("invalid movie ID parameter", "paramID", paramID, "error", err)
		http.Error(w, "Invalid movie ID", http.StatusBadRequest)
		return
	}

	movie, err := h.tmdbService.GetMovieDetails(r.Context(), id)
	if err != nil {
		log.Error("failed to retrieve movie details", "movieID", id, "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	log.Info("successfully retrieved movie details", "movieID", id, "title", movie.Movie.Title)
	templ.Handler(ui.MoviePage(*movie)).ServeHTTP(w, r)
}
