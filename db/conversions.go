package db

import (
	"gowatch/db/sqlc"
	"gowatch/internal/models"
)

// toModelsMovie converts sqlc.Movie to models.Movie
func toModelsMovie(movie sqlc.Movie) models.Movie {
	return models.Movie{
		ID:               movie.ID,
		IMDbID:           movie.ImdbID,
		Title:            movie.Title,
		OriginalTitle:    movie.OriginalTitle,
		ReleaseDate:      movie.ReleaseDate,
		OriginalLanguage: movie.OriginalLanguage,
		Overview:         movie.Overview,
		PosterPath:       movie.PosterPath,
		BackdropPath:     movie.BackdropPath,
		Budget:           movie.Budget,
		Revenue:          movie.Revenue,
		Runtime:          movie.Runtime,
		VoteAverage:      movie.VoteAverage,
		VoteCount:        movie.VoteCount,
		Popularity:       movie.Popularity,
		Homepage:         movie.Homepage,
		Status:           movie.Status,
		Tagline:          movie.Tagline,
	}
}

// toSqlcInsertMovieParams converts models.Movie to sqlc.InsertMovieParams
func toSqlcInsertMovieParams(movie models.Movie) sqlc.InsertMovieParams {
	return sqlc.InsertMovieParams{
		ID:               movie.ID,
		ImdbID:           movie.IMDbID,
		Title:            movie.Title,
		OriginalLanguage: movie.OriginalLanguage,
		Overview:         movie.Overview,
		PosterPath:       movie.PosterPath,
		ReleaseDate:      movie.ReleaseDate,
		OriginalTitle:    movie.OriginalTitle,
		BackdropPath:     movie.BackdropPath,
		Budget:           movie.Budget,
		Revenue:          movie.Revenue,
		Runtime:          movie.Runtime,
		VoteAverage:      movie.VoteAverage,
		VoteCount:        movie.VoteCount,
		Popularity:       movie.Popularity,
		Homepage:         movie.Homepage,
		Status:           movie.Status,
		Tagline:          movie.Tagline,
	}
}

// toSqlcInsertWatchedParams converts models.Watched to sqlc.InsertWatchedParams
func toSqlcInsertWatchedParams(watched models.Watched) sqlc.InsertWatchedParams {
	return sqlc.InsertWatchedParams{
		MovieID:     watched.MovieID,
		WatchedDate: watched.WatchedDate,
	}
}

func toSqlcInsertPersonParams(person models.Person) sqlc.InsertPersonParams {
	return sqlc.InsertPersonParams{
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
