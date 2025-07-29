// Package pages contains HTTP handlers that serve page contents in HTML.
package pages

import (
	"gowatch/internal/models"
	"gowatch/internal/services"
	"gowatch/internal/ui"
	"gowatch/logging"
	"net/http"
	"sort"
	"time"

	"github.com/a-h/templ"
	"github.com/go-chi/chi/v5"
)

var log = logging.Get("pages")

type Handlers struct {
	movieService *services.MovieService
}

func NewHandlers(movieService *services.MovieService) *Handlers {
	return &Handlers{
		movieService: movieService,
	}
}

func (h *Handlers) RegisterRoutes(r chi.Router) {
	r.Handle("/", templ.Handler(ui.HomePage()))
	r.Handle("/stats", templ.Handler(ui.StatsPage()))
	r.Handle("/search", templ.Handler(ui.SearchPage("TODO")))
	r.Get("/watched", h.WatchedPage)
}

func (h *Handlers) WatchedPage(w http.ResponseWriter, r *http.Request) {
	movies, err := h.movieService.GetAllWatchedMovies(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	templ.Handler(ui.WatchedPage(groupByDay(movies))).ServeHTTP(w, r)
}

func groupByDay(ms []models.WatchedMovie) []models.WatchedDay {
	sort.Slice(ms, func(i, j int) bool {
		return ms[i].WatchedDate.After(ms[j].WatchedDate)
	})

	var out []models.WatchedDay
	for _, m := range ms {
		d := m.WatchedDate.Truncate(24 * time.Hour) // midnight of that day
		if len(out) == 0 || !d.Equal(out[len(out)-1].Date) {
			out = append(out, models.WatchedDay{Date: d})
		}
		out[len(out)-1].Movies = append(out[len(out)-1].Movies, m.Movie)
	}
	return out
}
