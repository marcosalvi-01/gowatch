package models

import tmdb "github.com/cyruzin/golang-tmdb"

type Genre struct {
	ID   int64
	Name string
}

func GenreFromTMDBGenre(TMDBGenre tmdb.Genre) Genre {
	return Genre{
		ID:   TMDBGenre.ID,
		Name: TMDBGenre.Name,
	}
}

func GenreFromTMDBGenres(TMDBGenre []tmdb.Genre) []Genre {
	genres := make([]Genre, len(TMDBGenre))
	for i, genre := range TMDBGenre {
		genres[i] = GenreFromTMDBGenre(genre)
	}
	return genres
}
