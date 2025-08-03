package models

import "time"

// WatchedMovie is a movie and some more info relative for the specific watch
type WatchedMovie struct {
	MovieDetails MovieDetails
	Date         time.Time
	InTheaters   bool
}

// WatchedMoviesInDay represents a day and all the movies watched in that day.
// Used in the watched page to group movies by watched day
type WatchedMoviesInDay struct {
	Movies []MovieDetails
	Date   time.Time
}

type WatchedMovieRecord struct {
	Date       time.Time
	InTheaters bool
}

type WatchedMovieRecords struct {
	MovieDetails MovieDetails
	Records      []WatchedMovieRecord
}
