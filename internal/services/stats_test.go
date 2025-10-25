package services

import (
	"gowatch/internal/models"
	"log/slog"
	"reflect"
	"testing"
	"time"
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

	tests := []struct {
		name      string
		total     int64
		dateRange *models.DateRange
		expected  [3]float64 // [avgPerDay, avgPerWeek, avgPerMonth]
	}{
		{
			name:      "valid date range",
			total:     6,
			dateRange: &models.DateRange{MinDate: &minDate, MaxDate: &maxDate},
			expected:  [3]float64{2, 14, 60}, // 2 per day, 6/(3/7)=14 per week, 6/(3/30)=60 per month
		},
		{
			name:      "nil date range",
			total:     10,
			dateRange: nil,
			expected:  [3]float64{0, 0, 0},
		},
		{
			name:      "nil min date",
			total:     10,
			dateRange: &models.DateRange{MinDate: nil, MaxDate: &maxDate},
			expected:  [3]float64{0, 0, 0},
		},
		{
			name:      "nil max date",
			total:     10,
			dateRange: &models.DateRange{MinDate: &minDate, MaxDate: nil},
			expected:  [3]float64{0, 0, 0},
		},
		{
			name:      "same day",
			total:     5,
			dateRange: &models.DateRange{MinDate: &minDate, MaxDate: &minDate},
			expected:  [3]float64{5, 35, 150}, // 5 per day, 5/(1/7)=35 per week, 5/(1/30)=150 per month
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			avgPerDay, avgPerWeek, avgPerMonth := s.calculateAverages(tt.total, tt.dateRange)
			result := [3]float64{avgPerDay, avgPerWeek, avgPerMonth}
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("calculateAverages() = %v, want %v", result, tt.expected)
			}
		})
	}
}
