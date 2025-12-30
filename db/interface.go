package db

import (
	"context"
	"time"

	"gowatch/internal/models"
)

type DB interface {
	Close() error
	Health() error

	GetMovieDetailsByID(ctx context.Context, id int64) (*models.MovieDetails, error)
	UpsertMovie(ctx context.Context, movie *models.MovieDetails) error

	InsertWatched(ctx context.Context, watched InsertWatched) error
	GetWatchedJoinMovie(ctx context.Context) ([]models.WatchedMovie, error)
	GetWatchedJoinMovieByID(ctx context.Context, movieID int64) ([]models.WatchedMovie, error)
	GetRecentWatchedMovies(ctx context.Context, limit int) ([]models.WatchedMovieInDay, error)
	GetWatchedCount(ctx context.Context) (int64, error)
	GetWatchedPerMonthLastYear(ctx context.Context) ([]models.PeriodCount, error)
	GetWatchedPerYear(ctx context.Context) ([]models.PeriodCount, error)
	GetWeekdayDistribution(ctx context.Context) ([]models.PeriodCount, error)
	GetWatchedByGenre(ctx context.Context) ([]models.GenreCount, error)
	GetTheaterVsHomeCount(ctx context.Context) ([]models.TheaterCount, error)
	GetMostWatchedMovies(ctx context.Context, limit int) ([]models.TopMovie, error)
	GetMostWatchedDay(ctx context.Context) (*models.MostWatchedDay, error)
	GetMostWatchedMaleActors(ctx context.Context, limit int) ([]models.TopActor, error)
	GetMostWatchedFemaleActors(ctx context.Context, limit int) ([]models.TopActor, error)
	GetWatchedDateRange(ctx context.Context) (*models.DateRange, error)
	GetWatchedHoursPerMonthLastYear(ctx context.Context) ([]models.PeriodHours, error)
	GetTotalHoursWatched(ctx context.Context) (float64, error)

	InsertList(ctx context.Context, list InsertList) (int64, error)

	GetList(ctx context.Context, id int64) (*models.List, error)
	GetAllLists(ctx context.Context) ([]InsertList, error)
	AddMovieToList(ctx context.Context, insertMovieList InsertMovieList) error
	DeleteListByID(ctx context.Context, id int64) error
	DeleteMovieFromList(ctx context.Context, listID, movieID int64) error
}

type InsertList struct {
	ID          int64
	Name        string
	Description *string
}

type InsertMovieList struct {
	MovieID   int64
	ListID    int64
	DateAdded time.Time
	Position  *int64
	Note      *string
}

type InsertWatched struct {
	MovieID    int64
	Date       time.Time
	InTheaters bool
}
