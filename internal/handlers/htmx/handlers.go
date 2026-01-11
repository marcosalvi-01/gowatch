// Package htmx contains HTTP handlers for HTMX-based dynamic interactions, such as updating UI elements without full page reloads.
package htmx

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/marcosalvi-01/gowatch/internal/common"
	"github.com/marcosalvi-01/gowatch/internal/models"
	"github.com/marcosalvi-01/gowatch/internal/services"
	"github.com/marcosalvi-01/gowatch/internal/ui/components/addtolistdialog"
	"github.com/marcosalvi-01/gowatch/internal/ui/components/addtowatched"
	"github.com/marcosalvi-01/gowatch/internal/ui/components/listgrid"
	"github.com/marcosalvi-01/gowatch/internal/ui/components/liststats"
	"github.com/marcosalvi-01/gowatch/internal/ui/components/oobwrapper"
	"github.com/marcosalvi-01/gowatch/internal/ui/components/sidebar"
	"github.com/marcosalvi-01/gowatch/internal/ui/pages"
	"github.com/marcosalvi-01/gowatch/internal/utils"
	"github.com/marcosalvi-01/gowatch/logging"

	"github.com/a-h/templ"
	"github.com/go-chi/chi/v5"
)

var log = logging.Get("htmx handlers")

type Handlers struct {
	watchedService *services.WatchedService
	listService    *services.ListService
	homeService    *services.HomeService
	authService    *services.AuthService
}

func NewHandlers(watchedService *services.WatchedService, listService *services.ListService, homeService *services.HomeService, authService *services.AuthService) *Handlers {
	return &Handlers{
		watchedService: watchedService,
		listService:    listService,
		homeService:    homeService,
		authService:    authService,
	}
}

func (h *Handlers) RegisterRoutes(r chi.Router) {
	r.Post("/movies/watched", h.AddWatchedMovie)
	r.Post("/movies/import", h.ImportWatched)
	r.Post("/lists", h.CreateList)
	r.Delete("/lists", h.DeleteList)

	r.Post("/lists/items", h.AddMovieToList)
	r.Delete("/lists/items", h.DeleteMovieFromList)

	r.Post("/watchlist/add", h.AddToWatchlist)
	r.Get("/watchlist/{id}", h.RenderAddToWatchlistButton)

	r.Get("/sidebar", h.GetSidebar)
	r.Get("/lists/add-movie-dialog", h.RenderAddToListDialogContent)
	r.Get("/lists/home-lists", h.HomeLists)
	r.Get("/stats/top-lists", h.GetTopLists)
	r.Get("/lists/{id}/movie-grid", h.ListMovieGrid)
	r.Get("/lists/{id}/stats", h.ListStats)
}

func (h *Handlers) RenderAddToListDialogContent(w http.ResponseWriter, r *http.Request) {
	lists, err := h.listService.GetAllLists(r.Context())
	if err != nil {
		log.Error("failed to get all lists for dialog", "error", err)
		RenderErrorToast(w, r, "Failed to Load Lists", "An unexpected error occurred while fetching your lists.", 0)
		return
	}

	if err := addtolistdialog.AddToListDialog(lists).Render(r.Context(), w); err != nil {
		log.Error("failed to render add to list dialog", "error", err)
		http.Error(w, "Failed to render dialog", http.StatusInternalServerError)
		return
	}
}

