package services

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"
	"time"

	tmdb "github.com/cyruzin/golang-tmdb"
	"github.com/marcosalvi-01/gowatch/db"
	"github.com/marcosalvi-01/gowatch/internal/models"
)

func TestMovieService_GetMovieDetails_CacheHit(t *testing.T) {
	testDB, err := db.NewTestDB()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = testDB.Close() }()

	movieService := NewMovieService(testDB, nil, time.Hour)

	ctx := context.Background()

	// Insert movie with recent updatedAt
	now := time.Now()
	movie := &models.MovieDetails{
		Movie: models.Movie{
			ID:        1,
			Title:     "Test Movie",
			UpdatedAt: now,
		},
	}
	if err := testDB.UpsertMovie(ctx, movie); err != nil {
		t.Fatal(err)
	}

	// Get details, should hit cache
	details, err := movieService.GetMovieDetails(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}
	if details.Movie.Title != "Test Movie" {
		t.Errorf("expected title 'Test Movie', got %s", details.Movie.Title)
	}
}

func TestMapTMDBSearchMultiMovieToSearchResult(t *testing.T) {
	tests := []struct {
		name          string
		id            int64
		title         string
		originalTitle string
		posterPath    string
		releaseDate   string
		voteAverage   float32
		expected      models.SearchResult
	}{
		{
			name:          "uses movie title and release year",
			id:            1,
			title:         "Inception",
			originalTitle: "Inception",
			posterPath:    "/inception.jpg",
			releaseDate:   "2010-07-16",
			voteAverage:   8.7,
			expected: models.SearchResult{
				ID:            1,
				Title:         "Inception",
				ImagePath:     "/inception.jpg",
				SecondaryText: "2010",
				VoteAverage:   8.7,
				Type:          models.SearchResultTypeMovie,
			},
		},
		{
			name:          "falls back to original title and unknown year",
			id:            2,
			title:         "",
			originalTitle: "Original Name",
			posterPath:    "",
			releaseDate:   "",
			voteAverage:   0,
			expected: models.SearchResult{
				ID:            2,
				Title:         "Original Name",
				ImagePath:     "",
				SecondaryText: unknownYearSecondaryText,
				VoteAverage:   0,
				Type:          models.SearchResultTypeMovie,
			},
		},
		{
			name:          "uses fallback movie title when both titles are empty",
			id:            3,
			title:         "",
			originalTitle: "",
			posterPath:    "/fallback.jpg",
			releaseDate:   "2024",
			voteAverage:   6.1,
			expected: models.SearchResult{
				ID:            3,
				Title:         untitledMovieTitle,
				ImagePath:     "/fallback.jpg",
				SecondaryText: "2024",
				VoteAverage:   6.1,
				Type:          models.SearchResultTypeMovie,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mapTMDBSearchMultiMovieToSearchResult(
				tt.id,
				tt.title,
				tt.originalTitle,
				tt.posterPath,
				tt.releaseDate,
				tt.voteAverage,
			)

			if !reflect.DeepEqual(got, tt.expected) {
				t.Fatalf("unexpected mapped movie result\nexpected: %+v\ngot:      %+v", tt.expected, got)
			}
		})
	}
}

