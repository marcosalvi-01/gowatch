package services

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/marcosalvi-01/gowatch/internal/models"
)

func TestSeerrMetadataRequests(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get(seerrAPIKeyHeader); got != "test-key" {
			t.Errorf("API key header = %q, want %q", got, "test-key")
		}
		if got := r.Header.Get("Accept"); got != "application/json" {
			t.Errorf("Accept header = %q, want application/json", got)
		}

		switch r.URL.Path {
		case "/api/v1/search":
			if got := r.URL.Query().Get("query"); got != "The Matrix & Co" {
				t.Errorf("query = %q, want %q", got, "The Matrix & Co")
			}
			_, _ = w.Write([]byte(`{"results":[{"id":603,"mediaType":"movie","title":"The Matrix","releaseDate":"1999-03-31","voteAverage":8.26,"posterPath":"/matrix.jpg"},{"id":66633,"mediaType":"person","name":"Vince Gilligan","popularity":2.8919,"profilePath":"/vince.jpg"}]}`))
		case "/api/v1/movie/603":
			_, _ = w.Write([]byte(`{"id":603,"title":"The Matrix","releaseDate":"1999-03-31","genres":[{"id":28,"name":"Action"}],"relatedVideos":[{"url":"https://www.youtube.com/watch?v=abc123","key":"abc123","name":"Trailer","size":1080,"type":"Trailer","site":"YouTube"}],"productionCompanies":[{"id":1,"logoPath":"/studio.svg","originCountry":"US","name":"Warner Bros."}],"productionCountries":[{"iso_3166_1":"US","name":"United States"}],"credits":{"cast":[{"id":6384,"castId":34,"creditId":"cast-credit","character":"Neo","gender":2,"name":"Keanu Reeves","order":0,"profilePath":"/keanu.jpg"}],"crew":[{"id":1619,"creditId":"crew-credit","gender":2,"name":"Lana Wachowski","job":"Director","department":"Directing","profilePath":"/lana.jpg"}]}}`))
		case "/api/v1/movie/603/ratingscombined":
			_, _ = w.Write([]byte(`{"rt":{"url":"https://www.rottentomatoes.com/m/the_matrix","criticsScore":88,"criticsRating":"Certified Fresh","audienceScore":85,"audienceRating":"Upright"},"imdb":{"url":"https://www.imdb.com/title/tt0133093","criticsScore":8.7}}`))
		case "/api/v1/person/66633":
			_, _ = w.Write([]byte(`{"id":66633,"name":"Vince Gilligan","birthday":"1967-02-10","knownForDepartment":"Writing","popularity":2.8919,"placeOfBirth":"Richmond, Virginia, USA","profilePath":"/vince.jpg","imdbId":"nm0319213"}`))
		case "/api/v1/person/66633/combined_credits":
			_, _ = w.Write([]byte(`{"id":66633,"cast":[{"id":18347,"episodeCount":1,"name":"Community","originalName":"Community","mediaType":"tv","creditId":"tv-cast-credit","firstAirDate":"2009-09-17","voteAverage":8,"voteCount":2204,"character":"Devon"}],"crew":[{"id":5715,"episodeCount":6,"name":"The Lone Gunmen","originalName":"The Lone Gunmen","mediaType":"tv","creditId":"tv-crew-credit","firstAirDate":"2001-03-04","voteAverage":6.8,"voteCount":104,"department":"Writing","job":"Writer"}]}`))
		default:
			t.Errorf("unexpected request path %q", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	service, err := NewSeerrService(models.SeerrIntegration{BaseURL: server.URL, APIKey: "test-key"}, server.Client())
	if err != nil {
		t.Fatalf("NewSeerrService() error = %v", err)
	}

	search, err := service.searchMetadata(t.Context(), "The Matrix & Co")
	if err != nil {
		t.Fatalf("searchMetadata() error = %v", err)
	}
	if len(search.Results) != 2 || search.Results[0].Title != "The Matrix" || search.Results[1].Name != "Vince Gilligan" {
		t.Fatalf("searchMetadata() results = %+v", search.Results)
	}
	if search.Results[0].ProfilePath != "" || search.Results[1].ReleaseDate != "" {
		t.Fatalf("searchMetadata() optional fields were not zero-valued: %+v", search.Results)
	}

	movie, err := service.movieMetadata(t.Context(), 603)
	if err != nil {
		t.Fatalf("movieMetadata() error = %v", err)
	}
	if movie.ID != 603 || movie.Title != "The Matrix" || len(movie.Genres) != 1 || movie.Genres[0].Name != "Action" {
		t.Fatalf("movieMetadata() movie fields = %+v", movie)
	}
	if len(movie.Credits.Cast) != 1 || movie.Credits.Cast[0].Character != "Neo" || len(movie.Credits.Crew) != 1 || movie.Credits.Crew[0].Job != "Director" {
		t.Fatalf("movieMetadata() credits = %+v", movie.Credits)
	}
	if len(movie.RelatedVideos) != 1 || movie.RelatedVideos[0].Key != "abc123" || len(movie.ProductionCompanies) != 1 || movie.ProductionCompanies[0].Name != "Warner Bros." || len(movie.ProductionCountries) != 1 || movie.ProductionCountries[0].ISOCode != "US" {
		t.Fatalf("movieMetadata() extra fields = videos:%+v companies:%+v countries:%+v", movie.RelatedVideos, movie.ProductionCompanies, movie.ProductionCountries)
	}

	ratings, err := service.MovieRatings(t.Context(), 603)
	if err != nil {
		t.Fatalf("MovieRatings() error = %v", err)
	}
	if ratings.IMDb == nil || ratings.IMDb.Score == nil || *ratings.IMDb.Score != 8.7 || ratings.RottenTomatoes == nil || ratings.RottenTomatoes.CriticsScore == nil || *ratings.RottenTomatoes.CriticsScore != 88 || ratings.RottenTomatoes.AudienceRating != "Upright" {
		t.Fatalf("MovieRatings() = %+v", ratings)
	}

	person, err := service.personMetadata(t.Context(), 66633)
	if err != nil {
		t.Fatalf("personMetadata() error = %v", err)
	}
	if person.ID != 66633 || person.Name != "Vince Gilligan" || person.Birthday != "1967-02-10" {
		t.Fatalf("personMetadata() fields = %+v", person)
	}
	if person.Deathday != "" || person.Homepage != "" {
		t.Fatalf("personMetadata() missing optional fields = %+v", person)
	}

	credits, err := service.personCombinedCredits(t.Context(), 66633)
	if err != nil {
		t.Fatalf("personCombinedCredits() error = %v", err)
	}
	if credits.ID != 66633 || len(credits.Cast) != 1 || credits.Cast[0].Name != "Community" || credits.Cast[0].EpisodeCount != 1 {
		t.Fatalf("personCombinedCredits() cast = %+v", credits)
	}
	if len(credits.Crew) != 1 || credits.Crew[0].Name != "The Lone Gunmen" || credits.Crew[0].EpisodeCount != 6 || credits.Crew[0].FirstAirDate != "2001-03-04" {
		t.Fatalf("personCombinedCredits() crew = %+v", credits)
	}
}

func TestSeerrMetadataErrors(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		want       error
	}{
		{name: "unauthorized", statusCode: http.StatusUnauthorized, want: ErrSeerrUnauthorized},
		{name: "forbidden", statusCode: http.StatusForbidden, want: ErrSeerrForbidden},
		{name: "not found", statusCode: http.StatusNotFound, want: ErrSeerrRejected},
		{name: "server error", statusCode: http.StatusServiceUnavailable, want: ErrSeerrUnavailable},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tt.statusCode)
			}))
			defer server.Close()

			service := newTestSeerrMetadataService(t, server)
			_, err := service.movieMetadata(t.Context(), 603)
			if !errors.Is(err, tt.want) {
				t.Fatalf("movieMetadata() error = %v, want errors.Is(_, %v)", err, tt.want)
			}
		})
	}
}

