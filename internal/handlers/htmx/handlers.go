package htmx

import (
	"fmt"
	"gowatch/internal/services"
	"gowatch/internal/ui/components/page"
	"gowatch/internal/ui/components/toast"
	"gowatch/logging"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
)

var log = logging.Get("htmx handlers")

type Handlers struct {
	watchedService *services.WatchedService
	listService    *services.ListService
}

func NewHandlers(watchedService *services.WatchedService, listService *services.ListService) *Handlers {
	return &Handlers{
		watchedService: watchedService,
		listService:    listService,
	}
}

func (h *Handlers) RegisterRoutes(r chi.Router) {
	r.Post("/movies/watched", h.AddWatchedMovie)
	r.Post("/lists/create", h.CreateList)
	r.Get("/lists", h.GetAllLists)
	r.Get("/watchedCount", h.GetWatchedCount)
	r.Post("/list/add", h.AddMovieToList)
}

func (h *Handlers) AddWatchedMovie(w http.ResponseWriter, r *http.Request) {
	movieIDParam := r.FormValue("movie_id")
	movieID, err := strconv.Atoi(movieIDParam)
	if err != nil {
		log.Error("invalid movie ID parameter", "movieID", movieIDParam, "error", err)
		toast.Toast(toast.Props{
			Title:         "Invalid Movie ID",
			Description:   "The movie ID provided is not valid. Please try again.",
			Variant:       toast.VariantError,
			Position:      toast.PositionTopRight,
			Duration:      4000,
			ShowIndicator: true,
			Icon:          true,
		}).Render(r.Context(), w)
		return
	}

	watchedDateParam := r.FormValue("watched_date")
	if watchedDateParam == "" {
		log.Error("missing watched_date parameter")
		toast.Toast(toast.Props{
			Title:         "Missing Date",
			Description:   "Please select the date when you watched the movie.",
			Variant:       toast.VariantError,
			Position:      toast.PositionTopRight,
			Duration:      4000,
			ShowIndicator: true,
			Icon:          true,
		}).Render(r.Context(), w)
		return
	}

	watchedDate, err := time.Parse("2006-01-02", watchedDateParam)
	if err != nil {
		log.Error("invalid watched_date parameter", "watchedDate", watchedDateParam, "error", err)
		toast.Toast(toast.Props{
			Title:         "Invalid Date Format",
			Description:   "The date format is invalid. Please select a valid date.",
			Variant:       toast.VariantError,
			Position:      toast.PositionTopRight,
			Duration:      4000,
			ShowIndicator: true,
			Icon:          true,
		}).Render(r.Context(), w)
		return
	}

	if watchedDate.After(time.Now()) {
		log.Error("watched date is in the future", "watchedDate", watchedDate)
		toast.Toast(toast.Props{
			Title:         "Future Date Not Allowed",
			Description:   "You cannot mark a movie as watched for a future date.",
			Variant:       toast.VariantError,
			Position:      toast.PositionTopRight,
			Duration:      4000,
			ShowIndicator: true,
			Icon:          true,
		}).Render(r.Context(), w)
		return
	}

	watchedAtTheater := r.FormValue("watched_at_theater") != ""

	err = h.watchedService.AddWatched(r.Context(), int64(movieID), watchedDate, watchedAtTheater)
	if err != nil {
		log.Error("failed to add new watched movie", "movieID", movieID, "watchedDate", watchedDate, "theater", watchedAtTheater, "error", err)
		// TODO: Check for specific error types

		toast.Toast(toast.Props{
			Title:         "Unexpected error",
			Description:   "An unexpected error occurred, please try again",
			Variant:       toast.VariantError,
			Position:      toast.PositionTopRight,
			Duration:      5000,
			ShowIndicator: true,
			Icon:          true,
		}).Render(r.Context(), w)
		return
	}

	var successMessage string
	if watchedAtTheater {
		successMessage = fmt.Sprintf("Movie marked as watched on %s", watchedDate.Format("Jan 2, 2006"))
	} else {
		successMessage = fmt.Sprintf("Movie marked as watched on %s", watchedDate.Format("Jan 2, 2006"))
	}

	w.Header().Add("HX-Trigger", "newWatched")

	toast.Toast(toast.Props{
		Title:         "Movie Added Successfully",
		Description:   successMessage,
		Variant:       toast.VariantSuccess,
		Position:      toast.PositionTopRight,
		Duration:      3000,
		ShowIndicator: true,
		Icon:          true,
	}).Render(r.Context(), w)
}

