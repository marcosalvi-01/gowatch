package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"
	"sort"
	"time"

	"github.com/marcosalvi-01/gowatch/internal/common"
	"github.com/marcosalvi-01/gowatch/internal/models"
	"golang.org/x/sync/errgroup"
)

func (s *WatchedService) getTotalStats(ctx context.Context) (*models.TotalStats, error) {
	s.log.Debug("retrieving total stats")
	user, err := common.GetUser(ctx)
	if err != nil {
		s.log.Error("failed to get user", "error", err)
		return nil, err
	}
	stats, err := s.db.GetTotalWatchedStats(ctx, user.ID)
	if err != nil {
		s.log.Error("failed to retrieve total stats", "error", err)
		return nil, fmt.Errorf("failed to get total stats: %w", err)
	}
	return stats, nil
}

func (s *WatchedService) getTheaterVsHome(ctx context.Context) ([]models.TheaterCount, error) {
	s.log.Debug("retrieving theater vs home counts")
	user, err := common.GetUser(ctx)
	if err != nil {
		s.log.Error("failed to get user", "error", err)
		return nil, err
	}
	data, err := s.db.GetTheaterVsHomeCount(ctx, user.ID)
	if err != nil {
		s.log.Error("failed to retrieve theater vs home counts", "error", err)
		return nil, fmt.Errorf("failed to get theater vs home: %w", err)
	}
	return data, nil
}

func (s *WatchedService) getMonthlyStats(ctx context.Context) ([]models.PeriodStats, error) {
	s.log.Debug("retrieving monthly stats")
	user, err := common.GetUser(ctx)
	if err != nil {
		s.log.Error("failed to get user", "error", err)
		return nil, err
	}
	data, err := s.db.GetWatchedStatsPerMonthLastYear(ctx, user.ID)
	if err != nil {
		s.log.Error("failed to retrieve monthly stats", "error", err)
		return nil, fmt.Errorf("failed to get monthly stats: %w", err)
	}
	return data, nil
}

func (s *WatchedService) getYearlyAllTime(ctx context.Context) ([]models.PeriodCount, error) {
	s.log.Debug("retrieving yearly watched data")
	user, err := common.GetUser(ctx)
	if err != nil {
		s.log.Error("failed to get user", "error", err)
		return nil, err
	}
	data, err := s.db.GetWatchedPerYear(ctx, user.ID)
	if err != nil {
		s.log.Error("failed to retrieve yearly watched data", "error", err)
		return nil, fmt.Errorf("failed to get yearly data: %w", err)
	}
	return data, nil
}

func (s *WatchedService) getWeekdayDistribution(ctx context.Context) ([]models.PeriodCount, error) {
	s.log.Debug("retrieving weekday distribution")
	user, err := common.GetUser(ctx)
	if err != nil {
		s.log.Error("failed to get user", "error", err)
		return nil, err
	}
	data, err := s.db.GetWeekdayDistribution(ctx, user.ID)
	if err != nil {
		s.log.Error("failed to retrieve weekday distribution", "error", err)
		return nil, fmt.Errorf("failed to get weekday distribution: %w", err)
	}
	return data, nil
}

func (s *WatchedService) getGenres(ctx context.Context) ([]models.GenreCount, error) {
	s.log.Debug("retrieving watched by genre data")
	user, err := common.GetUser(ctx)
	if err != nil {
		s.log.Error("failed to get user", "error", err)
		return nil, err
	}
	genreData, err := s.db.GetWatchedByGenre(ctx, user.ID)
	if err != nil {
		s.log.Error("failed to retrieve watched by genre data", "error", err)
		return nil, fmt.Errorf("failed to get genre data: %w", err)
	}
	return s.aggregateGenres(genreData, MaxGenresDisplayed), nil
}

func (s *WatchedService) getMostWatchedMovies(ctx context.Context, limit int) ([]models.TopMovie, error) {
	s.log.Debug("retrieving most watched movies", "limit", limit)
	user, err := common.GetUser(ctx)
	if err != nil {
		s.log.Error("failed to get user", "error", err)
		return nil, err
	}
	data, err := s.db.GetMostWatchedMovies(ctx, user.ID, limit)
	if err != nil {
		s.log.Error("failed to retrieve most watched movies", "error", err)
		return nil, fmt.Errorf("failed to get most watched movies: %w", err)
	}
	return data, nil
}

func (s *WatchedService) getMostWatchedDay(ctx context.Context) (*models.MostWatchedDay, error) {
	s.log.Debug("retrieving most watched day")
	user, err := common.GetUser(ctx)
	if err != nil {
		s.log.Error("failed to get user", "error", err)
		return nil, err
	}
	dayData, err := s.db.GetMostWatchedDay(ctx, user.ID)
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

	user, err := common.GetUser(ctx)
	if err != nil {
		s.log.Error("failed to get user", "error", err)
		return nil, err
	}

	var maleActors, femaleActors []models.TopActor
	g, ctx := errgroup.WithContext(ctx)

	// Fetch males and females separately to ensure we get enough of each
	// Gender 2 = Male, Gender 1 = Female in TMDB
	g.Go(func() error {
		var err error
		maleActors, err = s.db.GetMostWatchedActorsByGender(ctx, user.ID, 2, limit)
		if err != nil {
			s.log.Error("failed to retrieve most watched male actors", "error", err)
			return fmt.Errorf("failed to get most watched male actors: %w", err)
		}
		return nil
	})

	g.Go(func() error {
		var err error
		femaleActors, err = s.db.GetMostWatchedActorsByGender(ctx, user.ID, 1, limit)
		if err != nil {
			s.log.Error("failed to retrieve most watched female actors", "error", err)
			return fmt.Errorf("failed to get most watched female actors: %w", err)
		}
		return nil
	})

	if err := g.Wait(); err != nil {
		return nil, err
	}

	allActors := append(maleActors, femaleActors...)

	return allActors, nil
}

