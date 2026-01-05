package services

import (
	"log/slog"
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
