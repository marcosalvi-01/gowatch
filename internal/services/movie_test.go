package services

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sync/atomic"
	"testing"
	"time"

	"github.com/marcosalvi-01/gowatch/db"
	"github.com/marcosalvi-01/gowatch/internal/models"
)

func newMovieServiceWithTestDB(t *testing.T, seerr *SeerrService, cacheTTL time.Duration) *MovieService {
	t.Helper()

	testDB, err := db.NewTestDB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := testDB.Close(); err != nil {
			t.Errorf("failed to close test database: %v", err)
		}
	})

	return NewMovieService(testDB, seerr, cacheTTL)
}

func TestMovieService_GetMovieDetails_CacheHit(t *testing.T) {
	testDB, err := db.NewTestDB()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = testDB.Close() }()

	movieService := NewMovieService(testDB, newUnconfiguredTestSeerrService(), time.Hour)

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

func TestMovieService_GetMovieDetailsUsesSeerrMetadataAndPreservesRelatedData(t *testing.T) {
	testDB, err := db.NewTestDB()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = testDB.Close() }()

	var requestCount atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/movie/603" {
			t.Errorf("request path = %q, want /api/v1/movie/603", r.URL.Path)
		}
		if got := r.Header.Get(seerrAPIKeyHeader); got != "test-key" {
			t.Errorf("API key header = %q, want %q", got, "test-key")
		}
		requestCount.Add(1)
		_, _ = w.Write([]byte(`{"id":603,"backdropPath":"/backdrop.jpg","budget":63000000,"genres":[{"id":18,"name":"Drama"}],"homepage":"https://example.com","imdbId":"tt0133093","originalLanguage":"en","originalTitle":"The Matrix","overview":"A hacker learns the truth.","popularity":42.5,"posterPath":"/poster.jpg","releaseDate":"1999-03-31","revenue":463000000,"runtime":136,"status":"Released","tagline":"Welcome to the Real World.","title":"The Matrix","voteAverage":8.2,"voteCount":10000,"credits":{"cast":[{"id":6384,"castId":1,"character":"Neo","creditId":"cast-credit","gender":2,"name":"Keanu Reeves","order":0,"profilePath":"/keanu.jpg"}],"crew":[{"id":9339,"creditId":"crew-credit","gender":2,"name":"Lana Wachowski","job":"Director","department":"Directing","profilePath":"/lana.jpg"}]}}`))
	}))
	defer server.Close()

	seerrService, err := NewSeerrService(models.SeerrIntegration{BaseURL: server.URL, APIKey: "test-key"}, server.Client())
	if err != nil {
		t.Fatalf("NewSeerrService() error = %v", err)
	}
	movieService := NewMovieService(testDB, seerrService, time.Hour)

	ctx := context.Background()
	movie, err := movieService.GetMovieDetails(ctx, 603)
	if err != nil {
		t.Fatalf("GetMovieDetails() error = %v", err)
	}
	if movie.Movie.ID != 603 || movie.Movie.Title != "The Matrix" || movie.Movie.ReleaseDate == nil || movie.Movie.ReleaseDate.Format("2006-01-02") != "1999-03-31" || movie.Movie.Popularity != 42.5 || movie.Budget != 63000000 || movie.Revenue != 463000000 || movie.Runtime != 136 || movie.IMDbID != "tt0133093" {
		t.Fatalf("mapped movie details = %+v", movie)
	}
	if len(movie.Genres) != 1 || movie.Genres[0] != (models.Genre{ID: 18, Name: "Drama"}) {
		t.Fatalf("mapped genres = %+v", movie.Genres)
	}
	if len(movie.Credits.Cast) != 1 || movie.Credits.Cast[0].Person.Name != "Keanu Reeves" || movie.Credits.Cast[0].Person.ProfilePath != "/keanu.jpg" || movie.Credits.Cast[0].Character != "Neo" || movie.Credits.Cast[0].CastOrder != 0 {
		t.Fatalf("mapped cast = %+v", movie.Credits.Cast)
	}
	if len(movie.Credits.Crew) != 1 || movie.Credits.Crew[0].Person.Name != "Lana Wachowski" || movie.Credits.Crew[0].Job != "Director" || movie.Credits.Crew[0].Department != "Directing" {
		t.Fatalf("mapped crew = %+v", movie.Credits.Crew)
	}

	if _, err := movieService.GetMovieDetails(ctx, 603); err != nil {
		t.Fatalf("GetMovieDetails() cache hit error = %v", err)
	}
	if got := requestCount.Load(); got != 1 {
		t.Fatalf("Seerr requests after cache hit = %d, want 1", got)
	}

	user, err := testDB.CreateUser(ctx, "movie-metadata@example.com", "Movie User", "hash")
	if err != nil {
		t.Fatalf("CreateUser() error = %v", err)
	}
	rating := 4.5
	if err := testDB.InsertWatched(ctx, db.InsertWatched{UserID: user.ID, MovieID: 603, Date: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), Rating: &rating}); err != nil {
		t.Fatalf("InsertWatched() error = %v", err)
	}
	listID, err := testDB.InsertList(ctx, db.InsertList{UserID: user.ID, Name: "Watch later", DisplaySort: models.ListMovieSortCustom})
	if err != nil {
		t.Fatalf("InsertList() error = %v", err)
	}
	if err := testDB.AddMovieToList(ctx, user.ID, db.InsertMovieList{MovieID: 603, ListID: listID, DateAdded: time.Now()}); err != nil {
		t.Fatalf("AddMovieToList() error = %v", err)
	}

	expiredMovieService := NewMovieService(testDB, seerrService, -time.Second)
	if _, err := expiredMovieService.GetMovieDetails(ctx, 603); err != nil {
		t.Fatalf("GetMovieDetails() expired-cache refresh error = %v", err)
	}
	if got := requestCount.Load(); got != 2 {
		t.Fatalf("Seerr requests after expired-cache refresh = %d, want 2", got)
	}

	cachedMovie, err := testDB.GetMovieDetailsByID(ctx, 603)
	if err != nil {
		t.Fatalf("GetMovieDetailsByID() error = %v", err)
	}
	if len(cachedMovie.Credits.Cast) != 1 || len(cachedMovie.Credits.Crew) != 1 {
		t.Fatalf("refreshed movie lost cast/crew relationships: %+v", cachedMovie.Credits)
	}
	watchedMovies, err := testDB.GetWatchedJoinMovie(ctx, user.ID)
	if err != nil {
		t.Fatalf("GetWatchedJoinMovie() error = %v", err)
	}
	if len(watchedMovies) != 1 || watchedMovies[0].MovieDetails.Movie.ID != 603 {
		t.Fatalf("watch history after movie refresh = %+v", watchedMovies)
	}
	list, err := testDB.GetList(ctx, user.ID, listID)
	if err != nil {
		t.Fatalf("GetList() error = %v", err)
	}
	if len(list.Movies) != 1 || list.Movies[0].MovieDetails.Movie.ID != 603 {
		t.Fatalf("list contents after movie refresh = %+v", list.Movies)
	}
}

