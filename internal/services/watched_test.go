package services

import (
	"context"
	"errors"
	"math"
	"reflect"
	"testing"
	"time"

	"github.com/marcosalvi-01/gowatch/db"
	"github.com/marcosalvi-01/gowatch/internal/common"
	"github.com/marcosalvi-01/gowatch/internal/models"
)

func TestWatchedService_AddWatched(t *testing.T) {
	testDB, err := db.NewTestDB()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = testDB.Close() }()
	ctx := setupTestUser(t, testDB)

	movieService := NewMovieService(testDB, nil, time.Hour) // No TMDB for test
	listService := NewListService(testDB, movieService)
	watchedService := NewWatchedService(testDB, listService, movieService)

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

	// Add watched
	date := time.Date(2023, 10, 1, 0, 0, 0, 0, time.UTC)
	if err := watchedService.AddWatched(ctx, 1, date, true, nil); err != nil {
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

func TestWatchedService_UpdateWatchedEntry(t *testing.T) {
	testDB, err := db.NewTestDB()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = testDB.Close() }()
	ctx := setupTestUser(t, testDB)

	movieService := NewMovieService(testDB, nil, time.Hour)
	listService := NewListService(testDB, movieService)
	watchedService := NewWatchedService(testDB, listService, movieService)

	movie := &models.MovieDetails{
		Movie: models.Movie{
			ID:    1,
			Title: "Test Movie",
		},
	}
	if err := testDB.UpsertMovie(ctx, movie); err != nil {
		t.Fatal(err)
	}

	originalDate := time.Date(2024, 1, 10, 0, 0, 0, 0, time.UTC)
	if err := watchedService.AddWatched(ctx, 1, originalDate, false, nil); err != nil {
		t.Fatal(err)
	}

	records, err := watchedService.GetWatchedMovieRecordsByID(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(records.Records) != 1 {
		t.Fatalf("expected 1 watched record, got %d", len(records.Records))
	}

	updatedDate := time.Date(2024, 1, 12, 0, 0, 0, 0, time.UTC)
	rating := 4.5
	movieID, err := watchedService.UpdateWatchedEntry(ctx, records.Records[0].ID, updatedDate, true, &rating)
	if err != nil {
		t.Fatal(err)
	}
	if movieID != 1 {
		t.Fatalf("expected movie ID 1, got %d", movieID)
	}

	updatedRecords, err := watchedService.GetWatchedMovieRecordsByID(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(updatedRecords.Records) != 1 {
		t.Fatalf("expected 1 watched record after update, got %d", len(updatedRecords.Records))
	}

	updatedRecord := updatedRecords.Records[0]
	if updatedRecord.ID != records.Records[0].ID {
		t.Fatalf("expected watched record ID %d, got %d", records.Records[0].ID, updatedRecord.ID)
	}
	if !updatedRecord.Date.Equal(updatedDate) {
		t.Fatalf("expected watched date %s, got %s", updatedDate, updatedRecord.Date)
	}
	if !updatedRecord.InTheaters {
		t.Fatal("expected watched entry to be marked as in theaters")
	}
	if updatedRecord.Rating == nil || math.Abs(*updatedRecord.Rating-rating) > 0.0001 {
		t.Fatalf("expected rating %.1f, got %+v", rating, updatedRecord.Rating)
	}
}

func TestWatchedService_UpdateWatchedEntry_ZeroRatingClearsRating(t *testing.T) {
	testDB, err := db.NewTestDB()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = testDB.Close() }()
	ctx := setupTestUser(t, testDB)

	movieService := NewMovieService(testDB, nil, time.Hour)
	listService := NewListService(testDB, movieService)
	watchedService := NewWatchedService(testDB, listService, movieService)

	movie := &models.MovieDetails{
		Movie: models.Movie{
			ID:    1,
			Title: "Test Movie",
		},
	}
	if err := testDB.UpsertMovie(ctx, movie); err != nil {
		t.Fatal(err)
	}

	rating := 4.0
	watchedDate := time.Date(2024, 1, 10, 0, 0, 0, 0, time.UTC)
	if err := watchedService.AddWatched(ctx, 1, watchedDate, false, &rating); err != nil {
		t.Fatal(err)
	}

	records, err := watchedService.GetWatchedMovieRecordsByID(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(records.Records) != 1 {
		t.Fatalf("expected 1 watched record, got %d", len(records.Records))
	}

	zeroRating := 0.0
	if _, err := watchedService.UpdateWatchedEntry(ctx, records.Records[0].ID, watchedDate, false, &zeroRating); err != nil {
		t.Fatal(err)
	}

	updatedRecords, err := watchedService.GetWatchedMovieRecordsByID(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}
	if updatedRecords.Records[0].Rating != nil {
		t.Fatalf("expected rating to be cleared, got %+v", updatedRecords.Records[0].Rating)
	}
}

func TestWatchedService_UpdateWatchedEntry_DuplicateDateConflict(t *testing.T) {
	testDB, err := db.NewTestDB()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = testDB.Close() }()
	ctx := setupTestUser(t, testDB)

	movieService := NewMovieService(testDB, nil, time.Hour)
	listService := NewListService(testDB, movieService)
	watchedService := NewWatchedService(testDB, listService, movieService)

	movie := &models.MovieDetails{
		Movie: models.Movie{
			ID:    1,
			Title: "Test Movie",
		},
	}
	if err := testDB.UpsertMovie(ctx, movie); err != nil {
		t.Fatal(err)
	}

	firstDate := time.Date(2024, 1, 10, 0, 0, 0, 0, time.UTC)
	secondDate := time.Date(2024, 1, 12, 0, 0, 0, 0, time.UTC)
	if err := watchedService.AddWatched(ctx, 1, firstDate, false, nil); err != nil {
		t.Fatal(err)
	}
	if err := watchedService.AddWatched(ctx, 1, secondDate, true, nil); err != nil {
		t.Fatal(err)
	}

	records, err := watchedService.GetWatchedMovieRecordsByID(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(records.Records) != 2 {
		t.Fatalf("expected 2 watched records, got %d", len(records.Records))
	}

	var firstRecordID int64
	for _, record := range records.Records {
		if record.Date.Equal(firstDate) {
			firstRecordID = record.ID
			break
		}
	}
	if firstRecordID == 0 {
		t.Fatal("expected to find watched record for first date")
	}

	_, err = watchedService.UpdateWatchedEntry(ctx, firstRecordID, secondDate, false, nil)
	if !errors.Is(err, ErrWatchedEntryConflict) {
		t.Fatalf("expected ErrWatchedEntryConflict, got %v", err)
	}
}

func TestWatchedService_UpdateWatchedEntry_NotFound(t *testing.T) {
	testDB, err := db.NewTestDB()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = testDB.Close() }()
	ctx := setupTestUser(t, testDB)

	movieService := NewMovieService(testDB, nil, time.Hour)
	listService := NewListService(testDB, movieService)
	watchedService := NewWatchedService(testDB, listService, movieService)

	_, err = watchedService.UpdateWatchedEntry(ctx, 999, time.Date(2024, 1, 10, 0, 0, 0, 0, time.UTC), false, nil)
	if !errors.Is(err, ErrWatchedEntryNotFound) {
		t.Fatalf("expected ErrWatchedEntryNotFound, got %v", err)
	}
}

func TestWatchedService_DeleteWatchedEntry(t *testing.T) {
	testDB, err := db.NewTestDB()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = testDB.Close() }()
	ctx := setupTestUser(t, testDB)

	movieService := NewMovieService(testDB, nil, time.Hour)
	listService := NewListService(testDB, movieService)
	watchedService := NewWatchedService(testDB, listService, movieService)

	movie := &models.MovieDetails{
		Movie: models.Movie{
			ID:    1,
			Title: "Test Movie",
		},
	}
	if err := testDB.UpsertMovie(ctx, movie); err != nil {
		t.Fatal(err)
	}

	watchedDate := time.Date(2024, 1, 10, 0, 0, 0, 0, time.UTC)
	if err := watchedService.AddWatched(ctx, 1, watchedDate, false, nil); err != nil {
		t.Fatal(err)
	}

	records, err := watchedService.GetWatchedMovieRecordsByID(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(records.Records) != 1 {
		t.Fatalf("expected 1 watched record, got %d", len(records.Records))
	}

	movieID, err := watchedService.DeleteWatchedEntry(ctx, records.Records[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	if movieID != 1 {
		t.Fatalf("expected movie ID 1, got %d", movieID)
	}

	updatedRecords, err := watchedService.GetWatchedMovieRecordsByID(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(updatedRecords.Records) != 0 {
		t.Fatalf("expected 0 watched records after delete, got %d", len(updatedRecords.Records))
	}

	count, err := watchedService.GetWatchedCount(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("expected watched count 0 after delete, got %d", count)
	}
}

func TestWatchedService_DeleteWatchedEntry_NotFound(t *testing.T) {
	testDB, err := db.NewTestDB()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = testDB.Close() }()
	ctx := setupTestUser(t, testDB)

	movieService := NewMovieService(testDB, nil, time.Hour)
	listService := NewListService(testDB, movieService)
	watchedService := NewWatchedService(testDB, listService, movieService)

	_, err = watchedService.DeleteWatchedEntry(ctx, 999)
	if !errors.Is(err, ErrWatchedEntryNotFound) {
		t.Fatalf("expected ErrWatchedEntryNotFound, got %v", err)
	}
}

func TestWatchedService_DeleteWatchedEntry_NonOwned(t *testing.T) {
	testDB, err := db.NewTestDB()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = testDB.Close() }()
	ctx := setupTestUser(t, testDB)
	otherUser, err := testDB.CreateUser(context.Background(), "other@example.com", "Other User", "hash")
	if err != nil {
		t.Fatal(err)
	}
	otherCtx := context.WithValue(context.Background(), common.UserKey, otherUser)

	movieService := NewMovieService(testDB, nil, time.Hour)
	listService := NewListService(testDB, movieService)
	watchedService := NewWatchedService(testDB, listService, movieService)

	movie := &models.MovieDetails{
		Movie: models.Movie{
			ID:    1,
			Title: "Test Movie",
		},
	}
	if err := testDB.UpsertMovie(ctx, movie); err != nil {
		t.Fatal(err)
	}

	watchedDate := time.Date(2024, 1, 10, 0, 0, 0, 0, time.UTC)
	if err := watchedService.AddWatched(ctx, 1, watchedDate, false, nil); err != nil {
		t.Fatal(err)
	}

	records, err := watchedService.GetWatchedMovieRecordsByID(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(records.Records) != 1 {
		t.Fatalf("expected 1 watched record, got %d", len(records.Records))
	}

	_, err = watchedService.DeleteWatchedEntry(otherCtx, records.Records[0].ID)
	if !errors.Is(err, ErrWatchedEntryNotFound) {
		t.Fatalf("expected ErrWatchedEntryNotFound, got %v", err)
	}
}

func TestWatchedService_ImportExportWatched(t *testing.T) {
	testDB, err := db.NewTestDB()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = testDB.Close() }()
	ctx := setupTestUser(t, testDB)

	movieService := NewMovieService(testDB, nil, time.Hour)
	listService := NewListService(testDB, movieService)
	watchedService := NewWatchedService(testDB, listService, movieService)

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
				{MovieID: 1, InTheaters: true, Rating: nil},
			},
		},
		{
			Date: time.Date(2023, 10, 2, 0, 0, 0, 0, time.UTC),
			Movies: []models.ImportWatchedMovieRef{
				{MovieID: 2, InTheaters: false, Rating: nil},
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
	defer func() { _ = testDB.Close() }()
	ctx := setupTestUser(t, testDB)

	movieService := NewMovieService(testDB, nil, time.Hour)
	listService := NewListService(testDB, movieService)
	watchedService := NewWatchedService(testDB, listService, movieService)

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
	if err := watchedService.AddWatched(ctx, 1, date, true, nil); err != nil {
		t.Fatal(err)
	}

	// Get stats
	stats, err := watchedService.GetWatchedStats(ctx, 5)
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

func TestWatchedService_GetWatchedStats_NewFields(t *testing.T) {
	testDB, err := db.NewTestDB()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = testDB.Close() }()
	ctx := setupTestUser(t, testDB)

	movieService := NewMovieService(testDB, nil, time.Hour)
	listService := NewListService(testDB, movieService)
	watchedService := NewWatchedService(testDB, listService, movieService)

	releaseDate := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)

	movie := &models.MovieDetails{
		Movie: models.Movie{
			ID:               100,
			Title:            "Stats Test Movie",
			OriginalTitle:    "Stats Test Movie",
			OriginalLanguage: "en",
			ReleaseDate:      &releaseDate,
		},
		Budget:  5_000_000,
		Revenue: 20_000_000,
		Runtime: 120,
		Credits: models.MovieCredits{
			Crew: []models.Crew{
				{
					MovieID:    100,
					PersonID:   1,
					CreditID:   "director-credit",
					Job:        "Director",
					Department: "Directing",
					Person: models.Person{
						ID:                 1,
						Name:               "Director Person",
						OriginalName:       "Director Person",
						KnownForDepartment: "Directing",
					},
				},
				{
					MovieID:    100,
					PersonID:   2,
					CreditID:   "writer-credit",
					Job:        "Writer",
					Department: "Writing",
					Person: models.Person{
						ID:                 2,
						Name:               "Writer Person",
						OriginalName:       "Writer Person",
						KnownForDepartment: "Writing",
					},
				},
				{
					MovieID:    100,
					PersonID:   3,
					CreditID:   "composer-credit",
					Job:        "Composer",
					Department: "Sound",
					Person: models.Person{
						ID:                 3,
						Name:               "Composer Person",
						OriginalName:       "Composer Person",
						KnownForDepartment: "Sound",
					},
				},
				{
					MovieID:    100,
					PersonID:   4,
					CreditID:   "camera-credit",
					Job:        "Director of Photography",
					Department: "Camera",
					Person: models.Person{
						ID:                 4,
						Name:               "Camera Person",
						OriginalName:       "Camera Person",
						KnownForDepartment: "Camera",
					},
				},
			},
		},
	}

	if err := testDB.UpsertMovie(ctx, movie); err != nil {
		t.Fatal(err)
	}

	now := time.Now().UTC()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	yesterday := today.AddDate(0, 0, -1)

	if err := watchedService.AddWatched(ctx, 100, yesterday, false, nil); err != nil {
		t.Fatal(err)
	}
	if err := watchedService.AddWatched(ctx, 100, today, true, nil); err != nil {
		t.Fatal(err)
	}

	stats, err := watchedService.GetWatchedStats(ctx, 5)
	if err != nil {
		t.Fatal(err)
	}

	if stats.RewatchStats.RewatchCount != 1 {
		t.Fatalf("expected rewatch count 1, got %d", stats.RewatchStats.RewatchCount)
	}
	if stats.LongestStreak.LongestDays != 2 {
		t.Fatalf("expected longest streak 2, got %d", stats.LongestStreak.LongestDays)
	}
	if len(stats.DailyWatchCountsLastYear) == 0 {
		t.Fatal("expected daily watch counts for heatmap")
	}
	if len(stats.YearlyAllTime) == 0 {
		t.Fatal("expected watches by year data")
	}
	if len(stats.TopDirectors) == 0 {
		t.Fatal("expected top directors data")
	}
	if len(stats.TopWriters) == 0 {
		t.Fatal("expected top writers data")
	}
	if len(stats.TopComposers) == 0 {
		t.Fatal("expected top composers data")
	}
	if len(stats.TopCinematographers) == 0 {
		t.Fatal("expected top cinematographers data")
	}
	if len(stats.TopLanguages) == 0 || stats.TopLanguages[0].Language != "en" {
		t.Fatalf("expected top language 'en', got %+v", stats.TopLanguages)
	}
	if len(stats.ReleaseYearDistribution) == 0 || stats.ReleaseYearDistribution[0].Year != 2020 {
		t.Fatalf("expected release year distribution for 2020, got %+v", stats.ReleaseYearDistribution)
	}
	if stats.LongestMovieWatched == nil || stats.LongestMovieWatched.RuntimeMinutes != 120 {
		t.Fatalf("expected longest movie runtime 120, got %+v", stats.LongestMovieWatched)
	}
	if stats.ShortestMovieWatched == nil || stats.ShortestMovieWatched.RuntimeMinutes != 120 {
		t.Fatalf("expected shortest movie runtime 120, got %+v", stats.ShortestMovieWatched)
	}
	if len(stats.BudgetTierDistribution) == 0 {
		t.Fatal("expected budget tier distribution data")
	}
	if len(stats.TopReturnOnInvestmentMovies) == 0 {
		t.Fatal("expected top ROI movies data")
	}
	if len(stats.BiggestBudgetMovies) == 0 {
		t.Fatal("expected biggest budget movies data")
	}
}

func TestWatchedService_GetWatchedStats_RatingFields(t *testing.T) {
	testDB, err := db.NewTestDB()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = testDB.Close() }()
	ctx := setupTestUser(t, testDB)

	movieService := NewMovieService(testDB, nil, time.Hour)
	listService := NewListService(testDB, movieService)
	watchedService := NewWatchedService(testDB, listService, movieService)

	makeMovie := func(id int64, title string, releaseDate time.Time, voteAverage float32, voteCount int64, directorID int64, directorName string, actorID int64, actorName string) *models.MovieDetails {
		return &models.MovieDetails{
			Movie: models.Movie{
				ID:               id,
				Title:            title,
				OriginalTitle:    title,
				OriginalLanguage: "en",
				ReleaseDate:      &releaseDate,
				VoteAverage:      voteAverage,
				VoteCount:        voteCount,
			},
			Runtime: 100,
			Credits: models.MovieCredits{
				Crew: []models.Crew{{
					MovieID:    id,
					PersonID:   directorID,
					CreditID:   title + "-director",
					Job:        "Director",
					Department: "Directing",
					Person: models.Person{
						ID:                 directorID,
						Name:               directorName,
						OriginalName:       directorName,
						KnownForDepartment: "Directing",
					},
				}},
				Cast: []models.Cast{{
					MovieID:   id,
					PersonID:  actorID,
					CastID:    id * 10,
					CreditID:  title + "-actor",
					Character: "Lead",
					CastOrder: 0,
					Person: models.Person{
						ID:                 actorID,
						Name:               actorName,
						OriginalName:       actorName,
						KnownForDepartment: "Acting",
						Gender:             2,
					},
				}},
			},
		}
	}

	movies := []*models.MovieDetails{
		makeMovie(201, "Alpha", time.Date(1995, 6, 15, 0, 0, 0, 0, time.UTC), 8.0, 500, 1, "Director One", 11, "Favorite Actor"),
		makeMovie(202, "Beta", time.Date(2005, 7, 20, 0, 0, 0, 0, time.UTC), 6.0, 400, 1, "Director One", 11, "Favorite Actor"),
		makeMovie(203, "Gamma", time.Date(2015, 8, 25, 0, 0, 0, 0, time.UTC), 9.0, 1000, 2, "Director Two", 11, "Favorite Actor"),
	}

	for _, movie := range movies {
		if err := testDB.UpsertMovie(ctx, movie); err != nil {
			t.Fatal(err)
		}
	}

	now := time.Now().UTC()
	firstWatch := time.Date(now.Year(), now.Month(), 5, 0, 0, 0, 0, time.UTC).AddDate(0, -2, 0)
	secondWatch := time.Date(now.Year(), now.Month(), 10, 0, 0, 0, 0, time.UTC).AddDate(0, -1, 0)
	thirdWatch := time.Date(now.Year(), now.Month(), 15, 0, 0, 0, 0, time.UTC)

	ratingFour := 4.0
	ratingFourHalf := 4.5
	ratingFive := 5.0
	ratingTwoHalf := 2.5

	entries := []struct {
		movieID     int64
		watchedDate time.Time
		inTheater   bool
		rating      *float64
	}{
		{movieID: 201, watchedDate: firstWatch, inTheater: false, rating: &ratingFour},
		{movieID: 201, watchedDate: secondWatch, inTheater: true, rating: &ratingFourHalf},
		{movieID: 201, watchedDate: thirdWatch, inTheater: false, rating: &ratingFive},
		{movieID: 202, watchedDate: secondWatch.AddDate(0, 0, 2), inTheater: false, rating: &ratingTwoHalf},
		{movieID: 203, watchedDate: firstWatch.AddDate(0, 0, 1), inTheater: true, rating: nil},
		{movieID: 203, watchedDate: thirdWatch.AddDate(0, 0, 1), inTheater: true, rating: &ratingFour},
	}

	for _, entry := range entries {
		if err := watchedService.AddWatched(ctx, entry.movieID, entry.watchedDate, entry.inTheater, entry.rating); err != nil {
			t.Fatal(err)
		}
	}

	stats, err := watchedService.GetWatchedStats(ctx, 5)
	if err != nil {
		t.Fatal(err)
	}

	if stats.Ratings.Summary.RatedCount != 5 {
		t.Fatalf("expected rated count 5, got %d", stats.Ratings.Summary.RatedCount)
	}
	if stats.Ratings.Summary.UnratedCount != 1 {
		t.Fatalf("expected unrated count 1, got %d", stats.Ratings.Summary.UnratedCount)
	}
	if math.Abs(stats.Ratings.Summary.AverageRating-4.0) > 0.0001 {
		t.Fatalf("expected average rating 4.0, got %f", stats.Ratings.Summary.AverageRating)
	}
	if math.Abs(stats.Ratings.Summary.Coverage-(5.0/6.0)) > 0.0001 {
		t.Fatalf("expected coverage close to %f, got %f", 5.0/6.0, stats.Ratings.Summary.Coverage)
	}

	distribution := make(map[float64]int64)
	for _, item := range stats.Ratings.Distribution {
		distribution[item.Rating] = item.Count
	}
	if distribution[2.5] != 1 || distribution[4.0] != 2 || distribution[4.5] != 1 || distribution[5.0] != 1 {
		t.Fatalf("unexpected rating distribution: %+v", distribution)
	}

	if len(stats.Ratings.MonthlyAverage) < 3 {
		t.Fatalf("expected monthly average rating data, got %+v", stats.Ratings.MonthlyAverage)
	}

	var theaterStats models.TheaterRating
	var homeStats models.TheaterRating
	for _, item := range stats.Ratings.TheaterVsHome {
		if item.InTheater {
			theaterStats = item
		} else {
			homeStats = item
		}
	}
	if theaterStats.RatedCount != 2 || math.Abs(theaterStats.AverageRating-4.25) > 0.0001 {
		t.Fatalf("unexpected theater rating stats: %+v", theaterStats)
	}
	if homeStats.RatedCount != 3 || math.Abs(homeStats.AverageRating-3.8333333333) > 0.0001 {
		t.Fatalf("unexpected home rating stats: %+v", homeStats)
	}

	if len(stats.Ratings.HighestRatedMovies) == 0 || stats.Ratings.HighestRatedMovies[0].ID != 201 {
		t.Fatalf("expected Alpha to be highest rated, got %+v", stats.Ratings.HighestRatedMovies)
	}
	if math.Abs(stats.Ratings.HighestRatedMovies[0].AverageRating-4.5) > 0.0001 {
		t.Fatalf("expected Alpha average rating 4.5, got %+v", stats.Ratings.HighestRatedMovies[0])
	}

	if stats.Ratings.VsTMDB.ComparedMovieCount != 3 {
		t.Fatalf("expected TMDB comparison count 3, got %+v", stats.Ratings.VsTMDB)
	}

	if len(stats.Ratings.ReleaseDecades) != 3 {
		t.Fatalf("expected 3 release decade buckets, got %+v", stats.Ratings.ReleaseDecades)
	}

	if len(stats.Ratings.FavoriteDirectors) != 1 || stats.Ratings.FavoriteDirectors[0].Name != "Director One" {
		t.Fatalf("expected Director One as favorite director, got %+v", stats.Ratings.FavoriteDirectors)
	}
	if len(stats.Ratings.FavoriteActors) != 1 || stats.Ratings.FavoriteActors[0].Name != "Favorite Actor" {
		t.Fatalf("expected Favorite Actor as favorite actor, got %+v", stats.Ratings.FavoriteActors)
	}

	if len(stats.Ratings.RewatchDrift) != 1 {
		t.Fatalf("expected one rewatch drift item, got %+v", stats.Ratings.RewatchDrift)
	}
	drift := stats.Ratings.RewatchDrift[0]
	if drift.MovieID != 201 || math.Abs(drift.RatingChange-1.0) > 0.0001 {
		t.Fatalf("expected Alpha drift +1.0, got %+v", drift)
	}
}

