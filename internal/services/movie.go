package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"gowatch/db"
	"gowatch/internal/models"
	"gowatch/logging"
	"log/slog"
	"time"

	tmdb "github.com/cyruzin/golang-tmdb"
)

type MovieService struct {
	client   *tmdb.Client
	db       db.DB
	log      *slog.Logger
	cacheTTL time.Duration
}

func NewMovieService(db db.DB, client *tmdb.Client, cacheTTL time.Duration) *MovieService {
	log := logging.Get("movie service")
	return &MovieService{
		client:   client,
		db:       db,
		log:      log,
		cacheTTL: cacheTTL,
	}
}

func (s *MovieService) SearchMovies(query string) ([]models.Movie, error) {
	s.log.Debug("searching movies", "query", query)

	search, err := s.client.GetSearchMovies(query, nil)
	if err != nil {
		s.log.Error("TMDB search failed", "query", query, "error", err)
		return nil, fmt.Errorf("error searching TMDB for query '%s': %w", query, err)
	}

	s.log.Info("movie search completed", "query", query, "results", search.TotalResults)

	movies := make([]models.Movie, len(search.Results))
	for i, m := range search.Results {
		var releaseDate *time.Time
		if m.ReleaseDate != "" {
			date, err := time.Parse("2006-01-02", m.ReleaseDate)
			if err != nil {
				s.log.Error("failed to parse movie release date", "movieID", m.ID, "releaseDate", m.ReleaseDate, "error", err)
				return nil, fmt.Errorf("failed to parse movie release date '%s': %w", m.ReleaseDate, err)
			}
			releaseDate = &date
		}
		movies[i] = models.Movie{
			ID:               m.ID,
			Title:            m.Title,
			OriginalTitle:    m.OriginalTitle,
			OriginalLanguage: m.OriginalLanguage,
			Overview:         m.Overview,
			ReleaseDate:      releaseDate,
			PosterPath:       m.PosterPath,
			BackdropPath:     m.BackdropPath,
			Popularity:       m.Popularity,
			VoteCount:        m.VoteCount,
			VoteAverage:      m.VoteAverage,
		}
	}

	s.log.Debug("converted TMDB search results to internal models", "movieCount", len(movies))
	return movies, nil
}

func (s *MovieService) GetMovieDetails(ctx context.Context, id int64) (*models.MovieDetails, error) {
	s.log.Debug("getting movie details", "movieID", id)

	movie, err := s.db.GetMovieDetailsByID(ctx, id)
	if err == nil && time.Since(movie.Movie.UpdatedAt) <= s.cacheTTL {
		s.log.Debug("found movie details in database cache", "movieID", id)
		return movie, nil
	}

	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		s.log.Error("failed to get movie details from database. Fetching from TMDB", "movieID", id, "error", err)
	}

	s.log.Debug("movie not found in cache, fetching from TMDB", "movieID", id)

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
