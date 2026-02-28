package services

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"sort"

	"github.com/marcosalvi-01/gowatch/internal/models"
	"github.com/marcosalvi-01/gowatch/logging"
)

const (
	RecentMoviesLimit    = 10
	WatchNextMoviesLimit = 5
)

// HomeService handles aggregating data for the home page
type HomeService struct {
	watched *WatchedService
	list    *ListService
	log     *slog.Logger
}

func NewHomeService(watched *WatchedService, list *ListService) *HomeService {
	log := logging.Get("home service")
	log.Debug("creating new HomeService instance")
	return &HomeService{
		watched: watched,
		list:    list,
		log:     log,
	}
}

func (s *HomeService) GetHomeData(ctx context.Context) (*models.HomeData, error) {
	s.log.Debug("aggregating home page data")

	// Get recent movies
	recentMovies, err := s.watched.GetRecentWatchedMovies(ctx, RecentMoviesLimit)
	if err != nil {
		s.log.Error("failed to retrieve recent movies", "error", err)
		return nil, err
	}

	// Get stats summary
	statsSummary, err := s.watched.GetHomeStatsSummary(ctx)
	if err != nil {
		s.log.Error("failed to retrieve stats summary", "error", err)
		return nil, err
	}

	// Get daily watch counts for heatmap
	dailyWatchCountsLastYear, err := s.watched.GetDailyWatchCountsLastYear(ctx)
	if err != nil {
		s.log.Error("failed to retrieve daily watch counts for home", "error", err)
		return nil, err
	}

	watchlist, err := s.list.GetWatchlist(ctx)
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			s.log.Error("failed to retrieve watchlist", "error", err)
			return nil, err
		}

		s.log.Warn("watchlist not found, defaulting to empty watchlist")
	}

	watchNextMovies := []models.MovieItem{}
	watchlistMovieCount := 0
	if watchlist != nil {
		watchlistMovieCount = len(watchlist.Movies)
		watchNextMovies = selectWatchNextMovies(watchlist.Movies, WatchNextMoviesLimit)
	}

	homeData := &models.HomeData{
		RecentMovies:             recentMovies,
		WatchNextMovies:          watchNextMovies,
		WatchlistMovieCount:      watchlistMovieCount,
		DailyWatchCountsLastYear: dailyWatchCountsLastYear,
		Stats:                    *statsSummary,
	}

	s.log.Info("successfully aggregated home data")
	return homeData, nil
}

func selectWatchNextMovies(movies []models.MovieItem, limit int) []models.MovieItem {
	if len(movies) == 0 || limit <= 0 {
		return []models.MovieItem{}
	}

	sorted := make([]models.MovieItem, len(movies))
	copy(sorted, movies)

	sort.Slice(sorted, func(i, j int) bool {
		left := sorted[i]
		right := sorted[j]

		leftHasPosition := left.Position != nil
		rightHasPosition := right.Position != nil

		switch {
		case leftHasPosition && rightHasPosition:
			if *left.Position != *right.Position {
				return *left.Position < *right.Position
			}
		case leftHasPosition:
			return true
		case rightHasPosition:
			return false
		}

		if !left.DateAdded.Equal(right.DateAdded) {
			return left.DateAdded.Before(right.DateAdded)
		}

		if left.MovieDetails.Movie.Title != right.MovieDetails.Movie.Title {
			return left.MovieDetails.Movie.Title < right.MovieDetails.Movie.Title
		}

		return left.MovieDetails.Movie.ID < right.MovieDetails.Movie.ID
	})

	if len(sorted) > limit {
		return sorted[:limit]
	}

	return sorted
}
