package api

import (
	"bytes"
	"context"
	"encoding/json"
	"gowatch/db"
	"gowatch/internal/models"
	"gowatch/internal/services"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestHandlers_HealthCheck(t *testing.T) {
	testDB, err := db.NewTestDB()
	if err != nil {
		t.Fatal(err)
	}
	defer testDB.Close()

	watchedService := services.NewWatchedService(testDB, nil)
	handlers := NewHandlers(testDB, watchedService)

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	handlers.healthCheck(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var resp map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp["status"] != "healthy" {
		t.Errorf("expected status healthy, got %s", resp["status"])
	}
}

func TestHandlers_ExportWatched(t *testing.T) {
	testDB, err := db.NewTestDB()
	if err != nil {
		t.Fatal(err)
	}
	defer testDB.Close()

	movieService := services.NewMovieService(testDB, nil, time.Hour)
	watchedService := services.NewWatchedService(testDB, movieService)
	handlers := NewHandlers(testDB, watchedService)

	// Insert movie and watched
	movie := &models.MovieDetails{
		Movie: models.Movie{
			ID:    1,
			Title: "Test Movie",
		},
	}
	ctx := context.Background()
	if err := testDB.UpsertMovie(ctx, movie); err != nil {
		t.Fatal(err)
	}
	if err := watchedService.AddWatched(ctx, 1, time.Now(), true); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest("GET", "/movies/export", nil)
	w := httptest.NewRecorder()

	handlers.exportWatched(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var exported models.ImportWatchedMoviesLog
	if err := json.Unmarshal(w.Body.Bytes(), &exported); err != nil {
		t.Fatal(err)
	}
	if len(exported) != 1 {
		t.Errorf("expected 1 day, got %d", len(exported))
	}
}

func TestHandlers_ImportWatched_InvalidJSON(t *testing.T) {
	testDB, err := db.NewTestDB()
	if err != nil {
		t.Fatal(err)
	}
	defer testDB.Close()

	watchedService := services.NewWatchedService(testDB, nil)
	handlers := NewHandlers(testDB, watchedService)

	req := httptest.NewRequest("POST", "/movies/import", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handlers.importWatched(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestHandlers_ExportWatched_Empty(t *testing.T) {
	testDB, err := db.NewTestDB()
	if err != nil {
		t.Fatal(err)
	}
	defer testDB.Close()

	movieService := services.NewMovieService(testDB, nil, time.Hour)
	watchedService := services.NewWatchedService(testDB, movieService)
	handlers := NewHandlers(testDB, watchedService)

	req := httptest.NewRequest("GET", "/movies/export", nil)
	w := httptest.NewRecorder()

	handlers.exportWatched(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var exported models.ImportWatchedMoviesLog
	if err := json.Unmarshal(w.Body.Bytes(), &exported); err != nil {
		t.Fatal(err)
	}
	if len(exported) != 0 {
		t.Errorf("expected 0 days, got %d", len(exported))
	}
}