func TestMovieService_GetPersonDetailsUsesSeerrCombinedCredits(t *testing.T) {
	seen := make(map[string]bool)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get(seerrAPIKeyHeader); got != "test-key" {
			t.Errorf("API key header = %q, want %q", got, "test-key")
		}
		seen[r.URL.Path] = true
		switch r.URL.Path {
		case "/api/v1/person/66633":
			_, _ = w.Write([]byte(`{"id":66633,"name":"Vince Gilligan","birthday":"1967-02-10","knownForDepartment":"Writing","popularity":2.8919,"biography":" A writer ","placeOfBirth":"Richmond, Virginia, USA","profilePath":"/vince.jpg","imdbId":"nm0319213"}`))
		case "/api/v1/person/66633/combined_credits":
			_, _ = w.Write([]byte(`{"id":66633,"cast":[{"id":18347,"mediaType":"tv","name":"Community","character":"Devon","firstAirDate":"2009-09-17","episodeCount":1,"voteAverage":8,"voteCount":2204}],"crew":[{"id":5715,"mediaType":"tv","name":"The Lone Gunmen","originalName":"The Lone Gunmen","creditId":"crew-credit","firstAirDate":"2001-03-04","episodeCount":6,"voteAverage":6.8,"voteCount":104,"department":"Writing","job":"Writer"}]}`))
		default:
			t.Errorf("unexpected request path %q", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	seerrService, err := NewSeerrService(models.SeerrIntegration{BaseURL: server.URL, APIKey: "test-key"}, server.Client())
	if err != nil {
		t.Fatalf("NewSeerrService() error = %v", err)
	}
	movieService := newMovieServiceWithTestDB(t, seerrService, time.Hour)

	person, err := movieService.GetPersonDetails(t.Context(), 66633)
	if err != nil {
		t.Fatalf("GetPersonDetails() error = %v", err)
	}
	if person.Name != "Vince Gilligan" || person.Birthday == nil || person.Birthday.Format("2006-01-02") != "1967-02-10" {
		t.Fatalf("person details = %+v", person)
	}
	if len(person.ActingCredits) != 1 || person.ActingCredits[0].Title != "Community" || person.ActingCredits[0].EpisodeCount != 1 {
		t.Fatalf("acting credits = %+v", person.ActingCredits)
	}
	if len(person.CrewCredits) != 1 || person.CrewCredits[0].Title != "The Lone Gunmen" || person.CrewCredits[0].EpisodeCount != 6 || person.CrewCredits[0].ReleaseDate == nil || person.CrewCredits[0].ReleaseDate.Format("2006-01-02") != "2001-03-04" {
		t.Fatalf("crew credits = %+v", person.CrewCredits)
	}
	if !seen["/api/v1/person/66633"] || !seen["/api/v1/person/66633/combined_credits"] || len(seen) != 2 {
		t.Fatalf("Seerr request paths = %+v, want person and combined credits only", seen)
	}
}

func TestMovieService_SearchMultiUsesSeerrResults(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/search" {
			t.Errorf("request path = %q, want /api/v1/search", r.URL.Path)
		}
		if got := r.URL.Query().Get("query"); got != "The Matrix & Co" {
			t.Errorf("query = %q, want %q", got, "The Matrix & Co")
		}
		if got := r.Header.Get(seerrAPIKeyHeader); got != "test-key" {
			t.Errorf("API key header = %q, want %q", got, "test-key")
		}
		_, _ = w.Write([]byte(`{"totalResults":6,"results":[{"id":603,"mediaType":"movie","title":"The Matrix","originalTitle":"The Matrix","posterPath":"/matrix.jpg","releaseDate":"1999-03-31","voteAverage":8.26},{"id":123,"mediaType":"tv","name":"Ignored TV result"},{"id":66633,"mediaType":"person","name":"Vince Gilligan","popularity":2.8919,"profilePath":"/vince.jpg"},{"id":42,"mediaType":"movie"},{"id":43,"mediaType":"person","originalName":"Fallback Person"},{"id":99,"mediaType":"collection","title":"Ignored collection"}]}`))
	}))
	defer server.Close()

	seerrService, err := NewSeerrService(models.SeerrIntegration{BaseURL: server.URL, APIKey: "test-key"}, server.Client())
	if err != nil {
		t.Fatalf("NewSeerrService() error = %v", err)
	}
	movieService := newMovieServiceWithTestDB(t, seerrService, time.Hour)

	results, err := movieService.SearchMulti(t.Context(), "The Matrix & Co")
	if err != nil {
		t.Fatalf("SearchMulti() error = %v", err)
	}
	if len(results) != 4 {
		t.Fatalf("SearchMulti() returned %d results, want 4: %+v", len(results), results)
	}
	if results[0].ID != 603 || results[0].Title != "The Matrix" || results[0].SecondaryText != "1999" || results[0].VoteAverage != 8.26 || results[0].ImagePath != "/matrix.jpg" {
		t.Fatalf("first result = %+v, want mapped Matrix result", results[0])
	}
	if results[1].ID != 66633 || results[1].Title != "Vince Gilligan" || results[1].SecondaryText != "" || results[1].ImagePath != "/vince.jpg" {
		t.Fatalf("second result = %+v, want mapped person result", results[1])
	}
	if results[2].Title != untitledMovieTitle || results[2].SecondaryText != unknownYearSecondaryText {
		t.Fatalf("movie with missing optional fields = %+v", results[2])
	}
	if results[3].Title != unknownPersonName || results[3].SecondaryText != "" {
		t.Fatalf("person with missing optional fields = %+v", results[3])
	}
}

