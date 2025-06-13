package server

import (
	"net/http"
	"time"
)

type watchedMovie struct {
	MovieID   int64 `json:"movie_id"`
	InTheater bool  `json:"in_theater,omitempty"`
}

type exportMovie struct {
	Date   time.Time      `json:"date"`
	Movies []watchedMovie `json:"movies"`
}

// exportWatched returns all watched movies grouped by watched date
//
//	@Summary		Export watched movies grouped by date
//	@Description	Export all watched movies grouped by their watched date. Returns an array of objects containing date and movie IDs watched on that date.
//	@Tags			Movies
//	@Produce		json
//	@Success		200	{array}		exportMovie	"Array of objects with date and movie IDs watched on that date"
//	@Failure		500	{string}	string		"Server error while fetching watched movies"
func (s *Server) exportWatched(w http.ResponseWriter, r *http.Request) {
	movies, err := s.query.GetAllWatched(r.Context())
	if err != nil {
		log.Error("Failed to retrieve watched movies from database", "error", err)
		http.Error(w, "Failed to retrieve watched movies", http.StatusInternalServerError)
		return
	}

	// Group movies by watched date
	movieMap := make(map[time.Time][]watchedMovie)
	for _, movie := range movies {
		watchedMovie := watchedMovie{
			MovieID: movie.MovieID,
			// InTheater: movie.InTheater,
		}
		movieMap[movie.WatchedDate] = append(movieMap[movie.WatchedDate], watchedMovie)
	}

	// Convert map to slice of exportMovie
	result := make([]exportMovie, 0, len(movieMap))
	for date, movieList := range movieMap {
		result = append(result, exportMovie{
			Date:   date,
			Movies: movieList,
		})
	}

	jsonResponse(w, http.StatusOK, result)
} //	@Router	/api/watched/export [get]