func TestMapTMDBSearchMultiPersonToSearchResult(t *testing.T) {
	tests := []struct {
		name         string
		id           int64
		personName   string
		originalName string
		profilePath  string
		popularity   float32
		expected     models.SearchResult
	}{
		{
			name:         "uses name and popularity",
			id:           10,
			personName:   "Christopher Nolan",
			originalName: "Christopher Nolan",
			profilePath:  "/nolan.jpg",
			popularity:   34.7,
			expected: models.SearchResult{
				ID:            10,
				Title:         "Christopher Nolan",
				ImagePath:     "/nolan.jpg",
				SecondaryText: "Popularity 34.7",
				Type:          models.SearchResultTypePerson,
			},
		},
		{
			name:         "falls back to original name and unknown popularity",
			id:           11,
			personName:   "",
			originalName: "Original Person",
			profilePath:  "",
			popularity:   0,
			expected: models.SearchResult{
				ID:            11,
				Title:         "Original Person",
				ImagePath:     "",
				SecondaryText: unknownPopularityText,
				Type:          models.SearchResultTypePerson,
			},
		},
		{
			name:         "uses fallback person name when names are empty",
			id:           12,
			personName:   "",
			originalName: "",
			profilePath:  "/unknown.jpg",
			popularity:   -1,
			expected: models.SearchResult{
				ID:            12,
				Title:         unknownPersonName,
				ImagePath:     "/unknown.jpg",
				SecondaryText: unknownPopularityText,
				Type:          models.SearchResultTypePerson,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mapTMDBSearchMultiPersonToSearchResult(
				tt.id,
				tt.personName,
				tt.originalName,
				tt.profilePath,
				tt.popularity,
			)

			if !reflect.DeepEqual(got, tt.expected) {
				t.Fatalf("unexpected mapped person result\nexpected: %+v\ngot:      %+v", tt.expected, got)
			}
		})
	}
}

func TestFormatPersonPopularity(t *testing.T) {
	tests := []struct {
		name       string
		popularity float32
		expected   string
	}{
		{
			name:       "valid popularity",
			popularity: 57.23,
			expected:   "Popularity 57.2",
		},
		{
			name:       "zero popularity",
			popularity: 0,
			expected:   unknownPopularityText,
		},
		{
			name:       "negative popularity",
			popularity: -3,
			expected:   unknownPopularityText,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatPersonPopularity(tt.popularity)
			if got != tt.expected {
				t.Fatalf("expected %q, got %q", tt.expected, got)
			}
		})
	}
}

func TestMapTMDBPersonDetailsToPersonDetailsPage_InvalidDatesDoNotFail(t *testing.T) {
	person := mapTMDBPersonDetailsToPersonDetailsPage(tmdb.PersonDetails{
		ID:                 55,
		Name:               "",
		Biography:          "  Biographical text  ",
		KnownForDepartment: "Acting",
		Birthday:           "not-a-date",
		Deathday:           "2025-31-31",
	}, nil, nil)

	if person == nil {
		t.Fatal("expected mapped person details, got nil")
	}

	if person.Name != unknownPersonName {
		t.Fatalf("expected fallback person name %q, got %q", unknownPersonName, person.Name)
	}

	if person.Birthday != nil {
		t.Fatalf("expected birthday to be nil for invalid date, got %v", person.Birthday)
	}

	if person.Deathday != nil {
		t.Fatalf("expected deathday to be nil for invalid date, got %v", person.Deathday)
	}

	if person.Biography != "Biographical text" {
		t.Fatalf("expected trimmed biography, got %q", person.Biography)
	}

	if len(person.ActingCredits) != 0 || len(person.CrewCredits) != 0 || len(person.KnownFor) != 0 {
		t.Fatalf(
			"expected no credits for nil combined credits, got acting=%d crew=%d knownFor=%d",
			len(person.ActingCredits),
			len(person.CrewCredits),
			len(person.KnownFor),
		)
	}
}