func (h *Handlers) GetSidebar(w http.ResponseWriter, r *http.Request) {
	currentURL := r.Header.Get("HX-Current-URL")
	if r.Header.Get("HX-Current-URL") == "" {
		http.Error(w, "HX-Current-URL not set", http.StatusBadRequest)
		return
	}
	location := getFirstPathElement(currentURL)

	ctx := r.Context()

	log.Debug("getting sidebar", "location", location)

	cookie, err := r.Cookie("sidebar_state")
	if err != nil {
		log.Warn("Cookie sidebar_state not set")
	}
	collapsed := cookie != nil && cookie.Value == "false"

	count, err := h.watchedService.GetWatchedCount(ctx)
	if err != nil {
		log.Error("failed to get watched count for sidebar", "error", err)
		RenderErrorToast(w, r, "Sidebar Error", "Could not load watched count.", 0)
		return
	}

	user, err := common.GetUser(ctx)
	if err != nil {
		log.Error("user not found in context", "error", err)
		http.Error(w, "User not authenticated", http.StatusUnauthorized)
		return
	}

	lists, err := h.listService.GetAllLists(ctx)
	if err != nil {
		log.Error("failed to get all lists for sidebar", "userID", user.ID, "error", err)
		RenderErrorToast(w, r, "Sidebar Error", "Could not load lists.", 0)
		return
	}

	log.Debug("retrieved lists for sidebar", "userID", user.ID, "listCount", len(lists))

	watchlist, err := h.listService.GetWatchlist(ctx)
	if err != nil {
		log.Warn("failed to get watchlist for sidebar", "userID", user.ID, "error", err)
		RenderErrorToast(w, r, "Sidebar Error", "Could not load watchlist.", 0)
	}

	err = sidebar.Sidebar(sidebar.Props{
		CurrentPage:  currentURL,
		Collapsed:    collapsed,
		WatchedCount: count,
		Lists:        lists,
		Watchlist:    watchlist,
		User:         user,
	}).Render(ctx, w)
	if err != nil {
		log.Error("failed to render sidebar", "error", err)
		http.Error(w, "Failed to render sidebar", http.StatusInternalServerError)
		return
	}

	log.Debug("rendered sidebar", "location", location, "watchedCount", count, "listCount", len(lists))
}

func (h *Handlers) AddWatchedMovie(w http.ResponseWriter, r *http.Request) {
	movieIDParam := r.FormValue("movie_id")
	movieID, err := strconv.Atoi(movieIDParam)
	if err != nil {
		log.Error("invalid movie ID parameter", "movieID", movieIDParam, "error", err)
		RenderErrorToast(w, r, "Invalid Movie ID", "The movie ID provided is not valid. Please try again.", 4000)
		return
	}

	watchedDateParam := r.FormValue("watched_date")
	if watchedDateParam == "" {
		log.Error("missing watched_date parameter")
		RenderErrorToast(w, r, "Missing Date", "Please select the date when you watched the movie.", 4000)
		return
	}

	watchedDate, err := time.Parse("2006-01-02", watchedDateParam)
	if err != nil {
		log.Error("invalid watched_date parameter", "watchedDate", watchedDateParam, "error", err)
		RenderErrorToast(w, r, "Invalid Date Format", "The date format is invalid. Please select a valid date.", 4000)
		return
	}

	if watchedDate.After(time.Now()) {
		log.Error("watched date is in the future", "watchedDate", watchedDate)
		RenderErrorToast(w, r, "Future Date Not Allowed", "You cannot mark a movie as watched for a future date.", 4000)
		return
	}

	watchedAtTheater := r.FormValue("watched_at_theater") != ""

	ratingParam := r.FormValue("rating")
	var rating *float64
	if ratingParam != "" {
		parsedRating, err := strconv.ParseFloat(ratingParam, 64)
		if err != nil {
			log.Error("invalid rating parameter", "rating", ratingParam, "error", err)
			RenderErrorToast(w, r, "Invalid Rating", "The rating provided is not a valid number.", 4000)
			return
		}
		if parsedRating < 0 || parsedRating > 5 {
			log.Error("rating out of range", "rating", parsedRating)
			RenderErrorToast(w, r, "Invalid Rating", "Rating must be between 0 and 5.", 4000)
			return
		}
		rating = &parsedRating
	}
	if *rating == 0 {
		rating = nil
	}

	log.Debug("adding watched movie", "movieID", movieID, "watchedDate", watchedDate, "theater", watchedAtTheater, "rating", rating)

	err = h.watchedService.AddWatched(r.Context(), int64(movieID), watchedDate, watchedAtTheater, rating)
	if err != nil {
		log.Error("failed to add new watched movie", "movieID", movieID, "watchedDate", watchedDate, "theater", watchedAtTheater, "error", err)
		RenderErrorToast(w, r, "Unexpected error", "An unexpected error occurred, please try again", 0)
		return
	}

	log.Info("successfully added watched movie", "movieID", movieID, "rating", rating)

	successMessage := fmt.Sprintf("Movie marked as watched on %s", watchedDate.Format("Jan 2, 2006"))
	if rating != nil {
		successMessage += fmt.Sprintf(" with rating %.1f/5", *rating)
	}

	w.Header().Add("HX-Trigger", "refreshSidebar")
	RenderSuccessToast(w, r, "Movie Added Successfully", successMessage, 0)
}

