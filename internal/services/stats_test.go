package services

import (
	"log/slog"
	"math"
	"reflect"
	"testing"
	"time"

	"github.com/marcosalvi-01/gowatch/internal/models"
)

func TestAggregateGenres(t *testing.T) {
	s := &WatchedService{} // No dependencies needed

	tests := []struct {
		name     string
		input    []models.GenreCount
		max      int
		expected []models.GenreCount
	}{
		{
			name:     "no aggregation needed",
			input:    []models.GenreCount{{Name: "Action", Count: 10}, {Name: "Drama", Count: 5}},
			max:      5,
			expected: []models.GenreCount{{Name: "Action", Count: 10}, {Name: "Drama", Count: 5}},
		},
		{
			name:  "aggregation with others",
			input: []models.GenreCount{{Name: "Action", Count: 10}, {Name: "Drama", Count: 5}, {Name: "Comedy", Count: 3}},
			max:   2,
			expected: []models.GenreCount{
				{Name: "Action", Count: 10},
				{Name: "Drama", Count: 5},
				{Name: "Others", Count: 3},
			},
		},
		{
			name:     "empty input",
			input:    []models.GenreCount{},
			max:      5,
			expected: []models.GenreCount{},
		},
		{
			name:     "single genre",
			input:    []models.GenreCount{{Name: "Horror", Count: 1}},
			max:      5,
			expected: []models.GenreCount{{Name: "Horror", Count: 1}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := s.aggregateGenres(tt.input, tt.max)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("aggregateGenres() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestCalculateAverages(t *testing.T) {
	s := &WatchedService{log: slog.New(slog.DiscardHandler)} // Dummy logger

	minDate := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	maxDate := time.Date(2023, 1, 3, 0, 0, 0, 0, time.UTC) // 3 days
	now := maxDate                                         // For standard tests, now = maxDate

	tests := []struct {
		name      string
		total     int64
		dateRange *models.DateRange
		now       time.Time
		expected  [3]float64 // [avgPerDay, avgPerWeek, avgPerMonth]
	}{
		{
			name:      "valid date range",
			total:     6,
			dateRange: &models.DateRange{MinDate: &minDate, MaxDate: &maxDate},
			now:       now,
			expected:  [3]float64{2, 6, 6}, // 2 per day, 6/1=6 per week, 6/1=6 per month
		},
		{
			name:      "nil date range",
			total:     10,
			dateRange: nil,
			now:       now,
			expected:  [3]float64{0, 0, 0},
		},
		{
			name:      "same_day",
			total:     5,
			dateRange: &models.DateRange{MinDate: &minDate, MaxDate: &minDate},
			now:       minDate,             // Now = MinDate, so 1 day span
			expected:  [3]float64{5, 5, 5}, // 5/1=5 per day, 5/1=5 per week, 5/1=5 per month
		},
		{
			name:      "same_day_with_gap",
			total:     5,
			dateRange: &models.DateRange{MinDate: &minDate, MaxDate: &minDate},
			now:       now,                                  // Now = Jan 3 (3 days span)
			expected:  [3]float64{1.6666666666666667, 5, 5}, // 5/3 per day, 5/1=5 per week, 5/1=5 per month
		},
		{
			name:      "long gap since last movie",
			total:     30,
			dateRange: &models.DateRange{MinDate: &minDate, MaxDate: &maxDate}, // 3 days activity
			now:       time.Date(2023, 1, 30, 0, 0, 0, 0, time.UTC),            // 30 days total span
			expected:  [3]float64{1, 7, 30},                                    // 30/30=1 per day, 30/(30/7)=7 per week, 30/(30/30)=30 per month
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			avgPerDay, avgPerWeek, avgPerMonth := s.calculateAverages(tt.total, tt.dateRange, tt.now)
			result := [3]float64{avgPerDay, avgPerWeek, avgPerMonth}
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("calculateAverages() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestFinalizeRatingSummary(t *testing.T) {
	s := &WatchedService{}

	summary := s.finalizeRatingSummary(&models.RatingSummary{
		AverageRating: 4.0,
		RatedCount:    5,
	}, 6)

	if summary.UnratedCount != 1 {
		t.Fatalf("expected unrated count 1, got %d", summary.UnratedCount)
	}

	if math.Abs(summary.Coverage-(5.0/6.0)) > 0.0001 {
		t.Fatalf("expected coverage close to %f, got %f", 5.0/6.0, summary.Coverage)
	}
}

func TestNormalizeRatingDistribution(t *testing.T) {
	s := &WatchedService{}

	result := s.normalizeRatingDistribution([]models.RatingBucketCount{
		{Rating: 1.5, Count: 2},
		{Rating: 4.0, Count: 3},
		{Rating: 5.0, Count: 1},
	})

	if len(result) != 10 {
		t.Fatalf("expected 10 rating buckets, got %d", len(result))
	}

	if result[2].Rating != 1.5 || result[2].Count != 2 {
		t.Fatalf("expected 1.5 bucket count 2, got %+v", result[2])
	}

	if result[7].Rating != 4.0 || result[7].Count != 3 {
		t.Fatalf("expected 4.0 bucket count 3, got %+v", result[7])
	}

	if result[9].Rating != 5.0 || result[9].Count != 1 {
		t.Fatalf("expected 5.0 bucket count 1, got %+v", result[9])
	}
}

func TestWatchedActorRankByGender(t *testing.T) {
	actors := []models.TopActor{
		{ID: 1, Name: "Female One", Gender: tmdbGenderFemale, WatchCount: 5},
		{ID: 2, Name: "Female Two", Gender: tmdbGenderFemale, WatchCount: 5},
		{ID: 3, Name: "Female Three", Gender: tmdbGenderFemale, WatchCount: 3},
		{ID: 4, Name: "Male One", Gender: tmdbGenderMale, WatchCount: 4},
		{ID: 5, Name: "Male Two", Gender: tmdbGenderMale, WatchCount: 2},
	}

	femaleRank := watchedActorRankByGender(actors, 3)
	if femaleRank == nil || *femaleRank != 2 {
		t.Fatalf("expected female actor rank 2, got %v", femaleRank)
	}

	maleRank := watchedActorRankByGender(actors, 5)
	if maleRank == nil || *maleRank != 2 {
		t.Fatalf("expected male actor rank 2, got %v", maleRank)
	}

	missingRank := watchedActorRankByGender(actors, 99)
	if missingRank != nil {
		t.Fatalf("expected nil rank for missing actor, got %v", *missingRank)
	}
}

func TestCalculateMonthlyMoviesTrend(t *testing.T) {
	s := &WatchedService{log: slog.New(slog.DiscardHandler)}

	tests := []struct {
		name          string
		input         []models.PeriodCount
		expectedDir   models.TrendDirection
		expectedValue int64
	}{
		{
			name:          "empty input",
			input:         []models.PeriodCount{},
			expectedDir:   models.TrendNeutral,
			expectedValue: 0,
		},
		{
			name:          "single month",
			input:         []models.PeriodCount{{Period: "2023-01", Count: 5}},
			expectedDir:   models.TrendUp,
			expectedValue: 5,
		},
		{
			name: "trend up",
			input: []models.PeriodCount{
				{Period: "2023-01", Count: 5},
				{Period: "2023-02", Count: 8},
			},
			expectedDir:   models.TrendUp,
			expectedValue: 3,
		},
		{
			name: "trend down",
			input: []models.PeriodCount{
				{Period: "2023-01", Count: 10},
				{Period: "2023-02", Count: 4},
			},
			expectedDir:   models.TrendDown,
			expectedValue: -6,
		},
		{
			name: "unsorted input",
			input: []models.PeriodCount{
				{Period: "2023-02", Count: 8},
				{Period: "2023-01", Count: 5},
			},
			expectedDir:   models.TrendUp,
			expectedValue: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir, val := s.calculateMonthlyMoviesTrend(tt.input)
			if dir != tt.expectedDir || val != tt.expectedValue {
				t.Errorf("calculateMonthlyMoviesTrend() = (%v, %v), want (%v, %v)", dir, val, tt.expectedDir, tt.expectedValue)
			}
		})
	}
}

func TestCalculateStreakStats(t *testing.T) {
	s := &WatchedService{log: slog.New(slog.DiscardHandler)}

	tests := []struct {
		name            string
		watchedDates    []time.Time
		now             time.Time
		expectedCurrent int64
		expectedLongest int64
		expectedStart   string
		expectedEnd     string
	}{
		{
			name:            "empty dates",
			watchedDates:    nil,
			now:             time.Date(2024, 1, 10, 12, 0, 0, 0, time.UTC),
			expectedCurrent: 0,
			expectedLongest: 0,
		},
		{
			name: "single day streak",
			watchedDates: []time.Time{
				time.Date(2024, 1, 10, 8, 0, 0, 0, time.UTC),
			},
			now:             time.Date(2024, 1, 10, 12, 0, 0, 0, time.UTC),
			expectedCurrent: 1,
			expectedLongest: 1,
			expectedStart:   "2024-01-10",
			expectedEnd:     "2024-01-10",
		},
		{
			name: "consecutive with duplicates",
			watchedDates: []time.Time{
				time.Date(2024, 1, 8, 8, 0, 0, 0, time.UTC),
				time.Date(2024, 1, 8, 21, 0, 0, 0, time.UTC),
				time.Date(2024, 1, 9, 8, 0, 0, 0, time.UTC),
				time.Date(2024, 1, 10, 8, 0, 0, 0, time.UTC),
			},
			now:             time.Date(2024, 1, 10, 23, 0, 0, 0, time.UTC),
			expectedCurrent: 3,
			expectedLongest: 3,
			expectedStart:   "2024-01-08",
			expectedEnd:     "2024-01-10",
		},
		{
			name: "current streak ends yesterday",
			watchedDates: []time.Time{
				time.Date(2024, 1, 8, 8, 0, 0, 0, time.UTC),
				time.Date(2024, 1, 9, 8, 0, 0, 0, time.UTC),
			},
			now:             time.Date(2024, 1, 10, 12, 0, 0, 0, time.UTC),
			expectedCurrent: 2,
			expectedLongest: 2,
			expectedStart:   "2024-01-08",
			expectedEnd:     "2024-01-09",
		},
		{
			name: "longest streak in the past",
			watchedDates: []time.Time{
				time.Date(2024, 1, 1, 8, 0, 0, 0, time.UTC),
				time.Date(2024, 1, 2, 8, 0, 0, 0, time.UTC),
				time.Date(2024, 1, 3, 8, 0, 0, 0, time.UTC),
				time.Date(2024, 1, 6, 8, 0, 0, 0, time.UTC),
				time.Date(2024, 1, 7, 8, 0, 0, 0, time.UTC),
			},
			now:             time.Date(2024, 1, 10, 8, 0, 0, 0, time.UTC),
			expectedCurrent: 0,
			expectedLongest: 3,
			expectedStart:   "2024-01-01",
			expectedEnd:     "2024-01-03",
		},
		{
			name: "watched today but streak already broken before yesterday",
			watchedDates: []time.Time{
				time.Date(2024, 1, 7, 8, 0, 0, 0, time.UTC),
				time.Date(2024, 1, 8, 8, 0, 0, 0, time.UTC),
				time.Date(2024, 1, 10, 8, 0, 0, 0, time.UTC),
			},
			now:             time.Date(2024, 1, 10, 12, 0, 0, 0, time.UTC),
			expectedCurrent: 1,
			expectedLongest: 2,
			expectedStart:   "2024-01-07",
			expectedEnd:     "2024-01-08",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stats := s.calculateStreakStats(tt.watchedDates, tt.now)

			if stats.CurrentDays != tt.expectedCurrent {
				t.Errorf("CurrentDays = %d, want %d", stats.CurrentDays, tt.expectedCurrent)
			}
			if stats.LongestDays != tt.expectedLongest {
				t.Errorf("LongestDays = %d, want %d", stats.LongestDays, tt.expectedLongest)
			}

			if tt.expectedStart == "" {
				if stats.LongestStart != nil || stats.LongestEnd != nil {
					t.Errorf("expected nil longest start/end, got %v - %v", stats.LongestStart, stats.LongestEnd)
				}
				return
			}

			if stats.LongestStart == nil || stats.LongestEnd == nil {
				t.Fatalf("expected non-nil longest start/end")
			}

			if got := stats.LongestStart.Format("2006-01-02"); got != tt.expectedStart {
				t.Errorf("LongestStart = %s, want %s", got, tt.expectedStart)
			}
			if got := stats.LongestEnd.Format("2006-01-02"); got != tt.expectedEnd {
				t.Errorf("LongestEnd = %s, want %s", got, tt.expectedEnd)
			}
		})
	}
}

func TestNormalizeBudgetTierDistribution(t *testing.T) {
	s := &WatchedService{log: slog.New(slog.DiscardHandler)}

	input := []models.BudgetTierCount{
		{Tier: models.BudgetTierMid, Count: 4},
		{Tier: models.BudgetTier("unexpected"), Count: 2},
		{Tier: models.BudgetTierIndie, Count: 1},
	}

	expected := []models.BudgetTierCount{
		{Tier: models.BudgetTierIndie, Count: 1},
		{Tier: models.BudgetTierMid, Count: 4},
		{Tier: models.BudgetTierBlockbuster, Count: 0},
		{Tier: models.BudgetTierUnknown, Count: 2},
	}

	result := s.normalizeBudgetTierDistribution(input)
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("normalizeBudgetTierDistribution() = %v, want %v", result, expected)
	}
}

func TestSortMoviesByROIDesc(t *testing.T) {
	s := &WatchedService{log: slog.New(slog.DiscardHandler)}

	input := []models.MovieFinancial{
		{Title: "A", Budget: 100, Revenue: 400, ROI: 3.0},
		{Title: "B", Budget: 100, Revenue: 500, ROI: 3.0},
		{Title: "C", Budget: 100, Revenue: 300, ROI: 2.0},
		{Title: "D", Budget: 0, Revenue: 999, ROI: 10.0},
	}

	result := s.sortMoviesByROIDesc(input)

	if len(result) != 3 {
		t.Fatalf("expected 3 movies after filtering, got %d", len(result))
	}

	if result[0].Title != "B" || result[1].Title != "A" || result[2].Title != "C" {
		t.Errorf("unexpected ROI sort order: %v", []string{result[0].Title, result[1].Title, result[2].Title})
	}
}

func TestSortMoviesByBudgetDesc(t *testing.T) {
	s := &WatchedService{log: slog.New(slog.DiscardHandler)}

	input := []models.MovieFinancial{
		{Title: "A", Budget: 100, Revenue: 500},
		{Title: "B", Budget: 300, Revenue: 100},
		{Title: "C", Budget: 300, Revenue: 200},
		{Title: "D", Budget: 0, Revenue: 999},
	}

	result := s.sortMoviesByBudgetDesc(input)

	if len(result) != 3 {
		t.Fatalf("expected 3 movies after filtering, got %d", len(result))
	}

	if result[0].Title != "C" || result[1].Title != "B" || result[2].Title != "A" {
		t.Errorf("unexpected budget sort order: %v", []string{result[0].Title, result[1].Title, result[2].Title})
	}
}
