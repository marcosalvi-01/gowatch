// Package services contains business logic services for managing movies, watched lists, and user data.
package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"time"

	"github.com/marcosalvi-01/gowatch/db"
	"github.com/marcosalvi-01/gowatch/internal/models"
	"github.com/marcosalvi-01/gowatch/logging"
)

type MovieService struct {
	seerr    *SeerrService
	db       db.DB
	log      *slog.Logger
	cacheTTL time.Duration
}

const (
	unknownYearSecondaryText = "Unknown year"
	untitledMovieTitle       = "Untitled movie"
	unknownPersonName        = "Unknown person"
	untitledPersonCredit     = "Untitled credit"
	unknownPersonRole        = "Role unavailable"
	metadataDateLayout       = "2006-01-02"
	personCreditMinimumVotes = 100
	personCreditMediaTypeTV  = "tv"
	personCreditMediumRoleEP = 2
	personCreditHighRoleEP   = 5
)

func NewMovieService(db db.DB, seerr *SeerrService, cacheTTL time.Duration) *MovieService {
	if db == nil {
		panic("database is required")
	}
	if seerr == nil {
		panic("seerr service is required")
	}

	log := logging.Get("movie service")
	return &MovieService{
		seerr:    seerr,
		db:       db,
		log:      log,
		cacheTTL: cacheTTL,
	}
}

func (s *MovieService) SearchMulti(ctx context.Context, query string) ([]models.SearchResult, error) {
	s.log.Debug("searching movies and people", "query", query)

	search, err := s.seerr.searchMetadata(ctx, query)
	if err != nil {
		s.log.Error("Seerr multi search failed", "query", query, "error", err)
		return nil, fmt.Errorf("error searching Seerr for query %q: %w", query, err)
	}

	results := make([]models.SearchResult, 0, len(search.Results))
	movieCount := 0
	personCount := 0

	for _, result := range search.Results {
		switch result.MediaType {
		case string(models.SearchResultTypeMovie):
			results = append(results, mapSearchMovieToSearchResult(
				result.ID,
				result.Title,
				result.OriginalTitle,
				result.PosterPath,
				result.ReleaseDate,
				result.VoteAverage,
			))
			movieCount++
		case string(models.SearchResultTypePerson):
			results = append(results, mapSearchPersonToSearchResult(
				result.ID,
				result.Name,
				result.ProfilePath,
			))
			personCount++
		}
	}

	s.log.Info(
		"multi search completed",
		"query", query,
		"seerrResultCount", search.TotalResults,
		"movieCount", movieCount,
		"personCount", personCount,
		"resultCount", len(results),
	)

	return results, nil
}

func mapSearchMovieToSearchResult(
	id int64,
	title string,
	originalTitle string,
	posterPath string,
	releaseDate string,
	voteAverage float32,
) models.SearchResult {
	searchTitle := title
	if searchTitle == "" {
		searchTitle = originalTitle
	}
	if searchTitle == "" {
		searchTitle = untitledMovieTitle
	}

	secondaryText := unknownYearSecondaryText
	if len(releaseDate) >= 4 {
		secondaryText = releaseDate[:4]
	}

	return models.SearchResult{
		ID:            id,
		Title:         searchTitle,
		ImagePath:     posterPath,
		SecondaryText: secondaryText,
		VoteAverage:   voteAverage,
		Type:          models.SearchResultTypeMovie,
	}
}

func mapSearchPersonToSearchResult(
	id int64,
	name string,
	profilePath string,
) models.SearchResult {
	searchName := name
	if searchName == "" {
		searchName = unknownPersonName
	}

	return models.SearchResult{
		ID:        id,
		Title:     searchName,
		ImagePath: profilePath,
		Type:      models.SearchResultTypePerson,
	}
}

