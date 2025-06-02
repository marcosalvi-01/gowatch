package server

import (
	"database/sql"
	"encoding/json"
	"errors"
	"gowatch/db"
	"gowatch/model"
	"net/http"
	"time"
)

// postWatched handles marking a movie as watched
//
//	@Summary		Mark a movie as watched
//	@Description	Mark a movie as watched by providing a TMDB ID. If the movie doesn't exist in the database, it will be fetched from TMDB and created. If no date is provided, the current date is used.
//	@Tags			Movies
//	@Accept			json
//	@Produce		json
//	@Param			watched	body		model.Watched	true	"Watched movie data (must include `id` as TMDB ID; optional `date` in YYYY-MM-DD format)"
//	@Success		201		{object}	model.Watched	"Successfully marked movie as watched"
//	@Failure		400		{string}	string			"Invalid request body or missing required `id` field"
//	@Failure		500		{string}	string			"Internal server error, database error, or TMDB API error"
//	@Router			/api/watched [post]
func (s *Server) postWatched(w http.ResponseWriter, r *http.Request) {
	log.Info("Received request to mark movie as watched")

	var watched model.Watched
	if err := json.NewDecoder(r.Body).Decode(&watched); err != nil {
		log.Error("Failed to decode request body", "error", err)
		http.Error(w, "Invalid JSON format", http.StatusBadRequest)
		return
	}

	log.Debug("Parsed watched request", "tmdb_id", watched.ID, "date", watched.Date)

	// Try to get existing movie
	log.Debug("Looking up movie in database", "tmdb_id", watched.ID)
	movie, err := s.query.GetMovieFromReference(r.Context(), watched.ID)
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			log.Error("Database error while looking up movie", "tmdb_id", watched.ID, "error", err)
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}

		log.Info("Movie not found in database, fetching from TMDB", "tmdb_id", watched.ID)
		// Movie doesn't exist, fetch from TMDB and create
		details, err := s.tmdb.GetMovieDetails(int(watched.ID), nil)
		if err != nil {
			log.Error("Failed to fetch movie from TMDB", "tmdb_id", watched.ID, "error", err)
			http.Error(w, "Failed to fetch from TMDB", http.StatusInternalServerError)
			return
		}

		log.Debug("Successfully fetched movie from TMDB", "tmdb_id", watched.ID, "title", details.Title)

		releaseDate, err := time.Parse("2006-01-02", details.ReleaseDate)
		if err != nil {
			log.Error("Failed to parse release date from TMDB", "tmdb_id", watched.ID, "release_date", details.ReleaseDate, "error", err)
			http.Error(w, "Invalid release date from TMDB", http.StatusInternalServerError)
			return
		}

		// Create movie in database
		log.Debug("Creating movie in database", "tmdb_id", watched.ID, "title", details.Title)
		movie, err = s.query.InsertMovie(r.Context(), db.InsertMovieParams{
			ID:               watched.ID,
			Title:            details.Title,
			ReleaseDate:      releaseDate,
			ImdbID:           details.IMDbID,
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
			http.Error(w, "Failed to create movie", http.StatusInternalServerError)
			return
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
			http.Error(w, "Invalid date format", http.StatusBadRequest)
			return
		}
	} else {
		w := watchDate.Format("2006-01-02")
		watched.Date = &w
	}

	// Record watched entry
	log.Debug("Recording movie as watched", "movie_id", movie.ID, "watch_date", watchDate)
	_, err = s.query.InsertWatched(r.Context(), db.InsertWatchedParams{
		MovieID:     movie.ID,
		WatchedDate: watchDate,
	})
	if err != nil {
		log.Error("Failed to record watched movie", "movie_id", movie.ID, "error", err)
		http.Error(w, "Failed to record movie as watched", http.StatusInternalServerError)
		return
	}

	log.Info("Successfully marked movie as watched", "tmdb_id", watched.ID, "title", movie.Title, "watch_date", watchDate)
	jsonResponse(w, http.StatusCreated, watched)
}

// getWatched returns all watched movies
//
//	@Summary	Get all watched movies
//	@Tags		Movies
//	@Produce	json
//	@Success	200	{array}	model.Movie	"List of all watched movies (array may be empty)"
//	@Failure	500	{string}	string			"Server error while fetching watched movies"
//	@Router		/api/watched [get]
func (s *Server) getWatched(w http.ResponseWriter, r *http.Request) {
	log.Info("Received request to get all watched movies")

	watched, err := s.query.GetWatchedJoinMovie(r.Context())
	if err != nil {
		log.Error("Failed to retrieve watched movies from database", "error", err)
		http.Error(w, "Failed to retrieve watched movies", http.StatusInternalServerError)
		return
	}

	log.Debug("Retrieved watched movies from database", "count", len(watched))

	movies := make([]model.Movie, len(watched))
	for i, w := range watched {
		movies[i] = model.Movie{
			ID:               w.ID,
			Title:            w.Title,
			ReleaseDate:      w.ReleaseDate,
			OriginalLanguage: w.OriginalLanguage,
			Overview:         w.Overview,
			PosterPath:       w.PosterPath,
			Budget:           w.Budget,
			Revenue:          w.Revenue,
			Runtime:          w.Runtime,
			VoteAverage:      w.VoteAverage,
			WatchedDate:      &w.WatchedDate,
		}
	}

	log.Info("Successfully returning watched movies", "count", len(movies))
	jsonResponse(w, http.StatusOK, movies)
}
