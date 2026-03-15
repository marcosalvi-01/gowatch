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

	tmdb "github.com/cyruzin/golang-tmdb"
)

type MovieService struct {
	client   *tmdb.Client
	db       db.DB
	log      *slog.Logger
	cacheTTL time.Duration
}

const (
	unknownYearSecondaryText = "Unknown year"
	unknownPopularityText    = "Popularity unknown"
	popularityPrefix         = "Popularity "
	untitledMovieTitle       = "Untitled movie"
	unknownPersonName        = "Unknown person"
	untitledPersonCredit     = "Untitled credit"
	unknownPersonRole        = "Role unavailable"
	tmdbDateLayout           = "2006-01-02"
	personCreditMinimumVotes = 100
	personCreditMediaTypeTV  = "tv"
	personCreditMediumRoleEP = 2
	personCreditHighRoleEP   = 5
)

func NewMovieService(db db.DB, client *tmdb.Client, cacheTTL time.Duration) *MovieService {
	log := logging.Get("movie service")
	return &MovieService{
		client:   client,
		db:       db,
		log:      log,
		cacheTTL: cacheTTL,
	}
}

func (s *MovieService) SearchMulti(query string) ([]models.SearchResult, error) {
	s.log.Debug("searching movies and people", "query", query)

	search, err := s.client.GetSearchMulti(query, nil)
	if err != nil {
		s.log.Error("TMDB multi search failed", "query", query, "error", err)
		return nil, fmt.Errorf("error searching TMDB multi for query '%s': %w", query, err)
	}

	results := make([]models.SearchResult, 0, len(search.Results))
	movieCount := 0
	personCount := 0

	for _, result := range search.Results {
		switch result.MediaType {
		case string(models.SearchResultTypeMovie):
			results = append(results, mapTMDBSearchMultiMovieToSearchResult(
				result.ID,
				result.Title,
				result.OriginalTitle,
				result.PosterPath,
				result.ReleaseDate,
				result.VoteAverage,
			))
			movieCount++
		case string(models.SearchResultTypePerson):
			results = append(results, mapTMDBSearchMultiPersonToSearchResult(
				result.ID,
				result.Name,
				result.OriginalName,
				result.ProfilePath,
				result.Popularity,
			))
			personCount++
		}
	}

	s.log.Info(
		"multi search completed",
		"query", query,
		"tmdbResultCount", search.TotalResults,
		"movieCount", movieCount,
		"personCount", personCount,
		"resultCount", len(results),
	)

	return results, nil
}

func mapTMDBSearchMultiMovieToSearchResult(
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

func mapTMDBSearchMultiPersonToSearchResult(
	id int64,
	name string,
	originalName string,
	profilePath string,
	popularity float32,
) models.SearchResult {
	searchName := name
	if searchName == "" {
		searchName = originalName
	}
	if searchName == "" {
		searchName = unknownPersonName
	}

	secondaryText := formatPersonPopularity(popularity)

	return models.SearchResult{
		ID:            id,
		Title:         searchName,
		ImagePath:     profilePath,
		SecondaryText: secondaryText,
		Type:          models.SearchResultTypePerson,
	}
}

func formatPersonPopularity(popularity float32) string {
	if popularity <= 0 {
		return unknownPopularityText
	}

	return fmt.Sprintf("%s%.1f", popularityPrefix, popularity)
}

func (s *MovieService) GetMovieDetails(ctx context.Context, id int64) (*models.MovieDetails, error) {
	s.log.Debug("getting movie details", "movieID", id)

	movie, err := s.db.GetMovieDetailsByID(ctx, id)
	if err == nil && time.Since(movie.Movie.UpdatedAt) <= s.cacheTTL {
		s.log.Info("movie details cache hit", "movieID", id, "title", movie.Movie.Title)
		return movie, nil
	}

	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		s.log.Error("failed to get movie details from database. Fetching from TMDB", "movieID", id, "error", err)
	}

	if s.client == nil {
		err := fmt.Errorf("tmdb client not configured")
		s.log.Error("tmdb client not configured", "movieID", id)
		return nil, err
	}

	s.log.Info("movie details cache miss, fetching from TMDB", "movieID", id)

	// cache miss
	details, err := s.client.GetMovieDetails(int(id), nil)
	if err != nil {
		s.log.Error("failed to get movie details from TMDB", "movieID", id, "error", err)
		return nil, fmt.Errorf("error getting TMDB movie details for id '%d': %w", id, err)
	}

	s.log.Debug("successfully fetched movie details from TMDB", "movieID", id, "title", details.Title)

	movie, err = models.MovieDetailsFromTMDBMovieDetails(*details)
	if err != nil {
		s.log.Error("failed to convert TMDB movie details to internal model", "movieID", id, "error", err)
		return nil, fmt.Errorf("failed to convert from TMDB results to internal model: %w", err)
	}

	// remember the credits
	credits, err := s.getMovieCredits(id)
	if err != nil {
		s.log.Error("failed to get movie credits", "movieID", id, "error", err)
		return nil, fmt.Errorf("failed to get movie credits: %w", err)
	}

	movie.Credits = credits

	err = s.db.UpsertMovie(ctx, movie)
	if err != nil {
		s.log.Error("failed to save movie to database", "movieID", id, "error", err)
		return nil, fmt.Errorf("failed to save movie in db: %w", err)
	}

	s.log.Info("successfully cached movie details", "movieID", id, "title", movie.Movie.Title)
	return movie, nil
}