func TestMovieService_SearchMultiReturnsSeerrErrorsWithoutFallback(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	seerrService, err := NewSeerrService(models.SeerrIntegration{BaseURL: server.URL, APIKey: "test-key"}, server.Client())
	if err != nil {
		t.Fatalf("NewSeerrService() error = %v", err)
	}
	movieService := newMovieServiceWithTestDB(t, seerrService, time.Hour)

	results, err := movieService.SearchMulti(t.Context(), "The Matrix")
	if !errors.Is(err, ErrSeerrUnavailable) {
		t.Fatalf("SearchMulti() error = %v, want ErrSeerrUnavailable", err)
	}
	if results != nil {
		t.Fatalf("SearchMulti() results = %+v, want nil on error", results)
	}
}

func TestNewMovieServicePanicsWithoutSeerr(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("NewMovieService() did not panic without Seerr service")
		}
	}()

	testDB, err := db.NewTestDB()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = testDB.Close() }()

	NewMovieService(testDB, nil, time.Hour)
}

func TestMapSearchMovieToSearchResult(t *testing.T) {
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
			got := mapSearchMovieToSearchResult(
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

func TestMapSearchPersonToSearchResult(t *testing.T) {
	tests := []struct {
		name        string
		id          int64
		personName  string
		profilePath string
		expected    models.SearchResult
	}{
		{
			name:        "uses person name",
			id:          10,
			personName:  "Christopher Nolan",
			profilePath: "/nolan.jpg",
			expected: models.SearchResult{
				ID:        10,
				Title:     "Christopher Nolan",
				ImagePath: "/nolan.jpg",
				Type:      models.SearchResultTypePerson,
			},
		},
		{
			name:        "uses unknown name when person name missing",
			id:          11,
			personName:  "",
			profilePath: "",
			expected: models.SearchResult{
				ID:        11,
				Title:     unknownPersonName,
				ImagePath: "",
				Type:      models.SearchResultTypePerson,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mapSearchPersonToSearchResult(
				tt.id,
				tt.personName,
				tt.profilePath,
			)

			if !reflect.DeepEqual(got, tt.expected) {
				t.Fatalf("unexpected mapped person result\nexpected: %+v\ngot:      %+v", tt.expected, got)
			}
		})
	}
}

func TestMapSeerrPersonDetailsToPersonDetailsPage_InvalidDatesDoNotFail(t *testing.T) {
	person := mapSeerrPersonDetailsToPersonDetailsPage(seerrPersonMetadata{
		ID:        55,
		Name:      "",
		Biography: "  Biographical text  ",
		Birthday:  "not-a-date",
		Deathday:  "2025-31-31",
	}, nil)

	if person == nil {
		t.Fatal("expected mapped person details, got nil")
		return
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

func TestMapPersonCastCreditToPersonCredit_FallbacksAndDateSelection(t *testing.T) {
	credit := mapPersonCastCreditToPersonCredit(
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

func TestMapPersonCrewCreditToPersonCredit_Fallbacks(t *testing.T) {
	credit := mapPersonCrewCreditToPersonCredit(
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

func TestMapPersonCrewCreditToPersonCredit_UsesTVFallbackTitleAndDate(t *testing.T) {
	credit := mapPersonCrewCreditToPersonCredit(
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

func TestMapSeerrPersonDetailsToPersonDetailsPage_SortsCreditsByVotesAndInvolvement(t *testing.T) {
	credits := mustPersonCombinedCreditsFromJSON(t, `
		{
			"cast": [
				{"id": 300, "mediaType": "movie", "title": "Third Item", "character": "Role 3", "releaseDate": "2011-01-01", "voteAverage": 6.1, "voteCount": 1000},
				{"id": 100, "mediaType": "movie", "title": "First Item", "character": "Role 1", "releaseDate": "2024-01-01", "voteAverage": 8.7, "voteCount": 12},
				{"id": 200, "mediaType": "tv", "name": "Second Show", "character": "Role 2", "firstAirDate": "2022-01-01", "episodeCount": 12, "voteAverage": 7.4, "voteCount": 320}
			],
			"crew": [
				{"id": 90, "mediaType": "movie", "title": "Crew Ninety", "job": "Producer", "department": "Production", "releaseDate": "2023-01-01", "voteAverage": 6.7, "voteCount": 150},
				{"id": 70, "mediaType": "movie", "title": "Crew Seventy", "job": "Director", "department": "Directing", "releaseDate": "2025-01-01", "voteAverage": 8.9, "voteCount": 40}
			]
		}
	`)

	person := mapSeerrPersonDetailsToPersonDetailsPage(seerrPersonMetadata{
		ID:   1,
		Name: "Person",
	}, credits)

	if person == nil {
		t.Fatal("expected mapped person details, got nil")
		return
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

func TestMapSeerrPersonDetailsToPersonDetailsPage_PrioritizesRecurringTVOverGuestAppearances(t *testing.T) {
	credits := mustPersonCombinedCreditsFromJSON(t, `
		{
			"cast": [
				{"id": 10, "mediaType": "tv", "name": "Guest Hit Show", "character": "Guest", "episodeCount": 1, "voteAverage": 9.2, "voteCount": 1200},
				{"id": 20, "mediaType": "tv", "name": "Recurring Show", "character": "Lead", "episodeCount": 8, "voteAverage": 8.1, "voteCount": 900},
				{"id": 30, "mediaType": "movie", "title": "Popular Movie", "character": "Lead", "voteAverage": 7.9, "voteCount": 1400}
			],
			"crew": []
		}
	`)

	person := mapSeerrPersonDetailsToPersonDetailsPage(seerrPersonMetadata{
		ID:   5,
		Name: "Person",
	}, credits)

	if person == nil {
		t.Fatal("expected mapped person details, got nil")
		return
	}

	if gotActingIDs, expectedActingIDs := personCreditIDs(person.ActingCredits), []int64{20, 30, 10}; !reflect.DeepEqual(gotActingIDs, expectedActingIDs) {
		t.Fatalf("unexpected acting order for recurring vs guest credits\nexpected: %v\ngot:      %v", expectedActingIDs, gotActingIDs)
	}
}

func TestMapSeerrPersonDetailsToPersonDetailsPage_PrioritizesRecurringTVCrewOverGuestCrew(t *testing.T) {
	credits := mustPersonCombinedCreditsFromJSON(t, `
		{
			"cast": [],
			"crew": [
				{"id": 10, "mediaType": "tv", "title": "Guest Hit Show", "name": "Guest Hit Show", "job": "Director", "department": "Directing", "creditId": "tv-guest", "episodeCount": 1, "voteAverage": 9.1, "voteCount": 1200},
				{"id": 20, "mediaType": "tv", "title": "Recurring Show", "name": "Recurring Show", "job": "Director", "department": "Directing", "creditId": "tv-recurring", "episodeCount": 12, "voteAverage": 8.3, "voteCount": 900},
				{"id": 30, "mediaType": "movie", "title": "Popular Movie", "job": "Director", "department": "Directing", "creditId": "movie-main", "voteAverage": 8.0, "voteCount": 1400}
			]
		}
	`)

	person := mapSeerrPersonDetailsToPersonDetailsPage(seerrPersonMetadata{
		ID:   6,
		Name: "Person",
	}, credits)

	if person == nil {
		t.Fatal("expected mapped person details, got nil")
		return
	}

	if gotCrewIDs, expectedCrewIDs := personCreditIDs(person.CrewCredits), []int64{20, 30, 10}; !reflect.DeepEqual(gotCrewIDs, expectedCrewIDs) {
		t.Fatalf("unexpected crew order for recurring vs guest tv crew credits\nexpected: %v\ngot:      %v", expectedCrewIDs, gotCrewIDs)
	}
}

func TestMapSeerrPersonDetailsToPersonDetailsPage_UsesTVCrewMetadata(t *testing.T) {
	credits := mustPersonCombinedCreditsFromJSON(t, `
		{
			"cast": [],
			"crew": [
				{"id": 20, "mediaType": "tv", "name": "Recurring Show", "originalName": "Recurring Show Original", "firstAirDate": "2021-02-03", "episodeCount": 12, "job": "Director", "department": "Directing", "creditId": "tv-recurring", "voteAverage": 8.3, "voteCount": 900}
			]
		}
	`)

	person := mapSeerrPersonDetailsToPersonDetailsPage(seerrPersonMetadata{
		ID:   7,
		Name: "Person",
	}, credits)

	if person == nil {
		t.Fatal("expected mapped person details, got nil")
		return
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

func TestMapSeerrPersonDetailsToPersonDetailsPage_KnownForFallsBackToSortedCrewCredits(t *testing.T) {
	credits := mustPersonCombinedCreditsFromJSON(t, `
		{
			"cast": [],
			"crew": [
				{"id": 11, "mediaType": "movie", "title": "Crew First", "job": "Producer", "department": "Production", "voteAverage": 4.5, "voteCount": 300},
				{"id": 22, "mediaType": "movie", "title": "Crew Second", "job": "Director", "department": "Directing", "voteAverage": 8.8, "voteCount": 500}
			]
		}
	`)

	person := mapSeerrPersonDetailsToPersonDetailsPage(seerrPersonMetadata{
		ID:   2,
		Name: "Person",
	}, credits)

	if person == nil {
		t.Fatal("expected mapped person details, got nil")
		return
	}

	if gotKnownForIDs, expectedKnownForIDs := personCreditIDs(person.KnownFor), []int64{22, 11}; !reflect.DeepEqual(gotKnownForIDs, expectedKnownForIDs) {
		t.Fatalf("unexpected known-for fallback order\nexpected: %v\ngot:      %v", expectedKnownForIDs, gotKnownForIDs)
	}
}

func TestMapSeerrPersonDetailsToPersonDetailsPage_PreservesResponseOrderWhenVoteAverageTies(t *testing.T) {
	credits := mustPersonCombinedCreditsFromJSON(t, `
		{
			"cast": [
				{"id": 31, "mediaType": "movie", "title": "Tie First", "character": "Role A", "voteAverage": 7.0, "voteCount": 180},
				{"id": 42, "mediaType": "movie", "title": "Tie Second", "character": "Role B", "voteAverage": 7.0, "voteCount": 180}
			],
			"crew": []
		}
	`)

	person := mapSeerrPersonDetailsToPersonDetailsPage(seerrPersonMetadata{
		ID:   3,
		Name: "Person",
	}, credits)

	if person == nil {
		t.Fatal("expected mapped person details, got nil")
		return
	}

	if gotActingIDs, expectedActingIDs := personCreditIDs(person.ActingCredits), []int64{31, 42}; !reflect.DeepEqual(gotActingIDs, expectedActingIDs) {
		t.Fatalf("unexpected acting order for equal vote averages\nexpected: %v\ngot:      %v", expectedActingIDs, gotActingIDs)
	}
}

func TestMapSeerrPersonDetailsToPersonDetailsPage_LowVoteOutliersDoNotLeadSorting(t *testing.T) {
	credits := mustPersonCombinedCreditsFromJSON(t, `
		{
			"cast": [
				{"id": 1, "mediaType": "movie", "title": "Reliable", "character": "Lead", "voteAverage": 7.4, "voteCount": 500},
				{"id": 2, "mediaType": "movie", "title": "One Vote", "character": "Lead", "voteAverage": 9.8, "voteCount": 1}
			],
			"crew": []
		}
	`)

	person := mapSeerrPersonDetailsToPersonDetailsPage(seerrPersonMetadata{
		ID:   4,
		Name: "Person",
	}, credits)

	if person == nil {
		t.Fatal("expected mapped person details, got nil")
		return
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

func mustPersonCombinedCreditsFromJSON(t *testing.T, payload string) *seerrPersonCombinedCredits {
	t.Helper()

	var credits seerrPersonCombinedCredits
	if err := json.Unmarshal([]byte(payload), &credits); err != nil {
		t.Fatalf("failed to unmarshal person combined credits fixture: %v", err)
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
