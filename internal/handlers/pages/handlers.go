// Package pages contains HTTP handlers that serve page contents in HTML.
package pages

import (
	"gowatch/internal/services"
	"gowatch/internal/ui/fragments"
	"gowatch/internal/ui/pages"
	"gowatch/logging"
	"net/http"

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

	r.Get("/", h.HomePage)
	r.Get("/watched", h.WatchedPage)
	r.Get("/home", h.HomePage)

	// r.Get("/", func(w http.ResponseWriter, r *http.Request) {
	// 	http.Redirect(w, r, "/watched", http.StatusFound) // 302 redirect
	// })
	// r.Get("/search", h.SearchPage)
	// r.Get("/movie/{id}", h.MoviePage)
	// r.Get("/list/{id}", h.ListPage)
}

func (h *Handlers) HomePage(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("HX-Request") == "true" {
		fragments.Home(true).Render(r.Context(), w)
	} else {
		templ.Handler(pages.Home()).ServeHTTP(w, r)
	}
}

func (h *Handlers) WatchedPage(w http.ResponseWriter, r *http.Request) {
	movies, err := h.watchedService.GetAllWatchedMoviesInDay(r.Context())
	if err != nil {
		log.Error("failed to retrieve watched movies", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if r.Header.Get("HX-Request") == "true" {
		fragments.Watched(true, movies).Render(r.Context(), w)
	} else {
		templ.Handler(pages.Watched(movies)).ServeHTTP(w, r)
	}
}

func (h *Handlers) MoviePage(w http.ResponseWriter, r *http.Request) {
	// ctx := r.Context()
	// paramID := chi.URLParam(r, "id")
	//
	// id, err := strconv.ParseInt(paramID, 10, 64)
	// if err != nil {
	// 	http.Error(w, "Invalid movie ID", http.StatusBadRequest)
	// 	return
	// }
	//
	// movie, err := h.tmdbService.GetMovieDetails(ctx, id)
	// if err != nil {
	// 	http.Error(w, "Internal server error", http.StatusInternalServerError)
	// 	return
	// }
	//
	// rec, err := h.watchedService.GetWatchedMovieRecordsByID(ctx, id)
	// if err != nil {
	// 	http.Error(w, "Internal server error", http.StatusInternalServerError)
	// 	return
	// }
	//
	// listEntries, err := h.listService.GetAllLists(r.Context())
	// if err != nil {
	// 	log.Error("failed to retrieve list entries", "error", err)
	// 	http.Error(w, "Internal server error", http.StatusInternalServerError)
	// 	return
	// }
	//
	// watchedCount, err := h.watchedService.GetWatchedCount(r.Context())
	// if err != nil {
	// 	log.Error("failed to retrieve watched count", "error", err)
	// 	http.Error(w, "Internal server error", http.StatusInternalServerError)
	// 	return
	// }
	//
	// templ.Handler(ui.MoviePage(*movie, rec, listEntries, watchedCount)).ServeHTTP(w, r)
}

func (h *Handlers) SearchPage(w http.ResponseWriter, r *http.Request) {
	// query := r.URL.Query().Get("q")
	//
	// results, err := h.tmdbService.SearchMovies(query)
	// if err != nil {
	// 	log.Error("failed to search for movie", "query", query, "error", err)
	// 	http.Error(w, "Internal server error", http.StatusInternalServerError)
	// 	return
	// }
	//
	// listEntries, err := h.listService.GetAllLists(r.Context())
	// if err != nil {
	// 	log.Error("failed to retrieve list entries", "error", err)
	// 	http.Error(w, "Internal server error", http.StatusInternalServerError)
	// 	return
	// }
	//
	// templ.Handler(ui.SearchPage(query, results, listEntries)).ServeHTTP(w, r)
}

func (h *Handlers) ListPage(w http.ResponseWriter, r *http.Request) {
	// ctx := r.Context()
	// paramID := chi.URLParam(r, "id")
	//
	// id, err := strconv.ParseInt(paramID, 10, 64)
	// if err != nil {
	// 	http.Error(w, "Invalid movie ID", http.StatusBadRequest)
	// 	return
	// }
	//
	// list, err := h.listService.GetListDetails(ctx, id)
	// if err != nil {
	// 	http.Error(w, "Internal server error", http.StatusInternalServerError)
	// 	return
	// }
	// log.Debug("fetched list details", "list", list, "listID", id)
	//
	// listEntries, err := h.listService.GetAllLists(r.Context())
	// if err != nil {
	// 	log.Error("failed to retrieve list entries", "error", err)
	// 	http.Error(w, "Internal server error", http.StatusInternalServerError)
	// 	return
	// }
	//
	// watchedCount, err := h.watchedService.GetWatchedCount(r.Context())
	// if err != nil {
	// 	log.Error("failed to retrieve watched count", "error", err)
	// 	http.Error(w, "Internal server error", http.StatusInternalServerError)
	// 	return
	// }
	//
	// templ.Handler(ui.ListPage(list, listEntries, watchedCount)).ServeHTTP(w, r)
}
