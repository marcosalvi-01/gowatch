package db

import (
	"context"
	"gowatch/internal/models"
	"time"
)

type DB interface {
	Close() error
	Health() error

	GetMovieDetailsByID(ctx context.Context, id int64) (*models.MovieDetails, error)
	UpsertMovie(ctx context.Context, movie *models.MovieDetails) error

	InsertWatched(ctx context.Context, watched InsertWatched) error
	GetWatchedJoinMovie(ctx context.Context) ([]models.WatchedMovie, error)
	GetWatchedJoinMovieByID(ctx context.Context, movieID int64) ([]models.WatchedMovie, error)
	GetWatchedCount(ctx context.Context) (int, error)

	InsertList(ctx context.Context, list InsertList) error
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
