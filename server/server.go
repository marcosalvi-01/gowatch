package server

import (
	"encoding/json"
	"fmt"
	"gowatch/db"
	"gowatch/logging"
	_ "gowatch/server/docs"
	"gowatch/ui"
	"net/http"
	"time"

	tmdb "github.com/cyruzin/golang-tmdb"
	"github.com/go-chi/chi"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	httpSwagger "github.com/swaggo/http-swagger"
)

//go:generate go tool swag init --parseDependency -g ./server.go

var log = logging.Get("server")

type Server struct {
	query  *db.Queries
	server http.Server
	tmdb   *tmdb.Client
	ui     *ui.App
}

// New creates a new Server. The TMDB API key is passed as an argument for better testability.
func New(port string, q *db.Queries, timeout time.Duration, tmdbClient *tmdb.Client, ui *ui.App) (*Server, error) {
	log.Info("Initializing new server instance",
		"port", port,
		"read_timeout", timeout,
	)

	srv := Server{
		query: q,
		server: http.Server{
			Addr:        port,
			ReadTimeout: timeout,
		},
		tmdb: tmdbClient,
		ui:   ui,
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
	r := chi.NewRouter()

	r.Handle("/metrics", promhttp.Handler())
	r.Handle("/swagger/*", httpSwagger.WrapHandler)

	r.Route("/api", func(api chi.Router) {
		api.Route("/watched", func(watched chi.Router) {
			watched.Get("/", s.getWatched)
			watched.Post("/", s.postWatched)
		})

		api.Get("/search/movie", s.searchMovie)
	})

	// ui
	r.Mount("/ui", s.ui.Routes())

	// home
	r.Handle("/", s.ui.Index())

	return r
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
