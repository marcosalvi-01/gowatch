package services

import (
	"testing"
	"time"

	"github.com/marcosalvi-01/gowatch/internal/models"
)

func TestSplitWatchlistMoviesForDisplay_SplitsAndSorts(t *testing.T) {
	now := time.Date(2026, 4, 13, 15, 0, 0, 0, time.UTC)

	releaseSoon := time.Date(2026, 4, 20, 0, 0, 0, 0, time.UTC)
	releaseLater := time.Date(2026, 5, 2, 0, 0, 0, 0, time.UTC)
	releasePast := time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC)

	movies := []models.MovieItem{
		newWatchlistMovieItem(1, "Later Upcoming", &releaseLater, time.Date(2026, 1, 2, 10, 0, 0, 0, time.UTC)),
		newWatchlistMovieItem(2, "Soon Upcoming 2", &releaseSoon, time.Date(2026, 1, 3, 10, 0, 0, 0, time.UTC)),
		newWatchlistMovieItem(3, "Soon Upcoming 1", &releaseSoon, time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)),
		newWatchlistMovieItem(4, "Released Newer", &releasePast, time.Date(2026, 1, 4, 10, 0, 0, 0, time.UTC)),
		newWatchlistMovieItem(5, "Unknown Release", nil, time.Date(2026, 1, 2, 10, 0, 0, 0, time.UTC)),
		newWatchlistMovieItem(6, "Released Older", &releasePast, time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)),
	}

	upcoming, released := splitWatchlistMoviesForDisplay(movies, now)

	assertWatchlistMovieIDs(t, upcoming, []int64{3, 2, 1})
	assertWatchlistMovieIDs(t, released, []int64{6, 5, 4})
}

func TestIsUpcomingWatchlistMovie_UsesDateBoundary(t *testing.T) {
	today := time.Date(2026, 4, 13, 0, 0, 0, 0, time.UTC)
	releaseToday := time.Date(2026, 4, 13, 21, 0, 0, 0, time.UTC)
	releaseTomorrow := time.Date(2026, 4, 14, 0, 0, 0, 0, time.UTC)

	if isUpcomingWatchlistMovie(newWatchlistMovieItem(1, "Today", &releaseToday, today), today) {
		t.Fatal("expected release date on same day to be treated as released")
	}

	if !isUpcomingWatchlistMovie(newWatchlistMovieItem(2, "Tomorrow", &releaseTomorrow, today), today) {
		t.Fatal("expected release date after today to be treated as upcoming")
	}
}

func TestSortWatchlistReleasedMovies_StableByTitleAndID(t *testing.T) {
	dateAdded := time.Date(2026, 1, 10, 10, 0, 0, 0, time.UTC)
	movies := []models.MovieItem{
		newWatchlistMovieItem(3, "B", nil, dateAdded),
		newWatchlistMovieItem(2, "A", nil, dateAdded),
		newWatchlistMovieItem(1, "A", nil, dateAdded),
	}

	sortWatchlistReleasedMovies(movies)

	assertWatchlistMovieIDs(t, movies, []int64{1, 2, 3})
}

func TestSortWatchlistUpcomingMovies_HandlesNilReleaseDate(t *testing.T) {
	releaseSoon := time.Date(2026, 4, 20, 0, 0, 0, 0, time.UTC)
	dateAdded := time.Date(2026, 1, 10, 10, 0, 0, 0, time.UTC)

	movies := []models.MovieItem{
		newWatchlistMovieItem(2, "No Date", nil, dateAdded),
		newWatchlistMovieItem(1, "With Date", &releaseSoon, dateAdded),
	}

	sortWatchlistUpcomingMovies(movies)

	assertWatchlistMovieIDs(t, movies, []int64{1, 2})
}

func TestBuildListGridData_BuildsWatchlistSections(t *testing.T) {
	now := time.Date(2026, 4, 13, 15, 0, 0, 0, time.UTC)
	releaseSoon := time.Date(2026, 4, 20, 0, 0, 0, 0, time.UTC)
	releasePast := time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC)

	list := &models.List{
		ID:          7,
		Name:        "Watchlist",
		IsWatchlist: true,
		Movies: []models.MovieItem{
			newWatchlistMovieItem(1, "Upcoming", &releaseSoon, time.Date(2026, 1, 2, 10, 0, 0, 0, time.UTC)),
			newWatchlistMovieItem(2, "Released", &releasePast, time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)),
		},
	}

	service := &ListService{}
	data := service.BuildListGridData(list, now)

	if data.List.ID != list.ID {
		t.Fatalf("expected list ID %d, got %d", list.ID, data.List.ID)
	}

	assertWatchlistMovieIDs(t, data.UpcomingMovies, []int64{1})
	assertWatchlistMovieIDs(t, data.ReleasedMovies, []int64{2})
}

func newWatchlistMovieItem(id int64, title string, releaseDate *time.Time, dateAdded time.Time) models.MovieItem {
	return models.MovieItem{
		MovieDetails: models.MovieDetails{
			Movie: models.Movie{
				ID:          id,
				Title:       title,
				ReleaseDate: releaseDate,
			},
		},
		DateAdded: dateAdded,
	}
}

func assertWatchlistMovieIDs(t *testing.T, movies []models.MovieItem, expected []int64) {
	t.Helper()

	if len(movies) != len(expected) {
		t.Fatalf("expected %d movies, got %d", len(expected), len(movies))
	}

	for i, expectedID := range expected {
		if movies[i].MovieDetails.Movie.ID != expectedID {
			t.Fatalf("expected movie ID %d at index %d, got %d", expectedID, i, movies[i].MovieDetails.Movie.ID)
		}
	}
}
