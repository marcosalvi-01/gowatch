package services

import (
	"gowatch/db"
)

// CreditsService handles cast and crew operations
type CreditsService struct {
	db   db.DB
	tmdb *TMDBService
}

func NewCreditsService(db db.DB, tmdbService *TMDBService) *CreditsService {
	return &CreditsService{
		db:   db,
		tmdb: tmdbService,
	}
}