func TestWatchedService_GetPersonWatchActivity(t *testing.T) {
	testDB, err := db.NewTestDB()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = testDB.Close() }()
	ctx := setupTestUser(t, testDB)

	movieService := NewMovieService(testDB, nil, time.Hour)
	listService := NewListService(testDB, movieService)
	watchedService := NewWatchedService(testDB, listService, movieService)

	const targetPersonID int64 = 77
	const competingMalePersonID int64 = 88
	const competingFemalePersonID int64 = 99

	movies := []*models.MovieDetails{
		testMovieDetailsWithCredits(
			1,
			"Alpha",
			[]models.Cast{{
				MovieID:   1,
				PersonID:  targetPersonID,
				CastID:    101,
				CreditID:  "alpha-cast",
				Character: "Lead",
				CastOrder: 0,
				Person:    testPerson(targetPersonID, "Target Person", "Acting", 2),
			}},
			[]models.Crew{{
				MovieID:    1,
				PersonID:   targetPersonID,
				CreditID:   "alpha-director",
				Job:        "Director",
				Department: "Directing",
				Person:     testPerson(targetPersonID, "Target Person", "Directing", 2),
			}, {
				MovieID:    1,
				PersonID:   targetPersonID,
				CreditID:   "alpha-producer",
				Job:        "Producer",
				Department: "Production",
				Person:     testPerson(targetPersonID, "Target Person", "Production", 2),
			}},
		),
		testMovieDetailsWithCredits(
			2,
			"Beta",
			nil,
			[]models.Crew{{
				MovieID:    2,
				PersonID:   targetPersonID,
				CreditID:   "beta-writer",
				Job:        "Writer",
				Department: "Writing",
				Person:     testPerson(targetPersonID, "Target Person", "Writing", 2),
			}},
		),
		testMovieDetailsWithCredits(
			3,
			"Gamma",
			[]models.Cast{{
				MovieID:   3,
				PersonID:  competingMalePersonID,
				CastID:    301,
				CreditID:  "gamma-cast",
				Character: "Lead",
				CastOrder: 0,
				Person:    testPerson(competingMalePersonID, "Other Male Actor", "Acting", 2),
			}},
			nil,
		),
		testMovieDetailsWithCredits(
			4,
			"Delta",
			[]models.Cast{{
				MovieID:   4,
				PersonID:  competingFemalePersonID,
				CastID:    401,
				CreditID:  "delta-cast",
				Character: "Lead",
				CastOrder: 0,
				Person:    testPerson(competingFemalePersonID, "Other Female Actor", "Acting", 1),
			}},
			nil,
		),
	}

	for _, movie := range movies {
		if err := testDB.UpsertMovie(ctx, movie); err != nil {
			t.Fatal(err)
		}
	}

	entries := []struct {
		movieID int64
		date    time.Time
	}{
		{movieID: 1, date: time.Date(2024, 1, 10, 0, 0, 0, 0, time.UTC)},
		{movieID: 1, date: time.Date(2024, 2, 15, 0, 0, 0, 0, time.UTC)},
		{movieID: 2, date: time.Date(2024, 3, 20, 0, 0, 0, 0, time.UTC)},
		{movieID: 3, date: time.Date(2024, 4, 5, 0, 0, 0, 0, time.UTC)},
		{movieID: 3, date: time.Date(2024, 4, 6, 0, 0, 0, 0, time.UTC)},
		{movieID: 3, date: time.Date(2024, 4, 7, 0, 0, 0, 0, time.UTC)},
		{movieID: 4, date: time.Date(2024, 4, 8, 0, 0, 0, 0, time.UTC)},
		{movieID: 4, date: time.Date(2024, 4, 9, 0, 0, 0, 0, time.UTC)},
		{movieID: 4, date: time.Date(2024, 4, 10, 0, 0, 0, 0, time.UTC)},
		{movieID: 4, date: time.Date(2024, 4, 11, 0, 0, 0, 0, time.UTC)},
	}

	for _, entry := range entries {
		if err := watchedService.AddWatched(ctx, entry.movieID, entry.date, false, nil); err != nil {
			t.Fatal(err)
		}
	}

	activity, err := watchedService.GetPersonWatchActivity(ctx, targetPersonID)
	if err != nil {
		t.Fatal(err)
	}

	if activity.TotalWatchCount != 3 {
		t.Fatalf("expected total watch count 3, got %d", activity.TotalWatchCount)
	}
	if activity.ActingMovieCount != 1 {
		t.Fatalf("expected acting movie count 1, got %d", activity.ActingMovieCount)
	}
	if activity.CrewMovieCount != 2 {
		t.Fatalf("expected crew movie count 2, got %d", activity.CrewMovieCount)
	}
	if activity.ActorRank == nil {
		t.Fatal("expected actor rank to be set")
	}
	if *activity.ActorRank != 2 {
		t.Fatalf("expected actor rank 2, got %d", *activity.ActorRank)
	}
	if len(activity.Movies) != 2 {
		t.Fatalf("expected 2 watched movies, got %d", len(activity.Movies))
	}

	firstMovie := activity.Movies[0]
	if firstMovie.ID != 1 {
		t.Fatalf("expected first movie ID 1, got %d", firstMovie.ID)
	}
	if firstMovie.WatchCount != 2 {
		t.Fatalf("expected first movie watch count 2, got %d", firstMovie.WatchCount)
	}
	if !firstMovie.LastWatchedDate.Equal(time.Date(2024, 2, 15, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("unexpected first movie last watched date: %s", firstMovie.LastWatchedDate)
	}
	if len(firstMovie.Roles) != 3 {
		t.Fatalf("expected 3 roles for first movie, got %d", len(firstMovie.Roles))
	}
	if firstMovie.Roles[0].Kind != models.PersonWatchRoleKindActing || firstMovie.Roles[0].Label != "Lead" {
		t.Fatalf("unexpected first role: %+v", firstMovie.Roles[0])
	}
	if firstMovie.Roles[1].Kind != models.PersonWatchRoleKindCrew || firstMovie.Roles[1].Label != "Director" {
		t.Fatalf("unexpected second role: %+v", firstMovie.Roles[1])
	}
	if firstMovie.Roles[2].Kind != models.PersonWatchRoleKindCrew || firstMovie.Roles[2].Label != "Producer" {
		t.Fatalf("unexpected third role: %+v", firstMovie.Roles[2])
	}

	secondMovie := activity.Movies[1]
	if secondMovie.ID != 2 {
		t.Fatalf("expected second movie ID 2, got %d", secondMovie.ID)
	}
	if secondMovie.WatchCount != 1 {
		t.Fatalf("expected second movie watch count 1, got %d", secondMovie.WatchCount)
	}
	if len(secondMovie.Roles) != 1 {
		t.Fatalf("expected 1 role for second movie, got %d", len(secondMovie.Roles))
	}
	if secondMovie.Roles[0].Kind != models.PersonWatchRoleKindCrew || secondMovie.Roles[0].Label != "Writer" {
		t.Fatalf("unexpected second movie role: %+v", secondMovie.Roles[0])
	}
}

