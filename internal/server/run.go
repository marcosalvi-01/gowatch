// Package server contains the server startup logic.
package server

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/marcosalvi-01/gowatch/db"
	"github.com/marcosalvi-01/gowatch/internal/models"
	"github.com/marcosalvi-01/gowatch/internal/routes"
	"github.com/marcosalvi-01/gowatch/internal/services"
	"github.com/marcosalvi-01/gowatch/logging"
)

// Config holds the application configuration.
type Config struct {
	Port                 string
	Timeout              time.Duration
	DBPath               string
	DBName               string
	SeerrBaseURL         string
	SeerrAPIKey          string
	CacheTTL             time.Duration
	ImageCacheTTL        time.Duration
	ImageCleanupInterval time.Duration
	SessionExpiry        time.Duration
	ShutdownTimeout      time.Duration
	HTTPS                bool
	AdminDefaultPassword string
}

// RunServer starts the HTTP server with the given configuration.
func RunServer(cfg Config) {
	defer func() {
		_ = logging.Close()
	}()

	log := logging.Get("server")
	log.Info("initializing application")

	log.Info(
		"configuration loaded",
		"port", cfg.Port,
		"dbPath", cfg.DBPath,
		"dbName", cfg.DBName,
		"cacheTTL", cfg.CacheTTL,
		"imageCacheTTL", cfg.ImageCacheTTL,
		"imageCleanupInterval", cfg.ImageCleanupInterval,
	)

	seerrService, err := services.NewSeerrService(models.SeerrIntegration{
		BaseURL: cfg.SeerrBaseURL,
		APIKey:  cfg.SeerrAPIKey,
	}, &http.Client{Timeout: cfg.Timeout})
	if err != nil {
		log.Error("invalid Seerr configuration", "error", err)
		panic(err)
	}

	db, err := db.NewSqliteDB(cfg.DBPath, cfg.DBName)
	if err != nil {
		log.Error("failed to initialize database", "error", err)
		panic(err)
	}
	defer func() {
		_ = db.Close()
	}()

	log.Debug("initializing services")
	movieService := services.NewMovieService(db, seerrService, cfg.CacheTTL)
	imageCacheDir := filepath.Join(cfg.DBPath, "tmdb-images")
	if err := os.MkdirAll(imageCacheDir, 0o750); err != nil {
		log.Error("failed to create image cache directory", "path", imageCacheDir, "error", err)
		panic(err)
	}
	tmdbImageService := services.NewTMDBImageService(imageCacheDir, cfg.ImageCacheTTL, &http.Client{Timeout: cfg.Timeout})
	listService := services.NewListService(db, movieService)
	watchedService := services.NewWatchedService(db, listService, movieService)
	authService := services.NewAuthService(db, listService, cfg.SessionExpiry, cfg.HTTPS, cfg.AdminDefaultPassword)
	router := routes.NewRouter(db, movieService, tmdbImageService, watchedService, listService, authService, seerrService)

	server := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      router,
		ReadTimeout:  cfg.Timeout,
		WriteTimeout: cfg.Timeout,
	}

	sessionCleanupTicker := time.NewTicker(cfg.SessionExpiry)
	imageCacheCleanupTicker := time.NewTicker(cfg.ImageCleanupInterval)
	done := make(chan struct{})

	// clean expired sessions
	go func() {
		for {
			select {
			case <-done:
				return
			case <-sessionCleanupTicker.C:
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

	go func() {
		for {
			select {
			case <-done:
				return
			case <-imageCacheCleanupTicker.C:
				err := tmdbImageService.CleanupExpiredCache()
				if err != nil {
					log.Error("failed to cleanup expired image cache", "error", err)
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
	close(done)
	sessionCleanupTicker.Stop()
	imageCacheCleanupTicker.Stop()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Error("server shutdown failed", "error", err)
		os.Exit(1)
	}

	log.Info("server shutdown complete")
}

func (cfg Config) Validate() error {
	if strings.TrimSpace(cfg.SeerrBaseURL) == "" {
		return errors.New("SEERR_BASE_URL is required")
	}
	if strings.TrimSpace(cfg.SeerrAPIKey) == "" {
		return errors.New("SEERR_API_KEY is required")
	}
	return nil
}
