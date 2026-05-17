package services

import (
	"fmt"
	"sort"
	"strings"

	"github.com/marcosalvi-01/gowatch/internal/models"
)

const (
	listMovieMoveFirst = "first"
	listMovieMoveUp    = "up"
	listMovieMoveDown  = "down"
	listMovieMoveLast  = "last"
)

func ParseListMovieSort(raw string) models.ListMovieSort {
	switch models.ListMovieSort(raw) {
	case models.ListMovieSortCustom,
		models.ListMovieSortDateAddedDesc,
		models.ListMovieSortDateAddedAsc,
		models.ListMovieSortTitleAsc,
		models.ListMovieSortRatingDesc,
		models.ListMovieSortReleaseDateDesc:
		return models.ListMovieSort(raw)
	default:
		return models.ListMovieSortCustom
	}
}

func IsListMovieOrderEditing(raw string, sortMode models.ListMovieSort) bool {
	if sortMode != models.ListMovieSortCustom {
		return false
	}

	return raw == "1" || strings.EqualFold(raw, "true")
}

func sortListMovies(movies []models.MovieItem, sortMode models.ListMovieSort) {
	sort.Slice(movies, func(i, j int) bool {
		left := movies[i]
		right := movies[j]

		switch sortMode {
		case models.ListMovieSortDateAddedDesc:
			if !left.DateAdded.Equal(right.DateAdded) {
				return left.DateAdded.After(right.DateAdded)
			}
		case models.ListMovieSortDateAddedAsc:
			if !left.DateAdded.Equal(right.DateAdded) {
				return left.DateAdded.Before(right.DateAdded)
			}
		case models.ListMovieSortTitleAsc:
			if left.MovieDetails.Movie.Title != right.MovieDetails.Movie.Title {
				return left.MovieDetails.Movie.Title < right.MovieDetails.Movie.Title
			}
		case models.ListMovieSortRatingDesc:
			if left.MovieDetails.Movie.VoteAverage != right.MovieDetails.Movie.VoteAverage {
				return left.MovieDetails.Movie.VoteAverage > right.MovieDetails.Movie.VoteAverage
			}
		case models.ListMovieSortReleaseDateDesc:
			leftDate := left.MovieDetails.Movie.ReleaseDate
			rightDate := right.MovieDetails.Movie.ReleaseDate

			switch {
			case leftDate == nil && rightDate != nil:
				return false
			case leftDate != nil && rightDate == nil:
				return true
			case leftDate != nil && rightDate != nil && !leftDate.Equal(*rightDate):
				return leftDate.After(*rightDate)
			}
		default:
			if compareByCustomPosition(left, right) != 0 {
				return compareByCustomPosition(left, right) < 0
			}
		}

		return compareListMovieFallback(left, right) < 0
	})
}

func compareByCustomPosition(left, right models.MovieItem) int {
	leftHasPosition := left.Position != nil
	rightHasPosition := right.Position != nil

	switch {
	case leftHasPosition && rightHasPosition:
		if *left.Position != *right.Position {
			if *left.Position < *right.Position {
				return -1
			}
			return 1
		}
	case leftHasPosition:
		return -1
	case rightHasPosition:
		return 1
	}

	return 0
}

func compareListMovieFallback(left, right models.MovieItem) int {
	if !left.DateAdded.Equal(right.DateAdded) {
		if left.DateAdded.Before(right.DateAdded) {
			return -1
		}
		return 1
	}

	if left.MovieDetails.Movie.Title != right.MovieDetails.Movie.Title {
		if left.MovieDetails.Movie.Title < right.MovieDetails.Movie.Title {
			return -1
		}
		return 1
	}

	if left.MovieDetails.Movie.ID < right.MovieDetails.Movie.ID {
		return -1
	}
	if left.MovieDetails.Movie.ID > right.MovieDetails.Movie.ID {
		return 1
	}

	return 0
}

func nextCustomListPosition(movies []models.MovieItem) int64 {
	var maxPosition int64
	for _, movie := range movies {
		if movie.Position != nil && *movie.Position > maxPosition {
			maxPosition = *movie.Position
		}
	}

	return maxPosition + 1
}

func reorderMovieIDsForCustomSort(movies []models.MovieItem, movieID int64, move string) ([]int64, error) {
	if len(movies) == 0 {
		return nil, fmt.Errorf("list has no movies")
	}

	ordered := make([]models.MovieItem, len(movies))
	copy(ordered, movies)
	sortListMovies(ordered, models.ListMovieSortCustom)

	currentIndex := -1
	for i, movie := range ordered {
		if movie.MovieDetails.Movie.ID == movieID {
			currentIndex = i
			break
		}
	}

	if currentIndex == -1 {
		return nil, fmt.Errorf("movie %d not found in list", movieID)
	}

	targetIndex := currentIndex
	switch move {
	case listMovieMoveFirst:
		targetIndex = 0
	case listMovieMoveUp:
		if currentIndex > 0 {
			targetIndex = currentIndex - 1
		}
	case listMovieMoveDown:
		if currentIndex < len(ordered)-1 {
			targetIndex = currentIndex + 1
		}
	case listMovieMoveLast:
		targetIndex = len(ordered) - 1
	default:
		return nil, fmt.Errorf("invalid move action %q", move)
	}

	if targetIndex != currentIndex {
		moved := ordered[currentIndex]
		ordered = append(ordered[:currentIndex], ordered[currentIndex+1:]...)
		ordered = append(ordered[:targetIndex], append([]models.MovieItem{moved}, ordered[targetIndex:]...)...)
	}

	movieIDs := make([]int64, len(ordered))
	for i, movie := range ordered {
		movieIDs[i] = movie.MovieDetails.Movie.ID
	}

	return movieIDs, nil
}
