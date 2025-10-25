package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"gowatch/internal/models"
)

func (s *WatchedService) getTotalWatched(ctx context.Context) (int64, error) {
	s.log.Debug("retrieving total watched count")
	total, err := s.db.GetWatchedCount(ctx)
	if err != nil {
		s.log.Error("failed to retrieve total watched count", "error", err)
		return 0, fmt.Errorf("failed to get total watched: %w", err)
	}
	return total, nil
}

func (s *WatchedService) getTheaterVsHome(ctx context.Context) ([]models.TheaterCount, error) {
	s.log.Debug("retrieving theater vs home counts")
	data, err := s.db.GetTheaterVsHomeCount(ctx)
	if err != nil {
		s.log.Error("failed to retrieve theater vs home counts", "error", err)
		return nil, fmt.Errorf("failed to get theater vs home: %w", err)
	}
	return data, nil
}

func (s *WatchedService) getMonthlyLastYear(ctx context.Context) ([]models.PeriodCount, error) {
	s.log.Debug("retrieving monthly watched data")
	data, err := s.db.GetWatchedPerMonthLastYear(ctx)
	if err != nil {
		s.log.Error("failed to retrieve monthly watched data", "error", err)
		return nil, fmt.Errorf("failed to get monthly data: %w", err)
	}
	return data, nil
}

func (s *WatchedService) getYearlyAllTime(ctx context.Context) ([]models.PeriodCount, error) {
	s.log.Debug("retrieving yearly watched data")
	data, err := s.db.GetWatchedPerYear(ctx)
	if err != nil {
		s.log.Error("failed to retrieve yearly watched data", "error", err)
		return nil, fmt.Errorf("failed to get yearly data: %w", err)
	}
	return data, nil
}

func (s *WatchedService) getGenres(ctx context.Context) ([]models.GenreCount, error) {
	s.log.Debug("retrieving watched by genre data")
	genreData, err := s.db.GetWatchedByGenre(ctx)
	if err != nil {
		s.log.Error("failed to retrieve watched by genre data", "error", err)
		return nil, fmt.Errorf("failed to get genre data: %w", err)
	}
	return s.aggregateGenres(genreData, MaxGenresDisplayed), nil
}

func (s *WatchedService) getMostWatchedMovies(ctx context.Context) ([]models.TopMovie, error) {
	s.log.Debug("retrieving most watched movies")
	data, err := s.db.GetMostWatchedMovies(ctx)
	if err != nil {
		s.log.Error("failed to retrieve most watched movies", "error", err)
		return nil, fmt.Errorf("failed to get most watched movies: %w", err)
	}
	return data, nil
}

func (s *WatchedService) getMostWatchedDay(ctx context.Context) (*models.MostWatchedDay, error) {
	s.log.Debug("retrieving most watched day")
	dayData, err := s.db.GetMostWatchedDay(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			s.log.Debug("no watched days found")
			return nil, nil
		}
		s.log.Error("failed to retrieve most watched day", "error", err)
		return nil, fmt.Errorf("failed to get most watched day: %w", err)
	}
	return dayData, nil
}

func (s *WatchedService) getMostWatchedActors(ctx context.Context) ([]models.TopActor, error) {
	s.log.Debug("retrieving most watched actors")
	data, err := s.db.GetMostWatchedActors(ctx)
	if err != nil {
		s.log.Error("failed to retrieve most watched actors", "error", err)
		return nil, fmt.Errorf("failed to get most watched actors: %w", err)
	}
	return data, nil
}

func (s *WatchedService) getAverages(ctx context.Context, total int64) (float64, float64, float64, error) {
	s.log.Debug("retrieving watched date range for average calculations")
	dateRange, err := s.db.GetWatchedDateRange(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			s.log.Debug("no valid watched dates found, skipping average calculations")
			return 0, 0, 0, nil
		}
		s.log.Error("failed to retrieve watched date range", "error", err)
		return 0, 0, 0, fmt.Errorf("failed to get date range: %w", err)
	}
	avgPerDay, avgPerWeek, avgPerMonth := s.calculateAverages(total, dateRange)
	return avgPerDay, avgPerWeek, avgPerMonth, nil
}

func (s *WatchedService) aggregateGenres(genreData []models.GenreCount, maxDisplayed int) []models.GenreCount {
	if len(genreData) <= maxDisplayed {
		return genreData
	}

	genres := make([]models.GenreCount, maxDisplayed+1)
	copy(genres, genreData[:maxDisplayed])
	var othersCount int64
	for i := maxDisplayed; i < len(genreData); i++ {
		othersCount += genreData[i].Count
	}
	genres[maxDisplayed] = models.GenreCount{
		Name:  "Others",
		Count: othersCount,
	}
	return genres
}

func (s *WatchedService) calculateAverages(total int64, dateRange *models.DateRange) (avgPerDay, avgPerWeek, avgPerMonth float64) {
	if dateRange == nil || dateRange.MinDate == nil || dateRange.MaxDate == nil {
		return 0, 0, 0
	}
	days := dateRange.MaxDate.Sub(*dateRange.MinDate).Hours()/24 + 1
	avgPerDay = float64(total) / days
	avgPerWeek = float64(total) / (days / 7)
	avgPerMonth = float64(total) / (days / 30)
	s.log.Debug("calculated averages", "avgPerDay", avgPerDay, "avgPerWeek", avgPerWeek, "avgPerMonth", avgPerMonth)
	return avgPerDay, avgPerWeek, avgPerMonth
}
