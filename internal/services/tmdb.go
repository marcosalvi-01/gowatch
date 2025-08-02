package services

import (
	"fmt"
	"gowatch/internal/models"

	tmdb "github.com/cyruzin/golang-tmdb"
)

// TMDBService handles all external TMDB API interactions
type TMDBService struct {
	client *tmdb.Client
}

func NewTMDBService(client *tmdb.Client) *TMDBService {
	return &TMDBService{client: client}
}

func (s *TMDBService) SearchMovies(query string) ([]models.SearchMovie, error) {
	log.Debug("searching movie", "query", query)

	search, err := s.client.GetSearchMovies(query, nil)
	if err != nil {
		log.Error("TMDB search failed", "query", query, "error", err)
		return nil, fmt.Errorf("error searching TMDB for query '%s': %w", query, err)
	}

	log.Info("movie search completed", "query", query, "results", search.TotalResults)

	movies := make([]models.SearchMovie, len(search.Results))
	for i, m := range search.Results {
		movies[i] = models.SearchMovie{
			ID:               m.ID,
			Title:            m.Title,
			OriginalTitle:    m.OriginalTitle,
			OriginalLanguage: m.OriginalLanguage,
			Overview:         m.Overview,
			ReleaseDate:      m.ReleaseDate,
			PosterPath:       m.PosterPath,
			BackdropPath:     m.BackdropPath,
			Popularity:       m.Popularity,
			VoteCount:        m.VoteCount,
			VoteAverage:      m.VoteAverage,
			GenreIDs:         m.GenreIDs,
			Adult:            m.Adult,
			Video:            m.Video,
		}
	}

	return movies, nil
}

func (s *TMDBService) GetMovieDetails(id int64) (models.Movie, error) {
	log.Debug("getting movie details", "id", id)

	details, err := s.client.GetMovieDetails(int(id), nil)
	if err != nil {
		log.Error("TMDB get movie details failed", "id", id, "error", err)
		return models.Movie{}, fmt.Errorf("error getting TMDB movie details for id '%d': %w", id, err)
	}

	movie, err := models.MovieFromTMDBMovieDetails(*details)
	if err != nil {
		return models.Movie{}, fmt.Errorf("failed to convert TMDB movie to model: %w", err)
	}

	return movie, nil
}

func (s *TMDBService) GetMovieCredits(id int64) (models.MovieCredits, error) {
	log.Debug("getting movie credits", "id", id)

	credits, err := s.client.GetMovieCredits(int(id), nil)
	if err != nil {
		log.Error("TMDB get movie credits failed", "id", id, "error", err)
		return models.MovieCredits{}, fmt.Errorf("error getting TMDB movie credits for id '%d': %w", id, err)
	}

	// Convert cast
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

	// Convert crew
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

	return models.MovieCredits{
		Crew: crew,
		Cast: cast,
	}, nil
}
