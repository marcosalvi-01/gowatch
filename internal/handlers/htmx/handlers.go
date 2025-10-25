// Package htmx contains HTTP handlers for HTMX-based dynamic interactions, such as updating UI elements without full page reloads.
package htmx

import (
	"bytes"
	"fmt"
	"gowatch/internal/services"
	"gowatch/internal/ui/components/addtolistdialog"
	"gowatch/internal/ui/components/oobwrapper"
	"gowatch/internal/ui/components/sidebar"
	"gowatch/internal/ui/pages"
	"gowatch/logging"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/a-h/templ"
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
	r.Delete("/lists", h.DeleteList)

	r.Post("/lists/items", h.AddMovieToList)
	r.Delete("/lists/items", h.DeleteMovieFromList)

	r.Get("/sidebar", h.GetSidebar)
	r.Get("/lists/add-movie-dialog", h.RenderAddToListDialogContent)
}

func (h *Handlers) RenderAddToListDialogContent(w http.ResponseWriter, r *http.Request) {
	lists, err := h.listService.GetAllLists(r.Context())
	if err != nil {
		log.Error("failed to get all lists for dialog", "error", err)
		h.renderErrorToast(w, r, "Failed to Load Lists", "An unexpected error occurred while fetching your lists.", 0)
		return
	}

	addtolistdialog.AddToListDialog(lists).Render(r.Context(), w)
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
		log.Error("failed to get watched count for sidebar", "error", err)
		h.renderErrorToast(w, r, "Sidebar Error", "Could not load watched count.", 0)
		return
	}

	lists, err := h.listService.GetAllLists(r.Context())
	if err != nil {
		log.Error("failed to get all lists for sidebar", "error", err)
		h.renderErrorToast(w, r, "Sidebar Error", "Could not load lists.", 0)
		return
	}

	sidebar.Sidebar(sidebar.Props{
		CurrentPage:  currentURL,
		Collapsed:    collapsed,
		WatchedCount: count,
		Lists:        lists,
	}).Render(r.Context(), w)

	log.Debug("rendered sidebar", "location", location, "watchedCount", count, "listCount", len(lists))
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

	log.Debug("adding watched movie", "movieID", movieID, "watchedDate", watchedDate, "theater", watchedAtTheater)

	err = h.watchedService.AddWatched(r.Context(), int64(movieID), watchedDate, watchedAtTheater)
	if err != nil {
		log.Error("failed to add new watched movie", "movieID", movieID, "watchedDate", watchedDate, "theater", watchedAtTheater, "error", err)
		h.renderErrorToast(w, r, "Unexpected error", "An unexpected error occurred, please try again", 0)
		return
	}

	log.Info("successfully added watched movie", "movieID", movieID)

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

	log.Debug("creating new list", "title", title, "descriptionLength", len(description))

	err := h.listService.CreateList(r.Context(), title, description)
	if err != nil {
		log.Error("failed to create list", "title", title, "description", description, "error", err)
		h.renderErrorToast(w, r, "Unexpected Error", "An unexpected error occurred, please try again.", 0)
		return
	}

	log.Info("successfully created list", "title", title)

	w.Header().Add("HX-Trigger", "refreshLists, refreshSidebar")
	h.renderSuccessToast(w, r, "List Created Successfully", fmt.Sprintf("List \"%s\" has been created.", title), 0)
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

	log.Info("successfully added movie to list", "listID", listID, "movieID", movieID)

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

	log.Debug("deleting list", "listID", id)

	err = h.listService.DeleteList(r.Context(), id)
	if err != nil {
		log.Error("failed to delete list from db", "listID", id, "error", err)
		h.renderErrorToast(w, r, "Unexpected error", "An unexpected error occurred, please try again", 0)
		return
	}

	log.Info("successfully deleted list", "listID", id)

	// add the headers before rendering the components
	w.Header().Set("HX-Push-Url", "/home")
	w.Header().Add("HX-Trigger", "refreshSidebar")

	var buf bytes.Buffer
	err = templ.RenderFragments(r.Context(), &buf, pages.Home(), "content")
	if err != nil {
		log.Error("failed to render home fragment", "error", err)
		h.renderErrorToast(w, r, "Unexpected error", "An unexpected error occurred, please try again", 0)
		return
	}
	ctx := templ.WithChildren(r.Context(), templ.Raw(buf.String()))
	oobwrapper.OOBWrapper("innerHTML:#main-content").Render(ctx, w)

	h.renderSuccessToast(w, r, "List Deleted Successfully", "The list has been deleted.", 2000)
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

	log.Debug("removing movie from list", "listID", listID, "movieID", movieID)

	err = h.listService.DeleteMovieFromList(r.Context(), listID, movieID)
	if err != nil {
		log.Error("failed to delete movie from list", "listID", listID, "movieID", movieID, "error", err)
		h.renderErrorToast(w, r, "Failed to Remove Movie", "An unexpected error occurred while removing the movie from the list", 0)
		return
	}

	log.Info("successfully removed movie from list", "listID", listID, "movieID", movieID)

	list, err := h.listService.GetListDetails(r.Context(), listID)
	if err != nil {
		log.Error("failed to get list details after delete", "listID", listID, "error", err)
		h.renderErrorToast(w, r, "Failed to Refresh List", "An unexpected error occurred while refreshing the list", 0)
		return
	}

	var buf bytes.Buffer
	err = templ.RenderFragments(r.Context(), &buf, pages.List(list), "content")
	if err != nil {
		log.Error("failed to render list fragment", "error", err)
		h.renderErrorToast(w, r, "Failed to Refresh List", "An unexpected error occurred while refreshing the list", 0)
		return
	}
	ctx := templ.WithChildren(r.Context(), templ.Raw(buf.String()))
	oobwrapper.OOBWrapper("innerHTML:#main-content").Render(ctx, w)

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
