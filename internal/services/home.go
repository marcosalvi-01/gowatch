package services

import (
	"context"
	"log/slog"

	"github.com/marcosalvi-01/gowatch/internal/models"
	"github.com/marcosalvi-01/gowatch/logging"
)

const RecentMoviesLimit = 10

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

	homeData := &models.HomeData{
		RecentMovies: recentMovies,
		Stats:        *statsSummary,
	}

	s.log.Info("successfully aggregated home data")
	return homeData, nil
}