func TestMapTMDBPersonCastCreditToPersonCredit_FallbacksAndDateSelection(t *testing.T) {
	credit := mapTMDBPersonCastCreditToPersonCredit(
		101,
		"credit-cast-1",
		" tv ",
		"",
		"Series Name",
		"",
		"",
		"",
		"",
		"2020-09-10",
		7,
		"/series-backdrop.jpg",
		"/series.jpg",
		8.1,
		321,
		53.4,
	)

	if credit.Title != "Series Name" {
		t.Fatalf("expected credit title fallback to series name, got %q", credit.Title)
	}

	if credit.Role != unknownPersonRole {
		t.Fatalf("expected role fallback %q, got %q", unknownPersonRole, credit.Role)
	}

	if credit.MediaType != "tv" {
		t.Fatalf("expected trimmed media type %q, got %q", "tv", credit.MediaType)
	}

	if credit.ReleaseDate == nil {
		t.Fatal("expected release date to use first air date fallback, got nil")
	}

	if credit.ReleaseDate.Format("2006-01-02") != "2020-09-10" {
		t.Fatalf("expected release date fallback to first air date, got %s", credit.ReleaseDate.Format("2006-01-02"))
	}

	if credit.BackdropPath != "/series-backdrop.jpg" {
		t.Fatalf("expected backdrop path to be mapped, got %q", credit.BackdropPath)
	}

	if credit.VoteCount != 321 {
		t.Fatalf("expected vote count to be mapped, got %d", credit.VoteCount)
	}

	if credit.EpisodeCount != 7 {
		t.Fatalf("expected episode count to be mapped, got %d", credit.EpisodeCount)
	}
}

func TestMapTMDBPersonCrewCreditToPersonCredit_Fallbacks(t *testing.T) {
	credit := mapTMDBPersonCrewCreditToPersonCredit(
		303,
		"credit-crew-1",
		"movie",
		"",
		"",
		"",
		"",
		"",
		"  Production  ",
		"invalid-date",
		"",
		4,
		" /crew-backdrop.jpg ",
		"",
		0,
		97,
		12,
	)

	if credit.Title != untitledPersonCredit {
		t.Fatalf("expected fallback title %q, got %q", untitledPersonCredit, credit.Title)
	}

	if credit.Role != unknownPersonRole {
		t.Fatalf("expected fallback role %q, got %q", unknownPersonRole, credit.Role)
	}

	if credit.Department != "Production" {
		t.Fatalf("expected trimmed department %q, got %q", "Production", credit.Department)
	}

	if credit.ReleaseDate != nil {
		t.Fatalf("expected release date to be nil for invalid date, got %v", credit.ReleaseDate)
	}

	if credit.BackdropPath != "/crew-backdrop.jpg" {
		t.Fatalf("expected trimmed backdrop path %q, got %q", "/crew-backdrop.jpg", credit.BackdropPath)
	}

	if credit.VoteCount != 97 {
		t.Fatalf("expected vote count to be mapped, got %d", credit.VoteCount)
	}

	if credit.EpisodeCount != 4 {
		t.Fatalf("expected episode count to be mapped, got %d", credit.EpisodeCount)
	}
}

func TestMapTMDBPersonCrewCreditToPersonCredit_UsesTVFallbackTitleAndDate(t *testing.T) {
	credit := mapTMDBPersonCrewCreditToPersonCredit(
		404,
		"credit-crew-tv-1",
		"tv",
		"",
		"Fallback Show",
		"",
		"Fallback Original Show",
		"Director",
		"Directing",
		"",
		"2023-04-02",
		8,
		"/tv-backdrop.jpg",
		"/tv-poster.jpg",
		8.5,
		450,
		21.3,
	)

	if credit.Title != "Fallback Show" {
		t.Fatalf("expected TV fallback title %q, got %q", "Fallback Show", credit.Title)
	}

	if credit.ReleaseDate == nil {
		t.Fatal("expected release date to use first air date fallback, got nil")
	}

	if credit.ReleaseDate.Format("2006-01-02") != "2023-04-02" {
		t.Fatalf("expected release date fallback to first air date, got %s", credit.ReleaseDate.Format("2006-01-02"))
	}
}