func TestWatchedService_GetPersonWatchActivity_CrewOnlyActorRankNil(t *testing.T) {
	testDB, err := db.NewTestDB()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = testDB.Close() }()
	ctx := setupTestUser(t, testDB)

	movieService := NewMovieService(testDB, nil, time.Hour)
	listService := NewListService(testDB, movieService)
	watchedService := NewWatchedService(testDB, listService, movieService)

	const targetPersonID int64 = 55

	movie := testMovieDetailsWithCredits(
		1,
		"Crew Movie",
		nil,
		[]models.Crew{{
			MovieID:    1,
			PersonID:   targetPersonID,
			CreditID:   "crew-director",
			Job:        "Director",
			Department: "Directing",
			Person:     testPerson(targetPersonID, "Crew Person", "Directing", 2),
		}},
	)

	if err := testDB.UpsertMovie(ctx, movie); err != nil {
		t.Fatal(err)
	}
	if err := watchedService.AddWatched(ctx, 1, time.Date(2024, 1, 10, 0, 0, 0, 0, time.UTC), false, nil); err != nil {
		t.Fatal(err)
	}

	activity, err := watchedService.GetPersonWatchActivity(ctx, targetPersonID)
	if err != nil {
		t.Fatal(err)
	}

	if activity.TotalWatchCount != 1 {
		t.Fatalf("expected total watch count 1, got %d", activity.TotalWatchCount)
	}
	if activity.ActingMovieCount != 0 {
		t.Fatalf("expected acting movie count 0, got %d", activity.ActingMovieCount)
	}
	if activity.CrewMovieCount != 1 {
		t.Fatalf("expected crew movie count 1, got %d", activity.CrewMovieCount)
	}
	if activity.ActorRank != nil {
		t.Fatalf("expected actor rank to be nil, got %d", *activity.ActorRank)
	}
	if len(activity.Movies) != 1 {
		t.Fatalf("expected 1 watched movie, got %d", len(activity.Movies))
	}
}