func (s *WatchedService) getDateRange(ctx context.Context) (*models.DateRange, error) {
	s.log.Debug("retrieving watched date range")
	user, err := common.GetUser(ctx)
	if err != nil {
		s.log.Error("failed to get user", "error", err)
		return nil, err
	}
	dateRange, err := s.db.GetWatchedDateRange(ctx, user.ID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			s.log.Debug("no valid watched dates found")
			return &models.DateRange{}, nil
		}
		s.log.Error("failed to retrieve watched date range", "error", err)
		return nil, fmt.Errorf("failed to get date range: %w", err)
	}
	return dateRange, nil
}

func (s *WatchedService) getMonthlyGenreBreakdown(ctx context.Context) ([]models.MonthlyGenreBreakdown, error) {
	s.log.Debug("retrieving monthly genre breakdown")
	user, err := common.GetUser(ctx)
	if err != nil {
		s.log.Error("failed to get user", "error", err)
		return nil, err
	}
	data, err := s.db.GetMonthlyGenreBreakdown(ctx, user.ID)
	if err != nil {
		s.log.Error("failed to retrieve monthly genre breakdown", "error", err)
		return nil, fmt.Errorf("failed to get monthly genre breakdown: %w", err)
	}

	aggregated := s.aggregateTopGenresForChart(data, MaxGenresDisplayed)
	return aggregated, nil
}

func (s *WatchedService) calculateHoursAverages(totalHours float64, dateRange *models.DateRange, now time.Time) (avgPerDay, avgPerWeek, avgPerMonth float64) {
	if dateRange == nil || dateRange.MinDate == nil || dateRange.MaxDate == nil {
		return 0, 0, 0
	}

	// Use the provided now time as the end date if it's later than the max watched date
	// to account for gaps in activity.
	endDate := *dateRange.MaxDate
	if now.After(endDate) {
		endDate = now
	}

	days := endDate.Sub(*dateRange.MinDate).Hours()/24 + 1
	avgPerDay = totalHours / days

	// Use a minimum divisor to avoid extreme projections for small datasets
	weekDivisor := days / 7
	if weekDivisor < 1 {
		weekDivisor = 1
	}
	avgPerWeek = totalHours / weekDivisor

	monthDivisor := days / 30
	if monthDivisor < 1 {
		monthDivisor = 1
	}
	avgPerMonth = totalHours / monthDivisor

	s.log.Debug("calculated hours averages", "avgPerDay", avgPerDay, "avgPerWeek", avgPerWeek, "avgPerMonth", avgPerMonth)
	return avgPerDay, avgPerWeek, avgPerMonth
}

func (s *WatchedService) calculateMonthlyHoursTrend(monthlyHours []models.PeriodHours) (direction models.TrendDirection, value float64) {
	if len(monthlyHours) == 0 {
		return models.TrendNeutral, 0
	}

	if len(monthlyHours) == 1 {
		return models.TrendUp, monthlyHours[0].Hours
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
	if len(monthlyMovies) == 0 {
		return models.TrendNeutral, 0
	}

	if len(monthlyMovies) == 1 {
		return models.TrendUp, monthlyMovies[0].Count
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

func (s *WatchedService) GetHomeStatsSummary(ctx context.Context) (*models.HomeStatsSummary, error) {
	s.log.Debug("retrieving home stats summary")

	summary := &models.HomeStatsSummary{}
	g, ctx := errgroup.WithContext(ctx)

	var totalStats *models.TotalStats
	var dateRange *models.DateRange
	var genres []models.GenreCount

	g.Go(func() error {
		var err error
		totalStats, err = s.getTotalStats(ctx)
		return err
	})

	g.Go(func() error {
		var err error
		dateRange, err = s.getDateRange(ctx)
		return err
	})

	g.Go(func() error {
		var err error
		genres, err = s.getGenres(ctx)
		return err
	})

	if err := g.Wait(); err != nil {
		return nil, err
	}

	summary.TotalWatched = totalStats.Count
	_, summary.AvgPerWeek, _ = s.calculateAverages(totalStats.Count, dateRange, time.Now())

	if len(genres) > 0 {
		summary.TopGenre = &genres[0]
	}

	return summary, nil
}

func (s *WatchedService) calculateAverages(total int64, dateRange *models.DateRange, now time.Time) (avgPerDay, avgPerWeek, avgPerMonth float64) {
	if dateRange == nil || dateRange.MinDate == nil || dateRange.MaxDate == nil {
		return 0, 0, 0
	}

	// Use the provided now time as the end date if it's later than the max watched date
	// to account for gaps in activity.
	endDate := *dateRange.MaxDate
	if now.After(endDate) {
		endDate = now
	}

	days := endDate.Sub(*dateRange.MinDate).Hours()/24 + 1
	avgPerDay = float64(total) / days

	// Use a minimum divisor to avoid extreme projections for small datasets
	weekDivisor := days / 7
	if weekDivisor < 1 {
		weekDivisor = 1
	}
	avgPerWeek = float64(total) / weekDivisor

	monthDivisor := days / 30
	if monthDivisor < 1 {
		monthDivisor = 1
	}
	avgPerMonth = float64(total) / monthDivisor

	s.log.Debug("calculated averages", "avgPerDay", avgPerDay, "avgPerWeek", avgPerWeek, "avgPerMonth", avgPerMonth)
	return avgPerDay, avgPerWeek, avgPerMonth
}
