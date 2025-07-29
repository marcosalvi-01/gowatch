// Package services contains the business logic layer of the application.
// Services implement core business rules, data validation, and coordinate
// between handlers and the database. They provide a clean interface for
// business operations that can be used by both API and HTMX handlers.
package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"gowatch/db"
	"gowatch/internal/models"
	"gowatch/logging"
	"sort"
	"time"

	tmdb "github.com/cyruzin/golang-tmdb"
)

var log = logging.Get("MovieService")

type MovieService struct {
	db   db.DB
	tmdb *tmdb.Client
}

func NewMovieService(db db.DB, tmdb *tmdb.Client) *MovieService {
	return &MovieService{
		db:   db,
		tmdb: tmdb,
	}
}

func (s *MovieService) SearchMovie(query string) ([]models.SearchMovie, error) {
	log.Debug("searching movie", "query", query)

	search, err := s.tmdb.GetSearchMovies(query, nil)
	if err != nil {
		log.Error("TMDB search failed", "query", query, "error", err)
		return nil, fmt.Errorf("error searching TMDB for query '%s': %w", query, err)
	}

	log.Info("movie search completed", "query", query, "results", search.TotalResults)

	movies := make([]models.SearchMovie, len(search.Results))
	for i, m := range search.Results {
		movies[i] = models.SearchMovie{
			ID:               m.ID,
			Title:            m.Title,
			OriginalTitle:    m.OriginalTitle,
			OriginalLanguage: m.OriginalLanguage,
			Overview:         m.Overview,
			ReleaseDate:      m.ReleaseDate,
			PosterPath:       m.PosterPath,
			BackdropPath:     m.BackdropPath,
			Popularity:       m.Popularity,
			VoteCount:        m.VoteCount,
			VoteAverage:      m.VoteAverage,
			GenreIDs:         m.GenreIDs,
			Adult:            m.Adult,
			Video:            m.Video,
		}
	}

	return movies, nil
}

func (s *MovieService) CreateMovie(ctx context.Context, movie models.Movie) error {
	log.Debug("creating movie", "movie_id", movie.ID, "title", movie.Title)

	err := s.db.InsertMovie(ctx, movie)
	if err != nil {
		log.Error("failed to insert movie", "movie_id", movie.ID, "error", err)
		return fmt.Errorf("failed to insert movie %d: %w", movie.ID, err)
	}

	log.Info("movie created", "movie_id", movie.ID, "title", movie.Title)
	return nil
}

// AddWatched records a movie as watched. If the movie doesn't exist in the database,
// it fetches the movie details from TMDB and adds it first. Used for tracking user's
// watched movie history.
func (s *MovieService) AddWatched(ctx context.Context, watchedEntry models.Watched) error {
	log.Debug("adding watched entry", "movie_id", watchedEntry.MovieID)

	_, err := s.db.GetMovieByID(ctx, watchedEntry.MovieID)
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			log.Error("database error checking movie", "movie_id", watchedEntry.MovieID, "error", err)
			return fmt.Errorf("failed to check if movie exists: %w", err)
		}

		log.Debug("movie not found, fetching from TMDB", "movie_id", watchedEntry.MovieID)

		// Movie doesn't exist, fetch from TMDB and add it
		movieDetails, err := s.tmdb.GetMovieDetails(int(watchedEntry.MovieID), nil)
		if err != nil {
			log.Error("TMDB fetch failed", "movie_id", watchedEntry.MovieID, "error", err)
			return fmt.Errorf("failed to fetch movie details from TMDB: %w", err)
		}

		newMovie, err := models.MovieFromTMDBMovieDetails(*movieDetails)
		if err != nil {
			log.Error("movie conversion failed", "movie_id", watchedEntry.MovieID, "error", err)
			return fmt.Errorf("failed to convert TMDB data to movie model: %w", err)
		}

		err = s.CreateMovie(ctx, newMovie)
		if err != nil {
			log.Error("movie creation failed", "movie_id", watchedEntry.MovieID, "error", err)
			return fmt.Errorf("failed to save movie to database: %w", err)
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

func (s *MovieService) ImportWatched(ctx context.Context, movies models.WatchedMoviesLog) error {
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
				// TODO: handle the error better. ImportMovies could be made idempotent?
				return fmt.Errorf("failed to import movie: %w", err)
			}
		}
	}

	log.Info("movie import completed", "total_movies", totalMovies)
	return nil
}

func (s *MovieService) ExportWatched(ctx context.Context) (models.WatchedMoviesLog, error) {
	watched, err := s.db.GetAllWatched(ctx)
	if err != nil {
		return nil, fmt.Errorf("TODO: %w", err)
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

func (s *MovieService) GetAllWatchedMovies(ctx context.Context) ([]models.WatchedMovie, error) {
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
