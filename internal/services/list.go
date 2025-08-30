package services

import (
	"context"
	"fmt"
	"gowatch/db"
	"gowatch/internal/models"
	"gowatch/logging"
	"log/slog"
	"time"
)

// ListService handles user's custom movie lists
type ListService struct {
	db   db.DB
	tmdb *MovieService
	log  *slog.Logger
}

func NewListService(db db.DB, tmdb *MovieService) *ListService {
	log := logging.Get("list service")
	log.Debug("creating new ListService instance")
	return &ListService{
		db:   db,
		tmdb: tmdb,
		log:  log,
	}
}

func (s *ListService) GetAllLists(ctx context.Context) ([]models.ListEntry, error) {
	s.log.Debug("retrieving all lists")

	results, err := s.db.GetAllLists(ctx)
	if err != nil {
		s.log.Error("failed to fetch lists from database", "error", err)
		return nil, fmt.Errorf("failed to get all lists: %w", err)
	}

	s.log.Debug("fetched lists from database", "count", len(results))

	lists := make([]models.ListEntry, len(results))
	for i, result := range results {
		lists[i] = models.ListEntry{
			ID:   result.ID,
			Name: result.Name,
		}
	}

	s.log.Info("successfully retrieved all lists", "count", len(lists))
	return lists, nil
}

func (s *ListService) CreateList(ctx context.Context, name, description string) error {
	s.log.Debug("creating new list", "name", name, "descriptionLength", len(description))

	err := s.db.InsertList(ctx, db.InsertList{
		Name:        name,
		Description: &description,
	})
	if err != nil {
		s.log.Error("failed to create list", "name", name, "error", err)
		return fmt.Errorf("failed to create list: %w", err)
	}

	s.log.Info("successfully created new list", "name", name)
	return nil
}

func (s *ListService) AddMovieToList(ctx context.Context, listID int64, movieID int64, note *string) error {
	err := s.db.AddMovieToList(ctx, db.InsertMovieList{
		MovieID:   movieID,
		ListID:    listID,
		DateAdded: time.Now(),
		Position:  nil,
		Note:      note,
	})
	if err != nil {
		return fmt.Errorf("failed to add movie '%d' to list '%d': %w", movieID, listID, err)
	}

	return nil
}