func (s *MovieService) GetPersonDetails(ctx context.Context, id int64) (*models.PersonDetailsPage, error) {
	if err := ctx.Err(); err != nil {
		s.log.Error("request context canceled before person details fetch", "personID", id, "error", err)
		return nil, fmt.Errorf("request context canceled before person details fetch: %w", err)
	}

	s.log.Debug("getting person details", "personID", id)

	if s.client == nil {
		err := fmt.Errorf("tmdb client not configured")
		s.log.Error("tmdb client not configured", "personID", id)
		return nil, err
	}

	details, err := s.client.GetPersonDetails(int(id), nil)
	if err != nil {
		s.log.Error("failed to get person details from TMDB", "personID", id, "error", err)
		return nil, fmt.Errorf("error getting TMDB person details for id '%d': %w", id, err)
	}

	if details == nil {
		err := fmt.Errorf("tmdb person details response was nil for id '%d'", id)
		s.log.Error("received nil person details from TMDB", "personID", id)
		return nil, err
	}

	credits, err := s.client.GetPersonCombinedCredits(int(id), nil)
	if err != nil {
		s.log.Error("failed to get person combined credits from TMDB", "personID", id, "error", err)
		return nil, fmt.Errorf("error getting TMDB person combined credits for id '%d': %w", id, err)
	}

	var tvCredits *tmdb.PersonTVCredits
	tvCredits, err = s.client.GetPersonTVCredits(int(id), nil)
	if err != nil {
		s.log.Warn("failed to get person tv credits from TMDB; continuing without tv episode counts", "personID", id, "error", err)
	}

	person := mapTMDBPersonDetailsToPersonDetailsPage(*details, credits, tvCredits)

	s.log.Info(
		"person details fetched successfully",
		"personID", id,
		"name", person.Name,
		"actingCreditCount", len(person.ActingCredits),
		"crewCreditCount", len(person.CrewCredits),
	)

	return person, nil
}

