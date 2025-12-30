package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"
	"sort"

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

func (s *WatchedService) getWeekdayDistribution(ctx context.Context) ([]models.PeriodCount, error) {
	s.log.Debug("retrieving weekday distribution")
	data, err := s.db.GetWeekdayDistribution(ctx)
	if err != nil {
		s.log.Error("failed to retrieve weekday distribution", "error", err)
		return nil, fmt.Errorf("failed to get weekday distribution: %w", err)
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

func (s *WatchedService) getMostWatchedMovies(ctx context.Context, limit int) ([]models.TopMovie, error) {
	s.log.Debug("retrieving most watched movies", "limit", limit)
	data, err := s.db.GetMostWatchedMovies(ctx, limit)
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

func (s *WatchedService) getMostWatchedActors(ctx context.Context, limit int) ([]models.TopActor, error) {
	s.log.Debug("retrieving most watched actors", "limit", limit)

	maleActors, err := s.db.GetMostWatchedMaleActors(ctx, limit)
	if err != nil {
		s.log.Error("failed to retrieve most watched male actors", "error", err)
		return nil, fmt.Errorf("failed to get most watched male actors: %w", err)
	}

	femaleActors, err := s.db.GetMostWatchedFemaleActors(ctx, limit)
	if err != nil {
		s.log.Error("failed to retrieve most watched female actors", "error", err)
		return nil, fmt.Errorf("failed to get most watched female actors: %w", err)
	}

	// Combine male and female actors
	allActors := append(maleActors, femaleActors...)

	// Sort by watch count descending
	sort.Slice(allActors, func(i, j int) bool {
		return allActors[i].WatchCount > allActors[j].WatchCount
	})

	s.log.Debug("retrieved most watched actors", "count", len(allActors))
	return allActors, nil
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

func (s *WatchedService) getTotalHoursWatched(ctx context.Context) (float64, error) {
	s.log.Debug("retrieving total hours watched")
	total, err := s.db.GetTotalHoursWatched(ctx)
	if err != nil {
		s.log.Error("failed to retrieve total hours watched", "error", err)
		return 0, fmt.Errorf("failed to get total hours watched: %w", err)
	}
	return total, nil
}

func (s *WatchedService) getMonthlyHoursLastYear(ctx context.Context) ([]models.PeriodHours, error) {
	s.log.Debug("retrieving monthly hours data")
	data, err := s.db.GetWatchedHoursPerMonthLastYear(ctx)
	if err != nil {
		s.log.Error("failed to retrieve monthly hours data", "error", err)
		return nil, fmt.Errorf("failed to get monthly hours data: %w", err)
	}
	return data, nil
}

func (s *WatchedService) getMonthlyGenreBreakdown(ctx context.Context) ([]models.MonthlyGenreBreakdown, error) {
	s.log.Debug("retrieving monthly genre breakdown")
	data, err := s.db.GetMonthlyGenreBreakdown(ctx)
	if err != nil {
		s.log.Error("failed to retrieve monthly genre breakdown", "error", err)
		return nil, fmt.Errorf("failed to get monthly genre breakdown: %w", err)
	}

	aggregated := s.aggregateTopGenresForChart(data, MaxGenresDisplayed)
	return aggregated, nil
}

func (s *WatchedService) getHoursAverages(ctx context.Context, totalHours float64) (float64, float64, float64, error) {
	s.log.Debug("retrieving watched date range for hours average calculations")
	dateRange, err := s.db.GetWatchedDateRange(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			s.log.Debug("no valid watched dates found, skipping hours average calculations")
			return 0, 0, 0, nil
		}
		s.log.Error("failed to retrieve watched date range", "error", err)
		return 0, 0, 0, fmt.Errorf("failed to get date range: %w", err)
	}
	avgHoursPerDay, avgHoursPerWeek, avgHoursPerMonth := s.calculateHoursAverages(totalHours, dateRange)
	return avgHoursPerDay, avgHoursPerWeek, avgHoursPerMonth, nil
}

func (s *WatchedService) calculateHoursAverages(totalHours float64, dateRange *models.DateRange) (avgPerDay, avgPerWeek, avgPerMonth float64) {
	if dateRange == nil || dateRange.MinDate == nil || dateRange.MaxDate == nil {
		return 0, 0, 0
	}
	days := dateRange.MaxDate.Sub(*dateRange.MinDate).Hours()/24 + 1
	avgPerDay = totalHours / days
	avgPerWeek = totalHours / (days / 7)
	avgPerMonth = totalHours / (days / 30)
	s.log.Debug("calculated hours averages", "avgPerDay", avgPerDay, "avgPerWeek", avgPerWeek, "avgPerMonth", avgPerMonth)
	return avgPerDay, avgPerWeek, avgPerMonth
}

func (s *WatchedService) calculateMonthlyHoursTrend(monthlyHours []models.PeriodHours) (direction models.TrendDirection, value float64) {
	if len(monthlyHours) < 2 {
		return models.TrendNeutral, 0
	}

	// Sort by period for chronological order
	sort.Slice(monthlyHours, func(i, j int) bool {
		return monthlyHours[i].Period < monthlyHours[j].Period
	})

	// Compare last month vs previous month
	lastIdx := len(monthlyHours) - 1
	prevIdx := lastIdx - 1

	lastMonth := monthlyHours[lastIdx].Hours
	prevMonth := monthlyHours[prevIdx].Hours

	diff := lastMonth - prevMonth

	// Determine direction based on the difference
	if diff > 0 {
		return models.TrendUp, diff
	} else if diff < 0 {
		return models.TrendDown, diff
	}
	return models.TrendNeutral, diff
}

func (s *WatchedService) calculateMonthlyMoviesTrend(monthlyMovies []models.PeriodCount) (direction models.TrendDirection, value int64) {
	if len(monthlyMovies) < 2 {
		return models.TrendNeutral, 0
	}

	// Sort by period for chronological order
	sort.Slice(monthlyMovies, func(i, j int) bool {
		return monthlyMovies[i].Period < monthlyMovies[j].Period
	})

	// Compare last month vs previous month
	lastIdx := len(monthlyMovies) - 1
	prevIdx := lastIdx - 1

	lastMonth := monthlyMovies[lastIdx].Count
	prevMonth := monthlyMovies[prevIdx].Count

	diff := lastMonth - prevMonth

	if diff > 0 {
		return models.TrendUp, diff
	} else if diff < 0 {
		return models.TrendDown, diff
	}
	return models.TrendNeutral, diff
}

func (s *WatchedService) aggregateTopGenresForChart(data []models.MonthlyGenreBreakdown, topN int) []models.MonthlyGenreBreakdown {
	// Calculate total counts across all months to determine top genres
	genreTotals := make(map[string]int)
	for _, month := range data {
		for genre, count := range month.Genres {
			genreTotals[genre] += count
		}
	}

	// Sort genres by total count to find top N
	type genreCount struct {
		name  string
		count int
	}
	var sortedGenres []genreCount
	for genre, count := range genreTotals {
		sortedGenres = append(sortedGenres, genreCount{name: genre, count: count})
	}
	sort.Slice(sortedGenres, func(i, j int) bool {
		return sortedGenres[i].count > sortedGenres[j].count
	})

	// Take top N genres, rest go to "Other"
	topGenres := make([]string, 0, topN)
	for i := 0; i < len(sortedGenres) && i < topN; i++ {
		topGenres = append(topGenres, sortedGenres[i].name)
	}

	// Create result with top genres + Other
	result := make([]models.MonthlyGenreBreakdown, len(data))
	for i, month := range data {
		genres := make(map[string]int)
		otherCount := 0

		for genre, count := range month.Genres {
			isTopGenre := false
			if slices.Contains(topGenres, genre) {
				genres[genre] = count
				isTopGenre = true
			}
			if !isTopGenre {
				otherCount += count
			}
		}

		if otherCount > 0 {
			genres["Others"] = otherCount
		}

		result[i] = models.MonthlyGenreBreakdown{
			Month:  month.Month,
			Genres: genres,
		}
	}

	return result
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
