package main

import (
	"gowatch/db"
	"gowatch/logging"
	"gowatch/server"
	"gowatch/ui"
	"log"
	"time"

	"github.com/caarlos0/env"
)

type Config struct {
	Port             string        `env:"PORT" envDefault:"8080"`
	Timeout          time.Duration `env:"REQUEST_TIMEOUT" envDefault:"30s"`
	DBPath           string        `env:"DB_PATH" envDefault:"./"`
	DBName           string        `env:"DB_NAME" envDefault:"db.db"`
	TmdbApiKey       string        `env:"TMDB_API_KEY"`
	TmdbPosterPrefix string        `env:"TMDB_POSTER_PREFIX" envDefault:"https://image.tmdb.org/t/p/w500"`
}

func main() {
	defer logging.Close()

	cfg := Config{}
	if err := env.Parse(&cfg); err != nil {
		log.Fatalf("Failed to parse env: %v", err)
	}

	q, err := db.Get(cfg.DBPath, cfg.DBName)
	if err != nil {
		panic(err)
	}

	u := ui.New(q)

	s, err := server.New(":"+cfg.Port, q, cfg.Timeout, cfg.TmdbApiKey, u)
	if err != nil {
		panic(err)
	}

	s.ListenAndServe()
}
