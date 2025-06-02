package server

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"gowatch/db"
	"gowatch/model"
	"net/http"
	"time"
)

// postWatched handles marking a movie as watched
//
//	@Summary		Mark a movie as watched
//	@Description	Mark a movie as watched by providing either a TMDB ID or name. If the movie doesn't exist in the database, it will be created. When using TMDB ID, movie details are fetched from TMDB API. When using name only, a basic movie record is created. If no date is provided, the current timestamp is used.
//	@Tags			Movies
//	@Accept			json
//	@Produce		json
//	@Param			watched	body		model.Watched	true	"Watched movie data - must include either tmdb_id or name"
//	@Success		201		{object}	model.Watched	"Successfully marked movie as watched"
//	@Failure		400		{string}	string			"Invalid request body or missing required fields (tmdb_id or name)"
//	@Failure		500		{string}	string			"Internal server error, database error, or TMDB API error"
//	@Router			/api/watched [post]
func (s *Server) postWatched(w http.ResponseWriter, r *http.Request) {
	var watched model.Watched
	if err := json.NewDecoder(r.Body).Decode(&watched); err != nil {
		http.Error(w, "Invalid JSON format in request body", http.StatusBadRequest)
		return
	}

	// Set default date if not provided
	if watched.Date == nil {
		now := time.Now().Format("2006-01-02")
		watched.Date = &now
	}

	// Get or create the movie
	movie, err := s.getOrCreateMovie(r.Context(), watched.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Record the watched entry
	if watched.Date == nil {
		_, err = s.query.InsertWatched(r.Context(), db.InsertWatchedParams{
			MovieID:     movie.ID,
			WatchedDate: time.Now(),
		})
	} else {
		var date time.Time
		date, err = time.Parse("2006-01-02", *watched.Date)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		_, err = s.query.InsertWatched(r.Context(), db.InsertWatchedParams{
			MovieID:     movie.ID,
			WatchedDate: date,
		})
	}
	if err != nil {
		http.Error(w, "Failed to record movie as watched", http.StatusInternalServerError)
		return
	}

	jsonResponse(w, http.StatusCreated, watched)
}

// getOrCreateMovie handles the logic of finding or creating a movie by TMDB ID or name
func (s *Server) getOrCreateMovie(ctx context.Context, tmdbID int64) (*db.Movie, error) {
	// Try to get existing movie
	movie, err := s.query.GetMovieFromReference(ctx, tmdbID)
	if err == nil {
		return &movie, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("database error while looking up movie by reference: %w", err)
	}

	// Movie doesn't exist, create it with TMDB details
	details, err := s.tmdb.GetMovieDetails(int(tmdbID), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch movie details from TMDB: %w", err)
	}

	release, err := time.Parse("2006-01-02", details.ReleaseDate)
	if err != nil {
		return nil, fmt.Errorf("failed to parse release date: %w", err)
	}

	runtime := int64(details.Runtime)
	voteAverage := float64(details.VoteAverage)

	movie, err = s.query.InsertMovie(ctx, db.InsertMovieParams{
		ID:               tmdbID,
		Title:            details.Title,
		ReleaseDate:      release,
		ImdbID:           details.IMDbID,
		OriginalLanguage: details.OriginalLanguage,
		Overview:         details.Overview,
		PosterPath:       details.PosterPath,
		Budget:           details.Budget,
		Revenue:          details.Revenue,
		Runtime:          runtime,
		VoteAverage:      voteAverage,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to insert movie: %w", err)
	}

	return &movie, nil
}

// return all watched movies
//
//	@Summary	Get all watched movies
//	@Tags		Movies
//	@Produce	json
//	@Success	200	{array}		server.uiWatched	"List of all watched movies (array may be empty)"
//	@Failure	500	{string}	string				"Server error while fetching watched movies"
//	@Router		/api/watched [get]
func (s *Server) getWatched(w http.ResponseWriter, r *http.Request) {
	watched, err := s.query.GetWatchedJoinMovie(r.Context())
	if err != nil {
		http.Error(w, "Failed to retrieve watched movies", http.StatusInternalServerError)
		return
	}

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

	jsonResponse(w, http.StatusOK, movies)
}