func mapTMDBPersonDetailsToPersonDetailsPage(
	details tmdb.PersonDetails,
	credits *tmdb.PersonCombinedCredits,
	tvCredits *tmdb.PersonTVCredits,
) *models.PersonDetailsPage {
	person := &models.PersonDetailsPage{
		ID:                 details.ID,
		Name:               personDisplayName(details.Name),
		Biography:          strings.TrimSpace(details.Biography),
		ProfilePath:        details.ProfilePath,
		KnownForDepartment: strings.TrimSpace(details.KnownForDepartment),
		Popularity:         details.Popularity,
		Birthday:           parseTMDBDate(details.Birthday),
		Deathday:           parseTMDBDate(details.Deathday),
		PlaceOfBirth:       strings.TrimSpace(details.PlaceOfBirth),
		IMDbID:             strings.TrimSpace(details.IMDbID),
		Homepage:           strings.TrimSpace(details.Homepage),
		KnownFor:           []models.PersonCredit{},
		ActingCredits:      []models.PersonCredit{},
		CrewCredits:        []models.PersonCredit{},
	}

	if credits == nil {
		return person
	}

	tvCrewMetadataByCreditID, tvCrewMetadataByID := personTVCrewMetadataLookup(tvCredits)

	person.ActingCredits = make([]models.PersonCredit, 0, len(credits.Cast))
	for _, credit := range credits.Cast {
		person.ActingCredits = append(person.ActingCredits, mapTMDBPersonCastCreditToPersonCredit(
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
		tvCrewMetadata := personTVCrewMetadata{}
		if strings.EqualFold(credit.MediaType, personCreditMediaTypeTV) {
			tvCrewMetadata = lookupPersonTVCrewMetadata(
				credit.CreditID,
				credit.ID,
				tvCrewMetadataByCreditID,
				tvCrewMetadataByID,
			)
		}

		person.CrewCredits = append(person.CrewCredits, mapTMDBPersonCrewCreditToPersonCredit(
			credit.ID,
			credit.CreditID,
			credit.MediaType,
			credit.Title,
			tvCrewMetadata.Name,
			credit.OriginalTitle,
			tvCrewMetadata.OriginalName,
			credit.Job,
			credit.Department,
			credit.ReleaseDate,
			tvCrewMetadata.FirstAirDate,
			tvCrewMetadata.EpisodeCount,
			credit.BackdropPath,
			credit.PosterPath,
			credit.VoteAverage,
			credit.VoteCount,
			credit.Popularity,
		))

	}

	sortPersonCreditsByVoteAverageDesc(person.ActingCredits)
	sortPersonCreditsByVoteAverageDesc(person.CrewCredits)

	person.KnownFor = append([]models.PersonCredit{}, person.ActingCredits...)
	if len(person.KnownFor) == 0 {
		person.KnownFor = append([]models.PersonCredit{}, person.CrewCredits...)
	}

	return person
}

func mapTMDBPersonCastCreditToPersonCredit(
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

func mapTMDBPersonCrewCreditToPersonCredit(
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

func parseTMDBDate(value string) *time.Time {
	trimmedValue := strings.TrimSpace(value)
	if trimmedValue == "" {
		return nil
	}

	parsedDate, err := time.Parse(tmdbDateLayout, trimmedValue)
	if err != nil {
		return nil
	}

	return &parsedDate
}

func firstKnownDate(primaryDate, fallbackDate string) *time.Time {
	if parsedPrimaryDate := parseTMDBDate(primaryDate); parsedPrimaryDate != nil {
		return parsedPrimaryDate
	}

	return parseTMDBDate(fallbackDate)
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

type personTVCrewMetadata struct {
	EpisodeCount int
	Name         string
	OriginalName string
	FirstAirDate string
}

func personTVCrewMetadataLookup(tvCredits *tmdb.PersonTVCredits) (map[string]personTVCrewMetadata, map[int64]personTVCrewMetadata) {
	if tvCredits == nil || len(tvCredits.Crew) == 0 {
		return nil, nil
	}

	metadataByCreditID := make(map[string]personTVCrewMetadata, len(tvCredits.Crew))
	metadataByID := make(map[int64]personTVCrewMetadata, len(tvCredits.Crew))

	for _, credit := range tvCredits.Crew {
		metadata := personTVCrewMetadata{
			EpisodeCount: credit.EpisodeCount,
			Name:         credit.Name,
			OriginalName: credit.OriginalName,
			FirstAirDate: credit.FirstAirDate,
		}

		if existingMetadata, exists := metadataByID[credit.ID]; !exists || credit.EpisodeCount > existingMetadata.EpisodeCount {
			metadataByID[credit.ID] = metadata
		}

		creditID := strings.TrimSpace(credit.CreditID)
		if creditID == "" {
			continue
		}

		if existingMetadata, exists := metadataByCreditID[creditID]; !exists || credit.EpisodeCount > existingMetadata.EpisodeCount {
			metadataByCreditID[creditID] = metadata
		}
	}

	return metadataByCreditID, metadataByID
}

func lookupPersonTVCrewMetadata(
	creditID string,
	id int64,
	metadataByCreditID map[string]personTVCrewMetadata,
	metadataByID map[int64]personTVCrewMetadata,
) personTVCrewMetadata {
	trimmedCreditID := strings.TrimSpace(creditID)
	if trimmedCreditID != "" {
		if metadata, exists := metadataByCreditID[trimmedCreditID]; exists {
			return metadata
		}
	}

	if metadata, exists := metadataByID[id]; exists {
		return metadata
	}

	return personTVCrewMetadata{}
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

func (s *MovieService) getMovieCredits(id int64) (models.MovieCredits, error) {
	s.log.Debug("getting movie credits from TMDB", "movieID", id)

	credits, err := s.client.GetMovieCredits(int(id), nil)
	if err != nil {
		s.log.Error("TMDB get movie credits failed", "movieID", id, "error", err)
		return models.MovieCredits{}, fmt.Errorf("error getting TMDB movie credits for id '%d': %w", id, err)
	}

	s.log.Debug("fetched movie credits from TMDB", "movieID", id, "castCount", len(credits.Cast), "crewCount", len(credits.Crew))

	cast := make([]models.Cast, len(credits.Cast))
	for i, c := range credits.Cast {
		cast[i] = models.Cast{
			MovieID:   id,
			PersonID:  c.ID,
			CastID:    c.CastID,
			CreditID:  c.CreditID,
			Character: c.Character,
			CastOrder: int64(c.Order),
			Person: models.Person{
				ID:                 c.ID,
				Name:               c.Name,
				OriginalName:       c.OriginalName,
				ProfilePath:        c.ProfilePath,
				KnownForDepartment: c.KnownForDepartment,
				Popularity:         float64(c.Popularity),
				Gender:             int64(c.Gender),
				Adult:              c.Adult,
			},
		}
	}

	crew := make([]models.Crew, len(credits.Crew))
	for i, c := range credits.Crew {
		crew[i] = models.Crew{
			MovieID:    id,
			PersonID:   c.ID,
			CreditID:   c.CreditID,
			Job:        c.Job,
			Department: c.Department,
			Person: models.Person{
				ID:                 c.ID,
				Name:               c.Name,
				OriginalName:       c.OriginalName,
				ProfilePath:        c.ProfilePath,
				KnownForDepartment: c.KnownForDepartment,
				Popularity:         float64(c.Popularity),
				Gender:             int64(c.Gender),
				Adult:              c.Adult,
			},
		}
	}

	s.log.Debug("converted TMDB credits to internal models", "movieID", id, "castCount", len(cast), "crewCount", len(crew))

	return models.MovieCredits{
		Crew: crew,
		Cast: cast,
	}, nil
}