func (s *MovieService) GetMovieDetails(ctx context.Context, id int64) (*models.MovieDetails, error) {
	s.log.Debug("getting movie details", "movieID", id)

	movie, err := s.db.GetMovieDetailsByID(ctx, id)
	if err == nil && !movie.Movie.UpdatedAt.IsZero() && time.Since(movie.Movie.UpdatedAt) <= s.cacheTTL {
		s.log.Info("movie details cache hit", "movieID", id, "title", movie.Movie.Title)
		return movie, nil
	}

	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		s.log.Error("failed to get movie details from database. Fetching from Seerr", "movieID", id, "error", err)
	}

	s.log.Info("movie details cache miss, fetching from Seerr", "movieID", id)

	details, err := s.seerr.movieMetadata(ctx, id)
	if err != nil {
		s.log.Error("failed to get movie details from Seerr", "movieID", id, "error", err)
		return nil, fmt.Errorf("get Seerr movie details for id %d: %w", id, err)
	}

	movie, err = mapSeerrMovieMetadataToMovieDetails(*details)
	if err != nil {
		s.log.Error("failed to convert Seerr movie details to internal model", "movieID", id, "error", err)
		return nil, fmt.Errorf("convert Seerr movie details: %w", err)
	}

	s.log.Debug("successfully mapped Seerr movie details", "movieID", id, "title", movie.Movie.Title)

	err = s.db.UpsertMovie(ctx, movie)
	if err != nil {
		s.log.Error("failed to save movie to database", "movieID", id, "error", err)
		return nil, fmt.Errorf("failed to save movie in db: %w", err)
	}

	s.log.Info("successfully cached movie details", "movieID", id, "title", movie.Movie.Title)
	return movie, nil
}

func (s *MovieService) GetMovieRecommendations(ctx context.Context, id int64) ([]models.SearchResult, error) {
	return s.getRelatedMovies(ctx, id, db.MovieRelationshipRecommendations, s.seerr.movieRecommendations)
}

func (s *MovieService) GetSimilarMovies(ctx context.Context, id int64) ([]models.SearchResult, error) {
	return s.getRelatedMovies(ctx, id, db.MovieRelationshipSimilar, s.seerr.similarMovies)
}

func (s *MovieService) getRelatedMovies(ctx context.Context, id int64, kind string, fetch func(context.Context, int64) (*seerrMovieResultsResponse, error)) ([]models.SearchResult, error) {
	if id <= 0 {
		return nil, fmt.Errorf("invalid TMDB movie ID %d", id)
	}
	movies, updatedAt, err := s.db.GetRelatedMovies(ctx, id, kind)
	if err == nil && !updatedAt.IsZero() && time.Since(updatedAt) <= s.cacheTTL {
		return mapMoviesToSearchResults(movies), nil
	}
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		s.log.Error("failed to read related movies, fetching from Seerr", "movieID", id, "kind", kind, "error", err)
	}
	results, err := fetch(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get Seerr movie %s for id %d: %w", kind, id, err)
	}
	movies = make([]models.Movie, len(results.Results))
	for i, result := range results.Results {
		movies[i] = models.Movie{
			ID:            result.ID,
			Title:         result.Title,
			OriginalTitle: result.OriginalTitle,
			ReleaseDate:   parseMetadataDate(result.ReleaseDate),
			PosterPath:    result.PosterPath,
			BackdropPath:  result.BackdropPath,
			Popularity:    result.Popularity,
			VoteCount:     result.VoteCount,
			VoteAverage:   result.VoteAverage,
		}
	}
	if err := s.db.UpsertRelatedMovies(ctx, id, kind, movies); err != nil {
		return nil, fmt.Errorf("save movie %d %s: %w", id, kind, err)
	}
	return mapMoviesToSearchResults(movies), nil
}

func (s *MovieService) GetMovieRatings(ctx context.Context, id int64) (*models.MovieRatings, error) {
	if id <= 0 {
		return nil, fmt.Errorf("invalid TMDB movie ID %d", id)
	}
	ratings, updatedAt, err := s.db.GetMovieRatings(ctx, id)
	if err == nil && !updatedAt.IsZero() && time.Since(updatedAt) <= s.cacheTTL {
		return ratings, nil
	}
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		s.log.Error("failed to read movie ratings, fetching from Seerr", "movieID", id, "error", err)
	}
	ratings, err = s.seerr.MovieRatings(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get Seerr movie ratings for id %d: %w", id, err)
	}
	if err := s.db.UpsertMovieRatings(ctx, id, ratings); err != nil {
		return nil, fmt.Errorf("save movie %d ratings: %w", id, err)
	}
	return ratings, nil
}

func mapMoviesToSearchResults(results []models.Movie) []models.SearchResult {
	movies := make([]models.SearchResult, len(results))
	for i, result := range results {
		var releaseDate string
		if result.ReleaseDate != nil {
			releaseDate = result.ReleaseDate.Format(metadataDateLayout)
		}
		movies[i] = mapSearchMovieToSearchResult(
			result.ID,
			result.Title,
			result.OriginalTitle,
			result.PosterPath,
			releaseDate,
			result.VoteAverage,
		)
	}
	return movies
}

