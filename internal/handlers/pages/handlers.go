// Package pages contains HTTP handlers that serve page contents in HTML.
package pages

import (
	"gowatch/internal/services"
	"gowatch/internal/ui/pages"
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
	listService    *services.ListService
}

func NewHandlers(
	tmdbService *services.MovieService,
	watchedService *services.WatchedService,
	listService *services.ListService,
) *Handlers {
	return &Handlers{
		tmdbService:    tmdbService,
		watchedService: watchedService,
		listService:    listService,
	}
}

func (h *Handlers) RegisterRoutes(r chi.Router) {
	log.Info("registering page routes")

	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/home", http.StatusFound)
	})
	r.Get("/watched", h.WatchedPage)
	r.Get("/home", h.HomePage)
	r.Get("/search", h.SearchPage)
	r.Get("/movie/{id}", h.MoviePage)
	r.Get("/list/{id}", h.ListPage)
	r.Get("/stats", h.StatsPage)

	// r.Get("/", func(w http.ResponseWriter, r *http.Request) {
	// 	http.Redirect(w, r, "/watched", http.StatusFound) // 302 redirect
	// })
	// r.Get("/search", h.SearchPage)
	// r.Get("/movie/{id}", h.MoviePage)
	// r.Get("/list/{id}", h.ListPage)
}

func (h *Handlers) HomePage(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("HX-Request") == "true" {
		templ.Handler(pages.Home(), templ.WithFragments("content")).ServeHTTP(w, r)
	} else {
		templ.Handler(pages.Home()).ServeHTTP(w, r)
	}
	log.Debug("serving home page")
}

func (h *Handlers) WatchedPage(w http.ResponseWriter, r *http.Request) {
	movies, err := h.watchedService.GetAllWatchedMoviesInDay(r.Context())
	if err != nil {
		log.Error("failed to retrieve watched movies", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	log.Debug("retrieved watched movies", "dayCount", len(movies))

	if r.Header.Get("HX-Request") == "true" {
		templ.Handler(pages.Watched(movies), templ.WithFragments("content")).ServeHTTP(w, r)
	} else {
		templ.Handler(pages.Watched(movies)).ServeHTTP(w, r)
	}
}

func (h *Handlers) MoviePage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	paramID := chi.URLParam(r, "id")

	id, err := strconv.ParseInt(paramID, 10, 64)
	if err != nil {
		log.Error("invalid movie ID", "id", paramID, "error", err)
		http.Error(w, "Invalid movie ID", http.StatusBadRequest)
		return
	}

	movie, err := h.tmdbService.GetMovieDetails(ctx, id)
	if err != nil {
		log.Error("failed to get movie details", "id", id, "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	rec, err := h.watchedService.GetWatchedMovieRecordsByID(ctx, id)
	if err != nil {
		log.Error("failed to get watched records", "id", id, "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if r.Header.Get("HX-Request") == "true" {
		templ.Handler(pages.Movie(*movie, rec), templ.WithFragments("content")).ServeHTTP(w, r)
	} else {
		templ.Handler(pages.Movie(*movie, rec)).ServeHTTP(w, r)
	}
}

func (h *Handlers) SearchPage(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")

	log.Debug("searching movies", "query", query)

	results, err := h.tmdbService.SearchMovies(query)
	if err != nil {
		log.Error("failed to search for movie", "query", query, "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	log.Debug("found movies", "count", len(results))

	if r.Header.Get("HX-Request") == "true" {
		w.Header().Add("HX-Trigger", "refreshSidebar")
		templ.Handler(pages.Search("", results), templ.WithFragments("content")).ServeHTTP(w, r)
	} else {
		templ.Handler(pages.Search("", results)).ServeHTTP(w, r)
	}
}

func (h *Handlers) ListPage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	paramID := chi.URLParam(r, "id")

	id, err := strconv.ParseInt(paramID, 10, 64)
	if err != nil {
		log.Error("invalid list ID", "id", paramID, "error", err)
		http.Error(w, "Invalid movie ID", http.StatusBadRequest)
		return
	}

	list, err := h.listService.GetListDetails(ctx, id)
	if err != nil {
		log.Error("failed to get list details", "id", id, "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	log.Debug("fetched list details", "list", list, "listID", id)

	if r.Header.Get("HX-Request") == "true" {
		templ.Handler(pages.List(list), templ.WithFragments("content")).ServeHTTP(w, r)
	} else {
		templ.Handler(pages.List(list)).ServeHTTP(w, r)
	}
}

func (h *Handlers) StatsPage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	stats, err := h.watchedService.GetWatchedStats(ctx)
	if err != nil {
		log.Error("failed to retrieve watched stats", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	log.Debug("retrieved watched stats")

	if r.Header.Get("HX-Request") == "true" {
		templ.Handler(pages.Stats(stats), templ.WithFragments("content")).ServeHTTP(w, r)
	} else {
		templ.Handler(pages.Stats(stats)).ServeHTTP(w, r)
	}
}
