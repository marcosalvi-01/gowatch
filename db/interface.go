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
	GetWatchedJoinMovie(ctx context.Context, userID int64) ([]models.WatchedMovie, error)
	GetWatchedJoinMovieByID(ctx context.Context, userID, movieID int64) ([]models.WatchedMovie, error)
	GetRecentWatchedMovies(ctx context.Context, userID int64, limit int) ([]models.WatchedMovieInDay, error)
	GetWatchedCount(ctx context.Context, userID int64) (int64, error)
	GetWatchedPerMonthLastYear(ctx context.Context, userID int64) ([]models.PeriodCount, error)
	GetWatchedPerYear(ctx context.Context, userID int64) ([]models.PeriodCount, error)
	GetWeekdayDistribution(ctx context.Context, userID int64) ([]models.PeriodCount, error)
	GetWatchedByGenre(ctx context.Context, userID int64) ([]models.GenreCount, error)
	GetTheaterVsHomeCount(ctx context.Context, userID int64) ([]models.TheaterCount, error)
	GetMostWatchedMovies(ctx context.Context, userID int64, limit int) ([]models.TopMovie, error)
	GetMostWatchedDay(ctx context.Context, userID int64) (*models.MostWatchedDay, error)
	GetMostWatchedMaleActors(ctx context.Context, userID int64, limit int) ([]models.TopActor, error)
	GetMostWatchedFemaleActors(ctx context.Context, userID int64, limit int) ([]models.TopActor, error)
	GetWatchedDateRange(ctx context.Context, userID int64) (*models.DateRange, error)
	GetWatchedHoursPerMonthLastYear(ctx context.Context, userID int64) ([]models.PeriodHours, error)
	GetTotalHoursWatched(ctx context.Context, userID int64) (float64, error)
	GetMonthlyGenreBreakdown(ctx context.Context, userID int64) ([]models.MonthlyGenreBreakdown, error)

	InsertList(ctx context.Context, list InsertList) (int64, error)

	GetList(ctx context.Context, userID, id int64) (*models.List, error)
	GetAllLists(ctx context.Context, userID int64) ([]InsertList, error)
	AddMovieToList(ctx context.Context, userID int64, insertMovieList InsertMovieList) error
	DeleteListByID(ctx context.Context, userID, id int64) error
	DeleteMovieFromList(ctx context.Context, userID, listID, movieID int64) error

	CreateSession(ctx context.Context, id string, userID int64, expiresAt time.Time) error
	GetSession(ctx context.Context, id string) (*models.Session, error)
	DeleteSession(ctx context.Context, id string) error
	CleanupExpiredSessions(ctx context.Context) error

	CreateUser(ctx context.Context, email, name, passwordHash string) (int64, error)
	GetUserByEmail(ctx context.Context, email string) (*models.User, error)
	GetUserByID(ctx context.Context, id int64) (*models.User, error)

	AssignNilUserLists(ctx context.Context, userID *int64) error
	AssignNilUserWatched(ctx context.Context, userID *int64) error
	CountUsers(ctx context.Context) (int64, error)
}

type InsertList struct {
	UserID      int64
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
	UserID     int64
	MovieID    int64
	Date       time.Time
	InTheaters bool
}