func mapSeerrMovieMetadataToMovieDetails(details seerrMovieMetadata) (*models.MovieDetails, error) {
	var releaseDate *time.Time
	if details.ReleaseDate != "" {
		parsedDate, err := time.Parse(metadataDateLayout, details.ReleaseDate)
		if err != nil {
			return nil, fmt.Errorf("parse release date %q: %w", details.ReleaseDate, err)
		}
		releaseDate = &parsedDate
	}

	movie := &models.MovieDetails{
		Movie: models.Movie{
			ID:               details.ID,
			Title:            details.Title,
			OriginalTitle:    details.OriginalTitle,
			OriginalLanguage: details.OriginalLanguage,
			Overview:         details.Overview,
			ReleaseDate:      releaseDate,
			PosterPath:       details.PosterPath,
			BackdropPath:     details.BackdropPath,
			Popularity:       details.Popularity,
			VoteCount:        details.VoteCount,
			VoteAverage:      details.VoteAverage,
		},
		Homepage:            details.Homepage,
		IMDbID:              details.IMDbID,
		Budget:              details.Budget,
		Revenue:             details.Revenue,
		Runtime:             details.Runtime,
		Status:              details.Status,
		Tagline:             details.Tagline,
		Genres:              make([]models.Genre, len(details.Genres)),
		RelatedVideos:       make([]models.RelatedVideo, len(details.RelatedVideos)),
		ProductionCompanies: make([]models.ProductionCompany, len(details.ProductionCompanies)),
		ProductionCountries: make([]models.ProductionCountry, len(details.ProductionCountries)),
		Credits: models.MovieCredits{
			Cast: make([]models.Cast, len(details.Credits.Cast)),
			Crew: make([]models.Crew, len(details.Credits.Crew)),
		},
	}

	for i, genre := range details.Genres {
		movie.Genres[i] = models.Genre{ID: genre.ID, Name: genre.Name}
	}
	for i, video := range details.RelatedVideos {
		movie.RelatedVideos[i] = models.RelatedVideo{
			URL:  video.URL,
			Key:  video.Key,
			Name: video.Name,
			Size: video.Size,
			Type: video.Type,
			Site: video.Site,
		}
	}
	for i, company := range details.ProductionCompanies {
		movie.ProductionCompanies[i] = models.ProductionCompany{
			ID:            company.ID,
			LogoPath:      company.LogoPath,
			OriginCountry: company.OriginCountry,
			Name:          company.Name,
		}
	}
	for i, country := range details.ProductionCountries {
		movie.ProductionCountries[i] = models.ProductionCountry{
			ISOCode: country.ISOCode,
			Name:    country.Name,
		}
	}
	for i, cast := range details.Credits.Cast {
		movie.Credits.Cast[i] = models.Cast{
			MovieID:   details.ID,
			PersonID:  cast.ID,
			CastID:    cast.CastID,
			CreditID:  cast.CreditID,
			Character: cast.Character,
			CastOrder: cast.Order,
			Person: models.Person{
				ID:          cast.ID,
				Name:        cast.Name,
				ProfilePath: cast.ProfilePath,
				Gender:      cast.Gender,
			},
		}
	}
	for i, crew := range details.Credits.Crew {
		movie.Credits.Crew[i] = models.Crew{
			MovieID:    details.ID,
			PersonID:   crew.ID,
			CreditID:   crew.CreditID,
			Job:        crew.Job,
			Department: crew.Department,
			Person: models.Person{
				ID:          crew.ID,
				Name:        crew.Name,
				ProfilePath: crew.ProfilePath,
				Gender:      crew.Gender,
			},
		}
	}

	return movie, nil
}

