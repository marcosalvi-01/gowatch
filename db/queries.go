package db

import (
	"context"
	"fmt"
	"gowatch/db/sqlc"
	"gowatch/internal/models"
	"time"
)

// InsertMovie adds a new movie to the database
func (d *SqliteDB) InsertMovie(ctx context.Context, movie *models.MovieDetails) error {
	log.Debug("inserting movie into database", "movieID", movie.Movie.ID, "title", movie.Movie.Title)

	tx, err := d.db.Begin()
	if err != nil {
		log.Error("failed to start database transaction for movie insert", "movieID", movie.Movie.ID, "error", err)
		return fmt.Errorf("failed to start db transaction: %w", err)
	}
	defer tx.Rollback()

	qtx := d.queries.WithTx(tx)

	err = qtx.InsertMovie(ctx, sqlc.InsertMovieParams{
		ID:               movie.Movie.ID,
		Title:            movie.Movie.Title,
		OriginalTitle:    movie.Movie.OriginalTitle,
		OriginalLanguage: movie.Movie.OriginalLanguage,
		Overview:         movie.Movie.Overview,
		ReleaseDate:      movie.Movie.ReleaseDate,
		PosterPath:       movie.Movie.PosterPath,
		BackdropPath:     movie.Movie.BackdropPath,
		Popularity:       float64(movie.Movie.Popularity),
		VoteCount:        movie.Movie.VoteCount,
		VoteAverage:      float64(movie.Movie.VoteAverage),
		Budget:           movie.Budget,
		Homepage:         movie.Homepage,
		ImdbID:           movie.IMDbID,
		Revenue:          movie.Revenue,
		Runtime:          int64(movie.Runtime),
		Status:           movie.Status,
		Tagline:          movie.Tagline,
	})
	if err != nil {
		log.Error("failed to insert movie record", "movieID", movie.Movie.ID, "error", err)
		return fmt.Errorf("failed to insert movie with ID %d: %w", movie.Movie.ID, err)
	}

	log.Debug("inserted movie record, processing genres", "movieID", movie.Movie.ID, "genreCount", len(movie.Genres))

	for _, genre := range movie.Genres {
		err := qtx.InsertGenre(ctx, sqlc.InsertGenreParams{
			ID:   genre.ID,
			Name: genre.Name,
		})
		if err != nil {
			log.Error("failed to insert genre", "movieID", movie.Movie.ID, "genreID", genre.ID, "error", err)
			return fmt.Errorf("failed to insert genre %d: %w", genre.ID, err)
		}

		err = qtx.InsertGenreMovie(ctx, sqlc.InsertGenreMovieParams{
			MovieID: movie.Movie.ID,
			GenreID: genre.ID,
		})
		if err != nil {
			log.Error("failed to insert movie-genre relationship", "movieID", movie.Movie.ID, "genreID", genre.ID, "error", err)
			return fmt.Errorf("failed to insert movie-genre relationship: %w", err)
		}
	}

	log.Debug("processed genres, inserting cast", "movieID", movie.Movie.ID, "castCount", len(movie.Credits.Cast))

	for _, cast := range movie.Credits.Cast {
		err := qtx.InsertCast(ctx, sqlc.InsertCastParams{
			MovieID:   cast.MovieID,
			PersonID:  cast.PersonID,
			CastID:    cast.CastID,
			CreditID:  cast.CreditID,
			Character: cast.Character,
			CastOrder: cast.CastOrder,
		})
		if err != nil {
			log.Error("failed to insert cast record", "movieID", movie.Movie.ID, "personID", cast.PersonID, "error", err)
			return fmt.Errorf("failed to insert cast record: %w", err)
		}

		err = qtx.InsertPerson(ctx, sqlc.InsertPersonParams{
			ID:                 cast.Person.ID,
			Name:               cast.Person.Name,
			OriginalName:       cast.Person.OriginalName,
			ProfilePath:        cast.Person.ProfilePath,
			KnownForDepartment: cast.Person.KnownForDepartment,
			Popularity:         cast.Person.Popularity,
			Gender:             cast.Person.Gender,
			Adult:              cast.Person.Adult,
		})
		if err != nil {
			log.Error("failed to insert person record from cast", "movieID", movie.Movie.ID, "personID", cast.Person.ID, "error", err)
			return fmt.Errorf("failed to insert person from cast: %w", err)
		}
	}

	log.Debug("processed cast, inserting crew", "movieID", movie.Movie.ID, "crewCount", len(movie.Credits.Crew))

	for _, crew := range movie.Credits.Crew {
		err := qtx.InsertCrew(ctx, sqlc.InsertCrewParams{
			MovieID:    crew.MovieID,
			PersonID:   crew.PersonID,
			CreditID:   crew.CreditID,
			Job:        crew.Job,
			Department: crew.Department,
		})
		if err != nil {
			log.Error("failed to insert crew record", "movieID", movie.Movie.ID, "personID", crew.PersonID, "error", err)
			return fmt.Errorf("failed to insert crew record: %w", err)
		}

		err = qtx.InsertPerson(ctx, sqlc.InsertPersonParams{
			ID:                 crew.Person.ID,
			Name:               crew.Person.Name,
			OriginalName:       crew.Person.OriginalName,
			ProfilePath:        crew.Person.ProfilePath,
			KnownForDepartment: crew.Person.KnownForDepartment,
			Popularity:         crew.Person.Popularity,
			Gender:             crew.Person.Gender,
			Adult:              crew.Person.Adult,
		})
		if err != nil {
			log.Error("failed to insert person record from crew", "movieID", movie.Movie.ID, "personID", crew.Person.ID, "error", err)
			return fmt.Errorf("failed to insert person from crew: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		log.Error("failed to commit movie insert transaction", "movieID", movie.Movie.ID, "error", err)
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	log.Info("successfully inserted movie with all related data", "movieID", movie.Movie.ID, "title", movie.Movie.Title,
		"genreCount", len(movie.Genres), "castCount", len(movie.Credits.Cast), "crewCount", len(movie.Credits.Crew))
	return nil
}

// InsertWatched records a movie as watched in the database
func (d *SqliteDB) InsertWatched(ctx context.Context, movieID int64, date time.Time, inTheaters bool) error {
	log.Debug("inserting watched record", "movieID", movieID, "date", date, "inTheaters", inTheaters)

	_, err := d.queries.InsertWatched(ctx, sqlc.InsertWatchedParams{
		MovieID:          movieID,
		WatchedDate:      date,
		WatchedInTheater: inTheaters,
	})
	if err != nil {
		log.Error("failed to insert watched record", "movieID", movieID, "error", err)
		return fmt.Errorf("failed to insert watched record for movie ID %d: %w", movieID, err)
	}

	log.Debug("successfully inserted watched record", "movieID", movieID)
	return nil
}

// GetMovieDetailsByID retrieves a specific movie by its ID
func (d *SqliteDB) GetMovieDetailsByID(ctx context.Context, id int64) (*models.MovieDetails, error) {
	log.Debug("retrieving movie details from database", "movieID", id)

	tx, err := d.db.Begin()
	if err != nil {
		log.Error("failed to start database transaction for movie retrieval", "movieID", id, "error", err)
		return nil, fmt.Errorf("failed to start db transaction: %w", err)
	}
	defer tx.Rollback()

	qtx := d.queries.WithTx(tx)

	sqlcMovie, err := qtx.GetMovieByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get movie with ID %d: %w", id, err)
	}

	log.Debug("found movie record, retrieving related data", "movieID", id, "title", sqlcMovie.Title)
	movie := toModelsMovieDetails(sqlcMovie)

	dbGenres, err := qtx.GetMovieGenre(ctx, id)
	if err != nil {
		log.Error("failed to get movie genres", "movieID", id, "error", err)
		return nil, fmt.Errorf("failed to get movie genres: %w", err)
	}

	log.Debug("retrieved movie genres", "movieID", id, "genreCount", len(dbGenres))

	genres := make([]models.Genre, len(dbGenres))
	for i, g := range dbGenres {
		genres[i] = models.Genre{
			ID:   g.GenreID,
			Name: g.Name,
		}
	}

	movie.Genres = genres

	dbCast, err := qtx.GetCastByMovieID(ctx, id)
	if err != nil {
		log.Error("failed to get movie cast", "movieID", id, "error", err)
		return nil, fmt.Errorf("failed to get movie cast: %w", err)
	}

	log.Debug("retrieved movie cast", "movieID", id, "castCount", len(dbCast))

	cast := make([]models.Cast, len(dbCast))
	for i, c := range dbCast {
		person, err := qtx.GetPerson(ctx, c.PersonID)
		if err != nil {
			log.Error("failed to get person for cast", "movieID", id, "personID", c.PersonID, "error", err)
			return nil, fmt.Errorf("failed to get person for cast: %w", err)
		}

		cast[i] = models.Cast{
			MovieID:   c.MovieID,
			PersonID:  c.PersonID,
			CastID:    c.CastID,
			CreditID:  c.CreditID,
			Character: c.Character,
			CastOrder: c.CastOrder,
			Person:    toModelsPerson(person),
		}
	}
	movie.Credits.Cast = cast

	dbCrew, err := qtx.GetCrewByMovieID(ctx, id)
	if err != nil {
		log.Error("failed to get movie crew", "movieID", id, "error", err)
		return nil, fmt.Errorf("failed to get movie crew: %w", err)
	}

	log.Debug("retrieved movie crew", "movieID", id, "crewCount", len(dbCrew))

	crew := make([]models.Crew, len(dbCrew))
	for i, c := range dbCrew {
		person, err := qtx.GetPerson(ctx, c.PersonID)
		if err != nil {
			log.Error("failed to get person for crew", "movieID", id, "personID", c.PersonID, "error", err)
			return nil, fmt.Errorf("failed to get person for crew: %w", err)
		}

		crew[i] = models.Crew{
			MovieID:    c.MovieID,
			PersonID:   c.PersonID,
			CreditID:   c.CreditID,
			Job:        c.Job,
			Department: c.Department,
			Person:     toModelsPerson(person),
		}
	}

	movie.Credits.Crew = crew

	log.Info("successfully retrieved complete movie details", "movieID", id, "title", movie.Movie.Title,
		"genreCount", len(movie.Genres), "castCount", len(movie.Credits.Cast), "crewCount", len(movie.Credits.Crew))
	return &movie, nil
}

func (d *SqliteDB) GetWatchedJoinMovie(ctx context.Context) ([]models.WatchedMovie, error) {
	log.Debug("retrieving all watched movies with details")

	results, err := d.queries.GetWatchedJoinMovie(ctx)
	if err != nil {
		log.Error("failed to get watched movies from database", "error", err)
		return nil, fmt.Errorf("failed to get watched movies: %w", err)
	}

	log.Debug("retrieved watched movies from database", "count", len(results))

	watched := make([]models.WatchedMovie, len(results))
	for i, result := range results {
		watched[i] = models.WatchedMovie{
			MovieDetails: toModelsMovieDetails(result.Movie),
			Date:         result.Watched.WatchedDate,
			InTheaters:   result.Watched.WatchedInTheater,
		}
	}

	log.Debug("converted watched movies to internal models", "count", len(watched))
	return watched, nil
}

func (d *SqliteDB) GetWatchedJoinMovieByID(ctx context.Context, movieID int64) ([]models.WatchedMovie, error) {
	log.Debug("retrieving watched rows for movie", "movieID", movieID)

	rows, err := d.queries.GetWatchedJoinMovieByID(ctx, movieID)
	if err != nil {
		log.Error("db query failed", "movieID", movieID, "error", err)
		return nil, fmt.Errorf("get watched by id: %w", err)
	}

	watched := make([]models.WatchedMovie, len(rows))
	for i, r := range rows {
		watched[i] = models.WatchedMovie{
			MovieDetails: toModelsMovieDetails(r.Movie),
			Date:         r.Watched.WatchedDate,
			InTheaters:   r.Watched.WatchedInTheater,
		}
	}

	return watched, nil
}
