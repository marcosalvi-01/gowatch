package db

import (
	"context"
	"database/sql"
	"io/fs"

	"github.com/marcosalvi-01/gowatch/db/sqlc"

	"github.com/pressly/goose/v3"
	_ "modernc.org/sqlite"
)

// NewTestDB creates an in-memory SQLite database for testing, with migrations applied.
func NewTestDB() (*SqliteDB, error) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, err
	}

	if err := runTestMigrations(db); err != nil {
		_ = db.Close()
		return nil, err
	}

	queries := sqlc.New(db)
	return &SqliteDB{
		queries: queries,
		db:      db,
	}, nil
}

// runTestMigrations runs migrations on the test DB.
func runTestMigrations(db *sql.DB) error {
	// Enable foreign keys
	if _, err := db.Exec("PRAGMA foreign_keys = ON;"); err != nil {
		return err
	}

	migrationsFS, err := fs.Sub(embedMigrations, "migrations")
	if err != nil {
		return err
	}

	provider, err := goose.NewProvider(goose.DialectSQLite3, db, migrationsFS)
	if err != nil {
		return err
	}

	ctx := context.Background()
	_, err = provider.Up(ctx)
	return err
}
