package htmx

import (
	"gowatch/internal/services"
	"gowatch/internal/ui/components/toast"
	"gowatch/logging"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
)

var log = logging.Get("htmx handlers")

type Handlers struct {
	watchedService *services.WatchedService
}

func NewHandlers(watchedService *services.WatchedService) *Handlers {
	return &Handlers{
		watchedService: watchedService,
	}
}

func (h *Handlers) RegisterRoutes(r chi.Router) {
	r.Post("/movies/watched", h.AddWatchedMovie)
}

func (h *Handlers) AddWatchedMovie(w http.ResponseWriter, r *http.Request) {
	movieIDParam := r.FormValue("movie_id")
	movieID, err := strconv.Atoi(movieIDParam)
	if err != nil {
		http.Error(w, "Bad movie id parameter", http.StatusBadRequest)
		return
	}

	watchedDateParam := r.FormValue("watched_date")
	watchedDate, err := time.Parse("2006-01-02", watchedDateParam)
	if err != nil {
		http.Error(w, "Bad watched_date param", http.StatusBadRequest)
		return
	}

	watchedAtTheater := r.FormValue("watched_at_theater") == "true"

	err = h.watchedService.AddWatched(r.Context(), int64(movieID), watchedDate, watchedAtTheater)
	if err != nil {
		log.Error("failed to add new watched movie", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	toast.Toast(toast.Props{
		Title:         "Movie Added Successfully",
		Variant:       toast.VariantSuccess,
		Position:      toast.PositionTopRight,
		Duration:      2000,
		ShowIndicator: true,
		Icon:          true,
	}).Render(r.Context(), w)
}
