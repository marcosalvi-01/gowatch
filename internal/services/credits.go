package services

import (
	"context"
	"fmt"
	"gowatch/db"
	"gowatch/internal/models"

)

// CreditsService handles cast and crew operations
type CreditsService struct {
	db   db.DB
	tmdb *TMDBService
}

func NewCreditsService(db db.DB, tmdbService *TMDBService) *CreditsService {
	return &CreditsService{
		db:   db,
		tmdb: tmdbService,
	}
}

func (s *CreditsService) CreateMovieCredits(ctx context.Context, credits models.MovieCredits) error {
	for _, c := range credits.Cast {
		if err := s.db.InsertCast(ctx, c); err != nil {
			log.Error("failed to insert cast member", "person_id", c.PersonID, "error", err)
			return fmt.Errorf("failed to insert cast member: %w", err)
		}
	}

	for _, c := range credits.Crew {
		if err := s.db.InsertCrew(ctx, c); err != nil {
			log.Error("failed to insert crew member", "person_id", c.PersonID, "error", err)
			return fmt.Errorf("failed to insert crew member: %w", err)
		}
	}

	log.Info("movie credits added", "cast_count", len(credits.Cast), "crew_count", len(credits.Crew))
	return nil
}

func (s *CreditsService) GetMovieCredits(ctx context.Context, movieID int64) (models.MovieCredits, error) {
	crew, err := s.GetMovieCrew(ctx, movieID)
	if err != nil {
		return models.MovieCredits{}, fmt.Errorf("failed to get crew for movie with id '%d': %w", movieID, err)
	}

	cast, err := s.GetMovieCast(ctx, movieID)
	if err != nil {
		return models.MovieCredits{}, fmt.Errorf("failed to get cast for movie with id '%d': %w", movieID, err)
	}

	return models.MovieCredits{
		Crew: crew,
		Cast: cast,
	}, nil
}

func (s *CreditsService) GetMovieCast(ctx context.Context, movieID int64) ([]models.Cast, error) {
	cast, err := s.db.GetMovieCast(ctx, movieID)
	if err != nil {
		return nil, fmt.Errorf("failed to get movie cast: %w", err)
	}
	return cast, nil
}

func (s *CreditsService) GetMovieCrew(ctx context.Context, movieID int64) ([]models.Crew, error) {
	crew, err := s.db.GetMovieCrew(ctx, movieID)
	if err != nil {
		return nil, fmt.Errorf("failed to get movie crew: %w", err)
	}
	return crew, nil
}
