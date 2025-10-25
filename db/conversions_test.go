package db

import (
	"reflect"
	"testing"
	"time"

	"gowatch/db/sqlc"
	"gowatch/internal/models"
)

func TestToModelsMovieDetails(t *testing.T) {
	releaseDate := time.Date(2023, 5, 15, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name     string
		input    sqlc.Movie
		expected models.MovieDetails
	}{
		{
			name: "full movie details",
			input: sqlc.Movie{
				ID:               123,
				Title:            "Test Movie",
				OriginalTitle:    "Original Test",
				OriginalLanguage: "en",
				Overview:         "A test movie",
				ReleaseDate:      &releaseDate,
				PosterPath:       "/poster.jpg",
				BackdropPath:     "/backdrop.jpg",
				Popularity:       7.5,
				VoteCount:        100,
				VoteAverage:      8.0,
				Budget:           1000000,
				Homepage:         "http://example.com",
				ImdbID:           "tt1234567",
				Revenue:          5000000,
				Runtime:          120,
				Status:           "Released",
				Tagline:          "A tagline",
				UpdatedAt:        &releaseDate,
			},
			expected: models.MovieDetails{
				Movie: models.Movie{
					ID:               123,
					Title:            "Test Movie",
					OriginalTitle:    "Original Test",
					OriginalLanguage: "en",
					Overview:         "A test movie",
					ReleaseDate:      &releaseDate,
					PosterPath:       "/poster.jpg",
					BackdropPath:     "/backdrop.jpg",
					Popularity:       7.5,
					VoteCount:        100,
					VoteAverage:      8.0,
					UpdatedAt:        releaseDate,
				},
				Budget:   1000000,
				Homepage: "http://example.com",
				IMDbID:   "tt1234567",
				Revenue:  5000000,
				Runtime:  120,
				Status:   "Released",
				Tagline:  "A tagline",
			},
		},
		{
			name: "nil release date",
			input: sqlc.Movie{
				ID:          456,
				Title:       "No Release",
				ReleaseDate: nil,
				UpdatedAt:   &releaseDate,
			},
			expected: models.MovieDetails{
				Movie: models.Movie{
					ID:        456,
					Title:     "No Release",
					UpdatedAt: releaseDate,
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
