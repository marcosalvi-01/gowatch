// Package pages contains HTTP handlers that serve page contents in HTML.
package pages

import (
	"fmt"
	"net/http"
	"strconv"
	"unicode"

	"github.com/marcosalvi-01/gowatch/internal/common"
	"github.com/marcosalvi-01/gowatch/internal/handlers/htmx"
	"github.com/marcosalvi-01/gowatch/internal/middleware"
	"github.com/marcosalvi-01/gowatch/internal/services"
	"github.com/marcosalvi-01/gowatch/internal/ui/pages"
	"github.com/marcosalvi-01/gowatch/internal/utils"
	"github.com/marcosalvi-01/gowatch/logging"

	"github.com/a-h/templ"
	"github.com/go-chi/chi/v5"
)

const htmxRequestHeaderValue = "true"

var log = logging.Get("pages")

type Handlers struct {
	tmdbService    *services.MovieService
	watchedService *services.WatchedService
	listService    *services.ListService
	homeService    *services.HomeService
	authService    *services.AuthService
}

func NewHandlers(
	tmdbService *services.MovieService,
	watchedService *services.WatchedService,
	listService *services.ListService,
	homeService *services.HomeService,
	authService *services.AuthService,
) *Handlers {
	return &Handlers{
		tmdbService:    tmdbService,
		watchedService: watchedService,
		listService:    listService,
		homeService:    homeService,
		authService:    authService,
	}
}

func (h *Handlers) RegisterRoutes(r chi.Router) {
	log.Info("registering page routes")

	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/home", http.StatusFound)
	})

	r.Get("/login", h.LoginPage)
	r.Get("/register", h.RegisterPage)
	r.Post("/register", h.RegisterPost)
	r.Post("/login", h.LoginPost)

	r.Group(func(r chi.Router) {
		r.Use(middleware.AuthMiddleware(*h.authService))

		r.Get("/watched", h.WatchedPage)
		r.Get("/home", h.HomePage)
		r.Get("/search", h.SearchPage)
		r.Get("/movie/{id}", h.MoviePage)
		r.Get("/list/{id}", h.ListPage)
		r.Get("/watchlist", h.Watchlist)
		r.Get("/stats", h.StatsPage)
		r.Post("/logout", h.LogoutPost)
		r.Get("/change-password", h.ChangePasswordPage)
		r.Post("/change-password", h.ChangePasswordPost)

		r.Route("/admin", func(r chi.Router) {
			r.Get("/users", h.AdminUsersPage)
			r.Delete("/users/{id}", h.AdminDeleteUser)
			r.Post("/users/{id}/reset-password", h.AdminResetPassword)
		})
	})
}

