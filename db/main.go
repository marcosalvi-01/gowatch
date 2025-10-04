package db

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"gowatch/db/sqlc"
	"gowatch/logging"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/pressly/goose/v3"
	_ "modernc.org/sqlite"
)

var log = logging.Get("database")

//go:embed migrations/*.sql
var embedMigrations embed.FS

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

	if err := runMigrations(db, dbPath, dbName); err != nil {
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

// runMigrations executes database migrations with automatic backup if needed
func runMigrations(db *sql.DB, dbPath, dbName string) error {
	ctx := context.Background()

	// Create a sub-filesystem pointing to the migrations directory
	migrationsFS, err := fs.Sub(embedMigrations, "migrations")
	if err != nil {
		log.Error("Failed to create migrations sub-filesystem", "error", err)
		return fmt.Errorf("failed to create migrations filesystem: %w", err)
	}

	provider, err := goose.NewProvider(goose.DialectSQLite3, db, migrationsFS)
	if err != nil {
		log.Error("Failed to create goose provider", "error", err)
		return fmt.Errorf("failed to create migration provider: %w", err)
	}

	hasPending, err := provider.HasPending(ctx)
	if err != nil {
		log.Error("Failed to check for pending migrations", "error", err)
		return fmt.Errorf("failed to check for pending migrations: %w", err)
	}

	if !hasPending {
		log.Debug("No pending migrations")
		return nil
	}

	current, target, err := provider.GetVersions(ctx)
	if err != nil {
		log.Error("Failed to get migration versions", "error", err)
		return fmt.Errorf("failed to get migration versions: %w", err)
	}

	log.Info("Pending migrations detected, creating backup", "currentVersion", current, "targetVersion", target)

	if current > 0 {
		log.Info("Creating backup before migration")
		if _, err := backupDatabase(dbPath, dbName, current, target); err != nil {
			log.Warn("Failed to create backup, continuing anyway", "error", err)
		}
	} else {
		log.Debug("Skipping backup for initial database setup")
	}

	results, err := provider.Up(ctx)
	if err != nil {
		log.Error("Failed to execute migrations", "error", err)
		return fmt.Errorf("failed to execute database migrations: %w", err)
	}

	log.Info("Migrations applied successfully", "from", current, "to", target, "applied", len(results))

	return nil
}

// backupDatabase creates a backup of the database file
func backupDatabase(dbPath, dbName string, currentVersion, targetVersion int64) (string, error) {
	sourceFile := filepath.Join(dbPath, dbName)

	timestamp := time.Now().Format("20060102_150405")
	backupName := fmt.Sprintf("%s.backup_%s_v%d_to_v%d", dbName, timestamp, currentVersion, targetVersion)
	backupFile := filepath.Join(dbPath, backupName)

	log.Info("Creating database backup", "source", sourceFile, "backup", backupFile)

	data, err := os.ReadFile(sourceFile)
	if err != nil {
		log.Error("Failed to read database file for backup", "file", sourceFile, "error", err)
		return "", fmt.Errorf("failed to read database file: %w", err)
	}

	if err := os.WriteFile(backupFile, data, 0644); err != nil {
		log.Error("Failed to write backup file", "file", backupFile, "error", err)
		return "", fmt.Errorf("failed to write backup file: %w", err)
	}

	log.Info("Database backup created successfully", "backup", backupFile)
	return backupFile, nil
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
