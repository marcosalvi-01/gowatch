// Package pages contains HTTP handlers that serve page contents in HTML.
package pages

import (
	"net/http"
	"strconv"

	"gowatch/internal/services"
	"gowatch/internal/ui/pages"
	"gowatch/internal/utils"
	"gowatch/logging"

	"github.com/a-h/templ"
	"github.com/go-chi/chi/v5"
)

const htmxRequestHeaderValue = "true"

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
}

func (h *Handlers) HomePage(w http.ResponseWriter, r *http.Request) {
	log.Debug("serving home page")

	if r.Header.Get("HX-Request") == htmxRequestHeaderValue {
		templ.Handler(pages.Home(), templ.WithFragments("content")).ServeHTTP(w, r)
	} else {
		templ.Handler(pages.Home()).ServeHTTP(w, r)
	}

	log.Info("home page served successfully")
}

func (h *Handlers) WatchedPage(w http.ResponseWriter, r *http.Request) {
	log.Debug("serving watched movies page")

	movies, err := h.watchedService.GetAllWatchedMoviesInDay(r.Context())
	if err != nil {
		log.Error("failed to retrieve watched movies", "error", err)
		render500Error(w, r)
		return
	}

	log.Debug("retrieved watched movies", "dayCount", len(movies))

	if r.Header.Get("HX-Request") == htmxRequestHeaderValue {
		templ.Handler(pages.Watched(movies), templ.WithFragments("content")).ServeHTTP(w, r)
	} else {
		templ.Handler(pages.Watched(movies)).ServeHTTP(w, r)
	}

	log.Info("watched movies page served successfully", "dayCount", len(movies))
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

	log.Debug("serving movie page", "movieID", id)

	movie, err := h.tmdbService.GetMovieDetails(ctx, id)
	if err != nil {
		log.Error("failed to get movie details", "movieID", id, "error", err)
		render500Error(w, r)
		return
	}

	rec, err := h.watchedService.GetWatchedMovieRecordsByID(ctx, id)
	if err != nil {
		log.Error("failed to get watched records", "movieID", id, "error", err)
		render500Error(w, r)
		return
	}

	if r.Header.Get("HX-Request") == htmxRequestHeaderValue {
		templ.Handler(pages.Movie(*movie, rec), templ.WithFragments("content")).ServeHTTP(w, r)
	} else {
		templ.Handler(pages.Movie(*movie, rec)).ServeHTTP(w, r)
	}

	log.Info("movie page served successfully", "movieID", id, "title", movie.Movie.Title)
}

func (h *Handlers) SearchPage(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")

	sanitizedQuery, err := utils.TrimAndValidateString(query, 255)
	if err != nil {
		log.Error("invalid search query", "query", query, "error", err)
		http.Error(w, "Invalid search query", http.StatusBadRequest)
		return
	}

	log.Debug("serving search page", "query", sanitizedQuery)

	results, err := h.tmdbService.SearchMovies(sanitizedQuery)
	if err != nil {
		log.Error("failed to search for movie", "query", sanitizedQuery, "error", err)
		render500Error(w, r)
		return
	}

	log.Debug("found movies", "query", sanitizedQuery, "count", len(results))

	if r.Header.Get("HX-Request") == htmxRequestHeaderValue {
		w.Header().Add("HX-Trigger", "refreshSidebar")
		templ.Handler(pages.Search("", results), templ.WithFragments("content")).ServeHTTP(w, r)
	} else {
		templ.Handler(pages.Search("", results)).ServeHTTP(w, r)
	}

	log.Info("search page served successfully", "query", sanitizedQuery, "resultCount", len(results))
}

func (h *Handlers) ListPage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	paramID := chi.URLParam(r, "id")

	id, err := strconv.ParseInt(paramID, 10, 64)
	if err != nil {
		log.Error("invalid list ID", "id", paramID, "error", err)
		http.Error(w, "Invalid list ID", http.StatusBadRequest)
		return
	}

	log.Debug("serving list page", "listID", id)

	list, err := h.listService.GetListDetails(ctx, id)
	if err != nil {
		log.Error("failed to get list details", "listID", id, "error", err)
		render500Error(w, r)
		return
	}
	log.Debug("fetched list details", "listID", id, "movieCount", len(list.Movies))

	if r.Header.Get("HX-Request") == htmxRequestHeaderValue {
		templ.Handler(pages.List(list), templ.WithFragments("content")).ServeHTTP(w, r)
	} else {
		templ.Handler(pages.List(list)).ServeHTTP(w, r)
	}

	log.Info("list page served successfully", "listID", id, "name", list.Name, "movieCount", len(list.Movies))
}

func (h *Handlers) StatsPage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	log.Debug("serving stats page")

	stats, err := h.watchedService.GetWatchedStats(ctx, 5)
	if err != nil {
		log.Error("failed to retrieve watched stats", "error", err)
		render500Error(w, r)
		return
	}

	log.Debug("retrieved watched stats", "totalWatched", stats.TotalWatched)

	if r.Header.Get("HX-Request") == htmxRequestHeaderValue {
		templ.Handler(pages.Stats(stats, 5), templ.WithFragments("content")).ServeHTTP(w, r)
	} else {
		templ.Handler(pages.Stats(stats, 5)).ServeHTTP(w, r)
	}

	log.Info("stats page served successfully")
}
