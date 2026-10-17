package services

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

const seerrMetadataResponseLimit = 8 << 20

type seerrSearchResponse struct {
	TotalResults int                 `json:"totalResults"`
	Results      []seerrSearchResult `json:"results"`
}

type seerrMovieResultsResponse struct {
	Results []seerrSearchResult `json:"results"`
}

type seerrSearchResult struct {
	ID            int64   `json:"id"`
	MediaType     string  `json:"mediaType"`
	Title         string  `json:"title"`
	OriginalTitle string  `json:"originalTitle"`
	Name          string  `json:"name"`
	PosterPath    string  `json:"posterPath"`
	ProfilePath   string  `json:"profilePath"`
	ReleaseDate   string  `json:"releaseDate"`
	VoteAverage   float32 `json:"voteAverage"`
}

type seerrMovieMetadata struct {
	ID               int64             `json:"id"`
	BackdropPath     string            `json:"backdropPath"`
	Budget           int64             `json:"budget"`
	Genres           []seerrGenre      `json:"genres"`
	Homepage         string            `json:"homepage"`
	IMDbID           string            `json:"imdbId"`
	OriginalLanguage string            `json:"originalLanguage"`
	OriginalTitle    string            `json:"originalTitle"`
	Overview         string            `json:"overview"`
	Popularity       float32           `json:"popularity"`
	PosterPath       string            `json:"posterPath"`
	ReleaseDate      string            `json:"releaseDate"`
	Revenue          int64             `json:"revenue"`
	Runtime          int               `json:"runtime"`
	Status           string            `json:"status"`
	Tagline          string            `json:"tagline"`
	Title            string            `json:"title"`
	VoteAverage      float32           `json:"voteAverage"`
	VoteCount        int64             `json:"voteCount"`
	Credits          seerrMovieCredits `json:"credits"`
}

type seerrGenre struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type seerrMovieCredits struct {
	Cast []seerrMovieCast `json:"cast"`
	Crew []seerrMovieCrew `json:"crew"`
}

type seerrMovieCast struct {
	ID          int64  `json:"id"`
	CastID      int64  `json:"castId"`
	CreditID    string `json:"creditId"`
	Character   string `json:"character"`
	Gender      int64  `json:"gender"`
	Name        string `json:"name"`
	Order       int64  `json:"order"`
	ProfilePath string `json:"profilePath"`
}

type seerrMovieCrew struct {
	ID          int64  `json:"id"`
	CreditID    string `json:"creditId"`
	Gender      int64  `json:"gender"`
	Name        string `json:"name"`
	Job         string `json:"job"`
	Department  string `json:"department"`
	ProfilePath string `json:"profilePath"`
}

type seerrPersonMetadata struct {
	ID           int64  `json:"id"`
	Name         string `json:"name"`
	Birthday     string `json:"birthday"`
	Deathday     string `json:"deathday"`
	Biography    string `json:"biography"`
	PlaceOfBirth string `json:"placeOfBirth"`
	ProfilePath  string `json:"profilePath"`
	IMDbID       string `json:"imdbId"`
	Homepage     string `json:"homepage"`
}

type seerrPersonCombinedCredits struct {
	ID   int64                   `json:"id"`
	Cast []seerrPersonCastCredit `json:"cast"`
	Crew []seerrPersonCrewCredit `json:"crew"`
}

type seerrPersonCredit struct {
	ID               int64   `json:"id"`
	OriginalLanguage string  `json:"originalLanguage"`
	EpisodeCount     int     `json:"episodeCount"`
	OriginalName     string  `json:"originalName"`
	VoteCount        int64   `json:"voteCount"`
	Name             string  `json:"name"`
	MediaType        string  `json:"mediaType"`
	Popularity       float32 `json:"popularity"`
	CreditID         string  `json:"creditId"`
	BackdropPath     string  `json:"backdropPath"`
	FirstAirDate     string  `json:"firstAirDate"`
	VoteAverage      float32 `json:"voteAverage"`
	PosterPath       string  `json:"posterPath"`
	OriginalTitle    string  `json:"originalTitle"`
	Title            string  `json:"title"`
	ReleaseDate      string  `json:"releaseDate"`
}

type seerrPersonCastCredit struct {
	seerrPersonCredit
	Character string `json:"character"`
}

