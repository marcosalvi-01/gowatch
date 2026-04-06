package db

import (
	"context"
	"time"

	"github.com/marcosalvi-01/gowatch/internal/models"
)

type DB interface {
	Close() error
	Health() error

	// Movie catalog and metadata.
	GetMovieDetailsByID(ctx context.Context, movieID int64) (*models.MovieDetails, error)
	UpsertMovie(ctx context.Context, movie *models.MovieDetails) error

	// Watched history and activity.
	InsertWatched(ctx context.Context, watched InsertWatched) error
	GetWatchedJoinMovie(ctx context.Context, userID int64) ([]models.WatchedMovie, error)
	GetWatchedJoinMovieByID(ctx context.Context, userID, movieID int64) ([]models.WatchedMovie, error)
	GetRecentWatchedMovies(ctx context.Context, userID int64, limit int) ([]models.WatchedMovieInDay, error)
	GetWatchedCount(ctx context.Context, userID int64) (int64, error)
	GetWatchedDateRange(ctx context.Context, userID int64) (*models.DateRange, error)
	GetWatchedDates(ctx context.Context, userID int64) ([]time.Time, error)

	// Watched stats.
	GetTotalWatchedStats(ctx context.Context, userID int64) (*models.TotalStats, error)
	GetWatchedStatsPerMonthLastYear(ctx context.Context, userID int64) ([]models.PeriodStats, error)
	GetWatchedPerYear(ctx context.Context, userID int64) ([]models.PeriodCount, error)
	GetWeekdayDistribution(ctx context.Context, userID int64) ([]models.PeriodCount, error)
	GetDailyWatchCountsLastYear(ctx context.Context, userID int64) ([]models.DailyWatchCount, error)
	GetRewatchStats(ctx context.Context, userID int64) (*models.RewatchStats, error)
	GetWatchedByGenre(ctx context.Context, userID int64) ([]models.GenreCount, error)
	GetTheaterVsHomeCount(ctx context.Context, userID int64) ([]models.TheaterCount, error)
	GetMostWatchedMovies(ctx context.Context, userID int64, limit int) ([]models.TopMovie, error)
	GetMostWatchedDay(ctx context.Context, userID int64) (*models.MostWatchedDay, error)
	GetMostWatchedActorsByGender(ctx context.Context, userID int64, gender int64, limit int) ([]models.TopActor, error)
	GetTopDirectors(ctx context.Context, userID int64, limit int) ([]models.TopCrewMember, error)
	GetTopWriters(ctx context.Context, userID int64, limit int) ([]models.TopCrewMember, error)
	GetTopComposers(ctx context.Context, userID int64, limit int) ([]models.TopCrewMember, error)
	GetTopCinematographers(ctx context.Context, userID int64, limit int) ([]models.TopCrewMember, error)
	GetTopLanguages(ctx context.Context, userID int64, limit int) ([]models.LanguageCount, error)
	GetReleaseYearDistribution(ctx context.Context, userID int64) ([]models.ReleaseYearCount, error)
	GetMonthlyGenreBreakdown(ctx context.Context, userID int64) ([]models.MonthlyGenreBreakdown, error)
	GetLongestWatchedMovie(ctx context.Context, userID int64) (*models.RuntimeMovie, error)
	GetShortestWatchedMovie(ctx context.Context, userID int64) (*models.RuntimeMovie, error)
	GetBudgetTierDistribution(ctx context.Context, userID int64) ([]models.BudgetTierCount, error)
	GetTopReturnOnInvestmentMovies(ctx context.Context, userID int64, limit int) ([]models.MovieFinancial, error)
	GetBiggestBudgetMovies(ctx context.Context, userID int64, limit int) ([]models.MovieFinancial, error)

	// Rating stats.
	GetRatingSummary(ctx context.Context, userID int64) (*models.RatingSummary, error)
	GetRatingDistribution(ctx context.Context, userID int64) ([]models.RatingBucketCount, error)
	GetMonthlyAverageRatingLastYear(ctx context.Context, userID int64) ([]models.PeriodRating, error)
	GetTheaterVsHomeAverageRating(ctx context.Context, userID int64) ([]models.TheaterRating, error)
	GetHighestRatedMovies(ctx context.Context, userID int64, limit int) ([]models.RatedMovie, error)
	GetRatingVsTMDB(ctx context.Context, userID int64, minVoteCount int) (*models.RatingVsTMDB, error)
	GetRatingByReleaseDecade(ctx context.Context, userID int64) ([]models.DecadeRating, error)
	GetFavoriteDirectorsByRating(ctx context.Context, userID int64, minRatedMovies, limit int) ([]models.RatedPerson, error)
	GetFavoriteActorsByRating(ctx context.Context, userID int64, minRatedMovies, limit int) ([]models.RatedPerson, error)
	GetRewatchRatingDrift(ctx context.Context, userID int64, minRatedWatches, limit int) ([]models.RewatchRatingDrift, error)

	// Lists and watchlist.
	InsertList(ctx context.Context, list InsertList) (int64, error)
	GetList(ctx context.Context, userID, listID int64) (*models.List, error)
	GetAllLists(ctx context.Context, userID int64) ([]InsertList, error)
	ExportLists(ctx context.Context, userID int64) ([]models.List, error)
	AddMovieToList(ctx context.Context, userID int64, insertMovieList InsertMovieList) error
	UpsertMovieInList(ctx context.Context, userID int64, insertMovieList InsertMovieList) error
	DeleteListByID(ctx context.Context, userID, listID int64) error
	DeleteMovieFromList(ctx context.Context, userID, listID, movieID int64) error
	GetWatchlistID(ctx context.Context, userID int64) (int64, error)

	// Sessions.
	CreateSession(ctx context.Context, sessionID string, userID int64, expiresAt time.Time) error
	GetSession(ctx context.Context, sessionID string) (*models.Session, error)
	DeleteSession(ctx context.Context, sessionID string) error
	CleanupExpiredSessions(ctx context.Context) error

	// Users and authentication.
	CreateUser(ctx context.Context, email, name, passwordHash string) (*models.User, error)
	GetUserByEmail(ctx context.Context, email string) (*models.User, error)
	GetUserByID(ctx context.Context, userID int64) (*models.User, error)

	// User data maintenance and admin.
	AssignNilUserLists(ctx context.Context, userID *int64) error
	AssignNilUserWatched(ctx context.Context, userID *int64) error
	CountUsers(ctx context.Context) (int64, error)
	SetAdmin(ctx context.Context, userID int64) error
	GetAllUsersWithStats(ctx context.Context) ([]models.UserWithStats, error)
	DeleteUser(ctx context.Context, userID int64) error
	UpdateUserPassword(ctx context.Context, userID int64, passwordHash string) error
	UpdatePasswordResetRequired(ctx context.Context, userID int64, reset bool) error
}

type InsertList struct {
	UserID      int64
	ID          int64
	Name        string
	Description *string
	IsWatchlist bool
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
	Rating     *float64
}
