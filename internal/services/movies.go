// Package services contains the business logic layer of the application.
// Services implement core business rules, data validation, and coordinate
// between handlers and the database. They provide a clean interface for
// business operations that can be used by both API and HTMX handlers.
package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"gowatch/db"
	"gowatch/internal/models"
	"gowatch/logging"
)

var log = logging.Get("services")

// MovieService handles movie database operations and business logic
type MovieService struct {
	db             db.DB
	tmdb           *TMDBService
	creditsService *CreditsService
}

func NewMovieService(db db.DB, tmdbService *TMDBService, creditsService *CreditsService) *MovieService {
	return &MovieService{
		db:             db,
		tmdb:           tmdbService,
		creditsService: creditsService,
	}
}

func (s *MovieService) CreateMovie(ctx context.Context, movie models.Movie) error {
	log.Debug("creating movie", "movie_id", movie.ID, "title", movie.Title)

	err := s.db.InsertMovie(ctx, movie)
	if err != nil {
		log.Error("failed to insert movie", "movie_id", movie.ID, "error", err)
		return fmt.Errorf("failed to insert movie %d: %w", movie.ID, err)
	}

	credits, err := s.tmdb.GetMovieCredits(movie.ID)
	if err != nil {
		return fmt.Errorf("TODO: %w", err)
	}

	err = s.creditsService.CreateMovieCredits(ctx, credits)
	if err != nil {
		return fmt.Errorf("failed to add movie credits to db: %w", err)
	}

	log.Info("movie created", "movie_id", movie.ID, "title", movie.Title)
	return nil
}

func (s *MovieService) GetMovieDetails(ctx context.Context, id int64) (models.WatchedMovieDetails, error) {
	movie, err := s.db.GetWatchedMovieDetails(ctx, id)
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			return models.WatchedMovieDetails{}, fmt.Errorf("failed to get movie details by id: %w", err)
		}

		tmdbMovie, err := s.tmdb.GetMovieDetails(id)
		if err != nil {
			return models.WatchedMovieDetails{}, fmt.Errorf("failed to get movie details from TMDB: %w", err)
		}

		err = s.CreateMovie(ctx, tmdbMovie)
		if err != nil {
			return models.WatchedMovieDetails{}, fmt.Errorf("failed to add new movie to db: %w", err)
		}

		movie = models.WatchedMovieDetails{
			Movie:     tmdbMovie,
			ViewCount: 0,
		}
	}

	credits, err := s.creditsService.GetMovieCredits(ctx, id)
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			return models.WatchedMovieDetails{}, fmt.Errorf("TODO: %w", err)
		}

		credits, err = s.tmdb.GetMovieCredits(id)
		if err != nil {
			return models.WatchedMovieDetails{}, fmt.Errorf("TODO: %w", err)
		}

		err = s.creditsService.CreateMovieCredits(ctx, credits)
		if err != nil {
			return models.WatchedMovieDetails{}, fmt.Errorf("TODO: %w", err)
		}
	}

	movie.Credits = credits

	return movie, nil
}
