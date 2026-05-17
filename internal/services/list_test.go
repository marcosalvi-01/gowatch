package services

import (
	"context"
	"fmt"
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

func TestListService_AddMovieToList_AssignsCustomPosition(t *testing.T) {
	testDB, err := db.NewTestDB()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = testDB.Close() }()

	movieService := NewMovieService(testDB, nil, time.Hour)
	listService := NewListService(testDB, movieService)
	ctx := setupTestUser(t, testDB)

	for i := 1; i <= 2; i++ {
		if err := testDB.UpsertMovie(ctx, &models.MovieDetails{Movie: models.Movie{ID: int64(i), Title: fmt.Sprintf("Movie %d", i)}}); err != nil {
			t.Fatal(err)
		}
	}

	list, err := listService.CreateList(ctx, "Ranked", nil, false)
	if err != nil {
		t.Fatal(err)
	}

	if err := listService.AddMovieToList(ctx, list.ID, 1, nil); err != nil {
		t.Fatal(err)
	}
	if err := listService.AddMovieToList(ctx, list.ID, 2, nil); err != nil {
		t.Fatal(err)
	}

	listDetails, err := listService.GetListDetails(ctx, list.ID)
	if err != nil {
		t.Fatal(err)
	}

	positions := make(map[int64]int64, len(listDetails.Movies))
	for _, movie := range listDetails.Movies {
		if movie.Position == nil {
			t.Fatalf("expected movie %d to have position", movie.MovieDetails.Movie.ID)
		}
		positions[movie.MovieDetails.Movie.ID] = *movie.Position
	}

	if positions[1] != 1 {
		t.Fatalf("expected movie 1 position 1, got %d", positions[1])
	}
	if positions[2] != 2 {
		t.Fatalf("expected movie 2 position 2, got %d", positions[2])
	}
}

func TestListService_GetListViewData_SortsMoviesForDisplay(t *testing.T) {
	testDB, err := db.NewTestDB()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = testDB.Close() }()

	movieService := NewMovieService(testDB, nil, time.Hour)
	listService := NewListService(testDB, movieService)
	ctx := setupTestUser(t, testDB)

	movie1Release := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	movie2Release := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	movie3Release := time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC)

	movies := []*models.MovieDetails{
		{Movie: models.Movie{ID: 1, Title: "Zulu", VoteAverage: 7.2, ReleaseDate: &movie1Release}},
		{Movie: models.Movie{ID: 2, Title: "Alpha", VoteAverage: 8.8, ReleaseDate: &movie2Release}},
		{Movie: models.Movie{ID: 3, Title: "Bravo", VoteAverage: 6.4, ReleaseDate: &movie3Release}},
	}
	for _, movie := range movies {
		if err := testDB.UpsertMovie(ctx, movie); err != nil {
			t.Fatal(err)
		}
	}

	list, err := listService.CreateList(ctx, "Favorites", nil, false)
	if err != nil {
		t.Fatal(err)
	}

	position := int64(1)
	if err := testDB.AddMovieToList(ctx, getTestUserID(t, ctx), db.InsertMovieList{
		MovieID:   1,
		ListID:    list.ID,
		DateAdded: time.Date(2024, 1, 3, 10, 0, 0, 0, time.UTC),
		Position:  &position,
	}); err != nil {
		t.Fatal(err)
	}
	if err := testDB.AddMovieToList(ctx, getTestUserID(t, ctx), db.InsertMovieList{
		MovieID:   2,
		ListID:    list.ID,
		DateAdded: time.Date(2024, 1, 2, 10, 0, 0, 0, time.UTC),
	}); err != nil {
		t.Fatal(err)
	}
	if err := testDB.AddMovieToList(ctx, getTestUserID(t, ctx), db.InsertMovieList{
		MovieID:   3,
		ListID:    list.ID,
		DateAdded: time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
	}); err != nil {
		t.Fatal(err)
	}

	testCases := []struct {
		name     string
		sortMode models.ListMovieSort
		wantIDs  []int64
	}{
		{name: "custom", sortMode: models.ListMovieSortCustom, wantIDs: []int64{1, 3, 2}},
		{name: "recently added", sortMode: models.ListMovieSortDateAddedDesc, wantIDs: []int64{1, 2, 3}},
		{name: "title", sortMode: models.ListMovieSortTitleAsc, wantIDs: []int64{2, 3, 1}},
		{name: "rating", sortMode: models.ListMovieSortRatingDesc, wantIDs: []int64{2, 1, 3}},
		{name: "release date", sortMode: models.ListMovieSortReleaseDateDesc, wantIDs: []int64{1, 2, 3}},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			viewData, err := listService.GetListViewData(ctx, list.ID, string(testCase.sortMode), "", time.Now())
			if err != nil {
				t.Fatal(err)
			}

			gotIDs := make([]int64, len(viewData.List.Movies))
			for i, movie := range viewData.List.Movies {
				gotIDs[i] = movie.MovieDetails.Movie.ID
			}

			if !reflect.DeepEqual(gotIDs, testCase.wantIDs) {
				t.Fatalf("expected ids %v, got %v", testCase.wantIDs, gotIDs)
			}
		})
	}
}

