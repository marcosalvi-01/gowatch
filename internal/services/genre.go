package services

import (
	"context"
	"fmt"
	"gowatch/db"
	"gowatch/internal/models"
)

type GenreService struct {
	db   db.DB
	tmdb *TMDBService
}

func NewGenreService(db db.DB, tmdbService *TMDBService) *GenreService {
	return &GenreService{
		db:   db,
		tmdb: tmdbService,
	}
}

func (g *GenreService) CreateGenre(ctx context.Context, genre models.Genre) error {
	err := g.db.InsertGenre(ctx, genre)
	if err != nil {
		return fmt.Errorf("failed to insert movie in db: %w", err)
	}

	return nil
}

func (g *GenreService) GetMovieGenres(ctx context.Context, movieID int64) ([]models.Genre, error) {
	genres, err := g.db.GetMovieGenre(ctx, movieID)
	if err != nil {
		return nil, fmt.Errorf("failed to get movie genres for movie with id '%d': %w", movieID, err)
	}

	return genres, nil
}
