package services

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/marcosalvi-01/gowatch/db"
	"github.com/marcosalvi-01/gowatch/internal/common"
	"github.com/marcosalvi-01/gowatch/internal/models"
	"github.com/marcosalvi-01/gowatch/logging"
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

// GetAllLists retrieves all user lists EXCEPT the watchlist
// The watchlist is managed separately and not included in normal list operations
func (s *ListService) GetAllLists(ctx context.Context) ([]models.ListEntry, error) {
	s.log.Debug("retrieving all lists")

	user, err := common.GetUser(ctx)
	if err != nil {
		s.log.Error("failed to get userID", "error", err)
		return nil, err
	}

	results, err := s.db.GetAllLists(ctx, user.ID)
	if err != nil {
		s.log.Error("failed to fetch lists from database", "error", err)
		return nil, fmt.Errorf("failed to get all lists: %w", err)
	}

	s.log.Debug("fetched lists from database", "totalCount", len(results))

	// Filter out the watchlist from the results
	var filteredResults []db.InsertList
	for _, result := range results {
		if !result.IsWatchlist {
			filteredResults = append(filteredResults, result)
		}
	}

	lists := make([]models.ListEntry, len(filteredResults))
	for i, result := range filteredResults {
		lists[i] = models.ListEntry{
			ID:   result.ID,
			Name: result.Name,
		}
	}

	s.log.Info("successfully retrieved all lists", "count", len(lists))
	return lists, nil
}

func (s *ListService) CreateList(ctx context.Context, name string, description *string, isWatchlist bool) (*models.List, error) {
	if name == "" {
		return nil, fmt.Errorf("list name cannot be empty")
	}
	s.log.Debug("creating new list", "name", name)

	user, err := common.GetUser(ctx)
	if err != nil {
		s.log.Error("failed to get userID", "error", err)
		return nil, err
	}

	id, err := s.db.InsertList(ctx, db.InsertList{
		UserID:      user.ID,
		Name:        name,
		Description: description,
		IsWatchlist: isWatchlist,
	})
	if err != nil {
		s.log.Error("failed to create list", "name", name, "error", err)
		return nil, fmt.Errorf("failed to create list: %w", err)
	}

	list, err := s.db.GetList(ctx, user.ID, id)
	if err != nil {
		s.log.Error("failed to retrieve created list", "id", id, "error", err)
		return nil, fmt.Errorf("failed to retrieve created list: %w", err)
	}

	s.log.Info("successfully created new list", "name", name, "id", id)
	return list, nil
}

func (s *ListService) AddMovieToList(ctx context.Context, listID, movieID int64, note *string) error {
	if listID <= 0 {
		return fmt.Errorf("invalid list ID")
	}
	if movieID <= 0 {
		return fmt.Errorf("invalid movie ID")
	}
	s.log.Debug("adding movie to list", "listID", listID, "movieID", movieID)

	user, err := common.GetUser(ctx)
	if err != nil {
		s.log.Error("failed to get userID", "error", err)
		return err
	}

	err = s.db.AddMovieToList(ctx, user.ID, db.InsertMovieList{
		MovieID:   movieID,
		ListID:    listID,
		DateAdded: time.Now(),
		Position:  nil,
		Note:      note,
	})
	if err != nil {
		s.log.Error("failed to add movie to list", "listID", listID, "movieID", movieID, "error", err)
		return fmt.Errorf("failed to add movie '%d' to list '%d': %w", movieID, listID, err)
	}

	s.log.Info("successfully added movie to list", "listID", listID, "movieID", movieID)
	return nil
}

func (s *ListService) GetListDetails(ctx context.Context, listID int64) (*models.List, error) {
	s.log.Debug("getting list details", "listID", listID)

	user, err := common.GetUser(ctx)
	if err != nil {
		s.log.Error("failed to get userID", "error", err)
		return nil, err
	}

	list, err := s.db.GetList(ctx, user.ID, listID)
	if err != nil {
		s.log.Error("failed to get list details", "listID", listID, "error", err)
		return nil, fmt.Errorf("failed to get list with id '%d' from db: %w", listID, err)
	}
	s.log.Debug("fetched list details", "listID", listID, "movieCount", len(list.Movies))

	return list, nil
}

