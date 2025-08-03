package services

import (
	"gowatch/db"
)

type GenreService struct {
	db   db.DB
	tmdb *TMDBService
}

func NewGenreService(db db.DB, tmdbService *TMDBService) *GenreService {
	return &GenreService{
		db:   db,
		tmdb: tmdbService,
	}
}
