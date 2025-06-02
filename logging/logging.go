package logging

import (
	"fmt"
	"log/slog"
	"os"
	"sync"
)

var (
	level = slog.LevelInfo

	base *slog.Logger
	once sync.Once
)

func init() {
	parseLevelFromEnv()
}

func initBase() {
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: level,
	})
	base = slog.New(handler)
	slog.SetDefault(base.With("component", "default"))
}

func parseLevelFromEnv() {
	v, ok := os.LookupEnv("LOG_LEVEL")
	if ok {
		switch v {
		case "DEBUG":
			level = slog.LevelDebug
		case "INFO":
			level = slog.LevelInfo
		default:
			slog.Warn("unknown LOG_LEVEL, falling back to INFO", "value", v)
			return
		}
	} else {
		slog.Warn("env variable LOG_LEVEL not set, falling back to INFO")
		return
	}
	fmt.Printf("setting up logging with level %s\n", v)
}

// Get returns a logger with the given component name.
//
// Use a package‑level variable to hold the logger:
//
//	var log = logging.Get("database")
func Get(component string) *slog.Logger {
	once.Do(initBase)
	return base.With("component", component)
}
