package db

import (
	"reflect"
	"testing"
	"time"

	"github.com/marcosalvi-01/gowatch/db/sqlc"
	"github.com/marcosalvi-01/gowatch/db/types/date"
	"github.com/marcosalvi-01/gowatch/internal/models"
)

func TestToModelsMovie(t *testing.T) {
	updatedAt := time.Date(2023, 10, 27, 0, 0, 0, 0, time.UTC)
	releaseDate := time.Date(2023, 10, 26, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name     string
		input    sqlc.Movie
		expected models.MovieDetails
	}{
		{
			name: "full movie details",
			input: sqlc.Movie{
				ID:               1,
				Title:            "Test Movie",
				OriginalTitle:    "Test Movie Original",
				OriginalLanguage: "en",
				Overview:         "Test overview",
				ReleaseDate:      date.New(releaseDate),
				PosterPath:       "/poster.jpg",
				BackdropPath:     "/backdrop.jpg",
				Popularity:       10.5,
				VoteCount:        100,
				VoteAverage:      7.5,
				Budget:           1000000,
				Homepage:         "https://example.com",
				ImdbID:           "tt1234567",
				Revenue:          2000000,
				Runtime:          120,
				Status:           "Released",
				Tagline:          "Test tagline",
				UpdatedAt:        &updatedAt,
			},
			expected: models.MovieDetails{
				Movie: models.Movie{
					ID:               1,
					Title:            "Test Movie",
					OriginalTitle:    "Test Movie Original",
					OriginalLanguage: "en",
					Overview:         "Test overview",
					ReleaseDate:      &releaseDate,
					PosterPath:       "/poster.jpg",
					BackdropPath:     "/backdrop.jpg",
					Popularity:       10.5,
					VoteCount:        100,
					VoteAverage:      7.5,
					UpdatedAt:        updatedAt,
				},
				Budget:   1000000,
				Homepage: "https://example.com",
				IMDbID:   "tt1234567",
				Revenue:  2000000,
				Runtime:  120,
				Status:   "Released",
				Tagline:  "Test tagline",
			},
		},
		{
			name: "minimal movie details",
			input: sqlc.Movie{
				ID:          1,
				Title:       "Test Movie",
				ReleaseDate: date.Date{},
				UpdatedAt:   &updatedAt,
			},
			expected: models.MovieDetails{
				Movie: models.Movie{
					ID:          1,
					Title:       "Test Movie",
					ReleaseDate: nil,
					UpdatedAt:   updatedAt,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := toModelsMovieDetails(tt.input)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("toModelsMovieDetails() = %+v, want %+v", result, tt.expected)
			}
		})
	}
}

func TestToModelsPerson(t *testing.T) {
	tests := []struct {
		name     string
		input    sqlc.Person
		expected models.Person
	}{
		{
			name: "full person details",
			input: sqlc.Person{
				ID:                 789,
				Name:               "Test Actor",
				OriginalName:       "Original Name",
				ProfilePath:        "/profile.jpg",
				KnownForDepartment: "Acting",
				Popularity:         9.0,
				Gender:             1,
				Adult:              false,
			},
			expected: models.Person{
				ID:                 789,
				Name:               "Test Actor",
				OriginalName:       "Original Name",
				ProfilePath:        "/profile.jpg",
				KnownForDepartment: "Acting",
				Popularity:         9.0,
				Gender:             1,
				Adult:              false,
			},
		},
		{
			name: "minimal person",
			input: sqlc.Person{
				ID:   101,
				Name: "Minimal",
			},
			expected: models.Person{
				ID:   101,
				Name: "Minimal",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := toModelsPerson(tt.input)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("toModelsPerson() = %+v, want %+v", result, tt.expected)
			}
		})
	}
}
