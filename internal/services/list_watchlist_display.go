package services

import (
	"sort"
	"time"

	"github.com/marcosalvi-01/gowatch/internal/models"
)

func (s *ListService) BuildListGridData(list *models.List, now time.Time) models.ListGridData {
	if list == nil {
		return models.ListGridData{}
	}

	data := models.ListGridData{
		List: *list,
	}

	if !list.IsWatchlist {
		return data
	}

	data.UpcomingMovies, data.ReleasedMovies = splitWatchlistMoviesForDisplay(list.Movies, now)
	return data
}

// splitWatchlistMoviesForDisplay separates watchlist movies into upcoming and released buckets and applies display ordering.
func splitWatchlistMoviesForDisplay(movies []models.MovieItem, now time.Time) ([]models.MovieItem, []models.MovieItem) {
	today := normalizeDayUTC(now)

	upcoming := make([]models.MovieItem, 0, len(movies))
	released := make([]models.MovieItem, 0, len(movies))

	for _, movie := range movies {
		if isUpcomingWatchlistMovie(movie, today) {
			upcoming = append(upcoming, movie)
			continue
		}

		released = append(released, movie)
	}

	sortWatchlistUpcomingMovies(upcoming)
	sortWatchlistReleasedMovies(released)

	return upcoming, released
}

func isUpcomingWatchlistMovie(movie models.MovieItem, today time.Time) bool {
	releaseDate := movie.MovieDetails.Movie.ReleaseDate
	if releaseDate == nil {
		return false
	}

	return normalizeDayUTC(*releaseDate).After(today)
}

func sortWatchlistUpcomingMovies(movies []models.MovieItem) {
	sort.Slice(movies, func(i, j int) bool {
		left := movies[i]
		right := movies[j]

		leftReleaseDate := left.MovieDetails.Movie.ReleaseDate
		rightReleaseDate := right.MovieDetails.Movie.ReleaseDate

		switch {
		case leftReleaseDate == nil && rightReleaseDate != nil:
			return false
		case leftReleaseDate != nil && rightReleaseDate == nil:
			return true
		case leftReleaseDate != nil && rightReleaseDate != nil:
			leftDay := normalizeDayUTC(*leftReleaseDate)
			rightDay := normalizeDayUTC(*rightReleaseDate)

			if !leftDay.Equal(rightDay) {
				return leftDay.Before(rightDay)
			}
		}

		if !left.DateAdded.Equal(right.DateAdded) {
			return left.DateAdded.Before(right.DateAdded)
		}

		if left.MovieDetails.Movie.Title != right.MovieDetails.Movie.Title {
			return left.MovieDetails.Movie.Title < right.MovieDetails.Movie.Title
		}

		return left.MovieDetails.Movie.ID < right.MovieDetails.Movie.ID
	})
}

func sortWatchlistReleasedMovies(movies []models.MovieItem) {
	sort.Slice(movies, func(i, j int) bool {
		left := movies[i]
		right := movies[j]

		if !left.DateAdded.Equal(right.DateAdded) {
			return left.DateAdded.Before(right.DateAdded)
		}

		if left.MovieDetails.Movie.Title != right.MovieDetails.Movie.Title {
			return left.MovieDetails.Movie.Title < right.MovieDetails.Movie.Title
		}

		return left.MovieDetails.Movie.ID < right.MovieDetails.Movie.ID
	})
}

func normalizeDayUTC(t time.Time) time.Time {
	utc := t.UTC()
	return time.Date(utc.Year(), utc.Month(), utc.Day(), 0, 0, 0, 0, time.UTC)
}
