package main

import (
	"gowatch/db"
	"gowatch/internal/routes"
	"gowatch/internal/services"
	"gowatch/logging"
	"log"
	"net/http"
	"time"

	"github.com/caarlos0/env/v11"
	tmdb "github.com/cyruzin/golang-tmdb"
)

type Config struct {
	Port             string        `env:"PORT" envDefault:"8080"`
	Timeout          time.Duration `env:"REQUEST_TIMEOUT" envDefault:"30s"`
	DBPath           string        `env:"DB_PATH" envDefault:"./"`
	DBName           string        `env:"DB_NAME" envDefault:"db.db"`
	TMDBAPIKey       string        `env:"TMDB_API_KEY"`
	TMDBPosterPrefix string        `env:"TMDB_POSTER_PREFIX" envDefault:"https://image.tmdb.org/t/p/w500"`
}

func main() {
	defer logging.Close()

	cfg := Config{}
	if err := env.Parse(&cfg); err != nil {
		log.Fatalf("Failed to parse env: %v", err)
	}

	db, err := db.NewSqliteDB(cfg.DBPath, cfg.DBName)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	tmdb, err := tmdb.Init(cfg.TMDBAPIKey)
	if err != nil {
		panic(err)
	}

	tmdbService := services.NewTMDBService(tmdb)
	creditsService := services.NewCreditsService(db, tmdbService)
	movieService := services.NewMovieService(db, tmdbService, creditsService)
	watchedService := services.NewWatchedService(db, movieService, creditsService)

	router := routes.NewRouter(movieService, watchedService)

	server := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      router,
		ReadTimeout:  cfg.Timeout,
		WriteTimeout: cfg.Timeout,
	}

	log.Printf("Server starting on port %s", cfg.Port)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal("Server failed to start:", err)
	}
}