func (h *Handlers) CreateList(w http.ResponseWriter, r *http.Request) {
	title := r.FormValue("title")
	description := r.FormValue("description")

	sanitizedTitle, err := utils.TrimAndValidateString(title, 100)
	if err != nil {
		log.Error("invalid list title", "title", title, "error", err)
		RenderErrorToast(w, r, "Invalid Title", "Please provide a valid title for your list.", 4000)
		return
	}

	sanitizedDescription, err := utils.TrimAndValidateString(description, 500)
	var descPtr *string
	if err == nil && sanitizedDescription != "" {
		descPtr = &sanitizedDescription
	}

	log.Debug("creating new list", "title", sanitizedTitle, "descriptionLength", len(sanitizedDescription))

	_, err = h.listService.CreateList(r.Context(), sanitizedTitle, descPtr, false)
	if err != nil {
		log.Error("failed to create list", "title", sanitizedTitle, "description", sanitizedDescription, "error", err)
		RenderErrorToast(w, r, "Unexpected Error", "An unexpected error occurred, please try again.", 0)
		return
	}

	log.Info("successfully created list", "title", sanitizedTitle)

	w.Header().Add("HX-Trigger", "refreshLists, refreshSidebar")
	RenderSuccessToast(w, r, "List Created Successfully", fmt.Sprintf("List \"%s\" has been created.", sanitizedTitle), 0)
}

func (h *Handlers) AddMovieToList(w http.ResponseWriter, r *http.Request) {
	listIDStr := r.FormValue("selected_list")
	movieIDStr := r.FormValue("movie_id")
	note := r.FormValue("note")

	sanitizedNote, err := utils.TrimAndValidateString(note, 500)
	if err != nil {
		sanitizedNote = ""
	}

	log.Debug("adding movie to list", "listID", listIDStr, "movieID", movieIDStr, "note", sanitizedNote)

	listID, err := strconv.ParseInt(listIDStr, 10, 64)
	if err != nil {
		log.Error("invalid list ID", "listID", listIDStr, "error", err)
		RenderErrorToast(w, r, "Invalid List", "Please select a valid list", 0)
		return
	}

	movieID, err := strconv.ParseInt(movieIDStr, 10, 64)
	if err != nil {
		log.Error("invalid movie ID", "movieID", movieIDStr, "error", err)
		RenderErrorToast(w, r, "Invalid Movie", "Movie not found", 0)
		return
	}

	var notePtr *string
	if sanitizedNote != "" {
		notePtr = &sanitizedNote
	}

	err = h.listService.AddMovieToList(r.Context(), listID, movieID, notePtr)
	if err != nil {
		log.Error("failed to add movie to list", "listID", listID, "movieID", movieID, "error", err)

		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			RenderWarningToast(w, r, "Movie Already in List", "This movie is already in this list", 4000)
			return
		}

		RenderErrorToast(w, r, "Failed to Add Movie", "An unexpected error occurred while adding the movie to your list", 0)
		return
	}

	log.Info("successfully added movie to list", "listID", listID, "movieID", movieID)

	RenderSuccessToast(w, r, "Added to List", "Movie has been added to the list", 0)
}