func TestListService_ReorderListMovie_PersistsCustomOrder(t *testing.T) {
	testDB, err := db.NewTestDB()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = testDB.Close() }()

	movieService := NewMovieService(testDB, nil, time.Hour)
	listService := NewListService(testDB, movieService)
	ctx := setupTestUser(t, testDB)
	userID := getTestUserID(t, ctx)

	for i := 1; i <= 3; i++ {
		if err := testDB.UpsertMovie(ctx, &models.MovieDetails{Movie: models.Movie{ID: int64(i), Title: fmt.Sprintf("Movie %d", i)}}); err != nil {
			t.Fatal(err)
		}
	}

	list, err := listService.CreateList(ctx, "Ranked", nil, false)
	if err != nil {
		t.Fatal(err)
	}

	for i := 1; i <= 3; i++ {
		position := int64(i)
		if err := testDB.AddMovieToList(ctx, userID, db.InsertMovieList{
			MovieID:   int64(i),
			ListID:    list.ID,
			DateAdded: time.Date(2024, 1, i, 10, 0, 0, 0, time.UTC),
			Position:  &position,
		}); err != nil {
			t.Fatal(err)
		}
	}

	if err := listService.ReorderListMovie(ctx, list.ID, 2, listMovieMoveUp); err != nil {
		t.Fatal(err)
	}

	viewData, err := listService.GetListViewData(ctx, list.ID, string(models.ListMovieSortCustom), "", time.Now())
	if err != nil {
		t.Fatal(err)
	}

	gotIDs := make([]int64, len(viewData.List.Movies))
	for i, movie := range viewData.List.Movies {
		gotIDs[i] = movie.MovieDetails.Movie.ID
		if movie.Position == nil {
			t.Fatalf("expected movie %d to have position", movie.MovieDetails.Movie.ID)
		}
		if *movie.Position != int64(i+1) {
			t.Fatalf("expected movie %d position %d, got %d", movie.MovieDetails.Movie.ID, i+1, *movie.Position)
		}
	}

	wantIDs := []int64{2, 1, 3}
	if !reflect.DeepEqual(gotIDs, wantIDs) {
		t.Fatalf("expected ids %v, got %v", wantIDs, gotIDs)
	}
}

func TestListService_ReorderListMovie_MovesToFirstAndLast(t *testing.T) {
	testDB, err := db.NewTestDB()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = testDB.Close() }()

	movieService := NewMovieService(testDB, nil, time.Hour)
	listService := NewListService(testDB, movieService)
	ctx := setupTestUser(t, testDB)
	userID := getTestUserID(t, ctx)

	for i := 1; i <= 4; i++ {
		if err := testDB.UpsertMovie(ctx, &models.MovieDetails{Movie: models.Movie{ID: int64(i), Title: fmt.Sprintf("Movie %d", i)}}); err != nil {
			t.Fatal(err)
		}
	}

	list, err := listService.CreateList(ctx, "Ranked", nil, false)
	if err != nil {
		t.Fatal(err)
	}

	for i := 1; i <= 4; i++ {
		position := int64(i)
		if err := testDB.AddMovieToList(ctx, userID, db.InsertMovieList{
			MovieID:   int64(i),
			ListID:    list.ID,
			DateAdded: time.Date(2024, 1, i, 10, 0, 0, 0, time.UTC),
			Position:  &position,
		}); err != nil {
			t.Fatal(err)
		}
	}

	if err := listService.ReorderListMovie(ctx, list.ID, 3, listMovieMoveFirst); err != nil {
		t.Fatal(err)
	}
	if err := listService.ReorderListMovie(ctx, list.ID, 1, listMovieMoveLast); err != nil {
		t.Fatal(err)
	}

	viewData, err := listService.GetListViewData(ctx, list.ID, string(models.ListMovieSortCustom), "", time.Now())
	if err != nil {
		t.Fatal(err)
	}

	gotIDs := make([]int64, len(viewData.List.Movies))
	for i, movie := range viewData.List.Movies {
		gotIDs[i] = movie.MovieDetails.Movie.ID
		if movie.Position == nil {
			t.Fatalf("expected movie %d to have position", movie.MovieDetails.Movie.ID)
		}
		if *movie.Position != int64(i+1) {
			t.Fatalf("expected movie %d position %d, got %d", movie.MovieDetails.Movie.ID, i+1, *movie.Position)
		}
	}

	wantIDs := []int64{3, 2, 4, 1}
	if !reflect.DeepEqual(gotIDs, wantIDs) {
		t.Fatalf("expected ids %v, got %v", wantIDs, gotIDs)
	}
}

func TestIsListMovieOrderEditing(t *testing.T) {
	testCases := []struct {
		name     string
		raw      string
		sortMode models.ListMovieSort
		want     bool
	}{
		{name: "custom edit enabled with one", raw: "1", sortMode: models.ListMovieSortCustom, want: true},
		{name: "custom edit enabled with true", raw: "true", sortMode: models.ListMovieSortCustom, want: true},
		{name: "custom edit disabled", raw: "0", sortMode: models.ListMovieSortCustom, want: false},
		{name: "non custom ignores edit", raw: "1", sortMode: models.ListMovieSortTitleAsc, want: false},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			got := IsListMovieOrderEditing(testCase.raw, testCase.sortMode)
			if got != testCase.want {
				t.Fatalf("expected %t, got %t", testCase.want, got)
			}
		})
	}
}

func getTestUserID(t *testing.T, ctx context.Context) int64 {
	t.Helper()

	user, err := common.GetUser(ctx)
	if err != nil {
		t.Fatal(err)
	}

	return user.ID
}