type seerrPersonCrewCredit struct {
	seerrPersonCredit
	Department string `json:"department"`
	Job        string `json:"job"`
}

func (s *SeerrService) searchMetadata(ctx context.Context, query string) (*seerrSearchResponse, error) {
	var result seerrSearchResponse
	queryValues := url.Values{"query": {query}}
	if err := s.getMetadata(ctx, "search", queryValues, "search metadata", &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (s *SeerrService) movieMetadata(ctx context.Context, movieID int64) (*seerrMovieMetadata, error) {
	if movieID <= 0 {
		return nil, fmt.Errorf("invalid TMDB movie ID %d", movieID)
	}
	var result seerrMovieMetadata
	if err := s.getMetadata(ctx, fmt.Sprintf("movie/%d", movieID), nil, "get movie metadata", &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (s *SeerrService) movieRecommendations(ctx context.Context, movieID int64) (*seerrMovieResultsResponse, error) {
	return s.movieResults(ctx, movieID, "recommendations")
}

func (s *SeerrService) similarMovies(ctx context.Context, movieID int64) (*seerrMovieResultsResponse, error) {
	return s.movieResults(ctx, movieID, "similar")
}

func (s *SeerrService) movieResults(ctx context.Context, movieID int64, relationship string) (*seerrMovieResultsResponse, error) {
	if movieID <= 0 {
		return nil, fmt.Errorf("invalid TMDB movie ID %d", movieID)
	}

	var result seerrMovieResultsResponse
	endpoint := fmt.Sprintf("movie/%d/%s", movieID, relationship)
	if err := s.getMetadata(ctx, endpoint, nil, "get movie "+relationship, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (s *SeerrService) personMetadata(ctx context.Context, personID int64) (*seerrPersonMetadata, error) {
	if personID <= 0 {
		return nil, fmt.Errorf("invalid TMDB person ID %d", personID)
	}
	var result seerrPersonMetadata
	if err := s.getMetadata(ctx, fmt.Sprintf("person/%d", personID), nil, "get person metadata", &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (s *SeerrService) personCombinedCredits(ctx context.Context, personID int64) (*seerrPersonCombinedCredits, error) {
	if personID <= 0 {
		return nil, fmt.Errorf("invalid TMDB person ID %d", personID)
	}
	var result seerrPersonCombinedCredits
	endpoint := fmt.Sprintf("person/%d/combined_credits", personID)
	if err := s.getMetadata(ctx, endpoint, nil, "get person credits", &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (s *SeerrService) getMetadata(ctx context.Context, endpoint string, query url.Values, action string, result any) error {
	if s.integration.BaseURL == "" {
		return ErrSeerrNotConfigured
	}

	requestURL, err := seerrEndpoint(s.integration.BaseURL, endpoint)
	if err != nil {
		return err
	}
	parsedURL, err := url.Parse(requestURL)
	if err != nil {
		return fmt.Errorf("parse Seerr metadata URL: %w", err)
	}
	parsedURL.RawQuery = query.Encode()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, parsedURL.String(), nil)
	if err != nil {
		return fmt.Errorf("create Seerr %s request: %w", action, err)
	}
	request.Header.Set(seerrAPIKeyHeader, s.integration.APIKey)
	request.Header.Set("Accept", "application/json")

	response, err := s.client.Do(request)
	if err != nil {
		return fmt.Errorf("%s: %w: %w", action, ErrSeerrUnavailable, err)
	}
	defer func() { _ = response.Body.Close() }()

	if response.StatusCode != http.StatusOK {
		return seerrResponseError(action, response.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(response.Body, seerrMetadataResponseLimit+1))
	if err != nil {
		return fmt.Errorf("read Seerr %s response: %w: %w", action, ErrSeerrUnavailable, err)
	}
	if len(body) > seerrMetadataResponseLimit {
		return fmt.Errorf("%s: %w: response exceeds %d bytes", action, ErrSeerrUnavailable, seerrMetadataResponseLimit)
	}
	if !strings.HasPrefix(strings.TrimSpace(string(body)), "{") {
		return fmt.Errorf("%s: %w: response must be a JSON object", action, ErrSeerrUnavailable)
	}
	if err := json.Unmarshal(body, result); err != nil {
		return fmt.Errorf("decode Seerr %s response: %w: %w", action, ErrSeerrUnavailable, err)
	}
	return nil
}
