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
	Date   time.Time
	Movies []WatchedMovieInDay
}

type WatchedMovieInDay struct {
	MovieDetails MovieDetails
	InTheaters   bool
}

type WatchedMovieRecord struct {
	Date       time.Time
	InTheaters bool
}

type WatchedMovieRecords struct {
	MovieDetails MovieDetails
	Records      []WatchedMovieRecord
}

// WatchedStats contains all statistics for watched movies
type WatchedStats struct {
	TotalWatched      int64
	TheaterVsHome     []TheaterCount
	MonthlyLastYear   []PeriodCount
	YearlyAllTime     []PeriodCount
	Genres            []GenreCount
	MostWatchedMovies []TopMovie
	MostWatchedDay    *MostWatchedDay
	MostWatchedActors []TopActor
	AvgPerDay         float64
	AvgPerWeek        float64
	AvgPerMonth       float64
}

type PeriodCount struct {
	Period string
	Count  int64
}

type GenreCount struct {
	Name  string
	Count int64
}

type TheaterCount struct {
	InTheater bool
	Count     int64
}

type TopMovie struct {
	Title      string
	ID         int64
	PosterPath string
	WatchCount int64
}

type TopActor struct {
	Name        string
	ID          int64
	ProfilePath string
	WatchCount  int64
}

type MostWatchedDay struct {
	Date  time.Time
	Count int64
}

type DateRange struct {
	MinDate *time.Time
	MaxDate *time.Time
}
