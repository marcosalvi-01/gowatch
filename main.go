// Package main is the entry point for the gowatch movie tracking application. It sets up configuration, initializes services, and starts the HTTP server.
package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"gowatch/db"
	"gowatch/internal/routes"
	"gowatch/internal/services"
	"gowatch/logging"

	"github.com/caarlos0/env/v11"
	tmdb "github.com/cyruzin/golang-tmdb"
)

var log = logging.Get("app")

type Config struct {
	Port                 string        `env:"PORT" envDefault:"8080"`
	Timeout              time.Duration `env:"REQUEST_TIMEOUT" envDefault:"30s"`
	DBPath               string        `env:"DB_PATH" envDefault:"/var/lib/gowatch"`
	DBName               string        `env:"DB_NAME" envDefault:"db.db"`
	TMDBAPIKey           string        `env:"TMDB_API_KEY"`
	TMDBPosterPrefix     string        `env:"TMDB_POSTER_PREFIX" envDefault:"https://image.tmdb.org/t/p/w500"`
	CacheTTL             time.Duration `env:"CACHE_TTL" envDefault:"168h"`
	SessionExpiry        time.Duration `env:"SESSION_EXPIRY" envDefault:"24h"`
	ShutdownTimeout      time.Duration `env:"SHUTDOWN_TIMEOUT" envDefault:"30s"`
	HTTPS                bool          `env:"HTTPS" envDefault:"false"`
	AdminDefaultPassword string        `env:"ADMIN_DEFAULT_PASSWORD" envDefault:"Welcome123!"`
}

func main() {
	defer func() {
		_ = logging.Close()
	}()

	log.Info("initializing application")

	cfg := Config{}
	if err := env.Parse(&cfg); err != nil {
		log.Error("failed to parse environment configuration", "error", err)
		panic(err)
	}

	log.Info("configuration loaded", "port", cfg.Port, "dbPath", cfg.DBPath, "dbName", cfg.DBName, "cacheTTL", cfg.CacheTTL)

	db, err := db.NewSqliteDB(cfg.DBPath, cfg.DBName)
	if err != nil {
		log.Error("failed to initialize database", "error", err)
		panic(err)
	}
	defer func() {
		_ = db.Close()
	}()

	tmdb, err := tmdb.Init(cfg.TMDBAPIKey)
	if err != nil {
		log.Error("failed to initialize TMDB client", "error", err)
		panic(err)
	}

	log.Debug("initializing services")
	movieService := services.NewMovieService(db, tmdb, cfg.CacheTTL)
	watchedService := services.NewWatchedService(db, movieService)
	listService := services.NewListService(db, movieService)
	authService := services.NewAuthService(db, cfg.SessionExpiry, cfg.HTTPS, cfg.AdminDefaultPassword)

	router := routes.NewRouter(db, movieService, watchedService, listService, authService)

	server := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      router,
		ReadTimeout:  cfg.Timeout,
		WriteTimeout: cfg.Timeout,
	}

	ticker := time.NewTicker(cfg.SessionExpiry)
	done := make(chan bool)

	// clean expired sessions
	go func() {
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				err := authService.CleanupExpiredSessions(context.Background())
				if err != nil {
					log.Error("failed to cleanup expired sessions", "error", err)
					// Continue to next tick instead of returning
				} else {
					log.Debug("successfully cleaned up expired sessions")
				}
			}
		}
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		log.Info("starting server", "port", cfg.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("server failed to start", "error", err)
			panic(err)
		}
	}()

	sig := <-sigChan
	log.Info("received signal, shutting down gracefully", "signal", sig)

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Error("server shutdown failed", "error", err)
		os.Exit(1)
	}

	log.Info("server shutdown complete")
}
