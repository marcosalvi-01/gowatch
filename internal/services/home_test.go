package services

import (
	"testing"
	"time"

	"github.com/marcosalvi-01/gowatch/internal/models"
)

func TestSelectWatchNextMovies_OrdersByPositionThenDateAdded(t *testing.T) {
	baseDate := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	movies := []models.MovieItem{
		newHomeMovieItem(10, "C", baseDate.AddDate(0, 0, 3), nil),
		newHomeMovieItem(11, "B", baseDate.AddDate(0, 0, 5), int64Ptr(2)),
		newHomeMovieItem(12, "A", baseDate.AddDate(0, 0, 4), int64Ptr(1)),
		newHomeMovieItem(13, "A", baseDate, nil),
		newHomeMovieItem(14, "A", baseDate, nil),
		newHomeMovieItem(15, "D", baseDate, nil),
	}

	result := selectWatchNextMovies(movies, 10)

	expectedOrder := []int64{12, 11, 13, 14, 15, 10}
	if len(result) != len(expectedOrder) {
		t.Fatalf("expected %d movies, got %d", len(expectedOrder), len(result))
	}

	for i, expectedID := range expectedOrder {
		if result[i].MovieDetails.Movie.ID != expectedID {
			t.Fatalf("expected movie ID %d at index %d, got %d", expectedID, i, result[i].MovieDetails.Movie.ID)
		}
	}
}

func TestSelectWatchNextMovies_AppliesLimit(t *testing.T) {
	baseDate := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	movies := []models.MovieItem{
		newHomeMovieItem(1, "A", baseDate, int64Ptr(1)),
		newHomeMovieItem(2, "B", baseDate, int64Ptr(2)),
		newHomeMovieItem(3, "C", baseDate, nil),
	}

	result := selectWatchNextMovies(movies, 2)

	if len(result) != 2 {
		t.Fatalf("expected 2 movies, got %d", len(result))
	}

	if result[0].MovieDetails.Movie.ID != 1 || result[1].MovieDetails.Movie.ID != 2 {
		t.Fatalf("unexpected order for limited results: got [%d, %d]", result[0].MovieDetails.Movie.ID, result[1].MovieDetails.Movie.ID)
	}
}

func TestSelectWatchNextMovies_ReturnsEmptyForNoMoviesOrInvalidLimit(t *testing.T) {
	if result := selectWatchNextMovies(nil, 5); len(result) != 0 {
		t.Fatalf("expected empty result for nil movies, got %d", len(result))
	}

	movies := []models.MovieItem{
		newHomeMovieItem(1, "A", time.Now(), nil),
	}

	if result := selectWatchNextMovies(movies, 0); len(result) != 0 {
		t.Fatalf("expected empty result for non-positive limit, got %d", len(result))
	}
}

func newHomeMovieItem(id int64, title string, dateAdded time.Time, position *int64) models.MovieItem {
	return models.MovieItem{
		MovieDetails: models.MovieDetails{
			Movie: models.Movie{
				ID:    id,
				Title: title,
			},
		},
		DateAdded: dateAdded,
		Position:  position,
	}
}

func int64Ptr(v int64) *int64 {
	return &v
}
