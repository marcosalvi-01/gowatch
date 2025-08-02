package services

import (
	"context"
	"fmt"
	"gowatch/db"
	"gowatch/internal/models"
)

// PersonService handles person-related operations
type PersonService struct {
	db   db.DB
	tmdb *TMDBService
}

func NewPersonService(db db.DB, tmdbService *TMDBService) *PersonService {
	return &PersonService{
		db:   db,
		tmdb: tmdbService,
	}
}

func (s *PersonService) GetMovieCast(ctx context.Context, personID int64) ([]models.Cast, error) {
	movies, err := s.db.GetMovieCast(ctx, personID)
	if err != nil {
		return nil, fmt.Errorf("failed to get person movies as cast: %w", err)
	}
	return movies, nil
}

func (s *PersonService) GetMovieCrew(ctx context.Context, personID int64) ([]models.Crew, error) {
	movies, err := s.db.GetMovieCrew(ctx, personID)
	if err != nil {
		return nil, fmt.Errorf("failed to get person movies as crew: %w", err)
	}
	return movies, nil
}