func (s *MovieService) GetPersonDetails(ctx context.Context, id int64) (*models.PersonDetailsPage, error) {
	if id <= 0 {
		return nil, fmt.Errorf("invalid TMDB person ID %d", id)
	}
	if err := ctx.Err(); err != nil {
		s.log.Error("request context canceled before person details fetch", "personID", id, "error", err)
		return nil, fmt.Errorf("request context canceled before person details fetch: %w", err)
	}

	s.log.Debug("getting person details", "personID", id)

	person, updatedAt, err := s.db.GetPersonDetails(ctx, id)
	if err == nil && !updatedAt.IsZero() && time.Since(updatedAt) <= s.cacheTTL {
		preparePersonFilmography(person)
		return person, nil
	}
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		s.log.Error("failed to read person details, fetching from Seerr", "personID", id, "error", err)
	}

	details, err := s.seerr.personMetadata(ctx, id)
	if err != nil {
		s.log.Error("failed to get person details from Seerr", "personID", id, "error", err)
		return nil, fmt.Errorf("get Seerr person details for id %d: %w", id, err)
	}

	credits, err := s.seerr.personCombinedCredits(ctx, id)
	if err != nil {
		s.log.Error("failed to get person combined credits from Seerr", "personID", id, "error", err)
		return nil, fmt.Errorf("get Seerr person combined credits for id %d: %w", id, err)
	}

	person = mapSeerrPersonDetailsToPersonDetailsPage(*details, credits)
	if err := s.db.UpsertPersonDetails(ctx, person); err != nil {
		return nil, fmt.Errorf("save person %d details: %w", id, err)
	}

	s.log.Info(
		"person details fetched successfully",
		"personID", id,
		"name", person.Name,
		"actingCreditCount", len(person.ActingCredits),
		"crewCreditCount", len(person.CrewCredits),
	)

	return person, nil
}

func mapSeerrPersonDetailsToPersonDetailsPage(
	details seerrPersonMetadata,
	credits *seerrPersonCombinedCredits,
) *models.PersonDetailsPage {
	person := &models.PersonDetailsPage{
		ID:            details.ID,
		Name:          personDisplayName(details.Name),
		Biography:     strings.TrimSpace(details.Biography),
		ProfilePath:   details.ProfilePath,
		Birthday:      parseMetadataDate(details.Birthday),
		Deathday:      parseMetadataDate(details.Deathday),
		PlaceOfBirth:  strings.TrimSpace(details.PlaceOfBirth),
		IMDbID:        strings.TrimSpace(details.IMDbID),
		Homepage:      strings.TrimSpace(details.Homepage),
		KnownFor:      []models.PersonCredit{},
		ActingCredits: []models.PersonCredit{},
		CrewCredits:   []models.PersonCredit{},
	}

	if credits == nil {
		return person
	}

	person.ActingCredits = make([]models.PersonCredit, 0, len(credits.Cast))
	for _, credit := range credits.Cast {
		person.ActingCredits = append(person.ActingCredits, mapPersonCastCreditToPersonCredit(
			credit.ID,
			credit.CreditID,
			credit.MediaType,
			credit.Title,
			credit.Name,
			credit.OriginalTitle,
			credit.OriginalName,
			credit.Character,
			credit.ReleaseDate,
			credit.FirstAirDate,
			credit.EpisodeCount,
			credit.BackdropPath,
			credit.PosterPath,
			credit.VoteAverage,
			credit.VoteCount,
			credit.Popularity,
		))
	}

	person.CrewCredits = make([]models.PersonCredit, 0, len(credits.Crew))
	for _, credit := range credits.Crew {
		person.CrewCredits = append(person.CrewCredits, mapPersonCrewCreditToPersonCredit(
			credit.ID,
			credit.CreditID,
			credit.MediaType,
			credit.Title,
			credit.Name,
			credit.OriginalTitle,
			credit.OriginalName,
			credit.Job,
			credit.Department,
			credit.ReleaseDate,
			credit.FirstAirDate,
			credit.EpisodeCount,
			credit.BackdropPath,
			credit.PosterPath,
			credit.VoteAverage,
			credit.VoteCount,
			credit.Popularity,
		))
	}

	preparePersonFilmography(person)
	return person
}

func preparePersonFilmography(person *models.PersonDetailsPage) {
	sortPersonCreditsByVoteAverageDesc(person.ActingCredits)
	sortPersonCreditsByVoteAverageDesc(person.CrewCredits)

	person.KnownFor = person.ActingCredits
	if len(person.KnownFor) == 0 {
		person.KnownFor = person.CrewCredits
	}
}

func mapPersonCastCreditToPersonCredit(
	id int64,
	creditID string,
	mediaType string,
	title string,
	name string,
	originalTitle string,
	originalName string,
	character string,
	releaseDate string,
	firstAirDate string,
	episodeCount int,
	backdropPath string,
	posterPath string,
	voteAverage float32,
	voteCount int64,
	popularity float32,
) models.PersonCredit {
	return models.PersonCredit{
		ID:           id,
		CreditID:     creditID,
		MediaType:    strings.TrimSpace(mediaType),
		Title:        personCreditDisplayTitle(title, name, originalTitle, originalName),
		Role:         personCreditDisplayRole(character),
		EpisodeCount: episodeCount,
		ReleaseDate:  firstKnownDate(releaseDate, firstAirDate),
		BackdropPath: strings.TrimSpace(backdropPath),
		PosterPath:   strings.TrimSpace(posterPath),
		VoteAverage:  voteAverage,
		VoteCount:    voteCount,
		Popularity:   popularity,
	}
}

