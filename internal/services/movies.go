// Package services contains the business logic layer of the application.
// Services implement core business rules, data validation, and coordinate
// between handlers and the database. They provide a clean interface for
// business operations that can be used by both API and HTMX handlers.
package services

import (
	"gowatch/logging"
)

var log = logging.Get("services")

// MovieService handles movie database operations and business logic
type MovieService struct {
	tmdb           *TMDBService
	creditsService *CreditsService
}

func NewMovieService(tmdbService *TMDBService, creditsService *CreditsService) *MovieService {
	return &MovieService{
		tmdb:           tmdbService,
		creditsService: creditsService,
	}
}
