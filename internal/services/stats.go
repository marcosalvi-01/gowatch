package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"slices"
	"sort"
	"time"

	"github.com/marcosalvi-01/gowatch/internal/common"
	"github.com/marcosalvi-01/gowatch/internal/models"
	"golang.org/x/sync/errgroup"
)

const (
	tmdbGenderFemale int64 = 1
	tmdbGenderMale   int64 = 2
)

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

func limitTopActorsByGender(actors []models.TopActor, gender int64, limit int) []models.TopActor {
	filtered := make([]models.TopActor, 0, len(actors))
	for _, actor := range actors {
		if actor.Gender != gender {
			continue
		}

		filtered = append(filtered, actor)
		if limit > 0 && len(filtered) >= limit {
			break
		}
	}

	return filtered
}

func watchedActorRankByGender(actors []models.TopActor, personID int64) *int64 {
	var currentGender int64
	var currentRank int64
	var previousWatchCount int64
	haveCurrentGender := false

	for _, actor := range actors {
		if !haveCurrentGender || actor.Gender != currentGender {
			currentGender = actor.Gender
			currentRank = 1
			previousWatchCount = actor.WatchCount
			haveCurrentGender = true
		} else if actor.WatchCount < previousWatchCount {
			currentRank++
			previousWatchCount = actor.WatchCount
		}

		if actor.ID == personID {
			rank := currentRank
			return &rank
		}
	}

	return nil
}

func filterTopCrewMembersByRole(data []models.TopCrewMemberStat, role models.TopCrewRole, limit int) []models.TopCrewMember {
	filtered := make([]models.TopCrewMember, 0, len(data))
	for _, member := range data {
		if member.RoleKey != role {
			continue
		}

		filtered = append(filtered, models.TopCrewMember{
			ID:          member.ID,
			Name:        member.Name,
			ProfilePath: member.ProfilePath,
			WatchCount:  member.WatchCount,
		})
		if limit > 0 && len(filtered) >= limit {
			break
		}
	}

	return filtered
}

func (s *WatchedService) getLongestWatchedMovie(ctx context.Context) (*models.RuntimeMovie, error) {
	s.log.Debug("retrieving longest watched movie")

	user, err := common.GetUser(ctx)
	if err != nil {
		s.log.Error("failed to get user", "error", err)
		return nil, err
	}

	movie, err := s.db.GetLongestWatchedMovie(ctx, user.ID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			s.log.Debug("no longest watched movie found")
			return nil, nil
		}
		s.log.Error("failed to retrieve longest watched movie", "error", err)
		return nil, fmt.Errorf("failed to get longest watched movie: %w", err)
	}

	return movie, nil
}

func (s *WatchedService) getShortestWatchedMovie(ctx context.Context) (*models.RuntimeMovie, error) {
	s.log.Debug("retrieving shortest watched movie")

	user, err := common.GetUser(ctx)
	if err != nil {
		s.log.Error("failed to get user", "error", err)
		return nil, err
	}

	movie, err := s.db.GetShortestWatchedMovie(ctx, user.ID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			s.log.Debug("no shortest watched movie found")
			return nil, nil
		}
		s.log.Error("failed to retrieve shortest watched movie", "error", err)
		return nil, fmt.Errorf("failed to get shortest watched movie: %w", err)
	}

	return movie, nil
}

func (s *WatchedService) getBudgetTierDistribution(ctx context.Context) ([]models.BudgetTierCount, error) {
	s.log.Debug("retrieving budget tier distribution")

	user, err := common.GetUser(ctx)
	if err != nil {
		s.log.Error("failed to get user", "error", err)
		return nil, err
	}

	data, err := s.db.GetBudgetTierDistribution(ctx, user.ID)
	if err != nil {
		s.log.Error("failed to retrieve budget tier distribution", "error", err)
		return nil, fmt.Errorf("failed to get budget tier distribution: %w", err)
	}

	return s.normalizeBudgetTierDistribution(data), nil
}

func (s *WatchedService) getTopReturnOnInvestmentMovies(ctx context.Context, limit int) ([]models.MovieFinancial, error) {
	s.log.Debug("retrieving top return on investment movies", "limit", limit)

	user, err := common.GetUser(ctx)
	if err != nil {
		s.log.Error("failed to get user", "error", err)
		return nil, err
	}

	data, err := s.db.GetTopReturnOnInvestmentMovies(ctx, user.ID, limit)
	if err != nil {
		s.log.Error("failed to retrieve top return on investment movies", "error", err)
		return nil, fmt.Errorf("failed to get top return on investment movies: %w", err)
	}

	return s.sortMoviesByROIDesc(data), nil
}

