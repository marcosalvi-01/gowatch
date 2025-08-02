package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"gowatch/db"
	"gowatch/internal/models"
	"sort"
	"time"
)

// WatchedService handles user's watched movie tracking
type WatchedService struct {
	db             db.DB
	movieService   *MovieService
	creditsService *CreditsService
}

func NewWatchedService(db db.DB, movieService *MovieService, creditService *CreditsService) *WatchedService {
	return &WatchedService{
		db:             db,
		movieService:   movieService,
		creditsService: creditService,
	}
}

func (s *WatchedService) AddWatched(ctx context.Context, watchedEntry models.Watched) error {
	log.Debug("adding watched entry", "movie_id", watchedEntry.MovieID)

	log.Debug("service", "service", s)
	log.Debug("db", "db", s.db)
	_, err := s.db.GetMovieByID(ctx, watchedEntry.MovieID)
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			log.Error("database error checking movie", "movie_id", watchedEntry.MovieID, "error", err)
			return fmt.Errorf("failed to check if movie exists: %w", err)
		}

		log.Debug("movie not found, fetching from TMDB", "movie_id", watchedEntry.MovieID)

		newMovie, err := s.movieService.tmdb.GetMovieDetails(watchedEntry.MovieID)
		if err != nil {
			return fmt.Errorf("failed to get movie details from TMDB: %w", err)
		}

		err = s.movieService.CreateMovie(ctx, newMovie)
		if err != nil {
			return fmt.Errorf("failed to add new movie to db: %w", err)
		}
	}

	err = s.db.InsertWatched(ctx, watchedEntry)
	if err != nil {
		log.Error("failed to insert watched entry", "movie_id", watchedEntry.MovieID, "error", err)
		return fmt.Errorf("failed to record watched entry: %w", err)
	}

	log.Info("watched entry added", "movie_id", watchedEntry.MovieID)
	return nil
}

func (s *WatchedService) GetAllWatchedMovies(ctx context.Context) ([]models.WatchedMovie, error) {
	movies, err := s.db.GetWatchedJoinMovie(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get watched join movies: %w", err)
	}

	// reverse the original slice to keep an order of first = last watched movie added
	for i, j := 0, len(movies)-1; i < j; i, j = i+1, j-1 {
		movies[i], movies[j] = movies[j], movies[i]
	}

	// sort keeping the original order for items that are equal
	sort.SliceStable(movies, func(i, j int) bool {
		return movies[i].WatchedDate.After(movies[j].WatchedDate)
	})

	return movies, nil
}

func (s *WatchedService) GetWatchedDayMovies(ctx context.Context) ([]models.WatchedDay, error) {
	movies, err := s.db.GetWatchedJoinMovie(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch watched join movie: %w", err)
	}

	sort.Slice(movies, func(i, j int) bool {
		return movies[i].WatchedDate.After(movies[j].WatchedDate)
	})

	var out []models.WatchedDay
	for _, m := range movies {
		d := m.WatchedDate.Truncate(24 * time.Hour)
		if len(out) == 0 || !d.Equal(out[len(out)-1].Date) {
			out = append(out, models.WatchedDay{Date: d})
		}
		out[len(out)-1].Movies = append(out[len(out)-1].Movies, m.Movie)
	}

	return out, nil
}

func (s *WatchedService) ImportWatched(ctx context.Context, movies models.WatchedMoviesLog) error {
	totalMovies := 0
	for _, importMovie := range movies {
		totalMovies += len(importMovie.Movies)
	}

	log.Info("starting movie import", "total_movies", totalMovies)

	for _, importMovie := range movies {
		for _, movieRef := range importMovie.Movies {
			err := s.AddWatched(ctx, models.Watched{
				MovieID:     int64(movieRef.MovieID),
				WatchedDate: importMovie.Date,
			})
			if err != nil {
				log.Warn("movie import failed", "movie_id", movieRef.MovieID, "error", err)
				return fmt.Errorf("failed to import movie: %w", err)
			}
		}
	}

	log.Info("movie import completed", "total_movies", totalMovies)
	return nil
}

func (s *WatchedService) ExportWatched(ctx context.Context) (models.WatchedMoviesLog, error) {
	watched, err := s.db.GetAllWatched(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get all watched movies for export: %w", err)
	}

	// Group movies by watched date
	movieMap := make(map[time.Time][]models.WatchedMovieRef)
	for _, w := range watched {
		movieMap[w.WatchedDate] = append(movieMap[w.WatchedDate], models.WatchedMovieRef{
			MovieID: int(w.MovieID),
		})
	}

	export := make(models.WatchedMoviesLog, 0, len(watched))
	for k, v := range movieMap {
		export = append(export, models.WatchedMoviesEntry{
			Date:   k,
			Movies: v,
		})
	}

	return export, nil
}
