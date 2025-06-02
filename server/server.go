// server/server.go
package server

import (
	"encoding/json"
	"gowatch/db"
	"gowatch/logging"
	"net/http"
	"time"

	tmdb "github.com/cyruzin/golang-tmdb"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	httpSwagger "github.com/swaggo/http-swagger"
)

var log = logging.Get("server")

type Server struct {
	query  *db.Queries
	server http.Server
	tmdb   *tmdb.Client
}

// New creates a new Server. The TMDB API key is passed as an argument for better testability.
func New(port string, q *db.Queries, timeout time.Duration, apiKey string) (Server, error) {
	tmdbClient, err := tmdb.Init(apiKey)
	if err != nil {
		return Server{}, err
	}
	return Server{
		query: q,
		server: http.Server{
			Addr:        port,
			ReadTimeout: timeout,
		},
		tmdb: tmdbClient,
	}, nil
}

// ListenAndServe starts the HTTP server.
func (s *Server) ListenAndServe() error {
	s.initRoutes()
	return s.server.ListenAndServe()
}

func (s *Server) initRoutes() http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	mux.Handle("/swagger/", httpSwagger.WrapHandler)

	apiMux := http.NewServeMux()
	apiMux.HandleFunc("GET /api/watched", s.getWatched)
	apiMux.HandleFunc("POST /api/watched", s.postWatched)
	// apiMux.HandleFunc("GET /api/search/movie", s.tmdbSearchMovie)

	mux.Handle("/api/", corsMiddlewarer(apiMux))

	return mux
}

func jsonResponse(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")

	// Encode to bytes first to check for errors before writing headers
	data, err := json.Marshal(body)
	if err != nil {
		// If encoding fails, send an error response instead
		http.Error(w, "Failed to encode response as JSON", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(status)
	w.Write(data)
}