func TestWatchedService_GetPersonWatchActivity_NoMatches(t *testing.T) {
	testDB, err := db.NewTestDB()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = testDB.Close() }()
	ctx := setupTestUser(t, testDB)

	movieService := NewMovieService(testDB, nil, time.Hour)
	listService := NewListService(testDB, movieService)
	watchedService := NewWatchedService(testDB, listService, movieService)

	movie := testMovieDetailsWithCredits(
		1,
		"Alpha",
		[]models.Cast{{
			MovieID:   1,
			PersonID:  88,
			CastID:    101,
			CreditID:  "alpha-cast",
			Character: "Lead",
			CastOrder: 0,
			Person:    testPerson(88, "Other Person", "Acting", 2),
		}},
		nil,
	)

	if err := testDB.UpsertMovie(ctx, movie); err != nil {
		t.Fatal(err)
	}
	if err := watchedService.AddWatched(ctx, 1, time.Date(2024, 1, 10, 0, 0, 0, 0, time.UTC), false, nil); err != nil {
		t.Fatal(err)
	}

	activity, err := watchedService.GetPersonWatchActivity(ctx, 99)
	if err != nil {
		t.Fatal(err)
	}

	if activity.TotalWatchCount != 0 {
		t.Fatalf("expected total watch count 0, got %d", activity.TotalWatchCount)
	}
	if activity.ActingMovieCount != 0 {
		t.Fatalf("expected acting movie count 0, got %d", activity.ActingMovieCount)
	}
	if activity.CrewMovieCount != 0 {
		t.Fatalf("expected crew movie count 0, got %d", activity.CrewMovieCount)
	}
	if activity.ActorRank != nil {
		t.Fatalf("expected actor rank to be nil, got %d", *activity.ActorRank)
	}
	if len(activity.Movies) != 0 {
		t.Fatalf("expected 0 watched movies, got %d", len(activity.Movies))
	}
}

