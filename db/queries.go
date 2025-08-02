package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"gowatch/db/sqlc"
	"gowatch/internal/models"
)

// GetAllWatched retrieves all watched movie records from the database
func (d *SqliteDB) GetAllWatched(ctx context.Context) ([]models.Watched, error) {
	rows, err := d.queries.GetAllWatched(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get all watched movies: %w", err)
	}
	watchedMovies := make([]models.Watched, len(rows))
	for i, row := range rows {
		watchedMovies[i] = models.Watched{
			MovieID:     row.MovieID,
			WatchedDate: row.WatchedDate,
		}
	}
	return watchedMovies, nil
}

// InsertMovie adds a new movie to the database
func (d *SqliteDB) InsertMovie(ctx context.Context, movie models.Movie) error {
	_, err := d.queries.InsertMovie(ctx, toSqlcInsertMovieParams(movie))
	if err != nil {
		return fmt.Errorf("failed to insert movie with ID %d: %w", movie.ID, err)
	}
	return nil
}

// InsertWatched records a movie as watched in the database
func (d *SqliteDB) InsertWatched(ctx context.Context, watched models.Watched) error {
	_, err := d.queries.InsertWatched(ctx, toSqlcInsertWatchedParams(watched))
	if err != nil {
		return fmt.Errorf("failed to insert watched record for movie ID %d: %w", watched.MovieID, err)
	}
	return nil
}

// GetMovieByID retrieves a specific movie by its ID
func (d *SqliteDB) GetMovieByID(ctx context.Context, id int64) (models.Movie, error) {
	sqlcMovie, err := d.queries.GetMovieByID(ctx, id)
	if err != nil {
		return models.Movie{}, fmt.Errorf("failed to get movie with ID %d: %w", id, err)
	}
	return toModelsMovie(sqlcMovie), nil
}

// GetWatchedJoinMovie retrieves all watched movies with their details
func (d *SqliteDB) GetWatchedJoinMovie(ctx context.Context) ([]models.WatchedMovie, error) {
	rows, err := d.queries.GetWatchedJoinMovie(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get watched movies with details: %w", err)
	}
	watchedMovies := make([]models.WatchedMovie, len(rows))
	for i, row := range rows {
		watchedMovies[i] = models.WatchedMovie{
			Movie:       toModelsMovie(row.Movie),
			WatchedDate: row.Watched.WatchedDate,
		}
	}
	return watchedMovies, nil
}

// GetMostWatchedMovies retrieves movies ordered by watch count
func (d *SqliteDB) GetMostWatchedMovies(ctx context.Context) ([]models.WatchedMovieDetails, error) {
	rows, err := d.queries.GetMostWatchedMovies(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get most watched movies: %w", err)
	}
	movieCounts := make([]models.WatchedMovieDetails, len(rows))
	for i, row := range rows {
		movieCounts[i] = models.WatchedMovieDetails{
			Movie:     toModelsMovie(row.Movie),
			ViewCount: int(row.ViewCount),
		}
	}
	return movieCounts, nil
}

func (d *SqliteDB) GetWatchedMovieDetails(ctx context.Context, id int64) (models.WatchedMovieDetails, error) {
	result, err := d.queries.GetWatchedMovieDetails(ctx, id)
	if err != nil {
		return models.WatchedMovieDetails{}, fmt.Errorf("failed to get most watched movies: %w", err)
	}
	return models.WatchedMovieDetails{
		Movie:     toModelsMovie(result.Movie),
		ViewCount: int(result.ViewCount),
	}, nil
}

func (d *SqliteDB) InsertCast(ctx context.Context, cast models.Cast) error {
	_, err := d.queries.GetPerson(ctx, cast.Person.ID)
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("failed to check if person exists with ID %d: %w", cast.Person.ID, err)
		}
		// insert the person too
		_, err := d.queries.InsertPerson(ctx, toSqlcInsertPersonParams(cast.Person))
		if err != nil {
			return fmt.Errorf("failed to insert person with ID %d: %w", cast.Person.ID, err)
		}
	}

	_, err = d.queries.InsertCast(ctx, sqlc.InsertCastParams{
		MovieID:   cast.MovieID,
		PersonID:  cast.PersonID,
		CastID:    cast.CastID,
		CreditID:  cast.CreditID,
		Character: cast.Character,
		CastOrder: cast.CastOrder,
	})
	if err != nil {
		return fmt.Errorf("failed to insert cast record for movie ID %d and person ID %d: %w", cast.MovieID, cast.PersonID, err)
	}

	return nil
}

func (d *SqliteDB) InsertCrew(ctx context.Context, crew models.Crew) error {
	_, err := d.queries.GetPerson(ctx, crew.Person.ID)
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("failed to check if person exists with ID %d: %w", crew.Person.ID, err)
		}
		// insert the person too
		_, err := d.queries.InsertPerson(ctx, toSqlcInsertPersonParams(crew.Person))
		if err != nil {
			return fmt.Errorf("failed to insert person with ID %d: %w", crew.Person.ID, err)
		}
	}

	_, err = d.queries.InsertCrew(ctx, sqlc.InsertCrewParams{
		MovieID:    crew.MovieID,
		PersonID:   crew.PersonID,
		CreditID:   crew.CreditID,
		Job:        crew.Job,
		Department: crew.Department,
	})
	if err != nil {
		return fmt.Errorf("failed to insert crew record for movie ID %d and person ID %d: %w", crew.MovieID, crew.PersonID, err)
	}

	return nil
}

func (d *SqliteDB) GetMovieCast(ctx context.Context, movieID int64) ([]models.Cast, error) {
	results, err := d.queries.GetCastByMovieID(ctx, movieID)
	if err != nil {
		return nil, fmt.Errorf("TODO: %w", err)
	}

	cast := make([]models.Cast, len(results))
	for _, result := range results {
		person, err := d.queries.GetPerson(ctx, result.PersonID)
		if err != nil {
			return nil, fmt.Errorf("TODO: %w", err)
		}
		cast = append(cast, models.Cast{
			MovieID:   result.MovieID,
			PersonID:  result.PersonID,
			CastID:    result.CastID,
			CreditID:  result.CreditID,
			Character: result.Character,
			CastOrder: result.CastOrder,
			Person:    toModelsPerson(person),
		})
	}

	return cast, nil
}

func (d *SqliteDB) GetMovieCrew(ctx context.Context, movieID int64) ([]models.Crew, error) {
	results, err := d.queries.GetCrewByMovieID(ctx, movieID)
	if err != nil {
		return nil, fmt.Errorf("TODO: %w", err)
	}

	crew := make([]models.Crew, len(results))
	for _, result := range results {
		person, err := d.queries.GetPerson(ctx, result.PersonID)
		if err != nil {
			return nil, fmt.Errorf("TODO: %w", err)
		}
		crew = append(crew, models.Crew{
			MovieID:    result.MovieID,
			PersonID:   result.PersonID,
			CreditID:   result.CreditID,
			Job:        result.Job,
			Department: result.Department,
			Person:     toModelsPerson(person),
		})
	}

	return crew, nil
}