func (s *WatchedService) getBiggestBudgetMovies(ctx context.Context, limit int) ([]models.MovieFinancial, error) {
	s.log.Debug("retrieving biggest budget movies", "limit", limit)

	user, err := common.GetUser(ctx)
	if err != nil {
		s.log.Error("failed to get user", "error", err)
		return nil, err
	}

	data, err := s.db.GetBiggestBudgetMovies(ctx, user.ID, limit)
	if err != nil {
		s.log.Error("failed to retrieve biggest budget movies", "error", err)
		return nil, fmt.Errorf("failed to get biggest budget movies: %w", err)
	}

	return s.sortMoviesByBudgetDesc(data), nil
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

func (s *WatchedService) getRatingDistribution(ctx context.Context) ([]models.RatingBucketCount, error) {
	s.log.Debug("retrieving rating distribution")

	user, err := common.GetUser(ctx)
	if err != nil {
		s.log.Error("failed to get user", "error", err)
		return nil, err
	}

	data, err := s.db.GetRatingDistribution(ctx, user.ID)
	if err != nil {
		s.log.Error("failed to retrieve rating distribution", "error", err)
		return nil, fmt.Errorf("failed to get rating distribution: %w", err)
	}

	return s.normalizeRatingDistribution(data), nil
}

func (s *WatchedService) finalizeRatingSummary(summary *models.RatingSummary, totalWatched int64) models.RatingSummary {
	if summary == nil {
		return models.RatingSummary{UnratedCount: totalWatched}
	}

	result := *summary
	if result.RatedCount < 0 {
		result.RatedCount = 0
	}

	result.UnratedCount = max(totalWatched-result.RatedCount, 0)

	if totalWatched > 0 {
		result.Coverage = float64(result.RatedCount) / float64(totalWatched)
	}

	return result
}

func (s *WatchedService) normalizeRatingDistribution(data []models.RatingBucketCount) []models.RatingBucketCount {
	countsByBucket := make(map[float64]int64, len(data))
	for _, item := range data {
		bucket := math.Round(item.Rating/ratingBucketSize) * ratingBucketSize
		countsByBucket[bucket] += item.Count
	}

	bucketCount := int(maxMovieRating / ratingBucketSize)
	result := make([]models.RatingBucketCount, 0, bucketCount)
	for i := 1; i <= bucketCount; i++ {
		bucket := float64(i) * ratingBucketSize
		result = append(result, models.RatingBucketCount{
			Rating: bucket,
			Count:  countsByBucket[bucket],
		})
	}

	return result
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

	user, err := common.GetUser(ctx)
	if err != nil {
		s.log.Error("failed to get user", "error", err)
		return nil, err
	}

	summary := &models.HomeStatsSummary{}
	g, ctx := errgroup.WithContext(ctx)

	var totalStats *models.TotalStats
	var dateRange *models.DateRange
	var genres []models.GenreCount

	g.Go(func() error {
		var fetchErr error
		totalStats, fetchErr = s.db.GetTotalWatchedStats(ctx, user.ID)
		if fetchErr != nil {
			return fmt.Errorf("failed to get total stats: %w", fetchErr)
		}
		return nil
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

func (s *WatchedService) calculateStreakStats(watchedDates []time.Time, now time.Time) models.StreakStats {
	if len(watchedDates) == 0 {
		return models.StreakStats{}
	}

	normalizedDates := make([]time.Time, len(watchedDates))
	for i, watchedDate := range watchedDates {
		normalizedDates[i] = normalizeDate(watchedDate)
	}

	sort.Slice(normalizedDates, func(i, j int) bool {
		return normalizedDates[i].Before(normalizedDates[j])
	})

	uniqueDates := make([]time.Time, 0, len(normalizedDates))
	for _, watchedDate := range normalizedDates {
		if len(uniqueDates) == 0 || !uniqueDates[len(uniqueDates)-1].Equal(watchedDate) {
			uniqueDates = append(uniqueDates, watchedDate)
		}
	}

	longestDays := int64(1)
	longestStartIdx := 0
	currentRunDays := int64(1)
	currentRunStartIdx := 0

	for i := 1; i < len(uniqueDates); i++ {
		if isConsecutiveDay(uniqueDates[i-1], uniqueDates[i]) {
			currentRunDays++
			continue
		}

		if currentRunDays > longestDays {
			longestDays = currentRunDays
			longestStartIdx = currentRunStartIdx
		}

		currentRunDays = 1
		currentRunStartIdx = i
	}

	if currentRunDays > longestDays {
		longestDays = currentRunDays
		longestStartIdx = currentRunStartIdx
	}

	longestEndIdx := longestStartIdx + int(longestDays) - 1
	longestStart := uniqueDates[longestStartIdx]
	longestEnd := uniqueDates[longestEndIdx]

	today := normalizeDate(now)
	currentDays := int64(0)
	endIdx := len(uniqueDates) - 1
	targetEnd := today.AddDate(0, 0, -1)
	if uniqueDates[endIdx].Equal(today) {
		targetEnd = today
	}
	if uniqueDates[endIdx].Equal(targetEnd) {
		currentDays = 1
		for i := endIdx; i > 0; i-- {
			if !isConsecutiveDay(uniqueDates[i-1], uniqueDates[i]) {
				break
			}
			currentDays++
		}
	}

	streakStats := models.StreakStats{
		CurrentDays:  currentDays,
		LongestDays:  longestDays,
		LongestStart: &longestStart,
		LongestEnd:   &longestEnd,
	}

	s.log.Debug("calculated streak stats",
		"currentDays", streakStats.CurrentDays,
		"longestDays", streakStats.LongestDays,
		"longestStart", streakStats.LongestStart,
		"longestEnd", streakStats.LongestEnd,
	)

	return streakStats
}

func (s *WatchedService) normalizeBudgetTierDistribution(data []models.BudgetTierCount) []models.BudgetTierCount {
	budgetTierCounts := map[models.BudgetTier]int64{
		models.BudgetTierIndie:       0,
		models.BudgetTierMid:         0,
		models.BudgetTierBlockbuster: 0,
		models.BudgetTierUnknown:     0,
	}

	for _, row := range data {
		switch row.Tier {
		case models.BudgetTierIndie, models.BudgetTierMid, models.BudgetTierBlockbuster, models.BudgetTierUnknown:
			budgetTierCounts[row.Tier] += row.Count
		default:
			budgetTierCounts[models.BudgetTierUnknown] += row.Count
		}
	}

	orderedTiers := []models.BudgetTier{
		models.BudgetTierIndie,
		models.BudgetTierMid,
		models.BudgetTierBlockbuster,
		models.BudgetTierUnknown,
	}

	result := make([]models.BudgetTierCount, 0, len(orderedTiers))
	for _, tier := range orderedTiers {
		result = append(result, models.BudgetTierCount{
			Tier:  tier,
			Count: budgetTierCounts[tier],
		})
	}

	return result
}

func (s *WatchedService) sortMoviesByROIDesc(movies []models.MovieFinancial) []models.MovieFinancial {
	result := make([]models.MovieFinancial, 0, len(movies))
	for _, movie := range movies {
		if movie.Budget > 0 {
			result = append(result, movie)
		}
	}

	sort.Slice(result, func(i, j int) bool {
		if result[i].ROI != result[j].ROI {
			return result[i].ROI > result[j].ROI
		}
		if result[i].Revenue != result[j].Revenue {
			return result[i].Revenue > result[j].Revenue
		}
		return result[i].Title < result[j].Title
	})

	return result
}

func (s *WatchedService) sortMoviesByBudgetDesc(movies []models.MovieFinancial) []models.MovieFinancial {
	result := make([]models.MovieFinancial, 0, len(movies))
	for _, movie := range movies {
		if movie.Budget > 0 {
			result = append(result, movie)
		}
	}

	sort.Slice(result, func(i, j int) bool {
		if result[i].Budget != result[j].Budget {
			return result[i].Budget > result[j].Budget
		}
		if result[i].Revenue != result[j].Revenue {
			return result[i].Revenue > result[j].Revenue
		}
		return result[i].Title < result[j].Title
	})

	return result
}

func normalizeDate(t time.Time) time.Time {
	year, month, day := t.Date()
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}

func isConsecutiveDay(previous, current time.Time) bool {
	return previous.AddDate(0, 0, 1).Equal(current)
}
