package models

import "time"

type ImportWatchedMoviesLog []ImportWatchedMoviesEntry

type ImportWatchedMoviesEntry struct {
	Date   time.Time               `json:"date"`
	Movies []ImportWatchedMovieRef `json:"movies"`
}

type ImportWatchedMovieRef struct {
	MovieID    int  `json:"movie_id"`
	InTheaters bool `json:"in_theaters,omitempty"`
}
