// Package routes defines the HTTP routing configuration for the application.
// It sets up the Chi router, applies middleware to route groups, and
// registers all handler routes. This is where the complete routing
// structure of the application is defined and organized.
package routes

import (
	"github.com/marcosalvi-01/gowatch/db"
	"github.com/marcosalvi-01/gowatch/internal/handlers/api"
	"github.com/marcosalvi-01/gowatch/internal/handlers/htmx"
	"github.com/marcosalvi-01/gowatch/internal/handlers/pages"
	"github.com/marcosalvi-01/gowatch/internal/handlers/static"
	"github.com/marcosalvi-01/gowatch/internal/middleware"
	"github.com/marcosalvi-01/gowatch/internal/services"
	"github.com/marcosalvi-01/gowatch/logging"

	"github.com/go-chi/chi/v5"
)

var log = logging.Get("routes")

func NewRouter(
	db db.DB,
	tmdbService *services.MovieService,
	watchedService *services.WatchedService,
	listService *services.ListService,
	authService *services.AuthService,
) chi.Router {
	log.Info("creating HTTP router")

	r := chi.NewRouter()

	log.Debug("applying global middleware")
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	homeService := services.NewHomeService(watchedService, listService)

	log.Debug("registering API routes")
	apiHandlers := api.NewHandlers(db, watchedService)
	r.Route("/api/v1", func(r chi.Router) {
		r.Use(middleware.AuthMiddleware(*authService))
		r.Use(middleware.JSONMiddleware)
		apiHandlers.RegisterRoutes(r)
	})

	log.Debug("registering pages routes")
	pagesHandlers := pages.NewHandlers(tmdbService, watchedService, listService, homeService, authService)
	r.Route("/", func(r chi.Router) {
		r.Use(middleware.HTMLMiddleware)
		pagesHandlers.RegisterRoutes(r)
	})

	log.Debug("registering static routes")
	staticHandlers := static.NewHandlers()
	r.Route("/static", func(r chi.Router) {
		staticHandlers.RegisterRoutes(r)
	})

	log.Debug("registering HTMX routes")
	htmxHandlers := htmx.NewHandlers(watchedService, listService, homeService, authService)
	r.Route("/htmx", func(r chi.Router) {
		r.Use(middleware.AuthMiddleware(*authService))
		r.Use(middleware.HTMLMiddleware)
		htmxHandlers.RegisterRoutes(r)
	})

	log.Info("router configuration complete")

	// Set custom 404 handler
	r.NotFound(pagesHandlers.Error404Page)

	return r
}
