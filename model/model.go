package model

import "time"

type Watched struct {
	ID   int64   `json:"id"`
	Date *string `json:"date"`
}

type Movie struct {
	ID               int64     `json:"id"`
	IMDbID           string    `json:"imdb_id"`
	Title            string    `json:"name"`
	ReleaseDate      time.Time `json:"release"`
	OriginalLanguage string    `json:"original_language"`
	Overview         string    `json:"overview"`
	PosterPath       string    `json:"poster_path"`
	Budget           int64     `json:"budget"`
	Revenue          int64     `json:"revenue"`
	Runtime          int64     `json:"runtime"`
	VoteAverage      float64   `json:"vote_average"`

	WatchedDate *time.Time `json:"watched_date,omitempty"`
}
