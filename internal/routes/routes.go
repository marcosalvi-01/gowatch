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

	"github.com/go-chi/chi/v5"
)

func NewRouter(tmdbService *services.MovieService, watchedService *services.WatchedService) chi.Router {
	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	apiHandlers := api.NewHandlers(watchedService)
	r.Route("/api/v1", func(r chi.Router) {
		r.Use(middleware.JSONMiddleware)
		apiHandlers.RegisterRoutes(r)
	})

	pagesHandlers := pages.NewHandlers(tmdbService, watchedService)
	r.Route("/", func(r chi.Router) {
		r.Use(middleware.HTMLMiddleware)
		pagesHandlers.RegisterRoutes(r)
	})

	staticHandlers := static.NewHandlers()
	r.Route("/static", func(r chi.Router) {
		staticHandlers.RegisterRoutes(r)
	})

	htmxHandlers := htmx.NewHandlers(watchedService)
	r.Route("/htmx", func(r chi.Router) {
		r.Use(middleware.HTMLMiddleware)
		htmxHandlers.RegisterRoutes(r)
	})

	return r
}
