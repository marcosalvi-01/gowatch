package services

import (
	"fmt"
	"sort"
	"time"

	"github.com/marcosalvi-01/gowatch/internal/models"
)

func (s *ListService) BuildListViewData(list *models.List, sortMode models.ListMovieSort, isEditingOrder bool, now time.Time) models.ListViewData {
	if list == nil {
		return models.ListViewData{}
	}

	data := models.ListViewData{
		List:    list,
		Sort:    sortMode,
		PageURL: fmt.Sprintf("/list/%d", list.ID),
	}
	data.GridURL = buildListMovieGridURL(list.ID, sortMode, isEditingOrder)

	if list.IsWatchlist {
		data.Sort = models.ListMovieSortCustom
		data.UpcomingMovies, data.ReleasedMovies = splitWatchlistMoviesForDisplay(list.Movies, now)
		return data
	}
	if sortMode != models.ListMovieSortCustom {
		isEditingOrder = false
	}

	data.IsEditingOrder = isEditingOrder
	data.ToggleEditURL = buildListPageURL(list.ID, sortMode, !isEditingOrder)
	sortListMovies(data.List.Movies, sortMode)
	data.AverageRating, data.HasAverageRating = averageMovieRating(data.List.Movies)

	return data
}

func buildListMovieGridURL(listID int64, sort models.ListMovieSort, isEditingOrder bool) string {
	return fmt.Sprintf("/htmx/lists/%d/movie-grid%s", listID, buildListQueryString(sort, isEditingOrder))
}

func buildListPageURL(listID int64, sort models.ListMovieSort, isEditingOrder bool) string {
	return fmt.Sprintf("/list/%d%s", listID, buildListQueryString(sort, isEditingOrder))
}

func buildListQueryString(sort models.ListMovieSort, isEditingOrder bool) string {
	query := fmt.Sprintf("?sort=%s", sort)
	if sort == models.ListMovieSortCustom && isEditingOrder {
		query += "&edit=1"
	}

	return query
}

func averageMovieRating(movies []models.MovieItem) (float32, bool) {
	var total float32
	count := 0

	for _, movie := range movies {
		if movie.MovieDetails.Movie.VoteAverage <= 0 {
			continue
		}

		total += movie.MovieDetails.Movie.VoteAverage
		count++
	}

	if count == 0 {
		return 0, false
	}

	return total / float32(count), true
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