func (h *Handlers) HomePage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log.Debug("serving home page")

	homeData, err := h.homeService.GetHomeData(ctx)
	if err != nil {
		log.Error("failed to retrieve home data", "error", err)
		render500Error(w, r)
		return
	}

	user, err := common.GetUser(ctx)
	if err != nil {
		log.Error("failed to retrieve user from context", "error", err)
		render500Error(w, r)
		return
	}

	if r.Header.Get("HX-Request") == htmxRequestHeaderValue {
		templ.Handler(pages.Home(user.Name, *homeData), templ.WithFragments("content")).ServeHTTP(w, r)
	} else {
		templ.Handler(pages.Home(user.Name, *homeData)).ServeHTTP(w, r)
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

	isInWatchlist := h.listService.IsMovieInWatchlist(ctx, id)

	if r.Header.Get("HX-Request") == htmxRequestHeaderValue {
		templ.Handler(pages.Movie(*movie, rec, isInWatchlist), templ.WithFragments("content")).ServeHTTP(w, r)
	} else {
		templ.Handler(pages.Movie(*movie, rec, isInWatchlist)).ServeHTTP(w, r)
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

func (h *Handlers) LoginPage(w http.ResponseWriter, r *http.Request) {
	log.Debug("serving login page")

	templ.Handler(pages.Login()).ServeHTTP(w, r)
}

func (h *Handlers) RegisterPage(w http.ResponseWriter, r *http.Request) {
	log.Debug("serving registration page")

	templ.Handler(pages.Register()).ServeHTTP(w, r)
}

func (h *Handlers) LoginPost(w http.ResponseWriter, r *http.Request) {
	log.Debug("processing login request")
	email := r.FormValue("email")
	password := r.FormValue("password")

	if email == "" || password == "" {
		htmx.RenderErrorToast(w, r, "Missing fields", "Please fill in all fields", 0)
		return
	}

	user, err := h.authService.AuthenticateUser(r.Context(), email, password)
	if err != nil {
		log.Error("Authentication failed", "error", err)
		htmx.RenderErrorToast(w, r, "Login failed", "Invalid email or password", 0)
		return
	}

	log.Info("user authenticated successfully", "userID", user.ID)

	sessionID, err := h.authService.CreateSession(r.Context(), user.ID)
	if err != nil {
		log.Error("Failed to create session", "error", err)
		htmx.RenderErrorToast(w, r, "Login failed", "Please try logging in manually", 0)
		return
	}

	log.Info("login session created", "userID", user.ID)

	http.SetCookie(w, &http.Cookie{
		Name:     "session_id",
		Value:    sessionID,
		Path:     "/",
		HttpOnly: true,
		Secure:   h.authService.HTTPS,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(h.authService.SessionExpiry),
	})

	if user.PasswordResetRequired {
		w.Header().Add("HX-Redirect", "/change-password")
		w.WriteHeader(http.StatusOK)
		return
	}

	w.Header().Add("HX-Redirect", "/home")
	w.WriteHeader(http.StatusOK)
}

func (h *Handlers) RegisterPost(w http.ResponseWriter, r *http.Request) {
	log.Debug("processing registration request")
	email := r.FormValue("email")
	name := r.FormValue("name")
	password := r.FormValue("password")
	confirmPassword := r.FormValue("confirm_password")

	if email == "" || name == "" || password == "" || confirmPassword == "" {
		htmx.RenderErrorToast(w, r, "Missing fields", "Please fill in all fields", 0)
		return
	}

	if password != confirmPassword {
		htmx.RenderErrorToast(w, r, "Passwords don't match", "Please make sure both passwords are the same", 0)
		return
	}

	if ok, why := ValidatePassword(password); !ok {
		htmx.RenderErrorToast(w, r, "Password is too weak", why, 0)
		return
	}

	userID, err := h.authService.CreateUser(r.Context(), email, name, password)
	if err != nil {
		log.Error("Failed to create user", "error", err)
		htmx.RenderErrorToast(w, r, "Registration failed", "This email might already be registered", 0)
		return
	}

	log.Info("user registered successfully", "userID", userID)

	// Check if this is the first user and assign any existing nil user data to them
	count, err := h.authService.CountUsers(r.Context())
	if err != nil {
		log.Error("failed to count users", "error", err, "userID", userID)
		// Continue with registration even if we can't check user count
		// The user is already created, so we shouldn't fail here
	} else if count <= 1 {
		// This is the first user, attempt to migrate any existing nil user data
		err := h.authService.AssignNilUserLists(r.Context(), &userID)
		if err != nil {
			log.Error("failed to assign nil user lists to first user", "error", err, "userID", userID)
			htmx.RenderErrorToast(w, r, "Data migration warning", "Your account was created but some data may not have transferred correctly", 0)
			return
		}
		log.Info("successfully assigned nil user lists to first user", "userID", userID)

		err = h.authService.AssignNilUserWatched(r.Context(), &userID)
		if err != nil {
			log.Error("failed to assign nil user watched items to first user", "error", err, "userID", userID)
			htmx.RenderErrorToast(w, r, "Data migration warning", "Your account was created but watched items may not have transferred correctly", 0)
			return
		}
		log.Info("successfully assigned nil user watched items to first user", "userID", userID)

		// set the user as admin
		err = h.authService.SetUserAsAdmin(r.Context(), userID)
		if err != nil {
			log.Error("failed to set user as admin", "error", err, "userID", userID)
			htmx.RenderErrorToast(w, r, "Admin setup failed", "Your account was created but admin privileges could not be set", 0)
			return
		}
	}

	sessionID, err := h.authService.CreateSession(r.Context(), userID)
	if err != nil {
		log.Error("Failed to create session", "error", err)
		htmx.RenderErrorToast(w, r, "Login failed", "Please try logging in manually", 0)
		return
	}

	log.Info("registration session created", "userID", userID)

	http.SetCookie(w, &http.Cookie{
		Name:     "session_id",
		Value:    sessionID,
		Path:     "/",
		MaxAge:   int(h.authService.SessionExpiry),
		HttpOnly: true,
		Secure:   h.authService.HTTPS,
		SameSite: http.SameSiteLaxMode,
	})

	w.Header().Add("HX-Redirect", "/home")
	w.WriteHeader(http.StatusOK)
}

func ValidatePassword(password string) (bool, string) {
	var (
		hasUpper  bool
		hasLower  bool
		hasNumber bool
	)

	if len(password) < 8 {
		return false, "Password must be at least 8 characters long"
	}

	for _, char := range password {
		switch {
		case unicode.IsUpper(char):
			hasUpper = true
		case unicode.IsLower(char):
			hasLower = true
		case unicode.IsNumber(char):
			hasNumber = true
		}
	}

	if !hasUpper {
		return false, "Password must contain at least one uppercase letter"
	}
	if !hasLower {
		return false, "Password must contain at least one lowercase letter"
	}
	if !hasNumber {
		return false, "Password must contain at least one number"
	}

	return true, "Password is valid"
}

func (h *Handlers) LogoutPost(w http.ResponseWriter, r *http.Request) {
	log.Debug("processing logout request")

	cookie, err := r.Cookie("session_id")
	if err != nil {
		log.Warn("no session cookie found during logout")
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}

	err = h.authService.Logout(r.Context(), cookie.Value)
	if err != nil {
		log.Error("failed to logout", "error", err)
		// Continue with logout even if deletion fails
	}

	// Clear the session cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "session_id",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   h.authService.HTTPS,
		SameSite: http.SameSiteLaxMode,
	})

	w.Header().Add("HX-Redirect", "/login")
	w.WriteHeader(http.StatusOK)
}