func mapPersonCrewCreditToPersonCredit(
	id int64,
	creditID string,
	mediaType string,
	title string,
	name string,
	originalTitle string,
	originalName string,
	job string,
	department string,
	releaseDate string,
	firstAirDate string,
	episodeCount int,
	backdropPath string,
	posterPath string,
	voteAverage float32,
	voteCount int64,
	popularity float32,
) models.PersonCredit {
	return models.PersonCredit{
		ID:           id,
		CreditID:     creditID,
		MediaType:    strings.TrimSpace(mediaType),
		Title:        personCreditDisplayTitle(title, name, originalTitle, originalName),
		Role:         personCreditDisplayRole(job),
		Department:   strings.TrimSpace(department),
		EpisodeCount: episodeCount,
		ReleaseDate:  firstKnownDate(releaseDate, firstAirDate),
		BackdropPath: strings.TrimSpace(backdropPath),
		PosterPath:   strings.TrimSpace(posterPath),
		VoteAverage:  voteAverage,
		VoteCount:    voteCount,
		Popularity:   popularity,
	}
}

func parseMetadataDate(value string) *time.Time {
	trimmedValue := strings.TrimSpace(value)
	if trimmedValue == "" {
		return nil
	}

	parsedDate, err := time.Parse(metadataDateLayout, trimmedValue)
	if err != nil {
		return nil
	}

	return &parsedDate
}

func firstKnownDate(primaryDate, fallbackDate string) *time.Time {
	if parsedPrimaryDate := parseMetadataDate(primaryDate); parsedPrimaryDate != nil {
		return parsedPrimaryDate
	}

	return parseMetadataDate(fallbackDate)
}

func personDisplayName(name string) string {
	displayName := strings.TrimSpace(name)
	if displayName == "" {
		return unknownPersonName
	}

	return displayName
}

func personCreditDisplayTitle(title, fallbackTitle, originalTitle, originalFallbackTitle string) string {
	possibleTitles := []string{title, fallbackTitle, originalTitle, originalFallbackTitle}
	for _, possibleTitle := range possibleTitles {
		if trimmedTitle := strings.TrimSpace(possibleTitle); trimmedTitle != "" {
			return trimmedTitle
		}
	}

	return untitledPersonCredit
}

func personCreditDisplayRole(role string) string {
	displayRole := strings.TrimSpace(role)
	if displayRole == "" {
		return unknownPersonRole
	}

	return displayRole
}

func sortPersonCreditsByVoteAverageDesc(credits []models.PersonCredit) {
	if len(credits) < 2 {
		return
	}

	sort.SliceStable(credits, func(i, j int) bool {
		leftCredit := credits[i]
		rightCredit := credits[j]

		leftHasEnoughVotes := personCreditHasEnoughVotes(leftCredit)
		rightHasEnoughVotes := personCreditHasEnoughVotes(rightCredit)

		if leftHasEnoughVotes != rightHasEnoughVotes {
			return leftHasEnoughVotes
		}

		if !leftHasEnoughVotes {
			return false
		}

		leftInvolvementTier := personCreditInvolvementTier(leftCredit)
		rightInvolvementTier := personCreditInvolvementTier(rightCredit)
		if leftInvolvementTier != rightInvolvementTier {
			return leftInvolvementTier > rightInvolvementTier
		}

		if leftCredit.VoteAverage != rightCredit.VoteAverage {
			return leftCredit.VoteAverage > rightCredit.VoteAverage
		}

		if leftCredit.VoteCount != rightCredit.VoteCount {
			return leftCredit.VoteCount > rightCredit.VoteCount
		}

		if leftCredit.EpisodeCount != rightCredit.EpisodeCount {
			return leftCredit.EpisodeCount > rightCredit.EpisodeCount
		}

		return false
	})
}

func personCreditHasEnoughVotes(credit models.PersonCredit) bool {
	return credit.VoteCount >= personCreditMinimumVotes
}

func personCreditInvolvementTier(credit models.PersonCredit) int {
	if !strings.EqualFold(credit.MediaType, personCreditMediaTypeTV) {
		return 2
	}

	switch {
	case credit.EpisodeCount >= personCreditHighRoleEP:
		return 2
	case credit.EpisodeCount >= personCreditMediumRoleEP:
		return 1
	default:
		return 0
	}
}
