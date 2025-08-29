package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"gowatch/db"
	"gowatch/internal/models"
	"gowatch/logging"
	"log/slog"
	"sort"
	"time"
)

// WatchedService handles user's watched movie tracking
type WatchedService struct {
	db   db.DB
	tmdb *MovieService
	log  *slog.Logger
}

func NewWatchedService(db db.DB, tmdb *MovieService) *WatchedService {
	log := logging.Get("watched service")
	log.Debug("creating new WatchedService instance")
	return &WatchedService{
		db:   db,
		tmdb: tmdb,
		log:  log,
	}
}

func (s *WatchedService) AddWatched(ctx context.Context, movieID int64, date time.Time, inTheaters bool) error {
	s.log.Debug("adding watched movie", "movieID", movieID, "date", date, "inTheaters", inTheaters)

	err := s.db.InsertWatched(ctx, db.InsertWatched{
		MovieID:    movieID,
		Date:       date,
		InTheaters: inTheaters,
	})
	if err != nil {
		s.log.Error("failed to insert watched entry", "movieID", movieID, "error", err)
		return fmt.Errorf("failed to record watched entry: %w", err)
	}

	s.log.Info("successfully added watched movie", "movieID", movieID)
	return nil
}

func (s *WatchedService) GetAllWatchedMoviesInDay(ctx context.Context) ([]models.WatchedMoviesInDay, error) {
	s.log.Debug("retrieving all watched movies grouped by day")

	movies, err := s.db.GetWatchedJoinMovie(ctx)
	if err != nil {
		s.log.Error("failed to fetch watched movies from database", "error", err)
		return nil, fmt.Errorf("failed to fetch watched join movie: %w", err)
	}

	s.log.Debug("fetched watched movies from database", "count", len(movies))

	sort.Slice(movies, func(i, j int) bool {
		return movies[i].Date.After(movies[j].Date)
	})

	var out []models.WatchedMoviesInDay
	for _, m := range movies {
		d := m.Date.Truncate(24 * time.Hour)
		if len(out) == 0 || !d.Equal(out[len(out)-1].Date) {
			out = append(out, models.WatchedMoviesInDay{Date: d})
		}
		out[len(out)-1].Movies = append(out[len(out)-1].Movies, models.WatchedMovieInDay{
			MovieDetails: m.MovieDetails,
			InTheaters:   m.InTheaters,
		})
	}

	s.log.Debug("grouped movies by day", "dayCount", len(out))
	return out, nil
}

func (s *WatchedService) ImportWatched(ctx context.Context, movies models.ImportWatchedMoviesLog) error {
	totalMovies := 0
	for _, importMovie := range movies {
		totalMovies += len(importMovie.Movies)
	}

	s.log.Info("starting watched movies import", "totalDays", len(movies), "totalMovies", totalMovies)

	for _, importMovie := range movies {
		for _, movieRef := range importMovie.Movies {
			err := s.AddWatched(ctx, int64(movieRef.MovieID), importMovie.Date, movieRef.InTheaters)
			if err != nil {
				s.log.Error("failed to import movie", "movieID", movieRef.MovieID, "date", importMovie.Date, "error", err)
				return fmt.Errorf("failed to import movie: %w", err)
			}

			_, err = s.tmdb.GetMovieDetails(ctx, int64(movieRef.MovieID))
			if err != nil {
				s.log.Error("failed to cache movie in db", "movieID", movieRef.MovieID, "date", importMovie.Date, "error", err)
				return fmt.Errorf("failed to cache movie in db: %w", err)
			}
		}
	}

	s.log.Info("successfully imported watched movies", "totalMovies", totalMovies)
	return nil
}

func (s *WatchedService) ExportWatched(ctx context.Context) (models.ImportWatchedMoviesLog, error) {
	s.log.Debug("starting watched movies export")

	watched, err := s.GetAllWatchedMoviesInDay(ctx)
	if err != nil {
		s.log.Error("failed to get watched movies for export", "error", err)
		return nil, fmt.Errorf("failed to get all watched movies for export: %w", err)
	}

	s.log.Debug("retrieved watched movies for export", "dayCount", len(watched))

	export := make(models.ImportWatchedMoviesLog, len(watched))
	totalMovies := 0

	for i, w := range watched {
		ids := make([]models.ImportWatchedMovieRef, len(w.Movies))
		for j, movieDetails := range w.Movies {
			ids[j] = models.ImportWatchedMovieRef{
				MovieID: int(movieDetails.MovieDetails.Movie.ID),
				// TODO: find a way to include the inTheaters in the export, we would need to modify the GetAllWatchedMoviesInDay model to include it
				InTheaters: movieDetails.InTheaters,
			}
		}
		totalMovies += len(w.Movies)
		export[i] = models.ImportWatchedMoviesEntry{
			Date:   w.Date,
			Movies: ids,
		}
	}

	s.log.Info("successfully exported watched movies", "dayCount", len(export), "totalMovies", totalMovies)
	return export, nil
}

func (s *WatchedService) GetWatchedMovieRecordsByID(ctx context.Context, movieID int64) (models.WatchedMovieRecords, error) {
	s.log.Debug("get watch records", "movieID", movieID)

	rows, err := s.db.GetWatchedJoinMovieByID(ctx, movieID)
	if errors.Is(err, sql.ErrNoRows) || len(rows) == 0 {
		return models.WatchedMovieRecords{}, nil
	}
	if err != nil {
		s.log.Error("db query failed", "movieID", movieID, "error", err)
		return models.WatchedMovieRecords{}, fmt.Errorf("get watched records: %w", err)
	}

	rec := models.WatchedMovieRecords{
		MovieDetails: rows[0].MovieDetails, // same in every row
		Records:      make([]models.WatchedMovieRecord, 0, len(rows)),
	}
	for _, r := range rows {
		rec.Records = append(rec.Records, models.WatchedMovieRecord{
			Date:       r.Date,
			InTheaters: r.InTheaters,
		})
	}

	sort.Slice(rec.Records, func(i, j int) bool {
		return rec.Records[i].Date.After(rec.Records[j].Date)
	})

	return rec, nil
}

func (s *WatchedService) GetWatchedCount(ctx context.Context) (int, error) {
	count, err := s.db.GetWatchedCount(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to get watched count from db: %w", err)
	}

	return count, nil
}