func TestMapTMDBPersonDetailsToPersonDetailsPage_SortsCreditsByVotesAndInvolvement(t *testing.T) {
	credits := mustPersonCombinedCreditsFromJSON(t, `
		{
			"cast": [
				{"id": 300, "media_type": "movie", "title": "Third Item", "character": "Role 3", "release_date": "2011-01-01", "vote_average": 6.1, "vote_count": 1000},
				{"id": 100, "media_type": "movie", "title": "First Item", "character": "Role 1", "release_date": "2024-01-01", "vote_average": 8.7, "vote_count": 12},
				{"id": 200, "media_type": "tv", "name": "Second Show", "character": "Role 2", "first_air_date": "2022-01-01", "episode_count": 12, "vote_average": 7.4, "vote_count": 320}
			],
			"crew": [
				{"id": 90, "media_type": "movie", "title": "Crew Ninety", "job": "Producer", "department": "Production", "release_date": "2023-01-01", "vote_average": 6.7, "vote_count": 150},
				{"id": 70, "media_type": "movie", "title": "Crew Seventy", "job": "Director", "department": "Directing", "release_date": "2025-01-01", "vote_average": 8.9, "vote_count": 40}
			]
		}
	`)

	person := mapTMDBPersonDetailsToPersonDetailsPage(tmdb.PersonDetails{
		ID:   1,
		Name: "Person",
	}, credits, nil)

	if person == nil {
		t.Fatal("expected mapped person details, got nil")
	}

	if gotActingIDs, expectedActingIDs := personCreditIDs(person.ActingCredits), []int64{200, 300, 100}; !reflect.DeepEqual(gotActingIDs, expectedActingIDs) {
		t.Fatalf("unexpected acting order\nexpected: %v\ngot:      %v", expectedActingIDs, gotActingIDs)
	}

	if gotCrewIDs, expectedCrewIDs := personCreditIDs(person.CrewCredits), []int64{90, 70}; !reflect.DeepEqual(gotCrewIDs, expectedCrewIDs) {
		t.Fatalf("unexpected crew order\nexpected: %v\ngot:      %v", expectedCrewIDs, gotCrewIDs)
	}

	if gotKnownForIDs, expectedKnownForIDs := personCreditIDs(person.KnownFor), []int64{200, 300, 100}; !reflect.DeepEqual(gotKnownForIDs, expectedKnownForIDs) {
		t.Fatalf("unexpected known-for order\nexpected: %v\ngot:      %v", expectedKnownForIDs, gotKnownForIDs)
	}
}

func TestMapTMDBPersonDetailsToPersonDetailsPage_PrioritizesRecurringTVOverGuestAppearances(t *testing.T) {
	credits := mustPersonCombinedCreditsFromJSON(t, `
		{
			"cast": [
				{"id": 10, "media_type": "tv", "name": "Guest Hit Show", "character": "Guest", "episode_count": 1, "vote_average": 9.2, "vote_count": 1200},
				{"id": 20, "media_type": "tv", "name": "Recurring Show", "character": "Lead", "episode_count": 8, "vote_average": 8.1, "vote_count": 900},
				{"id": 30, "media_type": "movie", "title": "Popular Movie", "character": "Lead", "vote_average": 7.9, "vote_count": 1400}
			],
			"crew": []
		}
	`)

	person := mapTMDBPersonDetailsToPersonDetailsPage(tmdb.PersonDetails{
		ID:   5,
		Name: "Person",
	}, credits, nil)

	if person == nil {
		t.Fatal("expected mapped person details, got nil")
	}

	if gotActingIDs, expectedActingIDs := personCreditIDs(person.ActingCredits), []int64{20, 30, 10}; !reflect.DeepEqual(gotActingIDs, expectedActingIDs) {
		t.Fatalf("unexpected acting order for recurring vs guest credits\nexpected: %v\ngot:      %v", expectedActingIDs, gotActingIDs)
	}
}