func TestSeerrMetadataRejectsMalformedAndOversizedResponses(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{name: "malformed", body: `{"id":`},
		{name: "non-object", body: `null`},
		{name: "oversized", body: `{"padding":"` + strings.Repeat("x", seerrMetadataResponseLimit) + `"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write([]byte(tt.body))
			}))
			defer server.Close()

			service := newTestSeerrMetadataService(t, server)
			_, err := service.movieMetadata(t.Context(), 603)
			if !errors.Is(err, ErrSeerrUnavailable) {
				t.Fatalf("movieMetadata() error = %v, want ErrSeerrUnavailable", err)
			}
		})
	}
}

func TestSeerrMetadataDoesNotFollowCrossHostRedirect(t *testing.T) {
	var redirected bool
	redirectTarget := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		redirected = true
		if got := r.Header.Get(seerrAPIKeyHeader); got != "" {
			t.Errorf("redirected request received API key %q", got)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id":603}`))
	}))
	defer redirectTarget.Close()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Location", redirectTarget.URL+"/api/v1/movie/603")
		w.WriteHeader(http.StatusFound)
	}))
	defer server.Close()

	service := newTestSeerrMetadataService(t, server)
	_, err := service.movieMetadata(t.Context(), 603)
	if !errors.Is(err, ErrSeerrUnavailable) {
		t.Fatalf("movieMetadata() error = %v, want ErrSeerrUnavailable", err)
	}
	if redirected {
		t.Fatal("metadata request followed a cross-host redirect")
	}
}

func TestSeerrMetadataPropagatesRequestContext(t *testing.T) {
	var requestReceived bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requestReceived = true
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"results":[]}`))
	}))
	defer server.Close()

	service := newTestSeerrMetadataService(t, server)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	_, err := service.searchMetadata(ctx, "The Matrix")
	if !errors.Is(err, ErrSeerrUnavailable) || !errors.Is(err, context.Canceled) {
		t.Fatalf("searchMetadata() error = %v, want ErrSeerrUnavailable and context.Canceled", err)
	}
	if requestReceived {
		t.Fatal("metadata request was sent with a canceled context")
	}
}

func TestSeerrMetadataRequiresConfiguration(t *testing.T) {
	service, err := NewSeerrService(models.SeerrIntegration{}, nil)
	if err != nil {
		t.Fatalf("NewSeerrService() error = %v", err)
	}
	if _, err := service.searchMetadata(t.Context(), "The Matrix"); !errors.Is(err, ErrSeerrNotConfigured) {
		t.Fatalf("searchMetadata() error = %v, want ErrSeerrNotConfigured", err)
	}
}

func newTestSeerrMetadataService(t *testing.T, server *httptest.Server) *SeerrService {
	t.Helper()
	service, err := NewSeerrService(models.SeerrIntegration{BaseURL: server.URL, APIKey: "test-key"}, server.Client())
	if err != nil {
		t.Fatalf("NewSeerrService() error = %v", err)
	}
	return service
}

func newUnconfiguredTestSeerrService() *SeerrService {
	service, err := NewSeerrService(models.SeerrIntegration{}, nil)
	if err != nil {
		panic(err)
	}
	return service
}
