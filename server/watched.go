package server

import (
	"encoding/json"
	"gowatch/model"
	"net/http"
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

	err := s.query.NewWatched(r.Context(), watched, s.tmdb)
	if err != nil {
		log.Error("Failed to insert new watched movie", "error", err)
		http.Error(w, "Invalid JSON format", http.StatusBadRequest)
		return
	}

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
