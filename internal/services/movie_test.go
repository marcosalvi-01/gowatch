package services

import (
	"context"
	"testing"
	"time"

	"github.com/marcosalvi-01/gowatch/db"
	"github.com/marcosalvi-01/gowatch/internal/models"
)

func TestMovieService_GetMovieDetails_CacheHit(t *testing.T) {
	testDB, err := db.NewTestDB()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = testDB.Close() }()

	movieService := NewMovieService(testDB, nil, time.Hour)

	ctx := context.Background()

	// Insert movie with recent updatedAt
	now := time.Now()
	movie := &models.MovieDetails{
		Movie: models.Movie{
			ID:        1,
			Title:     "Test Movie",
			UpdatedAt: now,
		},
	}
	if err := testDB.UpsertMovie(ctx, movie); err != nil {
		t.Fatal(err)
	}

	// Get details, should hit cache
	details, err := movieService.GetMovieDetails(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}
	if details.Movie.Title != "Test Movie" {
		t.Errorf("expected title 'Test Movie', got %s", details.Movie.Title)
	}
}
