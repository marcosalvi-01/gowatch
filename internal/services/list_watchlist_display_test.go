package services

import (
	"testing"
	"time"

	"github.com/marcosalvi-01/gowatch/internal/models"
)

func TestSplitMoviesByReleaseStatus_SplitsAndSorts(t *testing.T) {
	now := time.Date(2026, 4, 13, 15, 0, 0, 0, time.UTC)

	releaseSoon := time.Date(2026, 4, 20, 0, 0, 0, 0, time.UTC)
	releaseLater := time.Date(2026, 5, 2, 0, 0, 0, 0, time.UTC)
	releasePast := time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC)

	movies := []models.MovieItem{
		newListMovieItem(1, "Later Upcoming", &releaseLater, time.Date(2026, 1, 2, 10, 0, 0, 0, time.UTC)),
		newListMovieItem(2, "Soon Upcoming 2", &releaseSoon, time.Date(2026, 1, 3, 10, 0, 0, 0, time.UTC)),
		newListMovieItem(3, "Soon Upcoming 1", &releaseSoon, time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)),
		newListMovieItem(4, "Released Newer", &releasePast, time.Date(2026, 1, 4, 10, 0, 0, 0, time.UTC)),
		newListMovieItem(5, "Unknown Release", nil, time.Date(2026, 1, 2, 10, 0, 0, 0, time.UTC)),
		newListMovieItem(6, "Released Older", &releasePast, time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)),
	}

	upcoming, released := splitMoviesByReleaseStatus(movies, now)

	assertMovieIDs(t, upcoming, []int64{3, 2, 1})
	assertMovieIDs(t, released, []int64{6, 5, 4})
}

func TestIsUpcomingMovie_UsesDateBoundary(t *testing.T) {
	today := time.Date(2026, 4, 13, 0, 0, 0, 0, time.UTC)
	releaseToday := time.Date(2026, 4, 13, 21, 0, 0, 0, time.UTC)
	releaseTomorrow := time.Date(2026, 4, 14, 0, 0, 0, 0, time.UTC)

	if isUpcomingMovie(newListMovieItem(1, "Today", &releaseToday, today), today) {
		t.Fatal("expected release date on same day to be treated as released")
	}

	if !isUpcomingMovie(newListMovieItem(2, "Tomorrow", &releaseTomorrow, today), today) {
		t.Fatal("expected release date after today to be treated as upcoming")
	}
}

func TestSortReleasedMovies_StableByTitleAndID(t *testing.T) {
	dateAdded := time.Date(2026, 1, 10, 10, 0, 0, 0, time.UTC)
	movies := []models.MovieItem{
		newListMovieItem(3, "B", nil, dateAdded),
		newListMovieItem(2, "A", nil, dateAdded),
		newListMovieItem(1, "A", nil, dateAdded),
	}

	sortReleasedMovies(movies)

	assertMovieIDs(t, movies, []int64{1, 2, 3})
}

func TestSortUpcomingMovies_HandlesNilReleaseDate(t *testing.T) {
	releaseSoon := time.Date(2026, 4, 20, 0, 0, 0, 0, time.UTC)
	dateAdded := time.Date(2026, 1, 10, 10, 0, 0, 0, time.UTC)

	movies := []models.MovieItem{
		newListMovieItem(2, "No Date", nil, dateAdded),
		newListMovieItem(1, "With Date", &releaseSoon, dateAdded),
	}

	sortUpcomingMovies(movies)

	assertMovieIDs(t, movies, []int64{1, 2})
}

func TestBuildListViewData_BuildsWatchlistSections(t *testing.T) {
	now := time.Date(2026, 4, 13, 15, 0, 0, 0, time.UTC)
	releaseSoon := time.Date(2026, 4, 20, 0, 0, 0, 0, time.UTC)
	releasePast := time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC)

	list := &models.List{
		ID:          7,
		Name:        "Watchlist",
		IsWatchlist: true,
		DisplaySort: models.ListMovieSortReleaseStatus,
		Movies: []models.MovieItem{
			newListMovieItem(1, "Upcoming", &releaseSoon, time.Date(2026, 1, 2, 10, 0, 0, 0, time.UTC)),
			newListMovieItem(2, "Released", &releasePast, time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)),
		},
	}

	service := &ListService{}
	data := service.BuildListViewData(list, false, now)

	if data.List.ID != list.ID {
		t.Fatalf("expected list ID %d, got %d", list.ID, data.List.ID)
	}

	assertMovieIDs(t, data.UpcomingMovies, []int64{1})
	assertMovieIDs(t, data.ReleasedMovies, []int64{2})
	if data.IsEditing {
		t.Fatal("did not expect watchlist edit mode by default")
	}
	if data.ToggleEditURL != "/watchlist?edit=1" {
		t.Fatalf("expected watchlist edit URL, got %q", data.ToggleEditURL)
	}

	editData := service.BuildListViewData(list, true, now)
	if !editData.IsEditing {
		t.Fatal("expected watchlist edit mode")
	}
	if editData.GridURL != "/htmx/lists/7/movie-grid?edit=1" {
		t.Fatalf("expected watchlist grid to preserve edit mode, got %q", editData.GridURL)
	}
	if editData.ToggleEditURL != "/watchlist" {
		t.Fatalf("expected watchlist done URL, got %q", editData.ToggleEditURL)
	}

	list.DisplaySort = models.ListMovieSortCustom
	customData := service.BuildListViewData(list, false, now)
	if customData.Sort != models.ListMovieSortCustom {
		t.Fatalf("expected custom watchlist sort, got %q", customData.Sort)
	}
}

func TestBuildListViewData_BuildsReleaseStatusSectionsForCustomList(t *testing.T) {
	now := time.Date(2026, 4, 13, 15, 0, 0, 0, time.UTC)
	releaseSoon := time.Date(2026, 4, 20, 0, 0, 0, 0, time.UTC)
	releasePast := time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC)

	list := &models.List{
		ID:          7,
		DisplaySort: models.ListMovieSortReleaseStatus,
		Movies: []models.MovieItem{
			newListMovieItem(1, "Upcoming", &releaseSoon, time.Date(2026, 1, 2, 10, 0, 0, 0, time.UTC)),
			newListMovieItem(2, "Released", &releasePast, time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)),
		},
	}

	data := (&ListService{}).BuildListViewData(list, false, now)
	if data.Sort != models.ListMovieSortReleaseStatus {
		t.Fatalf("expected release status sort, got %q", data.Sort)
	}
	assertMovieIDs(t, data.UpcomingMovies, []int64{1})
	assertMovieIDs(t, data.ReleasedMovies, []int64{2})
}

func newListMovieItem(id int64, title string, releaseDate *time.Time, dateAdded time.Time) models.MovieItem {
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

func assertMovieIDs(t *testing.T, movies []models.MovieItem, expected []int64) {
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
