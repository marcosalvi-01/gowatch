// Package db provides database operations for the gowatch application.
// It manages SQLite database connections, migrations, and provides type-safe
// query operations with automatic conversion between sqlc generated types
// and application models.
package db

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"gowatch/db/sqlc"
	"gowatch/internal/models"
	"gowatch/logging"
	"os"
	"path/filepath"

	"github.com/pressly/goose/v3"
	_ "modernc.org/sqlite"
)

var log = logging.Get("database")

//go:embed migrations/*.sql
var embedMigrations embed.FS

func init() {
	goose.SetBaseFS(embedMigrations)
	goose.SetDialect("sqlite")
}

type DB interface {
	Close() error
	Health() error

	GetAllWatched(ctx context.Context) ([]models.Watched, error)
	InsertMovie(ctx context.Context, movie models.Movie) error
	InsertWatched(ctx context.Context, watched models.Watched) error
	GetMovieByID(ctx context.Context, id int64) (models.Movie, error)
	GetWatchedJoinMovie(ctx context.Context) ([]models.WatchedMovie, error)
	GetMostWatchedMovies(ctx context.Context) ([]models.WatchedMovieCount, error)
}

// SqliteDB wraps database connection and queries
type SqliteDB struct {
	queries *sqlc.Queries
	path    string
	name    string
	db      *sql.DB
}

// NewSqliteDB opens a new connection to the database. If the database file does not exist
// in the filesystem, it is created along with any necessary directories.
func NewSqliteDB(dbPath, dbName string) (*SqliteDB, error) {
	log.Debug("Ensuring DB path exists", "path", dbPath)
	if err := os.MkdirAll(dbPath, 0755); err != nil {
		log.Error("Failed to create DB path", "path", dbPath, "error", err)
		return nil, fmt.Errorf("failed to create database directory %s: %w", dbPath, err)
	}

	dbFile := filepath.Join(dbPath, dbName)
	log.Debug("Resolved DB file path", "dbFile", dbFile)

	if err := createDatabaseFileIfNotExists(dbFile); err != nil {
		return nil, err
	}

	db, err := sql.Open("sqlite", dbFile)
	if err != nil {
		log.Error("Failed to open DB connection", "dbFile", dbFile, "error", err)
		return nil, fmt.Errorf("failed to open database connection to %s: %w", dbFile, err)
	}

	// Test the connection
	if err := db.Ping(); err != nil {
		db.Close()
		log.Error("Failed to ping database", "dbFile", dbFile, "error", err)
		return nil, fmt.Errorf("failed to ping database %s: %w", dbFile, err)
	}

	log.Debug("DB connection opened", "dbFile", dbFile)

	if err := runMigrations(db); err != nil {
		db.Close()
		return nil, err
	}

	queries := sqlc.New(db)
	log.Info("DB initialized successfully", "dbFile", dbFile)

	return &SqliteDB{
		queries: queries,
		name:    dbName,
		path:    dbPath,
		db:      db,
	}, nil
}

// createDatabaseFileIfNotExists creates the database file if it doesn't exist
func createDatabaseFileIfNotExists(dbFile string) error {
	if _, err := os.Stat(dbFile); os.IsNotExist(err) {
		log.Info("DB file does not exist, creating", "dbFile", dbFile)
		file, err := os.Create(dbFile)
		if err != nil {
			log.Error("Failed to create DB file", "dbFile", dbFile, "error", err)
			return fmt.Errorf("failed to create database file %s: %w", dbFile, err)
		}
		defer file.Close()
	} else if err != nil {
		log.Error("Failed to check DB file existence", "dbFile", dbFile, "error", err)
		return fmt.Errorf("failed to check database file existence %s: %w", dbFile, err)
	} else {
		log.Debug("DB file already exists", "dbFile", dbFile)
	}
	return nil
}

// runMigrations executes database migrations
func runMigrations(db *sql.DB) error {
	if err := goose.Up(db, "migrations"); err != nil {
		log.Error("Failed to execute migrations", "error", err)
		return fmt.Errorf("failed to execute database migrations: %w", err)
	}
	log.Debug("Migrations executed successfully")
	return nil
}

// Close closes the database connection
func (d *SqliteDB) Close() error {
	if d.db == nil {
		return nil
	}
	return d.db.Close()
}

// Health checks if the database connection is healthy
func (d *SqliteDB) Health() error {
	if d.db == nil {
		return fmt.Errorf("database connection is nil")
	}
	return d.db.Ping()
}
