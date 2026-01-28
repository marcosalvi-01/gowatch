package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/marcosalvi-01/gowatch/db"
	"github.com/marcosalvi-01/gowatch/internal/common"
	"github.com/marcosalvi-01/gowatch/internal/models"
	"github.com/marcosalvi-01/gowatch/internal/services"
)

func getTestCtx() context.Context {
	return context.WithValue(context.Background(), common.UserKey, &models.User{ID: 1})
}

func setupTestUser(t *testing.T, testDB db.DB) {
	ctx := context.Background()
	_, err := testDB.CreateUser(ctx, "test@example.com", "Test User", "hash")
	if err != nil {
		t.Fatal(err)
	}
}

func TestHandlers_HealthCheck(t *testing.T) {
	testDB, err := db.NewTestDB()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = testDB.Close() }()

	listService := services.NewListService(testDB, nil)
	watchedService := services.NewWatchedService(testDB, listService, nil)
	handlers := NewHandlers(testDB, watchedService, listService)

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
	defer func() { _ = testDB.Close() }()
	setupTestUser(t, testDB)

	movieService := services.NewMovieService(testDB, nil, time.Hour)
	listService := services.NewListService(testDB, movieService)
	watchedService := services.NewWatchedService(testDB, listService, movieService)
	handlers := NewHandlers(testDB, watchedService, listService)

	// Insert movie and watched
	movie := &models.MovieDetails{
		Movie: models.Movie{
			ID:    1,
			Title: "Test Movie",
		},
	}
	ctx := getTestCtx()
	if err := testDB.UpsertMovie(ctx, movie); err != nil {
		t.Fatal(err)
	}
	if err := watchedService.AddWatched(ctx, 1, time.Now(), true, nil); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest("GET", "/movies/export", nil)
	req = req.WithContext(ctx)
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
	defer func() { _ = testDB.Close() }()

	listService := services.NewListService(testDB, nil)
	watchedService := services.NewWatchedService(testDB, listService, nil)
	handlers := NewHandlers(testDB, watchedService, listService)

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
	defer func() { _ = testDB.Close() }()
	setupTestUser(t, testDB)

	movieService := services.NewMovieService(testDB, nil, time.Hour)
	listService := services.NewListService(testDB, movieService)
	watchedService := services.NewWatchedService(testDB, listService, movieService)
	handlers := NewHandlers(testDB, watchedService, listService)

	ctx := getTestCtx()
	req := httptest.NewRequest("GET", "/movies/export", nil)
	req = req.WithContext(ctx)
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
