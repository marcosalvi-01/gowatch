package main

import (
	"gowatch/db"
	"gowatch/internal/routes"
	"gowatch/internal/services"
	"gowatch/logging"
	"net/http"
	"time"

	"github.com/caarlos0/env/v11"
	tmdb "github.com/cyruzin/golang-tmdb"
)

var log = logging.Get("app")

type Config struct {
	Port             string        `env:"PORT" envDefault:"8080"`
	Timeout          time.Duration `env:"REQUEST_TIMEOUT" envDefault:"30s"`
	DBPath           string        `env:"DB_PATH" envDefault:"/var/lib/gowatch"`
	DBName           string        `env:"DB_NAME" envDefault:"db.db"`
	TMDBAPIKey       string        `env:"TMDB_API_KEY"`
	TMDBPosterPrefix string        `env:"TMDB_POSTER_PREFIX" envDefault:"https://image.tmdb.org/t/p/w500"`
	CacheTTL         time.Duration `env:"CACHE_TTL" envDefault:"168h"`
}

func main() {
	defer logging.Close()

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
	defer db.Close()

	tmdb, err := tmdb.Init(cfg.TMDBAPIKey)
	if err != nil {
		log.Error("failed to initialize TMDB client", "error", err)
		panic(err)
	}

	log.Debug("initializing services")
	movieService := services.NewMovieService(db, tmdb, cfg.CacheTTL)
	watchedService := services.NewWatchedService(db, movieService)
	listService := services.NewListService(db, movieService)

	router := routes.NewRouter(db, movieService, watchedService, listService)

	server := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      router,
		ReadTimeout:  cfg.Timeout,
		WriteTimeout: cfg.Timeout,
	}

	log.Info("starting server", "port", cfg.Port)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Error("server failed to start", "error", err)
		panic(err)
	}
}