func (h *Handlers) DeleteList(w http.ResponseWriter, r *http.Request) {
	listIDstr := r.FormValue("list_id")
	id, err := strconv.ParseInt(listIDstr, 10, 64)
	if err != nil {
		log.Error("failed to parse list id to int", "listID", listIDstr, "error", err)
		RenderErrorToast(w, r, "Unexpected error", "An unexpected error occurred, please try again", 0)
		return
	}

	log.Debug("deleting list", "listID", id)

	err = h.listService.DeleteList(r.Context(), id)
	if err != nil {
		log.Error("failed to delete list from db", "listID", id, "error", err)
		RenderErrorToast(w, r, "Unexpected error", "An unexpected error occurred, please try again", 0)
		return
	}

	log.Info("successfully deleted list", "listID", id)

	// Fetch home data
	ctx := r.Context()

	homeData, err := h.homeService.GetHomeData(ctx)
	if err != nil {
		log.Error("failed to retrieve home data", "error", err)
		RenderErrorToast(w, r, "Unexpected error", "An unexpected error occurred, please try again", 0)
		return
	}

	// add the headers before rendering the components
	w.Header().Set("HX-Push-Url", "/home")
	w.Header().Add("HX-Trigger", "refreshSidebar")

	user, err := common.GetUser(ctx)
	if err != nil {
		log.Error("failed to retrieve user from context", "error", err)
		RenderErrorToast(w, r, "Unexpected error", "An unexpected error occurred, please try again", 0)
		return
	}

	var buf bytes.Buffer
	err = templ.RenderFragments(r.Context(), &buf, pages.Home(user.Name, *homeData), "content")
	if err != nil {
		log.Error("failed to render home fragment", "error", err)
		RenderErrorToast(w, r, "Unexpected error", "An unexpected error occurred, please try again", 0)
		return
	}

	oobCtx := templ.WithChildren(r.Context(), templ.Raw(buf.String()))
	if err := oobwrapper.OOBWrapper("innerHTML:#main-content").Render(oobCtx, w); err != nil {
		log.Error("failed to render OOB wrapper", "error", err)
		RenderErrorToast(w, r, "Unexpected error", "An unexpected error occurred, please try again", 0)
		return
	}

	RenderSuccessToast(w, r, "List Deleted Successfully", "The list has been deleted.", 2000)
}

func (h *Handlers) AddToWatchlist(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	movieIDStr := r.FormValue("movie_id")
	movieID, err := strconv.ParseInt(movieIDStr, 10, 64)
	if err != nil {
		log.Error("invalid movie ID for watchlist", "movieID", movieIDStr, "error", err)
		RenderErrorToast(w, r, "Invalid Request", "Invalid movie ID.", 3000)
		return
	}

	watchlist, err := h.listService.GetWatchlist(ctx)
	if err != nil {
		log.Error("failed to get watchlist", "error", err)
		RenderErrorToast(w, r, "Watchlist Error", "Could not access your watchlist.", 3000)
		return
	}

	err = h.listService.AddMovieToList(ctx, watchlist.ID, movieID, nil)
	if err != nil {
		log.Error("failed to add movie to watchlist", "movieID", movieID, "error", err)
		// Check if it's a duplicate
		if strings.Contains(err.Error(), "duplicate") || strings.Contains(err.Error(), "UNIQUE constraint") {
			RenderErrorToast(w, r, "Already in Watchlist", "This movie is already in your watchlist.", 3000)
		} else {
			RenderErrorToast(w, r, "Failed to Add", "Could not add movie to watchlist.", 3000)
		}
		return
	}

	log.Info("successfully added movie to watchlist", "movieID", movieID)
	w.Header().Add("HX-Trigger", "refreshSidebar, refreshWatchlist")
	RenderSuccessToast(w, r, "Added to Watchlist", "Movie added to your watchlist.", 2000)
}

