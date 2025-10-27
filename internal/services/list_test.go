package services

import (
	"context"
	"testing"
	"time"

	"gowatch/db"
	"gowatch/internal/models"
)

func TestListService_CRUD(t *testing.T) {
	testDB, err := db.NewTestDB()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = testDB.Close() }()

	movieService := NewMovieService(testDB, nil, time.Hour)
	listService := NewListService(testDB, movieService)

	ctx := context.Background()

	// Insert a movie
	movie := &models.MovieDetails{
		Movie: models.Movie{
			ID:    1,
			Title: "Test Movie",
		},
	}
	if err := testDB.UpsertMovie(ctx, movie); err != nil {
		t.Fatal(err)
	}

	// Create list
	desc := "A test list"
	if _, err := listService.CreateList(ctx, "Test List", &desc); err != nil {
		t.Fatal(err)
	}

	// Get all lists
	lists, err := listService.GetAllLists(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(lists) != 1 {
		t.Errorf("expected 1 list, got %d", len(lists))
	}
	listID := lists[0].ID

	// Add movie to list
	if err := listService.AddMovieToList(ctx, listID, 1, nil); err != nil {
		t.Fatal(err)
	}

	// Get list details
	details, err := listService.GetListDetails(ctx, listID)
	if err != nil {
		t.Fatal(err)
	}
	if details.Name != "Test List" {
		t.Errorf("expected name 'Test List', got %s", details.Name)
	}
	if len(details.Movies) != 1 {
		t.Errorf("expected 1 movie, got %d", len(details.Movies))
	}

	// Delete list
	if err := listService.DeleteList(ctx, listID); err != nil {
		t.Fatal(err)
	}

	// Check lists empty
	lists, err = listService.GetAllLists(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(lists) != 0 {
		t.Errorf("expected 0 lists, got %d", len(lists))
	}
}

func TestListService_CreateList_EmptyName(t *testing.T) {
	testDB, err := db.NewTestDB()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = testDB.Close() }()

	movieService := NewMovieService(testDB, nil, time.Hour)
	listService := NewListService(testDB, movieService)

	ctx := context.Background()

	// Try to create list with empty name
	desc := "desc"
	_, err = listService.CreateList(ctx, "", &desc)
	if err == nil {
		t.Error("expected error for empty list name")
	}
}

func TestListService_AddMovieToList_InvalidIDs(t *testing.T) {
	testDB, err := db.NewTestDB()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = testDB.Close() }()

	movieService := NewMovieService(testDB, nil, time.Hour)
	listService := NewListService(testDB, movieService)

	ctx := context.Background()

	// Try to add movie to non-existent list
	err = listService.AddMovieToList(ctx, 999, 1, nil)
	if err == nil {
		t.Error("expected error for invalid list ID")
	}

	// Create list
	desc := ""
	_, err = listService.CreateList(ctx, "Test", &desc)
	if err != nil {
		t.Fatal(err)
	}
	lists, _ := listService.GetAllLists(ctx)
	listID := lists[0].ID

	// Try to add non-existent movie
	err = listService.AddMovieToList(ctx, listID, 999, nil)
	if err == nil {
		t.Error("expected error for invalid movie ID")
	}
}
