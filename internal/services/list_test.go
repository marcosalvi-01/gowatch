package services

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/marcosalvi-01/gowatch/db"
	"github.com/marcosalvi-01/gowatch/internal/common"
	"github.com/marcosalvi-01/gowatch/internal/models"
)

func setupTestUser(t *testing.T, testDB db.DB) context.Context {
	ctx := context.Background()
	user, err := testDB.CreateUser(ctx, "test@example.com", "Test User", "hash")
	if err != nil {
		t.Fatal(err)
	}
	return context.WithValue(ctx, common.UserKey, user)
}

func TestListService_CRUD(t *testing.T) {
	testDB, err := db.NewTestDB()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = testDB.Close() }()

	movieService := NewMovieService(testDB, nil, time.Hour)
	listService := NewListService(testDB, movieService)

	ctx := setupTestUser(t, testDB)

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
	if _, err := listService.CreateList(ctx, "Test List", &desc, false); err != nil {
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

	ctx := setupTestUser(t, testDB)

	// Try to create list with empty name
	desc := "desc"
	_, err = listService.CreateList(ctx, "", &desc, false)
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

	ctx := setupTestUser(t, testDB)

	// Try to add movie to non-existent list
	err = listService.AddMovieToList(ctx, 999, 1, nil)
	if err == nil {
		t.Error("expected error for invalid list ID")
	}

	// Create list
	desc := ""
	_, err = listService.CreateList(ctx, "Test", &desc, false)
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

func TestListService_ImportLists_MergesByNameAndPreservesMetadata(t *testing.T) {
	testDB, err := db.NewTestDB()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = testDB.Close() }()

	movieService := NewMovieService(testDB, nil, time.Hour)
	listService := NewListService(testDB, movieService)
	ctx := setupTestUser(t, testDB)

	for i := 1; i <= 2; i++ {
		movie := &models.MovieDetails{
			Movie: models.Movie{
				ID:    int64(i),
				Title: "Test Movie",
			},
		}
		if err := testDB.UpsertMovie(ctx, movie); err != nil {
			t.Fatal(err)
		}
	}

	desc := "Favorites"
	note := "must watch"
	position := int64(1)
	firstDate := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)
	secondDate := time.Date(2024, 1, 2, 12, 0, 0, 0, time.UTC)

	importData := models.ImportListsLog{
		{
			Name:        "Favorites",
			Description: &desc,
			Movies: []models.ImportListMovieRef{
				{MovieID: 1, DateAdded: firstDate, Position: &position, Note: &note},
				{MovieID: 2, DateAdded: secondDate},
			},
		},
	}

	if err := listService.ImportLists(ctx, importData); err != nil {
		t.Fatal(err)
	}

	// Importing the same data twice should merge into the same list without duplicates.
	if err := listService.ImportLists(ctx, importData); err != nil {
		t.Fatal(err)
	}

	lists, err := listService.GetAllLists(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(lists) != 1 {
		t.Fatalf("expected 1 list after re-import, got %d", len(lists))
	}

	listDetails, err := listService.GetListDetails(ctx, lists[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(listDetails.Movies) != 2 {
		t.Fatalf("expected 2 movies in merged list, got %d", len(listDetails.Movies))
	}

	moviesByID := make(map[int64]models.MovieItem, len(listDetails.Movies))
	for _, movie := range listDetails.Movies {
		moviesByID[movie.MovieDetails.Movie.ID] = movie
	}

	firstMovie, ok := moviesByID[1]
	if !ok {
		t.Fatal("expected movie 1 in list")
	}
	if !firstMovie.DateAdded.Equal(firstDate) {
		t.Fatalf("expected movie 1 date_added %s, got %s", firstDate, firstMovie.DateAdded)
	}
	if firstMovie.Position == nil || *firstMovie.Position != position {
		t.Fatalf("expected movie 1 position %d, got %v", position, firstMovie.Position)
	}
	if firstMovie.Note == nil || *firstMovie.Note != note {
		t.Fatalf("expected movie 1 note %q, got %v", note, firstMovie.Note)
	}

	secondMovie, ok := moviesByID[2]
	if !ok {
		t.Fatal("expected movie 2 in list")
	}
	if !secondMovie.DateAdded.Equal(secondDate) {
		t.Fatalf("expected movie 2 date_added %s, got %s", secondDate, secondMovie.DateAdded)
	}
	if secondMovie.Position != nil {
		t.Fatalf("expected movie 2 position to be nil, got %v", *secondMovie.Position)
	}
}

func TestListService_ExportLists_IsStable(t *testing.T) {
	testDB, err := db.NewTestDB()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = testDB.Close() }()

	movieService := NewMovieService(testDB, nil, time.Hour)
	listService := NewListService(testDB, movieService)
	ctx := setupTestUser(t, testDB)

	for i := 1; i <= 2; i++ {
		movie := &models.MovieDetails{
			Movie: models.Movie{
				ID:    int64(i),
				Title: "Test Movie",
			},
		}
		if err := testDB.UpsertMovie(ctx, movie); err != nil {
			t.Fatal(err)
		}
	}

	importData := models.ImportListsLog{
		{
			Name: "List A",
			Movies: []models.ImportListMovieRef{
				{MovieID: 1, DateAdded: time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)},
			},
		},
		{
			Name: "List B",
			Movies: []models.ImportListMovieRef{
				{MovieID: 2, DateAdded: time.Date(2024, 1, 2, 10, 0, 0, 0, time.UTC)},
			},
		},
	}

	if err := listService.ImportLists(ctx, importData); err != nil {
		t.Fatal(err)
	}

	exported1, err := listService.ExportLists(ctx)
	if err != nil {
		t.Fatal(err)
	}
	exported2, err := listService.ExportLists(ctx)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(exported1, exported2) {
		t.Fatal("expected repeated exports to be stable")
	}
}