func TestMapTMDBPersonDetailsToPersonDetailsPage_PrioritizesRecurringTVCrewOverGuestCrew(t *testing.T) {
	credits := mustPersonCombinedCreditsFromJSON(t, `
		{
			"cast": [],
			"crew": [
				{"id": 10, "media_type": "tv", "title": "Guest Hit Show", "job": "Director", "department": "Directing", "credit_id": "tv-guest", "vote_average": 9.1, "vote_count": 1200},
				{"id": 20, "media_type": "tv", "title": "Recurring Show", "job": "Director", "department": "Directing", "credit_id": "tv-recurring", "vote_average": 8.3, "vote_count": 900},
				{"id": 30, "media_type": "movie", "title": "Popular Movie", "job": "Director", "department": "Directing", "credit_id": "movie-main", "vote_average": 8.0, "vote_count": 1400}
			]
		}
	`)

	tvCredits := mustPersonTVCreditsFromJSON(t, `
		{
			"cast": [],
			"crew": [
				{"id": 10, "credit_id": "tv-guest", "episode_count": 1},
				{"id": 20, "credit_id": "tv-recurring", "episode_count": 12}
			]
		}
	`)

	person := mapTMDBPersonDetailsToPersonDetailsPage(tmdb.PersonDetails{
		ID:   6,
		Name: "Person",
	}, credits, tvCredits)

	if person == nil {
		t.Fatal("expected mapped person details, got nil")
	}

	if gotCrewIDs, expectedCrewIDs := personCreditIDs(person.CrewCredits), []int64{20, 30, 10}; !reflect.DeepEqual(gotCrewIDs, expectedCrewIDs) {
		t.Fatalf("unexpected crew order for recurring vs guest tv crew credits\nexpected: %v\ngot:      %v", expectedCrewIDs, gotCrewIDs)
	}
}

func TestMapTMDBPersonDetailsToPersonDetailsPage_UsesTVCrewFallbackMetadata(t *testing.T) {
	credits := mustPersonCombinedCreditsFromJSON(t, `
		{
			"cast": [],
			"crew": [
				{"id": 20, "media_type": "tv", "job": "Director", "department": "Directing", "credit_id": "tv-recurring", "vote_average": 8.3, "vote_count": 900}
			]
		}
	`)

	tvCredits := mustPersonTVCreditsFromJSON(t, `
		{
			"cast": [],
			"crew": [
				{"id": 20, "credit_id": "tv-recurring", "name": "Recurring Show", "original_name": "Recurring Show Original", "first_air_date": "2021-02-03", "episode_count": 12}
			]
		}
	`)

	person := mapTMDBPersonDetailsToPersonDetailsPage(tmdb.PersonDetails{
		ID:   7,
		Name: "Person",
	}, credits, tvCredits)

	if person == nil {
		t.Fatal("expected mapped person details, got nil")
	}

	if len(person.CrewCredits) != 1 {
		t.Fatalf("expected 1 crew credit, got %d", len(person.CrewCredits))
	}

	credit := person.CrewCredits[0]

	if credit.Title != "Recurring Show" {
		t.Fatalf("expected TV fallback title %q, got %q", "Recurring Show", credit.Title)
	}

	if credit.ReleaseDate == nil {
		t.Fatal("expected TV fallback release date, got nil")
	}

	if credit.ReleaseDate.Format("2006-01-02") != "2021-02-03" {
		t.Fatalf("expected TV fallback release date %q, got %s", "2021-02-03", credit.ReleaseDate.Format("2006-01-02"))
	}

	if credit.EpisodeCount != 12 {
		t.Fatalf("expected episode count %d, got %d", 12, credit.EpisodeCount)
	}
}

func TestMapTMDBPersonDetailsToPersonDetailsPage_KnownForFallsBackToSortedCrewCredits(t *testing.T) {
	credits := mustPersonCombinedCreditsFromJSON(t, `
		{
			"cast": [],
			"crew": [
				{"id": 11, "media_type": "movie", "title": "Crew First", "job": "Producer", "department": "Production", "vote_average": 4.5, "vote_count": 300},
				{"id": 22, "media_type": "movie", "title": "Crew Second", "job": "Director", "department": "Directing", "vote_average": 8.8, "vote_count": 500}
			]
		}
	`)

	person := mapTMDBPersonDetailsToPersonDetailsPage(tmdb.PersonDetails{
		ID:   2,
		Name: "Person",
	}, credits, nil)

	if person == nil {
		t.Fatal("expected mapped person details, got nil")
	}

	if gotKnownForIDs, expectedKnownForIDs := personCreditIDs(person.KnownFor), []int64{22, 11}; !reflect.DeepEqual(gotKnownForIDs, expectedKnownForIDs) {
		t.Fatalf("unexpected known-for fallback order\nexpected: %v\ngot:      %v", expectedKnownForIDs, gotKnownForIDs)
	}
}

