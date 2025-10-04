// Package models defines the core data structures and types used throughout
// the application. This includes request/response DTOs, and shared data types.
// Models are used across all layers of the application.
package models

import (
	"fmt"
	"time"

	tmdb "github.com/cyruzin/golang-tmdb"
)

func MovieDetailsFromTMDBMovieDetails(movie tmdb.MovieDetails) (*MovieDetails, error) {
	var releaseDate *time.Time
	if movie.ReleaseDate != "" {
		date, err := time.Parse("2006-01-02", movie.ReleaseDate)
		if err != nil {
			return nil, fmt.Errorf("failed to parse movie release date '%s': %w", movie.ReleaseDate, err)
		}
		releaseDate = &date
	}
	return &MovieDetails{
		Movie: Movie{
			ID:               movie.ID,
			Title:            movie.Title,
			OriginalTitle:    movie.OriginalTitle,
			OriginalLanguage: movie.OriginalLanguage,
			Overview:         movie.Overview,
			ReleaseDate:      releaseDate,
			PosterPath:       movie.PosterPath,
			BackdropPath:     movie.BackdropPath,
			Popularity:       movie.Popularity,
			VoteCount:        movie.VoteCount,
			VoteAverage:      movie.VoteAverage,
		},
		Budget:   movie.Budget,
		Genres:   GenreFromTMDBGenres(movie.Genres),
		Homepage: movie.Homepage,
		IMDbID:   movie.IMDbID,
		Revenue:  movie.Revenue,
		Runtime:  movie.Runtime,
		Status:   movie.Status,
		Tagline:  movie.Tagline,
	}, nil
}

// Movie is a simple movie, the necessary things for a poster in every list view (search, watched, to watch, ...)
type Movie struct {
	// just simple stuff for the cards, the kind of things that a tmdb.search would give so to use it in both the search and in the watched list
	ID               int64
	Title            string
	OriginalTitle    string
	OriginalLanguage string
	Overview         string
	ReleaseDate      *time.Time
	PosterPath       string
	BackdropPath     string
	Popularity       float32
	VoteCount        int64
	VoteAverage      float32

	UpdatedAt time.Time
}

// MovieDetails is a more detailed movie with all the info necessary for a detailed view of it (e.g. when clicking on a movie after a search)
type MovieDetails struct {
	Movie Movie

	Budget   int64
	Homepage string
	IMDbID   string
	Revenue  int64
	Runtime  int
	Status   string
	Tagline  string

	Genres  []Genre
	Credits MovieCredits

	// OriginCountry []string
	// BelongsToCollection BelongsToCollection
	// ProductionCompanies []ProductionCompany
	// ProductionCountries []ProductionCountry
	// SpokenLanguages     []SpokenLanguage
}
