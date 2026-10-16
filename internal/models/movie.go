// Package models defines the core data structures and types used throughout
// the application. This includes request/response DTOs, and shared data types.
// Models are used across all layers of the application.
package models

import (
	"time"
)

// Movie is a simple movie, the necessary things for a poster in every list view (search, watched, to watch, ...)
type Movie struct {
	// Fields used by movie cards in search, watched, and list views.
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