func (h *Handlers) AdminUsersPage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user, err := common.GetUser(ctx)
	if err != nil || !user.Admin {
		log.Warn("unauthorized access attempt to admin page", "userID", user.ID)
		http.Redirect(w, r, "/home", http.StatusFound)
		return
	}

	users, err := h.authService.GetAllUsersWithStats(ctx)
	if err != nil {
		log.Error("failed to retrieve users for admin page", "error", err)
		render500Error(w, r)
		return
	}

	if r.Header.Get("HX-Request") == htmxRequestHeaderValue {
		templ.Handler(pages.AdminUsers(users), templ.WithFragments("content")).ServeHTTP(w, r)
	} else {
		templ.Handler(pages.AdminUsers(users)).ServeHTTP(w, r)
	}
}

func (h *Handlers) AdminDeleteUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	admin, err := common.GetUser(ctx)
	if err != nil || !admin.Admin {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	targetUserIDStr := chi.URLParam(r, "id")
	targetUserID, err := strconv.ParseInt(targetUserIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	if targetUserID == admin.ID {
		htmx.RenderErrorToast(w, r, "Action denied", "You cannot delete yourself!", 0)
		return
	}

	err = h.authService.DeleteUser(ctx, targetUserID)
	if err != nil {
		log.Error("failed to delete user", "userID", targetUserID, "error", err)
		htmx.RenderErrorToast(w, r, "Deletion failed", "Could not delete user", 0)
		return
	}

	htmx.RenderSuccessToast(w, r, "User deleted", "Account has been permanently removed", 0)
	w.WriteHeader(http.StatusOK)
}

func (h *Handlers) AdminResetPassword(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	admin, err := common.GetUser(ctx)
	if err != nil || !admin.Admin {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	targetUserIDStr := chi.URLParam(r, "id")
	targetUserID, err := strconv.ParseInt(targetUserIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	newPass, err := h.authService.RequirePasswordReset(ctx, targetUserID)
	if err != nil {
		log.Error("failed to reset user password", "userID", targetUserID, "error", err)
		htmx.RenderErrorToast(w, r, "Reset failed", "Could not reset password", 0)
		return
	}

	htmx.RenderSuccessToast(w, r, "Password reset", fmt.Sprintf("Password has been reset to: %s", newPass), 0)
}

func (h *Handlers) ChangePasswordPage(w http.ResponseWriter, r *http.Request) {
	log.Debug("serving change password page")

	ctx := r.Context()
	user, err := common.GetUser(ctx)
	if err != nil {
		log.Error("failed to get user from context", "error", err)
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	if !user.PasswordResetRequired {
		w.Header().Add("HX-Redirect", "/home")
		w.WriteHeader(http.StatusOK)
		return
	}

	templ.Handler(pages.ChangePassword()).ServeHTTP(w, r)
}

func (h *Handlers) ChangePasswordPost(w http.ResponseWriter, r *http.Request) {
	log.Debug("processing change password request")
	ctx := r.Context()
	user, err := common.GetUser(ctx)
	if err != nil {
		log.Error("failed to get user from context", "error", err)
		htmx.RenderErrorToast(w, r, "Authentication Error", "Please log in again", 0)
		return
	}
	password := r.FormValue("password")
	confirmPassword := r.FormValue("confirm_password")

	if password == "" || confirmPassword == "" {
		htmx.RenderErrorToast(w, r, "Missing fields", "Please fill in all fields", 0)
		return
	}

	if password != confirmPassword {
		htmx.RenderErrorToast(w, r, "Passwords don't match", "Please make sure both passwords are the same", 0)
		return
	}

	if ok, why := ValidatePassword(password); !ok {
		htmx.RenderErrorToast(w, r, "Password is too weak", why, 0)
		return
	}

	err = h.authService.UpdateUserPassword(ctx, user.ID, password)
	if err != nil {
		log.Error("failed to update user password", "userID", user.ID, "error", err)
		htmx.RenderErrorToast(w, r, "Password Update Failed", "An error occurred while updating your password. Please try again.", 0)
		return
	}

	err = h.authService.ClearPasswordResetRequired(ctx, user.ID)
	if err != nil {
		log.Error("failed to clear password reset flag", "userID", user.ID, "error", err)
		htmx.RenderErrorToast(w, r, "Setup Error", "Password updated but please contact admin if you see this message.", 0)
		return
	}

	log.Info("user password changed successfully", "userID", user.ID)

	sessionID, err := h.authService.CreateSession(r.Context(), user.ID)
	if err != nil {
		log.Error("Failed to create session", "error", err)
		htmx.RenderErrorToast(w, r, "Login failed", "Please try logging in manually", 0)
		return
	}

	log.Info("registration session created", "userID", user.ID)

	http.SetCookie(w, &http.Cookie{
		Name:     "session_id",
		Value:    sessionID,
		Path:     "/",
		MaxAge:   int(h.authService.SessionExpiry),
		HttpOnly: true,
		Secure:   h.authService.HTTPS,
		SameSite: http.SameSiteLaxMode,
	})

	w.Header().Add("HX-Redirect", "/home")
	w.WriteHeader(http.StatusOK)
}

func (h *Handlers) Watchlist(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	log.Debug("serving watchlist page")

	userID, err := common.GetUser(ctx)
	if err != nil {
		log.Error("failed to retrieve user from context", "error", err)
		render500Error(w, r)
		return
	}

	list, err := h.listService.GetWatchlist(ctx)
	if err != nil {
		log.Error("failed to get list details", "userID", userID, "error", err)
		render500Error(w, r)
		return
	}
	log.Debug("fetched watchlist details", "userID", userID, "movieCount", len(list.Movies))

	if r.Header.Get("HX-Request") == htmxRequestHeaderValue {
		templ.Handler(pages.Watchlist(list), templ.WithFragments("content")).ServeHTTP(w, r)
	} else {
		templ.Handler(pages.Watchlist(list)).ServeHTTP(w, r)
	}

	log.Info("watchlist page served successfully", "userID", userID, "movieCount", len(list.Movies))
}
