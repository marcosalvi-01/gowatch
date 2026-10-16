package db

import (
	"database/sql"
	"io/fs"
	"testing"

	"github.com/pressly/goose/v3"
)

func TestRemoveUnusedPersonColumns(t *testing.T) {
	for _, populated := range []bool{false, true} {
		name := "empty database"
		if populated {
			name = "populated database"
		}
		t.Run(name, func(t *testing.T) {
			db, provider := testMigrationProvider(t)
			if _, err := provider.UpTo(t.Context(), 9); err != nil {
				t.Fatalf("apply migrations through version 9: %v", err)
			}

			if populated {
				insertMigrationTestData(t, db)
			}

			if _, err := provider.Up(t.Context()); err != nil {
				t.Fatalf("apply person column migration: %v", err)
			}

			assertPersonSchema(t, db)
			if populated {
				assertMigrationTestDataPreserved(t, db)
			}
		})
	}
}

func testMigrationProvider(t *testing.T) (*sql.DB, *goose.Provider) {
	t.Helper()

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open migration test database: %v", err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Errorf("close migration test database: %v", err)
		}
	})
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		t.Fatalf("enable foreign keys: %v", err)
	}

	migrations, err := fs.Sub(embedMigrations, "migrations")
	if err != nil {
		t.Fatalf("open embedded migrations: %v", err)
	}
	provider, err := goose.NewProvider(goose.DialectSQLite3, db, migrations)
	if err != nil {
		t.Fatalf("create migration provider: %v", err)
	}
	return db, provider
}

func insertMigrationTestData(t *testing.T, db *sql.DB) {
	t.Helper()

	statements := []string{
		`INSERT INTO movie (id, title, original_title, original_language, overview, poster_path, backdrop_path, popularity, vote_count, vote_average, budget, homepage, imdb_id, revenue, runtime, status, tagline, updated_at)
		 VALUES (603, 'The Matrix', 'The Matrix', 'en', 'Overview', '/poster.jpg', '/backdrop.jpg', 42.5, 100, 8.2, 1, 'https://example.com', 'tt0133093', 2, 136, 'Released', 'Tagline', CURRENT_TIMESTAMP)`,
		`INSERT INTO person (id, name, original_name, profile_path, known_for_department, popularity, gender, adult, updated_at)
		 VALUES (1, 'Keanu Reeves', 'Keanu Reeves', '/keanu.jpg', 'Acting', 25.5, 2, FALSE, CURRENT_TIMESTAMP)`,
		`INSERT INTO cast (movie_id, person_id, cast_id, credit_id, character, cast_order, updated_at)
		 VALUES (603, 1, 7, 'cast-credit', 'Neo', 0, CURRENT_TIMESTAMP)`,
		`INSERT INTO crew (movie_id, person_id, credit_id, job, department, updated_at)
		 VALUES (603, 1, 'crew-credit', 'Producer', 'Production', CURRENT_TIMESTAMP)`,
		`INSERT INTO watched (movie_id, user_id, watched_date, watched_in_theater, rating)
		 VALUES (603, NULL, '2025-01-01', TRUE, 4.5)`,
		`INSERT INTO list (name, creation_date, description, user_id, is_watchlist, display_sort)
		 VALUES ('Keepers', '2025-01-01', 'Notes', NULL, FALSE, 'custom')`,
		`INSERT INTO list_movie (movie_id, list_id, date_added, position, note)
		 VALUES (603, 1, '2025-01-01', 1, 'Keep')`,
	}
	for _, statement := range statements {
		if _, err := db.Exec(statement); err != nil {
			t.Fatalf("insert migration test data: %v", err)
		}
	}
}

func assertPersonSchema(t *testing.T, db *sql.DB) {
	t.Helper()

	rows, err := db.Query("PRAGMA table_info(person)")
	if err != nil {
		t.Fatalf("read person schema: %v", err)
	}
	defer func() { _ = rows.Close() }()

	columns := make(map[string]bool)
	for rows.Next() {
		var cid, notNull, primaryKey int
		var name, dataType string
		var defaultValue any
		if err := rows.Scan(&cid, &name, &dataType, &notNull, &defaultValue, &primaryKey); err != nil {
			t.Fatalf("scan person schema: %v", err)
		}
		columns[name] = true
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate person schema: %v", err)
	}

	for _, column := range []string{"original_name", "known_for_department", "popularity", "adult"} {
		if columns[column] {
			t.Errorf("person schema still has dropped column %q", column)
		}
	}
	for _, column := range []string{"id", "name", "profile_path", "gender", "updated_at"} {
		if !columns[column] {
			t.Errorf("person schema lost retained column %q", column)
		}
	}
	if len(columns) != 5 {
		t.Errorf("person schema has %d columns, want exactly the 5 retained columns", len(columns))
	}
}

func assertMigrationTestDataPreserved(t *testing.T, db *sql.DB) {
	t.Helper()

	checks := []struct {
		name  string
		query string
		want  int
	}{
		{name: "movie", query: `SELECT COUNT(*) FROM movie WHERE id = 603 AND title = 'The Matrix'`, want: 1},
		{name: "person", query: `SELECT COUNT(*) FROM person WHERE id = 1 AND name = 'Keanu Reeves' AND profile_path = '/keanu.jpg' AND gender = 2`, want: 1},
		{name: "cast relationship", query: `SELECT COUNT(*) FROM cast WHERE movie_id = 603 AND person_id = 1 AND character = 'Neo'`, want: 1},
		{name: "crew relationship", query: `SELECT COUNT(*) FROM crew WHERE movie_id = 603 AND person_id = 1 AND job = 'Producer'`, want: 1},
		{name: "watch history", query: `SELECT COUNT(*) FROM watched WHERE movie_id = 603 AND rating = 4.5`, want: 1},
		{name: "list", query: `SELECT COUNT(*) FROM list WHERE name = 'Keepers'`, want: 1},
		{name: "list movie relationship", query: `SELECT COUNT(*) FROM list_movie WHERE movie_id = 603 AND list_id = 1`, want: 1},
	}
	for _, check := range checks {
		var got int
		if err := db.QueryRow(check.query).Scan(&got); err != nil {
			t.Errorf("query preserved %s: %v", check.name, err)
			continue
		}
		if got != check.want {
			t.Errorf("preserved %s count = %d, want %d", check.name, got, check.want)
		}
	}
}