func TestMapTMDBPersonDetailsToPersonDetailsPage_PreservesTMDBOrderWhenVoteAverageTies(t *testing.T) {
	credits := mustPersonCombinedCreditsFromJSON(t, `
		{
			"cast": [
				{"id": 31, "media_type": "movie", "title": "Tie First", "character": "Role A", "vote_average": 7.0, "vote_count": 180},
				{"id": 42, "media_type": "movie", "title": "Tie Second", "character": "Role B", "vote_average": 7.0, "vote_count": 180}
			],
			"crew": []
		}
	`)

	person := mapTMDBPersonDetailsToPersonDetailsPage(tmdb.PersonDetails{
		ID:   3,
		Name: "Person",
	}, credits, nil)

	if person == nil {
		t.Fatal("expected mapped person details, got nil")
	}

	if gotActingIDs, expectedActingIDs := personCreditIDs(person.ActingCredits), []int64{31, 42}; !reflect.DeepEqual(gotActingIDs, expectedActingIDs) {
		t.Fatalf("unexpected acting order for equal vote averages\nexpected: %v\ngot:      %v", expectedActingIDs, gotActingIDs)
	}
}

func TestMapTMDBPersonDetailsToPersonDetailsPage_LowVoteOutliersDoNotLeadSorting(t *testing.T) {
	credits := mustPersonCombinedCreditsFromJSON(t, `
		{
			"cast": [
				{"id": 1, "media_type": "movie", "title": "Reliable", "character": "Lead", "vote_average": 7.4, "vote_count": 500},
				{"id": 2, "media_type": "movie", "title": "One Vote", "character": "Lead", "vote_average": 9.8, "vote_count": 1}
			],
			"crew": []
		}
	`)

	person := mapTMDBPersonDetailsToPersonDetailsPage(tmdb.PersonDetails{
		ID:   4,
		Name: "Person",
	}, credits, nil)

	if person == nil {
		t.Fatal("expected mapped person details, got nil")
	}

	if gotActingIDs, expectedActingIDs := personCreditIDs(person.ActingCredits), []int64{1, 2}; !reflect.DeepEqual(gotActingIDs, expectedActingIDs) {
		t.Fatalf("unexpected acting order with low-vote outlier\nexpected: %v\ngot:      %v", expectedActingIDs, gotActingIDs)
	}
}

func TestPersonCreditDisplayTitle_UsesFallbacks(t *testing.T) {
	if got := personCreditDisplayTitle("", "", "", ""); got != untitledPersonCredit {
		t.Fatalf("expected untitled fallback %q, got %q", untitledPersonCredit, got)
	}

	if got := personCreditDisplayTitle("", "Display Name", "", ""); got != "Display Name" {
		t.Fatalf("expected fallback display name, got %q", got)
	}
}

func mustPersonCombinedCreditsFromJSON(t *testing.T, payload string) *tmdb.PersonCombinedCredits {
	t.Helper()

	var credits tmdb.PersonCombinedCredits
	if err := json.Unmarshal([]byte(payload), &credits); err != nil {
		t.Fatalf("failed to unmarshal person combined credits fixture: %v", err)
	}

	return &credits
}

func mustPersonTVCreditsFromJSON(t *testing.T, payload string) *tmdb.PersonTVCredits {
	t.Helper()

	var credits tmdb.PersonTVCredits
	if err := json.Unmarshal([]byte(payload), &credits); err != nil {
		t.Fatalf("failed to unmarshal person tv credits fixture: %v", err)
	}

	return &credits
}

func personCreditIDs(credits []models.PersonCredit) []int64 {
	ids := make([]int64, len(credits))
	for i, credit := range credits {
		ids[i] = credit.ID
	}

	return ids
}