func TestListService_ImportLists_SkipsMovieFailuresAndContinues(t *testing.T) {
	testDB, err := db.NewTestDB()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = testDB.Close() }()

	movieService := NewMovieService(testDB, nil, time.Hour)
	listService := NewListService(testDB, movieService)
	ctx := setupTestUser(t, testDB)

	validMovie := &models.MovieDetails{
		Movie: models.Movie{
			ID:    1,
			Title: "Valid Movie",
		},
	}
	if err := testDB.UpsertMovie(ctx, validMovie); err != nil {
		t.Fatal(err)
	}

	importData := models.ImportListsLog{
		{
			Name: "Mixed",
			Movies: []models.ImportListMovieRef{
				{MovieID: 1, DateAdded: time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)},
				{MovieID: 999, DateAdded: time.Date(2024, 1, 2, 10, 0, 0, 0, time.UTC)}, // not in DB, should fail and be skipped
			},
		},
	}

	if err := listService.ImportLists(ctx, importData); err != nil {
		t.Fatal(err)
	}

	lists, err := listService.GetAllLists(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(lists) != 1 {
		t.Fatalf("expected 1 list, got %d", len(lists))
	}

	listDetails, err := listService.GetListDetails(ctx, lists[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(listDetails.Movies) != 1 {
		t.Fatalf("expected only valid movie to be imported, got %d movies", len(listDetails.Movies))
	}
	if listDetails.Movies[0].MovieDetails.Movie.ID != 1 {
		t.Fatalf("expected imported movie ID 1, got %d", listDetails.Movies[0].MovieDetails.Movie.ID)
	}
}

func TestListService_ImportLists_SkipsInvalidListAndContinues(t *testing.T) {
	testDB, err := db.NewTestDB()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = testDB.Close() }()

	movieService := NewMovieService(testDB, nil, time.Hour)
	listService := NewListService(testDB, movieService)
	ctx := setupTestUser(t, testDB)

	validMovie := &models.MovieDetails{
		Movie: models.Movie{
			ID:    1,
			Title: "Valid Movie",
		},
	}
	if err := testDB.UpsertMovie(ctx, validMovie); err != nil {
		t.Fatal(err)
	}

	importData := models.ImportListsLog{
		{
			Name: "",
			Movies: []models.ImportListMovieRef{
				{MovieID: 1, DateAdded: time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)},
			},
		},
		{
			Name: "Valid List",
			Movies: []models.ImportListMovieRef{
				{MovieID: 1, DateAdded: time.Date(2024, 1, 2, 10, 0, 0, 0, time.UTC)},
			},
		},
	}

	if err := listService.ImportLists(ctx, importData); err != nil {
		t.Fatal(err)
	}

	lists, err := listService.GetAllLists(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(lists) != 1 {
		t.Fatalf("expected only the valid list to be created, got %d", len(lists))
	}
	if lists[0].Name != "Valid List" {
		t.Fatalf("expected list name 'Valid List', got %q", lists[0].Name)
	}
}
