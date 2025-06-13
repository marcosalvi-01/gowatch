package server

import (
	"net/http"
	"time"
)

// exportWatched returns all watched movies grouped by watched date
//
//	@Summary		Export watched movies grouped by date
//	@Description	Export all watched movies grouped by their watched date. Returns a map where keys are dates and values are arrays of TMDB movie IDs watched on that date.
//	@Tags			Movies
//	@Produce		json
//	@Success		200	{object}	map[string][]int64	"Map of watched dates to arrays of TMDB movie IDs (dates as keys, movie ID arrays as values)"
//	@Failure		500	{string}	string				"Server error while fetching watched movies"
//	@Router			/api/watched/export [get]
func (s *Server) exportWatched(w http.ResponseWriter, r *http.Request) {
	movies, err := s.query.GetAllWatched(r.Context())
	if err != nil {
		log.Error("Failed to retrieve watched movies from database", "error", err)
		http.Error(w, "Failed to retrieve watched movies", http.StatusInternalServerError)
		return
	}
	result := map[time.Time][]int64{}
	for _, movie := range movies {
		if _, ok := result[movie.WatchedDate]; !ok {
			result[movie.WatchedDate] = []int64{}
		}
		result[movie.WatchedDate] = append(result[movie.WatchedDate], movie.MovieID)
	}
	jsonResponse(w, http.StatusOK, result)
}
