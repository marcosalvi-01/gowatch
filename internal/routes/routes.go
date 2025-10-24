// Package routes defines the HTTP routing configuration for the application.
// It sets up the Chi router, applies middleware to route groups, and
// registers all handler routes. This is where the complete routing
// structure of the application is defined and organized.
package routes

import (
	"gowatch/internal/handlers/api"
	"gowatch/internal/handlers/htmx"
	"gowatch/internal/handlers/pages"
	"gowatch/internal/handlers/static"
	"gowatch/internal/middleware"
	"gowatch/internal/services"
	"gowatch/logging"

	"github.com/go-chi/chi/v5"
)

var log = logging.Get("routes")

func NewRouter(
	tmdbService *services.MovieService,
	watchedService *services.WatchedService,
	listService *services.ListService,
) chi.Router {
	log.Info("creating HTTP router")

	r := chi.NewRouter()

	log.Debug("applying global middleware")
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	log.Debug("registering API routes")
	apiHandlers := api.NewHandlers(watchedService)
	r.Route("/api/v1", func(r chi.Router) {
		r.Use(middleware.JSONMiddleware)
		apiHandlers.RegisterRoutes(r)
	})

	log.Debug("registering pages routes")
	pagesHandlers := pages.NewHandlers(tmdbService, watchedService, listService)
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
	htmxHandlers := htmx.NewHandlers(watchedService, listService)
	r.Route("/htmx", func(r chi.Router) {
		r.Use(middleware.HTMLMiddleware)
		htmxHandlers.RegisterRoutes(r)
	})

	log.Info("router configuration complete")
	return r
}
