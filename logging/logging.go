package logging

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"sync"
)

var (
	level   = slog.LevelInfo
	base    *slog.Logger
	once    sync.Once
	logFile *os.File
)

func init() {
	v, ok := os.LookupEnv("LOG_LEVEL")
	if ok {
		switch v {
		case "DEBUG":
			level = slog.LevelDebug
		case "INFO":
			level = slog.LevelInfo
		case "WARN":
			level = slog.LevelWarn
		case "ERROR":
			level = slog.LevelError
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

func initBase() {
	var writer io.Writer = os.Stdout

	// Check for LOG_FILE environment variable
	if logPath, ok := os.LookupEnv("LOG_FILE"); ok {
		file, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			slog.Error("failed to open log file, falling back to stdout", "path", logPath, "error", err)
		} else {
			logFile = file
			writer = file
			fmt.Printf("logging to file: %s\n", logPath)
		}
	}

	handler := slog.NewJSONHandler(writer, &slog.HandlerOptions{
		Level: level,
	})
	base = slog.New(handler)
	slog.SetDefault(base.With("component", "default"))
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

// Close closes the log file if one was opened.
// Call this function when your application shuts down to ensure
// the log file is properly closed.
func Close() error {
	if logFile != nil {
		return logFile.Close()
	}
	return nil
}
