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
	movieService   *services.MovieService
	watchedService *services.WatchedService
}

func NewHandlers(movieService *services.MovieService, watchedService *services.WatchedService) *Handlers {
	return &Handlers{
		movieService:   movieService,
		watchedService: watchedService,
	}
}

func (h *Handlers) RegisterRoutes(r chi.Router) {
	r.Handle("/", templ.Handler(ui.HomePage()))
	r.Handle("/stats", templ.Handler(ui.StatsPage()))
	r.Handle("/search", templ.Handler(ui.SearchPage("TODO")))
	r.Get("/watched", h.WatchedPage)
	r.Get("/movie/{id}", h.MoviePage)
}

func (h *Handlers) WatchedPage(w http.ResponseWriter, r *http.Request) {
	movies, err := h.watchedService.GetWatchedDayMovies(r.Context())
	if err != nil {
		// TODO: wrap error
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	templ.Handler(ui.WatchedPage(movies)).ServeHTTP(w, r)
}

func (h *Handlers) MoviePage(w http.ResponseWriter, r *http.Request) {
	paramId := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(paramId, 10, 64)
	if err != nil {
		// TODO: wrap error
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	movie, err := h.movieService.GetMovieDetails(r.Context(), id)
	if err != nil {
		// TODO: wrap error
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	templ.Handler(ui.MoviePage(movie)).ServeHTTP(w, r)
}
