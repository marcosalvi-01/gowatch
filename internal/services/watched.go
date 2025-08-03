package services

import (
	"context"
	"fmt"
	"gowatch/db"
	"gowatch/internal/models"
	"sort"
	"time"
)

// WatchedService handles user's watched movie tracking
type WatchedService struct {
	db   db.DB
	tmdb *TMDBService
}

func NewWatchedService(db db.DB, tmdb *TMDBService) *WatchedService {
	log.Debug("creating new WatchedService instance")
	return &WatchedService{
		db:   db,
		tmdb: tmdb,
	}
}

func (s *WatchedService) AddWatched(ctx context.Context, movieID int64, date time.Time, inTheaters bool) error {
	log.Debug("adding watched movie", "movieID", movieID, "date", date, "inTheaters", inTheaters)

	err := s.db.InsertWatched(ctx, movieID, date, inTheaters)
	if err != nil {
		log.Error("failed to insert watched entry", "movieID", movieID, "error", err)
		return fmt.Errorf("failed to record watched entry: %w", err)
	}

	log.Info("successfully added watched movie", "movieID", movieID)
	return nil
}

func (s *WatchedService) GetAllWatchedMoviesInDay(ctx context.Context) ([]models.WatchedMoviesInDay, error) {
	log.Debug("retrieving all watched movies grouped by day")

	movies, err := s.db.GetWatchedJoinMovie(ctx)
	if err != nil {
		log.Error("failed to fetch watched movies from database", "error", err)
		return nil, fmt.Errorf("failed to fetch watched join movie: %w", err)
	}

	log.Debug("fetched watched movies from database", "count", len(movies))

	sort.Slice(movies, func(i, j int) bool {
		return movies[i].Date.After(movies[j].Date)
	})

	var out []models.WatchedMoviesInDay
	for _, m := range movies {
		d := m.Date.Truncate(24 * time.Hour)
		if len(out) == 0 || !d.Equal(out[len(out)-1].Date) {
			out = append(out, models.WatchedMoviesInDay{Date: d})
		}
		out[len(out)-1].Movies = append(out[len(out)-1].Movies, m.MovieDetails)
	}

	log.Debug("grouped movies by day", "dayCount", len(out))
	return out, nil
}

func (s *WatchedService) ImportWatched(ctx context.Context, movies models.ImportWatchedMoviesLog) error {
	totalMovies := 0
	for _, importMovie := range movies {
		totalMovies += len(importMovie.Movies)
	}

	log.Info("starting watched movies import", "totalDays", len(movies), "totalMovies", totalMovies)

	for _, importMovie := range movies {
		for _, movieRef := range importMovie.Movies {
			err := s.AddWatched(ctx, int64(movieRef.MovieID), importMovie.Date, movieRef.InTheaters)
			if err != nil {
				log.Error("failed to import movie", "movieID", movieRef.MovieID, "date", importMovie.Date, "error", err)
				return fmt.Errorf("failed to import movie: %w", err)
			}

			_, err = s.tmdb.GetMovieDetails(ctx, int64(movieRef.MovieID))
			if err != nil {
				log.Error("failed to cache movie in db", "movieID", movieRef.MovieID, "date", importMovie.Date, "error", err)
				return fmt.Errorf("failed to cache movie in db: %w", err)
			}
		}
	}

	log.Info("successfully imported watched movies", "totalMovies", totalMovies)
	return nil
}

func (s *WatchedService) ExportWatched(ctx context.Context) (models.ImportWatchedMoviesLog, error) {
	log.Debug("starting watched movies export")

	watched, err := s.GetAllWatchedMoviesInDay(ctx)
	if err != nil {
		log.Error("failed to get watched movies for export", "error", err)
		return nil, fmt.Errorf("failed to get all watched movies for export: %w", err)
	}

	log.Debug("retrieved watched movies for export", "dayCount", len(watched))

	export := make(models.ImportWatchedMoviesLog, len(watched))
	totalMovies := 0

	for _, w := range watched {
		ids := make([]models.ImportWatchedMovieRef, len(w.Movies))
		for _, movieDetails := range w.Movies {
			ids = append(ids, models.ImportWatchedMovieRef{
				MovieID: int(movieDetails.Movie.ID),
				// TODO: find a way to include the inTheaters in the export, we would need to modify the GetAllWatchedMoviesInDay model to include it
				InTheaters: false,
			})
		}
		totalMovies += len(w.Movies)
		export = append(export, models.ImportWatchedMoviesEntry{
			Date:   w.Date,
			Movies: ids,
		})
	}

	log.Info("successfully exported watched movies", "dayCount", len(export), "totalMovies", totalMovies)
	return export, nil
}
