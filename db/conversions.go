package db

import (
	"gowatch/db/sqlc"
	"gowatch/internal/models"
)

// toModelsMovie converts sqlc.Movie to models.Movie
func toModelsMovieDetails(movie sqlc.Movie) models.MovieDetails {
	return models.MovieDetails{
		Movie: models.Movie{
			ID:               movie.ID,
			Title:            movie.Title,
			OriginalTitle:    movie.OriginalTitle,
			OriginalLanguage: movie.OriginalLanguage,
			Overview:         movie.Overview,
			ReleaseDate:      movie.ReleaseDate,
			PosterPath:       movie.PosterPath,
			BackdropPath:     movie.BackdropPath,
			Popularity:       float32(movie.Popularity),
			VoteCount:        movie.VoteCount,
			VoteAverage:      float32(movie.VoteAverage),
			UpdatedAt:        *movie.UpdatedAt,
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
		ID:                 person.ID,
		Name:               person.Name,
		OriginalName:       person.OriginalName,
		ProfilePath:        person.ProfilePath,
		KnownForDepartment: person.KnownForDepartment,
		Popularity:         person.Popularity,
		Gender:             person.Gender,
		Adult:              person.Adult,
	}
}
