package models

import "time"

type ImportWatchedMoviesLog []ImportWatchedMoviesEntry

type ImportWatchedMoviesEntry struct {
	Date   time.Time               `json:"date"`
	Movies []ImportWatchedMovieRef `json:"movies"`
}

type ImportWatchedMovieRef struct {
	MovieID    int64    `json:"movie_id"`
	InTheaters bool     `json:"in_theaters"`
	Rating     *float64 `json:"rating,omitempty"`
}

type ImportListsLog []ImportListEntry

type ImportListEntry struct {
	Name        string               `json:"name"`
	Description *string              `json:"description,omitempty"`
	IsWatchlist bool                 `json:"is_watchlist,omitempty"`
	Movies      []ImportListMovieRef `json:"movies"`
}

type ImportListMovieRef struct {
	MovieID   int64     `json:"movie_id"`
	DateAdded time.Time `json:"date_added"`
	Position  *int64    `json:"position,omitempty"`
	Note      *string   `json:"note,omitempty"`
}

type ImportAllData struct {
	Watched ImportWatchedMoviesLog `json:"watched"`
	Lists   ImportListsLog         `json:"lists"`
}
