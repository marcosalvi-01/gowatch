package htmx

import (
	"fmt"
	"gowatch/internal/services"
	"gowatch/internal/ui/components/sidebar"
	"gowatch/logging"
	"net/http"
	"net/url"
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
	r.Post("/lists", h.CreateList)

	// r.Get("/lists", h.GetAllLists)
	// // r.Get("/lists/{id}", h.GetList)
	// r.Delete("/lists", h.DeleteList)
	// // r.Put("/lists/{id}", h.UpdateList)
	//
	// r.Post("/lists/items", h.AddMovieToList)
	// r.Get("/lists/items", h.GetListDetails)
	// r.Delete("/lists/items", h.DeleteMovieFromList)

	r.Get("/sidebar", h.GetSidebar)
}

func (h *Handlers) GetSidebar(w http.ResponseWriter, r *http.Request) {
	currentURL := r.Header.Get("HX-Current-URL")
	if r.Header.Get("HX-Current-URL") == "" {
		http.Error(w, "HX-Current-URL not set", http.StatusBadRequest)
		return
	}
	location := getFirstPathElement(currentURL)

	log.Debug("getting sidebar", "location", location)

	cookie, err := r.Cookie("sidebar_state")
	if err != nil {
		log.Warn("Cookie sidebar_state not set")
	}
	collapsed := cookie != nil && cookie.Value == "false"

	count, err := h.watchedService.GetWatchedCount(r.Context())
	if err != nil {
		http.Error(w, "TODO", http.StatusInternalServerError)
		return
	}

	lists, err := h.listService.GetAllLists(r.Context())
	if err != nil {
		http.Error(w, "TODO", http.StatusInternalServerError)
	}

	sidebar.Sidebar(sidebar.Props{
		CurrentPage:  currentURL,
		Collapsed:    collapsed,
		WatchedCount: count,
		Lists:        lists,
	}).Render(r.Context(), w)
}

func (h *Handlers) AddWatchedMovie(w http.ResponseWriter, r *http.Request) {
	movieIDParam := r.FormValue("movie_id")
	movieID, err := strconv.Atoi(movieIDParam)
	if err != nil {
		log.Error("invalid movie ID parameter", "movieID", movieIDParam, "error", err)
		h.renderErrorToast(w, r, "Invalid Movie ID", "The movie ID provided is not valid. Please try again.", 4000)
		return
	}

	watchedDateParam := r.FormValue("watched_date")
	if watchedDateParam == "" {
		log.Error("missing watched_date parameter")
		h.renderErrorToast(w, r, "Missing Date", "Please select the date when you watched the movie.", 4000)
		return
	}

	watchedDate, err := time.Parse("2006-01-02", watchedDateParam)
	if err != nil {
		log.Error("invalid watched_date parameter", "watchedDate", watchedDateParam, "error", err)
		h.renderErrorToast(w, r, "Invalid Date Format", "The date format is invalid. Please select a valid date.", 4000)
		return
	}

	if watchedDate.After(time.Now()) {
		log.Error("watched date is in the future", "watchedDate", watchedDate)
		h.renderErrorToast(w, r, "Future Date Not Allowed", "You cannot mark a movie as watched for a future date.", 4000)
		return
	}

	watchedAtTheater := r.FormValue("watched_at_theater") != ""

	err = h.watchedService.AddWatched(r.Context(), int64(movieID), watchedDate, watchedAtTheater)
	if err != nil {
		log.Error("failed to add new watched movie", "movieID", movieID, "watchedDate", watchedDate, "theater", watchedAtTheater, "error", err)
		h.renderErrorToast(w, r, "Unexpected error", "An unexpected error occurred, please try again", 0)
		return
	}

	successMessage := fmt.Sprintf("Movie marked as watched on %s", watchedDate.Format("Jan 2, 2006"))

	w.Header().Add("HX-Trigger", "refreshSidebar")
	h.renderSuccessToast(w, r, "Movie Added Successfully", successMessage, 0)
}

func (h *Handlers) CreateList(w http.ResponseWriter, r *http.Request) {
	title := r.FormValue("title")
	description := r.FormValue("description")

	if title == "" {
		log.Error("missing list title")
		h.renderErrorToast(w, r, "Missing Title", "Please provide a title for your list.", 4000)
		return
	}

	if len(description) > 500 {
		log.Error("description too long", "length", len(description))
		h.renderErrorToast(w, r, "Description Too Long", "Please keep the description under 500 characters.", 4000)
		return
	}

	err := h.listService.CreateList(r.Context(), title, description)
	if err != nil {
		log.Error("failed to create list", "title", title, "description", description, "error", err)
		h.renderErrorToast(w, r, "Unexpected Error", "An unexpected error occurred, please try again.", 0)
		return
	}

	w.Header().Add("HX-Trigger", "refreshSidebar")
	h.renderSuccessToast(w, r, "List Created Successfully", fmt.Sprintf("List \"%s\" has been created.", title), 0)
}

func (h *Handlers) GetAllLists(w http.ResponseWriter, r *http.Request) {
	// lists, err := h.listService.GetAllLists(r.Context())
	// if err != nil {
	// 	log.Error("failed to get all lists", "error", err)
	// 	h.renderErrorToast(w, r, "Unexpected Error", "An unexpected error occurred", 0)
	// 	return
	// }
	return

	// page.SidebarListsList("", lists).Render(r.Context(), w)
}