func (h *Handlers) CreateList(w http.ResponseWriter, r *http.Request) {
	title := r.FormValue("title")
	description := r.FormValue("description")

	if title == "" {
		log.Error("missing list title")
		toast.Toast(toast.Props{
			Title:         "Missing Title",
			Description:   "Please provide a title for your list.",
			Variant:       toast.VariantError,
			Position:      toast.PositionTopRight,
			Duration:      4000,
			ShowIndicator: true,
			Icon:          true,
		}).Render(r.Context(), w)
		return
	}

	if len(description) > 500 {
		log.Error("description too long", "length", len(description))
		toast.Toast(toast.Props{
			Title:         "Description Too Long",
			Description:   "Please keep the description under 500 characters.",
			Variant:       toast.VariantError,
			Position:      toast.PositionTopRight,
			Duration:      4000,
			ShowIndicator: true,
			Icon:          true,
		}).Render(r.Context(), w)
		return
	}

	err := h.listService.CreateList(r.Context(), title, description)
	if err != nil {
		log.Error("failed to create list", "title", title, "description", description, "error", err)
		toast.Toast(toast.Props{
			Title:         "Unexpected Error",
			Description:   "An unexpected error occurred, please try again.",
			Variant:       toast.VariantError,
			Position:      toast.PositionTopRight,
			Duration:      5000,
			ShowIndicator: true,
			Icon:          true,
		}).Render(r.Context(), w)
		return
	}

	// trigger the newList event to update the list of lists in the sidebar
	w.Header().Add("HX-Trigger", "newList")

	toast.Toast(toast.Props{
		Title:         "List Created Successfully",
		Description:   fmt.Sprintf("List \"%s\" has been created.", title),
		Variant:       toast.VariantSuccess,
		Position:      toast.PositionTopRight,
		Duration:      3000,
		ShowIndicator: true,
		Icon:          true,
	}).Render(r.Context(), w)
}

func (h *Handlers) GetAllLists(w http.ResponseWriter, r *http.Request) {
	lists, err := h.listService.GetAllLists(r.Context())
	if err != nil {
		log.Error("failed to get all lists", "error", err)
		toast.Toast(toast.Props{
			Title:         "Unexpected Error",
			Description:   "An unexpected error occurred",
			Variant:       toast.VariantError,
			Position:      toast.PositionTopRight,
			Duration:      5000,
			ShowIndicator: true,
			Icon:          true,
		}).Render(r.Context(), w)
		return
	}

	page.SidebarListsList("", lists).Render(r.Context(), w)
}

func (h *Handlers) GetWatchedCount(w http.ResponseWriter, r *http.Request) {
	count, err := h.watchedService.GetWatchedCount(r.Context())
	if err != nil {
		log.Error("failed to get watched movie count", "error", err)
		toast.Toast(toast.Props{
			Title:         "Unexpected Error",
			Description:   "An unexpected error occurred",
			Variant:       toast.VariantError,
			Position:      toast.PositionTopRight,
			Duration:      5000,
			ShowIndicator: true,
			Icon:          true,
		}).Render(r.Context(), w)
		return
	}
	watchedCount := "<span>" + strconv.Itoa(count) + "</span>"
	w.Write([]byte(watchedCount))
}

func (h *Handlers) AddMovieToList(w http.ResponseWriter, r *http.Request) {
	listIDStr := r.FormValue("selected_list")
	movieIDStr := r.FormValue("movie_id")
	note := r.FormValue("note")

	log.Debug("adding movie to list", "listID", listIDStr, "movieID", movieIDStr, "note", note)

	listID, err := strconv.ParseInt(listIDStr, 10, 64)
	if err != nil {
		log.Error("invalid list ID", "listID", listIDStr, "error", err)
		toast.Toast(toast.Props{
			Title:         "Invalid List",
			Description:   "Please select a valid list",
			Variant:       toast.VariantError,
			Position:      toast.PositionTopRight,
			Duration:      5000,
			ShowIndicator: true,
			Icon:          true,
		}).Render(r.Context(), w)
		return
	}

	movieID, err := strconv.ParseInt(movieIDStr, 10, 64)
	if err != nil {
		log.Error("invalid movie ID", "movieID", movieIDStr, "error", err)
		toast.Toast(toast.Props{
			Title:         "Invalid Movie",
			Description:   "Movie not found",
			Variant:       toast.VariantError,
			Position:      toast.PositionTopRight,
			Duration:      5000,
			ShowIndicator: true,
			Icon:          true,
		}).Render(r.Context(), w)
		return
	}

	var notePtr *string
	if strings.TrimSpace(note) != "" {
		notePtr = &note
	}

	err = h.listService.AddMovieToList(r.Context(), listID, movieID, notePtr)
	if err != nil {
		log.Error("failed to add movie to list", "listID", listID, "movieID", movieID, "error", err)

		// TODO: use appropriate error types
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			toast.Toast(toast.Props{
				Title:         "Movie Already in List",
				Description:   "This movie is already in this list",
				Variant:       toast.VariantWarning,
				Position:      toast.PositionTopRight,
				Duration:      4000,
				ShowIndicator: true,
				Icon:          true,
			}).Render(r.Context(), w)
			return
		}

		toast.Toast(toast.Props{
			Title:         "Failed to Add Movie",
			Description:   "An unexpected error occurred while adding the movie to your list",
			Variant:       toast.VariantError,
			Position:      toast.PositionTopRight,
			Duration:      5000,
			ShowIndicator: true,
			Icon:          true,
		}).Render(r.Context(), w)
		return
	}

	toast.Toast(toast.Props{
		Title:         "Added to List",
		Description:   "Movie has been added to the list",
		Variant:       toast.VariantSuccess,
		Position:      toast.PositionTopRight,
		Duration:      3000,
		ShowIndicator: true,
		Icon:          true,
	}).Render(r.Context(), w)
}
