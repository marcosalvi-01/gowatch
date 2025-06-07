package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"gowatch/model"
	"time"

	tmdb "github.com/cyruzin/golang-tmdb"
)

func (q *Queries) NewWatched(ctx context.Context, watched model.Watched, tmdb *tmdb.Client) error {
	log.Info("Received request to mark movie as watched")
	log.Debug("Parsed watched request", "tmdb_id", watched.ID, "date", watched.Date)

	// Try to get existing movie
	log.Debug("Looking up movie in database", "tmdb_id", watched.ID)
	movie, err := q.GetMovieFromReference(ctx, watched.ID)
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			log.Error("Database error while looking up movie", "tmdb_id", watched.ID, "error", err)
			return fmt.Errorf("database error while looking up movie with tmdb_id %d: %w", watched.ID, err)
		}

		log.Info("Movie not found in database, fetching from TMDB", "tmdb_id", watched.ID)
		// Movie doesn't exist, fetch from TMDB and create
		details, err := tmdb.GetMovieDetails(int(watched.ID), nil)
		if err != nil {
			log.Error("Failed to fetch movie from TMDB", "tmdb_id", watched.ID, "error", err)
			return fmt.Errorf("failed to fetch movie from TMDB with tmdb_id %d: %w", watched.ID, err)
		}

		log.Debug("Successfully fetched movie from TMDB", "tmdb_id", watched.ID, "title", details.Title)
		var releaseDate time.Time
		if details.ReleaseDate != "" {
			releaseDate, err = time.Parse("2006-01-02", details.ReleaseDate)
			if err != nil {
				log.Error("Failed to parse release date from TMDB", "tmdb_id", watched.ID, "release_date", details.ReleaseDate, "error", err)
				return fmt.Errorf("failed to parse release date '%s' from TMDB for tmdb_id %d: %w", details.ReleaseDate, watched.ID, err)
			}
		}

		// Create movie in database
		log.Debug("Creating movie in database", "tmdb_id", watched.ID, "title", details.Title)
		movie, err = q.InsertMovie(ctx, InsertMovieParams{
			ID:               watched.ID,
			Title:            details.Title,
			ReleaseDate:      releaseDate,
			ImdbID:           &details.IMDbID,
			OriginalLanguage: details.OriginalLanguage,
			Overview:         details.Overview,
			PosterPath:       details.PosterPath,
			Budget:           details.Budget,
			Revenue:          details.Revenue,
			Runtime:          int64(details.Runtime),
			VoteAverage:      float64(details.VoteAverage),
		})
		if err != nil {
			log.Error("Failed to create movie in database", "tmdb_id", watched.ID, "error", err)
			return fmt.Errorf("failed to create movie in database with tmdb_id %d: %w", watched.ID, err)
		}

		log.Info("Successfully created movie", "tmdb_id", watched.ID, "title", details.Title)
	} else {
		log.Debug("Found existing movie in database", "tmdb_id", watched.ID, "title", movie.Title)
	}

	// Parse watch date
	watchDate := time.Now()
	if watched.Date != nil {
		watchDate, err = time.Parse("2006-01-02", *watched.Date)
		if err != nil {
			log.Error("Failed to parse watch date", "date", *watched.Date, "error", err)
			return fmt.Errorf("failed to parse watch date '%s': %w", *watched.Date, err)
		}
	} else {
		w := watchDate.Format("2006-01-02")
		watched.Date = &w
	}

	// Record watched entry
	log.Debug("Recording movie as watched", "movie_id", movie.ID, "watch_date", watchDate)
	_, err = q.InsertWatched(ctx, InsertWatchedParams{
		MovieID:     movie.ID,
		WatchedDate: watchDate,
	})
	if err != nil {
		log.Error("Failed to record watched movie", "movie_id", movie.ID, "error", err)
		return fmt.Errorf("failed to record watched movie with movie_id %d: %w", movie.ID, err)
	}

	log.Info("Successfully marked movie as watched", "tmdb_id", watched.ID, "title", movie.Title, "watch_date", watchDate)
	return nil
}
