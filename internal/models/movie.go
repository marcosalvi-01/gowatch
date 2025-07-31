// Package models defines the core data structures and types used throughout
// the application. This includes request/response DTOs, and shared data types.
// Models are used across all layers of the application.
package models

import (
	"fmt"
	"time"

	tmdb "github.com/cyruzin/golang-tmdb"
)

type Movie struct {
	ID               int64
	IMDbID           string
	Title            string
	OriginalTitle    string
	ReleaseDate      time.Time
	OriginalLanguage string
	Overview         string
	PosterPath       string
	BackdropPath     string
	Budget           int64
	Revenue          int64
	Runtime          int64
	VoteAverage      float64
	VoteCount        int64
	Popularity       float64
	Homepage         string
	Status           string
	Tagline          string
}

func MovieFromTMDBMovieDetails(TMDBMovie tmdb.MovieDetails) (Movie, error) {
	releaseDate, err := time.Parse("2006-01-02", TMDBMovie.ReleaseDate)
	if err != nil {
		return Movie{}, fmt.Errorf("failed to parse movie release date '%s': %w", TMDBMovie.ReleaseDate, err)
	}
	return Movie{
		ID:               TMDBMovie.ID,
		IMDbID:           TMDBMovie.IMDbID,
		Title:            TMDBMovie.Title,
		OriginalTitle:    TMDBMovie.OriginalTitle,
		ReleaseDate:      releaseDate,
		OriginalLanguage: TMDBMovie.OriginalLanguage,
		Overview:         TMDBMovie.Overview,
		PosterPath:       TMDBMovie.PosterPath,
		BackdropPath:     TMDBMovie.BackdropPath,
		Budget:           TMDBMovie.Budget,
		Revenue:          TMDBMovie.Revenue,
		Runtime:          int64(TMDBMovie.Runtime),
		VoteAverage:      float64(TMDBMovie.VoteAverage),
		VoteCount:        TMDBMovie.VoteCount,
		Popularity:       float64(TMDBMovie.Popularity),
		Homepage:         TMDBMovie.Homepage,
		Status:           TMDBMovie.Status,
		Tagline:          TMDBMovie.Tagline,
	}, nil
}

type SearchMovie struct {
	ID               int64
	Title            string
	OriginalTitle    string
	OriginalLanguage string
	Overview         string
	ReleaseDate      string
	PosterPath       string
	BackdropPath     string
	Popularity       float32
	VoteCount        int64
	VoteAverage      float32
	GenreIDs         []int64
	Adult            bool
	Video            bool
}

type WatchedMovie struct {
	Movie       Movie
	WatchedDate time.Time
}

// WatchedMovieDetails represents a watched movie and some additional details about it.
// Used in the movie page details about a specific movie (if watched)
type WatchedMovieDetails struct {
	Movie     Movie
	ViewCount int
}

// WatchedDay represents a day and all the movies watched in that day.
// Used in the watched page to group movies by watched day
type WatchedDay struct {
	Movies []Movie
	Date   time.Time
}

// For importing/exporting watched movie data from JSON

type WatchedMoviesLog []WatchedMoviesEntry

type WatchedMoviesEntry struct {
	Date   time.Time         `json:"date"`
	Movies []WatchedMovieRef `json:"movies"`
}

type WatchedMovieRef struct {
	MovieID int `json:"movie_id"`
}
