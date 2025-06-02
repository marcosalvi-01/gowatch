package server

import "net/http"

// searchMovie searches for movies using TMDB API
//
//	@Summary		Search movies via TMDB
//	@Description	Search for movies using The Movie Database (TMDB) API. Provide search terms via query parameter to find matching movies.
//	@Tags			Movies
//	@Produce		json
//	@Param			query	query		string				true	"Search query for movie titles"
//	@Success		200		{object}	tmdb.MovieResult	"TMDB search results containing matching movies"
//	@Failure		400		{string}	string				"Missing or empty query parameter"
//	@Failure		500		{string}	string				"TMDB API error or server error"
//	@Router			/api/search/movie [get]
func (s *Server) searchMovie(w http.ResponseWriter, r *http.Request) {
	log.Info("Received movie search request")

	// Parse query parameter
	query := r.URL.Query().Get("query")
	if query == "" {
		log.Warn("Movie search request with empty query")
		http.Error(w, "Query parameter cannot be empty", http.StatusBadRequest)
		return
	}

	log.Debug("Searching for movies", "query", query)

	// Search movies via TMDB API
	search, err := s.tmdb.GetSearchMovies(query, nil)
	if err != nil {
		log.Error("Failed to search movies via TMDB", "query", query, "error", err)
		http.Error(w, "Failed to search movies", http.StatusInternalServerError)
		return
	}

	log.Info("Successfully searched movies", "query", query, "results_count", len(search.Results))
	jsonResponse(w, http.StatusOK, search.Results)
}