func (s *ListService) DeleteList(ctx context.Context, id int64) error {
	s.log.Debug("deleting list", "listID", id)

	// Check if this list can be deleted
	if s.IsWatchlist(ctx, id) {
		s.log.Warn("attempted to delete watchlist", "listID", id)
		return fmt.Errorf("cannot delete watchlist")
	}

	user, err := common.GetUser(ctx)
	if err != nil {
		s.log.Error("failed to get userID", "error", err)
		return err
	}

	err = s.db.DeleteListByID(ctx, user.ID, id)
	if err != nil {
		s.log.Error("failed to delete list", "listID", id, "error", err)
		return fmt.Errorf("failed to delete list from db: %w", err)
	}
	s.log.Info("successfully deleted list", "listID", id)

	return nil
}

func (s *ListService) DeleteMovieFromList(ctx context.Context, listID, movieID int64) error {
	s.log.Debug("removing movie from list", "listID", listID, "movieID", movieID)

	user, err := common.GetUser(ctx)
	if err != nil {
		s.log.Error("failed to get userID", "error", err)
		return err
	}

	err = s.db.DeleteMovieFromList(ctx, user.ID, listID, movieID)
	if err != nil {
		s.log.Error("failed to remove movie from list", "listID", listID, "movieID", movieID, "error", err)
		return fmt.Errorf("failed to delete movie for list from db: %w", err)
	}
	s.log.Info("successfully removed movie from list", "listID", listID, "movieID", movieID)

	return nil
}

// GetWatchlist retrieves the user's watchlist
func (s *ListService) GetWatchlist(ctx context.Context) (*models.List, error) {
	s.log.Debug("getting watchlist")

	user, err := common.GetUser(ctx)
	if err != nil {
		s.log.Error("failed to get userID", "error", err)
		return nil, err
	}

	watchlistID, err := s.db.GetWatchlistID(ctx, user.ID)
	if err != nil {
		s.log.Error("failed to get watchlist ID", "error", err)
		return nil, fmt.Errorf("failed to get watchlist ID: %w", err)
	}

	watchlist, err := s.db.GetList(ctx, user.ID, watchlistID)
	if err != nil {
		s.log.Error("failed to get watchlist details", "watchlistID", watchlistID, "error", err)
		return nil, fmt.Errorf("failed to get watchlist: %w", err)
	}

	s.log.Debug("successfully retrieved watchlist", "watchlistID", watchlistID, "movieCount", len(watchlist.Movies))
	return watchlist, nil
}

// EnsureWatchlistExists creates a watchlist if one doesn't exist for the user
func (s *ListService) EnsureWatchlistExists(ctx context.Context) error {
	s.log.Debug("ensuring watchlist exists")

	user, err := common.GetUser(ctx)
	if err != nil {
		s.log.Error("failed to get userID", "error", err)
		return err
	}

	s.log.Debug("checking if watchlist exists", "userID", user.ID)

	_, err = s.db.GetWatchlistID(ctx, user.ID)
	if err == nil {
		s.log.Debug("watchlist already exists", "userID", user.ID)
		return nil
	}

	s.log.Info("watchlist doesn't exist, creating it", "userID", user.ID)

	description := "Your personal watchlist"
	listID, err := s.CreateList(ctx, "Watchlist", &description, true)
	if err != nil {
		s.log.Error("failed to create watchlist", "userID", user.ID, "error", err)
		return fmt.Errorf("failed to create watchlist: %w", err)
	}

	s.log.Info("successfully created watchlist", "userID", user.ID, "listID", listID)
	return nil
}

// RemoveMovieFromWatchlist removes a movie from the user's watchlist
func (s *ListService) RemoveMovieFromWatchlist(ctx context.Context, movieID int64) error {
	s.log.Debug("removing movie from watchlist", "movieID", movieID)

	watchlist, err := s.GetWatchlist(ctx)
	if err != nil {
		s.log.Error("failed to get watchlist", "error", err)
		return fmt.Errorf("failed to get watchlist: %w", err)
	}

	return s.DeleteMovieFromList(ctx, watchlist.ID, movieID)
}

// IsWatchlist returns true if the list is a watchlist (protected)
func (s *ListService) IsWatchlist(ctx context.Context, listID int64) bool {
	user, err := common.GetUser(ctx)
	if err != nil {
		s.log.Error("failed to get userID", "error", err)
		return false
	}

	watchlistID, err := s.db.GetWatchlistID(ctx, user.ID)
	if err != nil {
		s.log.Error("failed to get watchlist", "userID", user.ID, "error", err)
		return false
	}

	return listID == watchlistID
}

