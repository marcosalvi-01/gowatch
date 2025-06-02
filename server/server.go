package server

import (
	"encoding/json"
	"fmt"
	"gowatch/db"
	_ "gowatch/server/docs"
	"gowatch/logging"
	"net/http"
	"time"

	tmdb "github.com/cyruzin/golang-tmdb"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	httpSwagger "github.com/swaggo/http-swagger"
)

//go:generate go tool swag init --parseDependency -g ./server.go

var log = logging.Get("server")

type Server struct {
	query  *db.Queries
	server http.Server
	tmdb   *tmdb.Client
}

// New creates a new Server. The TMDB API key is passed as an argument for better testability.
func New(port string, q *db.Queries, timeout time.Duration, apiKey string) (*Server, error) {
	log.Info("Initializing new server instance",
		"port", port,
		"read_timeout", timeout,
	)
	log.Debug("Initializing TMDB client", "api_key_provided", apiKey != "")
	tmdbClient, err := tmdb.Init(apiKey)
	if err != nil {
		log.Error("Failed to initialize TMDB client", "error", err)
		return nil, err
	}
	log.Info("TMDB client initialized successfully")

	srv := Server{
		query: q,
		server: http.Server{
			Addr:        port,
			ReadTimeout: timeout,
		},
		tmdb: tmdbClient,
	}
	log.Debug("Server struct created", "server_addr", srv.server.Addr)
	return &srv, nil
}

// ListenAndServe starts the HTTP server.
func (s *Server) ListenAndServe() error {
	log.Info("Starting HTTP server", "addr", s.server.Addr)
	srvHandler := s.initRoutes()
	s.server.Handler = srvHandler

	err := s.server.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		log.Error("HTTP server terminated with error", "error", err)
		return err
	}
	log.Info("HTTP server shut down gracefully")
	return nil
}

func (s *Server) initRoutes() http.Handler {
	log.Debug("Initializing routes")
	mux := http.NewServeMux()

	log.Debug("Registering /metrics endpoint")
	mux.Handle("/metrics", promhttp.Handler())

	log.Debug("Registering /swagger/ endpoint")
	mux.Handle("/swagger/", httpSwagger.WrapHandler)

	apiMux := http.NewServeMux()
	log.Debug("Registering API handlers")
	apiMux.HandleFunc("GET /api/watched", s.getWatched)
	apiMux.HandleFunc("POST /api/watched", s.postWatched)

	log.Debug("Wrapping API mux with CORS middleware")
	mux.Handle("/api/", corsMiddlewarer(apiMux))

	return mux
}

func jsonResponse(w http.ResponseWriter, status int, body any) {
	log.Debug("Preparing JSON response", "status", status, "body_type", fmt.Sprintf("%T", body))
	w.Header().Set("Content-Type", "application/json")

	// Encode to bytes first to check for errors before writing headers
	data, err := json.Marshal(body)
	if err != nil {
		log.Error("Failed to marshal JSON response", "error", err)
		// If encoding fails, send an error response instead
		http.Error(w, "Failed to encode response as JSON", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(status)
	_, writeErr := w.Write(data)
	if writeErr != nil {
		log.Error("Failed to write JSON response", "error", writeErr)
	}
}
