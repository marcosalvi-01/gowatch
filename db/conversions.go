// Package db provides database connectivity, query operations, and data conversions between SQL and internal models.
package db

import (
	"time"

	"github.com/marcosalvi-01/gowatch/db/sqlc"
	"github.com/marcosalvi-01/gowatch/internal/models"
)

// toModelsMovie converts sqlc.Movie to models.Movie
func toModelsMovieDetails(movie sqlc.Movie) models.MovieDetails {
	var updatedAt time.Time
	if movie.UpdatedAt != nil {
		updatedAt = *movie.UpdatedAt
	}
	return models.MovieDetails{
		Movie: models.Movie{
			ID:               movie.ID,
			Title:            movie.Title,
			OriginalTitle:    movie.OriginalTitle,
			OriginalLanguage: movie.OriginalLanguage,
			Overview:         movie.Overview,
			ReleaseDate:      movie.ReleaseDate.ToTimePtr(),
			PosterPath:       movie.PosterPath,
			BackdropPath:     movie.BackdropPath,
			Popularity:       float32(movie.Popularity),
			VoteCount:        movie.VoteCount,
			VoteAverage:      float32(movie.VoteAverage),
			UpdatedAt:        updatedAt,
		},
		Budget:   movie.Budget,
		Homepage: movie.Homepage,
		IMDbID:   movie.ImdbID,
		Revenue:  movie.Revenue,
		Runtime:  int(movie.Runtime),
		Status:   movie.Status,
		Tagline:  movie.Tagline,
	}
}

func toModelsPerson(person sqlc.Person) models.Person {
	return models.Person{
		ID:          person.ID,
		Name:        person.Name,
		ProfilePath: person.ProfilePath,
		Gender:      person.Gender,
	}
}
