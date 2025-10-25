package services

import (
	"context"
	"reflect"
	"testing"
	"time"

	"gowatch/db"
	"gowatch/internal/models"
)

func TestWatchedService_AddWatched(t *testing.T) {
	testDB, err := db.NewTestDB()
	if err != nil {
		t.Fatal(err)
	}
	defer testDB.Close()

	movieService := NewMovieService(testDB, nil, time.Hour) // No TMDB for test
	watchedService := NewWatchedService(testDB, movieService)

	// Insert a movie
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

	// Add watched
	date := time.Date(2023, 10, 1, 0, 0, 0, 0, time.UTC)
	if err := watchedService.AddWatched(ctx, 1, date, true); err != nil {
		t.Fatal(err)
	}

	// Check count
	count, err := watchedService.GetWatchedCount(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Errorf("expected count 1, got %d", count)
	}
}

func TestWatchedService_ImportExportWatched(t *testing.T) {
	testDB, err := db.NewTestDB()
	if err != nil {
		t.Fatal(err)
	}
	defer testDB.Close()

	movieService := NewMovieService(testDB, nil, time.Hour)
	watchedService := NewWatchedService(testDB, movieService)

	ctx := context.Background()

	// Insert movies
	for i := 1; i <= 2; i++ {
		movie := &models.MovieDetails{
			Movie: models.Movie{
				ID:    int64(i),
				Title: "Test Movie " + string(rune(i+'0')),
			},
		}
		if err := testDB.UpsertMovie(ctx, movie); err != nil {
			t.Fatal(err)
		}
	}

	// Import data
	importData := models.ImportWatchedMoviesLog{
		{
			Date: time.Date(2023, 10, 1, 0, 0, 0, 0, time.UTC),
			Movies: []models.ImportWatchedMovieRef{
				{MovieID: 1, InTheaters: true},
			},
		},
		{
			Date: time.Date(2023, 10, 2, 0, 0, 0, 0, time.UTC),
			Movies: []models.ImportWatchedMovieRef{
				{MovieID: 2, InTheaters: false},
			},
		},
	}
	if err := watchedService.ImportWatched(ctx, importData); err != nil {
		t.Fatal(err)
	}

	// Export
	exported, err := watchedService.ExportWatched(ctx)
	if err != nil {
		t.Fatal(err)
	}

	// Sort exported by date ascending to match import order
	for i := 0; i < len(exported)-1; i++ {
		for j := i + 1; j < len(exported); j++ {
			if exported[i].Date.After(exported[j].Date) {
				exported[i], exported[j] = exported[j], exported[i]
			}
		}
	}

	// Check
	if len(exported) != 2 {
		t.Errorf("expected 2 days, got %d", len(exported))
	}
	if !reflect.DeepEqual(importData, exported) {
		t.Errorf("exported data does not match imported")
	}
}

func TestWatchedService_GetWatchedStats(t *testing.T) {
	testDB, err := db.NewTestDB()
	if err != nil {
		t.Fatal(err)
	}
	defer testDB.Close()

	movieService := NewMovieService(testDB, nil, time.Hour)
	watchedService := NewWatchedService(testDB, movieService)

	ctx := context.Background()

	// Insert movie with genres
	movie := &models.MovieDetails{
		Movie: models.Movie{
			ID:    1,
			Title: "Test Movie",
		},
	}
	if err := testDB.UpsertMovie(ctx, movie); err != nil {
		t.Fatal(err)
	}

	// Add watched
	date := time.Date(2023, 10, 1, 0, 0, 0, 0, time.UTC)
	if err := watchedService.AddWatched(ctx, 1, date, true); err != nil {
		t.Fatal(err)
	}

	// Get stats
	stats, err := watchedService.GetWatchedStats(ctx)
	if err != nil {
		t.Fatal(err)
	}

	if stats.TotalWatched != 1 {
		t.Errorf("expected total 1, got %d", stats.TotalWatched)
	}
	if len(stats.TheaterVsHome) != 1 {
		t.Errorf("expected 1 theater count, got %d", len(stats.TheaterVsHome))
	}
	if stats.TheaterVsHome[0].Count != 1 {
		t.Errorf("expected theater count 1, got %d", stats.TheaterVsHome[0].Count)
	}
}

func TestWatchedService_AddWatched_InvalidMovie(t *testing.T) {
	testDB, err := db.NewTestDB()
	if err != nil {
		t.Fatal(err)
	}
	defer testDB.Close()

	movieService := NewMovieService(testDB, nil, time.Hour)
	watchedService := NewWatchedService(testDB, movieService)

	ctx := context.Background()

	// Try to add watched for non-existent movie
	date := time.Date(2023, 10, 1, 0, 0, 0, 0, time.UTC)
	err = watchedService.AddWatched(ctx, 999, date, true)
	if err == nil {
		t.Error("expected error for invalid movie ID")
	}
}

func TestWatchedService_ImportExport_Empty(t *testing.T) {
	testDB, err := db.NewTestDB()
	if err != nil {
		t.Fatal(err)
	}
	defer testDB.Close()

	movieService := NewMovieService(testDB, nil, time.Hour)
	watchedService := NewWatchedService(testDB, movieService)

	ctx := context.Background()

	// Import empty data
	importData := models.ImportWatchedMoviesLog{}
	if err := watchedService.ImportWatched(ctx, importData); err != nil {
		t.Fatal(err)
	}

	// Export
	exported, err := watchedService.ExportWatched(ctx)
	if err != nil {
		t.Fatal(err)
	}

	if len(exported) != 0 {
		t.Errorf("expected 0 days, got %d", len(exported))
	}
}