func (h *Handlers) GetTopLists(w http.ResponseWriter, r *http.Request) {
	const maxLimit = 100
	limitStr := r.URL.Query().Get("limit")
	limit := 5
	if limitStr != "" {
		if parsed, err := strconv.Atoi(limitStr); err == nil && parsed > 0 {
			limit = min(parsed, maxLimit)
		}
	}

	stats, err := h.watchedService.GetWatchedStats(r.Context(), limit)
	if err != nil {
		log.Error("failed to retrieve watched stats for top lists", "error", err)
		http.Error(w, "Failed to load top lists", http.StatusInternalServerError)
		return
	}

	if err := pages.TopLists(stats, limit).Render(r.Context(), w); err != nil {
		log.Error("failed to render top lists", "error", err)
		http.Error(w, "Failed to render top lists", http.StatusInternalServerError)
		return
	}
}

func (h *Handlers) DeleteMovieFromList(w http.ResponseWriter, r *http.Request) {
	listIDstr := r.FormValue("list_id")
	movieIDstr := r.FormValue("movie_id")

	listID, err := strconv.ParseInt(listIDstr, 10, 64)
	if err != nil {
		log.Error("failed to parse list id to int", "listID", listIDstr, "error", err)
		RenderErrorToast(w, r, "Unexpected error", "An unexpected error occurred, please try again", 0)
		return
	}

	movieID, err := strconv.ParseInt(movieIDstr, 10, 64)
	if err != nil {
		log.Error("failed to parse movie id to int", "movieID", movieIDstr, "error", err)
		RenderErrorToast(w, r, "Unexpected error", "An unexpected error occurred, please try again", 0)
		return
	}

	log.Debug("removing movie from list", "listID", listID, "movieID", movieID)

	err = h.listService.DeleteMovieFromList(r.Context(), listID, movieID)
	if err != nil {
		log.Error("failed to delete movie from list", "listID", listID, "movieID", movieID, "error", err)
		RenderErrorToast(w, r, "Failed to Remove Movie", "An unexpected error occurred while removing the movie from the list", 0)
		return
	}

	log.Info("successfully removed movie from list", "listID", listID, "movieID", movieID)

	w.Header().Add("HX-Trigger", "refreshSidebar, refreshListGrid, refreshListStats")

	RenderSuccessToast(w, r, "Removed from List", "Movie has been removed from the list", 0)
}

func (h *Handlers) HomeLists(w http.ResponseWriter, r *http.Request) {
	lists, err := h.listService.GetAllLists(r.Context())
	if err != nil {
		log.Error("failed to get all lists for dialog", "error", err)
		RenderErrorToast(w, r, "Failed to Load Lists", "An unexpected error occurred while fetching your lists.", 0)
		return
	}

	if err := pages.HomeLists(lists).Render(r.Context(), w); err != nil {
		log.Error("failed to render sidebar", "error", err)
		http.Error(w, "Failed to render sidebar", http.StatusInternalServerError)
		return
	}
}

