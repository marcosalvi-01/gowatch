package services

import (
	"fmt"
	"sort"
	"time"

	"github.com/marcosalvi-01/gowatch/internal/models"
)

const watchlistPageURL = "/watchlist"

func defaultListMovieSort(isWatchlist bool) models.ListMovieSort {
	if isWatchlist {
		return models.ListMovieSortReleaseStatus
	}

	return models.ListMovieSortCustom
}

func (s *ListService) BuildListViewData(list *models.List, isEditing bool, now time.Time) models.ListViewData {
	if list == nil {
		return models.ListViewData{}
	}

	data := models.ListViewData{
		List:          list,
		Sort:          list.DisplaySort,
		IsEditing:     isEditing,
		GridURL:       buildListMovieGridURL(list.ID, isEditing),
		ToggleEditURL: buildListPageURL(list, !isEditing),
	}

	if data.Sort == models.ListMovieSortReleaseStatus {
		data.UpcomingMovies, data.ReleasedMovies = splitMoviesByReleaseStatus(data.List.Movies, now)
	} else {
		sortListMovies(data.List.Movies, data.Sort)
	}
	data.AverageRating, data.HasAverageRating = averageMovieRating(data.List.Movies)

	return data
}

func buildListMovieGridURL(listID int64, isEditing bool) string {
	url := fmt.Sprintf("/htmx/lists/%d/movie-grid", listID)
	if isEditing {
		return url + "?edit=1"
	}

	return url
}

func buildListPageURL(list *models.List, isEditing bool) string {
	pageURL := fmt.Sprintf("/list/%d", list.ID)
	if list.IsWatchlist {
		pageURL = watchlistPageURL
	}
	if isEditing {
		return pageURL + "?edit=1"
	}

	return pageURL
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

// splitMoviesByReleaseStatus separates movies into upcoming and released buckets and applies display ordering.
func splitMoviesByReleaseStatus(movies []models.MovieItem, now time.Time) ([]models.MovieItem, []models.MovieItem) {
	today := normalizeDayUTC(now)

	upcoming := make([]models.MovieItem, 0, len(movies))
	released := make([]models.MovieItem, 0, len(movies))

	for _, movie := range movies {
		if isUpcomingMovie(movie, today) {
			upcoming = append(upcoming, movie)
			continue
		}

		released = append(released, movie)
	}

	sortUpcomingMovies(upcoming)
	sortReleasedMovies(released)

	return upcoming, released
}

func isUpcomingMovie(movie models.MovieItem, today time.Time) bool {
	releaseDate := movie.MovieDetails.Movie.ReleaseDate
	if releaseDate == nil {
		return false
	}

	return normalizeDayUTC(*releaseDate).After(today)
}

func sortUpcomingMovies(movies []models.MovieItem) {
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

func sortReleasedMovies(movies []models.MovieItem) {
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