func TestWatchedService_AddWatched_InvalidMovie(t *testing.T) {
	testDB, err := db.NewTestDB()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = testDB.Close() }()
	ctx := setupTestUser(t, testDB)

	movieService := NewMovieService(testDB, nil, time.Hour)
	listService := NewListService(testDB, movieService)
	watchedService := NewWatchedService(testDB, listService, movieService)

	// Try to add watched for non-existent movie
	date := time.Date(2023, 10, 1, 0, 0, 0, 0, time.UTC)
	err = watchedService.AddWatched(ctx, 999, date, true, nil)
	if err == nil {
		t.Error("expected error for invalid movie ID")
	}
}

func TestWatchedService_ImportExport_Empty(t *testing.T) {
	testDB, err := db.NewTestDB()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = testDB.Close() }()
	ctx := setupTestUser(t, testDB)

	movieService := NewMovieService(testDB, nil, time.Hour)
	listService := NewListService(testDB, movieService)
	watchedService := NewWatchedService(testDB, listService, movieService)

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

func TestWatchedService_ImportWatched_ContinuesOnPerMovieErrors(t *testing.T) {
	testDB, err := db.NewTestDB()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = testDB.Close() }()
	ctx := setupTestUser(t, testDB)

	movieService := NewMovieService(testDB, nil, time.Hour)
	listService := NewListService(testDB, movieService)
	watchedService := NewWatchedService(testDB, listService, movieService)

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

	watchedDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	importData := models.ImportWatchedMoviesLog{
		{
			Date: watchedDate,
			Movies: []models.ImportWatchedMovieRef{
				{MovieID: 1, InTheaters: true},
				{MovieID: 1, InTheaters: false}, // duplicate day+movie, should fail and be skipped
				{MovieID: 2, InTheaters: false},
			},
		},
	}

	if err := watchedService.ImportWatched(ctx, importData); err != nil {
		t.Fatal(err)
	}

	count, err := watchedService.GetWatchedCount(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if count != 2 {
		t.Fatalf("expected watched count 2 after skipping duplicate, got %d", count)
	}

	movie1Records, err := watchedService.GetWatchedMovieRecordsByID(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(movie1Records.Records) != 1 {
		t.Fatalf("expected 1 watched record for movie 1, got %d", len(movie1Records.Records))
	}

	movie2Records, err := watchedService.GetWatchedMovieRecordsByID(ctx, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(movie2Records.Records) != 1 {
		t.Fatalf("expected 1 watched record for movie 2, got %d", len(movie2Records.Records))
	}
}

func TestWatchedService_ImportAll_ContinuesWithListsWhenWatchedHasErrors(t *testing.T) {
	testDB, err := db.NewTestDB()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = testDB.Close() }()
	ctx := setupTestUser(t, testDB)

	movieService := NewMovieService(testDB, nil, time.Hour)
	listService := NewListService(testDB, movieService)
	watchedService := NewWatchedService(testDB, listService, movieService)

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

	watchlistMovieDate := time.Date(2024, 1, 2, 15, 0, 0, 0, time.UTC)
	importData := models.ImportAllData{
		Watched: models.ImportWatchedMoviesLog{
			{
				Date: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
				Movies: []models.ImportWatchedMovieRef{
					{MovieID: 1, InTheaters: true},
					{MovieID: 1, InTheaters: false}, // duplicate day+movie, should fail and be skipped
				},
			},
		},
		Lists: models.ImportListsLog{
			{
				Name:        "Watchlist",
				IsWatchlist: true,
				Movies: []models.ImportListMovieRef{
					{MovieID: 2, DateAdded: watchlistMovieDate},
				},
			},
		},
	}

	if err := watchedService.ImportAll(ctx, importData); err != nil {
		t.Fatal(err)
	}

	count, err := watchedService.GetWatchedCount(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("expected watched count 1 after skipping duplicate, got %d", count)
	}

	watchlist, err := listService.GetWatchlist(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(watchlist.Movies) != 1 {
		t.Fatalf("expected 1 movie in imported watchlist, got %d", len(watchlist.Movies))
	}
	if watchlist.Movies[0].MovieDetails.Movie.ID != 2 {
		t.Fatalf("expected watchlist movie ID 2, got %d", watchlist.Movies[0].MovieDetails.Movie.ID)
	}
}

func TestWatchedService_ImportAll(t *testing.T) {
	testDB, err := db.NewTestDB()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = testDB.Close() }()
	ctx := setupTestUser(t, testDB)

	movieService := NewMovieService(testDB, nil, time.Hour)
	listService := NewListService(testDB, movieService)
	watchedService := NewWatchedService(testDB, listService, movieService)

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

	note := "great"
	position := int64(3)
	listMovieDate := time.Date(2024, 1, 2, 15, 0, 0, 0, time.UTC)

	importData := models.ImportAllData{
		Watched: models.ImportWatchedMoviesLog{
			{
				Date: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
				Movies: []models.ImportWatchedMovieRef{
					{MovieID: 1, InTheaters: true},
				},
			},
		},
		Lists: models.ImportListsLog{
			{
				Name: "Favorites",
				Movies: []models.ImportListMovieRef{
					{MovieID: 2, DateAdded: listMovieDate, Position: &position, Note: &note},
				},
			},
		},
	}

	if err := watchedService.ImportAll(ctx, importData); err != nil {
		t.Fatal(err)
	}

	count, err := watchedService.GetWatchedCount(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("expected watched count 1, got %d", count)
	}

	lists, err := listService.GetAllLists(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(lists) != 1 {
		t.Fatalf("expected 1 custom list, got %d", len(lists))
	}

	details, err := listService.GetListDetails(ctx, lists[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(details.Movies) != 1 {
		t.Fatalf("expected 1 movie in list, got %d", len(details.Movies))
	}

	movie := details.Movies[0]
	if movie.MovieDetails.Movie.ID != 2 {
		t.Fatalf("expected list movie ID 2, got %d", movie.MovieDetails.Movie.ID)
	}
	if movie.Position == nil || *movie.Position != position {
		t.Fatalf("expected imported position %d, got %v", position, movie.Position)
	}
	if movie.Note == nil || *movie.Note != note {
		t.Fatalf("expected imported note %q, got %v", note, movie.Note)
	}
	if !movie.DateAdded.Equal(listMovieDate) {
		t.Fatalf("expected imported date_added %s, got %s", listMovieDate, movie.DateAdded)
	}
}

func testMovieDetailsWithCredits(movieID int64, title string, cast []models.Cast, crew []models.Crew) *models.MovieDetails {
	return &models.MovieDetails{
		Movie: models.Movie{
			ID:               movieID,
			Title:            title,
			OriginalTitle:    title,
			OriginalLanguage: "en",
		},
		Credits: models.MovieCredits{
			Cast: cast,
			Crew: crew,
		},
	}
}

func testPerson(personID int64, name, department string, gender int64) models.Person {
	return models.Person{
		ID:                 personID,
		Name:               name,
		OriginalName:       name,
		KnownForDepartment: department,
		Gender:             gender,
	}
}