func (h *Handlers) ImportWatched(w http.ResponseWriter, r *http.Request) {
	err := r.ParseMultipartForm(32 << 20) // 32 MB
	if err != nil {
		log.Error("failed to parse multipart form", "error", err)
		RenderErrorToast(w, r, "Invalid Request", "Failed to process the upload.", 4000)
		return
	}

	file, _, err := r.FormFile("import-file")
	if err != nil {
		log.Error("failed to get file", "error", err)
		RenderErrorToast(w, r, "No File Selected", "Please select a file to import.", 4000)
		return
	}
	defer func() {
		if err := file.Close(); err != nil {
			log.Error("failed to close uploaded file", "error", err)
		}
	}()

	data, err := io.ReadAll(file)
	if err != nil {
		log.Error("failed to read file", "error", err)
		RenderErrorToast(w, r, "File Read Error", "Failed to read the uploaded file.", 4000)
		return
	}

	var importLog models.ImportWatchedMoviesLog
	err = json.Unmarshal(data, &importLog)
	if err != nil {
		log.Error("failed to parse JSON", "error", err)
		RenderErrorToast(w, r, "Invalid File Format", "The file must be valid JSON matching the expected format.", 4000)
		return
	}

	if len(importLog) == 0 {
		RenderErrorToast(w, r, "Empty File", "The file contains no movies to import.", 4000)
		return
	}

	ctx := context.WithoutCancel(r.Context())
	go func() {
		log.Info("HTMX import job started")
		if err := h.watchedService.ImportWatched(ctx, importLog); err != nil {
			log.Error("HTMX import job failed", "error", err)
			return
		}
		log.Info("HTMX import job finished successfully")
	}()

	RenderSuccessToast(w, r, "Import Started", "Your movies are being imported. This may take a few moments.", 0)
}

func (h *Handlers) RenderAddToWatchlistButton(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	movieIDParam := chi.URLParam(r, "id")

	movieID, err := strconv.ParseInt(movieIDParam, 10, 64)
	if err != nil {
		log.Error("invalid list ID", "id", movieIDParam, "error", err)
		http.Error(w, "Invalid list ID", http.StatusBadRequest)
		return
	}

	isMovieInWatchlist := h.listService.IsMovieInWatchlist(ctx, movieID)

	if err := addtowatched.AddToWatched(movieID, isMovieInWatchlist).Render(r.Context(), w); err != nil {
		log.Error("failed to render add to list dialog", "error", err)
		http.Error(w, "Failed to render dialog", http.StatusInternalServerError)
		return
	}
}

func (h *Handlers) ListMovieGrid(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	listIDStr := chi.URLParam(r, "id")

	log.Debug("rendering list movie grid", "listID", listIDStr)

	listID, err := strconv.ParseInt(listIDStr, 10, 64)
	if err != nil {
		log.Error("invalid list ID", "id", listIDStr, "error", err)
		RenderErrorToast(w, r, "Invalid List", "Invalid list ID.", 0)
		return
	}

	list, err := h.listService.GetListDetails(ctx, listID)
	if err != nil {
		log.Error("failed to get list details", "listID", listIDStr, "error", err)
		RenderErrorToast(w, r, "Unexpected Error", "An unexpected error occurred, please try again.", 0)
		return
	}

	if err := listgrid.ListGrid(*list).Render(ctx, w); err != nil {
		log.Error("failed to render list grid", "listID", listIDStr, "error", err)
		RenderErrorToast(w, r, "Unexpected Error", "An unexpected error occurred, please try again.", 0)
		return
	}
}

func (h *Handlers) ListStats(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	listIDStr := chi.URLParam(r, "id")

	log.Debug("rendering list stats", "listID", listIDStr)

	listID, err := strconv.ParseInt(listIDStr, 10, 64)
	if err != nil {
		log.Error("invalid list ID", "id", listIDStr, "error", err)
		RenderErrorToast(w, r, "Invalid List", "Invalid list ID.", 0)
		return
	}

	list, err := h.listService.GetListDetails(ctx, listID)
	if err != nil {
		log.Error("failed to get list details", "listID", listIDStr, "error", err)
		RenderErrorToast(w, r, "Unexpected Error", "An unexpected error occurred, please try again.", 0)
		return
	}

	if err := liststats.ListStats(*list).Render(ctx, w); err != nil {
		log.Error("failed to render list stats", "listID", listIDStr, "error", err)
		RenderErrorToast(w, r, "Unexpected Error", "An unexpected error occurred, please try again.", 0)
		return
	}
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
