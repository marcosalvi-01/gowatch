package main

import (
	"gowatch/db"
	"gowatch/server"
	"log"
	"time"

	"github.com/caarlos0/env"
)

type Config struct {
	Port       string        `env:"PORT" envDefault:"8080"`
	Timeout    time.Duration `env:"REQUEST_TIMEOUT" envDefault:"30s"`
	DBPath     string        `env:"DB_PATH" envDefault:"./"`
	DBName     string        `env:"DB_NAME" envDefault:"db.db"`
	TMDBApiKey string        `env:"TMDB_API_KEY"`
}

func main() {
	cfg := Config{}
	if err := env.Parse(&cfg); err != nil {
		log.Fatalf("Failed to parse env: %v", err)
	}

	q, err := db.Get(cfg.DBPath, cfg.DBName)
	if err != nil {
		panic(err)
	}

	s, err := server.New(":"+cfg.Port, q, cfg.Timeout, cfg.TMDBApiKey)
	if err != nil {
		panic(err)
	}

	s.ListenAndServe()
}