// IsMovieInWatchlist checks if the given movie is in the user's watchlist
func (s *ListService) IsMovieInWatchlist(ctx context.Context, movieID int64) bool {
	watchlist, err := s.GetWatchlist(ctx)
	if err != nil {
		s.log.Error("failed to get watchlist for movie check", "movieID", movieID, "error", err)
		return false
	}

	for _, movie := range watchlist.Movies {
		if movie.MovieDetails.Movie.ID == movieID {
			return true
		}
	}
	return false
}

// ExportLists exports all user lists with their movies in import format
func (s *ListService) ExportLists(ctx context.Context) (models.ImportListsLog, error) {
	s.log.Debug("exporting all lists")

	user, err := common.GetUser(ctx)
	if err != nil {
		s.log.Error("failed to get userID", "error", err)
		return nil, err
	}

	lists, err := s.db.ExportLists(ctx, user.ID)
	if err != nil {
		s.log.Error("failed to export lists from database", "error", err)
		return nil, fmt.Errorf("failed to export lists: %w", err)
	}

	exportLists := make(models.ImportListsLog, len(lists))
	for i, list := range lists {
		exportMovies := make([]models.ImportListMovieRef, len(list.Movies))
		for j, movie := range list.Movies {
			exportMovies[j] = models.ImportListMovieRef{
				MovieID:   movie.MovieDetails.Movie.ID,
				DateAdded: movie.DateAdded,
				Position:  movie.Position,
				Note:      movie.Note,
			}
		}

		exportLists[i] = models.ImportListEntry{
			Name:        list.Name,
			Description: list.Description,
			IsWatchlist: list.IsWatchlist,
			Movies:      exportMovies,
		}
	}

	s.log.Info("successfully exported lists", "listCount", len(exportLists))
	return exportLists, nil
}

// ImportLists imports lists from import format
func (s *ListService) ImportLists(ctx context.Context, lists models.ImportListsLog) error {
	s.log.Info("ImportLists: starting lists import", "totalLists", len(lists))

	totalMovies := 0
	for _, list := range lists {
		totalMovies += len(list.Movies)
	}
	s.log.Info("ImportLists: import details", "totalLists", len(lists), "totalMovies", totalMovies)

	for _, importList := range lists {
		var targetList *models.List
		var err error

		if importList.IsWatchlist {
			// For watchlist, use existing or create new
			targetList, err = s.GetWatchlist(ctx)
			if err != nil {
				// Create new watchlist
				targetList, err = s.CreateList(ctx, importList.Name, importList.Description, true)
				if err != nil {
					s.log.Error("ImportLists: failed to create watchlist", "error", err)
					return fmt.Errorf("ImportLists: failed to create watchlist: %w", err)
				}
			}
		} else {
			// For custom lists, create new
			targetList, err = s.CreateList(ctx, importList.Name, importList.Description, false)
			if err != nil {
				s.log.Error("ImportLists: failed to create list", "listName", importList.Name, "error", err)
				return fmt.Errorf("ImportLists: failed to create list %s: %w", importList.Name, err)
			}
		}

		// Add movies to the target list
		for _, movieRef := range importList.Movies {
			// Ensure movie exists in DB
			_, err := s.tmdb.GetMovieDetails(ctx, movieRef.MovieID)
			if err != nil {
				s.log.Error("ImportLists: failed to fetch movie details", "movieID", movieRef.MovieID, "listName", importList.Name, "error", err)
				return fmt.Errorf("ImportLists: failed to fetch movie details for ID %d: %w", movieRef.MovieID, err)
			}

			// For import, we ignore position and use current time for date_added
			err = s.AddMovieToList(ctx, targetList.ID, movieRef.MovieID, movieRef.Note)
			if err != nil {
				s.log.Error("ImportLists: failed to add movie to list", "movieID", movieRef.MovieID, "listID", targetList.ID, "error", err)
				return fmt.Errorf("ImportLists: failed to add movie %d to list %d: %w", movieRef.MovieID, targetList.ID, err)
			}
		}
	}

	s.log.Info("ImportLists: successfully imported lists", "totalLists", len(lists))
	return nil
}