func (h *Handlers) AddMovieToList(w http.ResponseWriter, r *http.Request) {
	listIDStr := r.FormValue("selected_list")
	movieIDStr := r.FormValue("movie_id")
	note := r.FormValue("note")

	log.Debug("adding movie to list", "listID", listIDStr, "movieID", movieIDStr, "note", note)

	listID, err := strconv.ParseInt(listIDStr, 10, 64)
	if err != nil {
		log.Error("invalid list ID", "listID", listIDStr, "error", err)
		h.renderErrorToast(w, r, "Invalid List", "Please select a valid list", 0)
		return
	}

	movieID, err := strconv.ParseInt(movieIDStr, 10, 64)
	if err != nil {
		log.Error("invalid movie ID", "movieID", movieIDStr, "error", err)
		h.renderErrorToast(w, r, "Invalid Movie", "Movie not found", 0)
		return
	}

	var notePtr *string
	if strings.TrimSpace(note) != "" {
		notePtr = &note
	}

	err = h.listService.AddMovieToList(r.Context(), listID, movieID, notePtr)
	if err != nil {
		log.Error("failed to add movie to list", "listID", listID, "movieID", movieID, "error", err)

		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			h.renderWarningToast(w, r, "Movie Already in List", "This movie is already in this list", 4000)
			return
		}

		h.renderErrorToast(w, r, "Failed to Add Movie", "An unexpected error occurred while adding the movie to your list", 0)
		return
	}

	h.renderSuccessToast(w, r, "Added to List", "Movie has been added to the list", 0)
}

func (h *Handlers) DeleteList(w http.ResponseWriter, r *http.Request) {
	listIDstr := r.FormValue("list_id")
	id, err := strconv.ParseInt(listIDstr, 10, 64)
	if err != nil {
		log.Error("failed to parse list id to int", "listID", listIDstr, "error", err)
		h.renderErrorToast(w, r, "Unexpected error", "An unexpected error occurred, please try again", 0)
		return
	}

	err = h.listService.DeleteList(r.Context(), id)
	if err != nil {
		log.Error("failed to delete list from db", "listID", id, "error", err)
		h.renderErrorToast(w, r, "Unexpected error", "An unexpected error occurred, please try again", 0)
		return
	}

	// TODO when the home is done this should redirect to there
	w.Header().Add("HX-Redirect", "/watched")

	// no success toast since it wouldn't be shown after the redirect
}

func (h *Handlers) DeleteMovieFromList(w http.ResponseWriter, r *http.Request) {
	listIDstr := r.FormValue("list_id")
	movieIDstr := r.FormValue("movie_id")

	listID, err := strconv.ParseInt(listIDstr, 10, 64)
	if err != nil {
		log.Error("failed to parse list id to int", "listID", listIDstr, "error", err)
		h.renderErrorToast(w, r, "Unexpected error", "An unexpected error occurred, please try again", 0)
		return
	}

	movieID, err := strconv.ParseInt(movieIDstr, 10, 64)
	if err != nil {
		log.Error("failed to parse movie id to int", "movieID", movieIDstr, "error", err)
		h.renderErrorToast(w, r, "Unexpected error", "An unexpected error occurred, please try again", 0)
		return
	}

	err = h.listService.DeleteMovieFromList(r.Context(), listID, movieID)
	if err != nil {
		log.Error("failed to delete movie from list", "listID", listID, "movieID", movieID, "error", err)
		h.renderErrorToast(w, r, "Failed to Remove Movie", "An unexpected error occurred while removing the movie from the list", 0)
		return
	}

	w.Header().Add("HX-Trigger", "refreshListContents")

	h.renderSuccessToast(w, r, "Removed from List", "Movie has been removed from the list", 0)
}

func (h *Handlers) GetListDetails(w http.ResponseWriter, r *http.Request) {
	listIDStr := r.FormValue("list_id")
	if listIDStr == "" {
		log.Error("missing list ID parameter")
		h.renderErrorToast(w, r, "Missing List", "Please select a valid list to view", 0)
		return
	}

	// listID, err := strconv.ParseInt(listIDStr, 10, 64)
	// if err != nil {
	// 	log.Error("invalid list ID", "listID", listIDStr, "error", err)
	// 	h.renderErrorToast(w, r, "Invalid List", "Please select a valid list", 0)
	// 	return
	// }

	// list, err := h.listService.GetListDetails(r.Context(), listID)
	// if err != nil {
	// 	log.Error("failed to fetch list details from db", "listID", listID, "error", err)
	// 	h.renderErrorToast(w, r, "Unexpected error", "An unexpected error occurred, please try again", 0)
	// 	return
	// }

	// ui.ListDetails(list).Render(r.Context(), w)
}

func getFirstPathElement(urlStr string) string {
	u, err := url.Parse(urlStr)
	if err != nil {
		return ""
	}

	// Remove leading slash and split
	path := strings.TrimPrefix(u.Path, "/")
	parts := strings.Split(path, "/")

	if len(parts) > 0 && parts[0] != "" {
		return parts[0]
	}
	return ""
}
