package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/marcosalvi-01/gowatch/db/sqlc"
	"github.com/marcosalvi-01/gowatch/db/types/date"
	"github.com/marcosalvi-01/gowatch/internal/models"
)

// UpsertMovie adds a new movie to the database
func (d *SqliteDB) UpsertMovie(ctx context.Context, movie *models.MovieDetails) error {
	log.Debug("inserting movie into database", "movieID", movie.Movie.ID, "title", movie.Movie.Title)

	tx, err := d.db.Begin()
	if err != nil {
		log.Error("failed to start database transaction for movie insert", "movieID", movie.Movie.ID, "error", err)
		return fmt.Errorf("failed to start db transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	qtx := d.queries.WithTx(tx)

	err = qtx.UpsertMovie(ctx, sqlc.UpsertMovieParams{
		ID:               movie.Movie.ID,
		Title:            movie.Movie.Title,
		OriginalTitle:    movie.Movie.OriginalTitle,
		OriginalLanguage: movie.Movie.OriginalLanguage,
		Overview:         movie.Movie.Overview,
		ReleaseDate:      date.NewFromPtr(movie.Movie.ReleaseDate),
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
		err := qtx.UpsertGenre(ctx, sqlc.UpsertGenreParams{
			ID:   genre.ID,
			Name: genre.Name,
		})
		if err != nil {
			log.Error("failed to insert genre", "movieID", movie.Movie.ID, "genreID", genre.ID, "error", err)
			return fmt.Errorf("failed to insert genre %d: %w", genre.ID, err)
		}

		err = qtx.UpsertGenreMovie(ctx, sqlc.UpsertGenreMovieParams{
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
		err = qtx.UpsertPerson(ctx, sqlc.UpsertPersonParams{
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

		err := qtx.UpsertCast(ctx, sqlc.UpsertCastParams{
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
	}

	log.Debug("processed cast, inserting crew", "movieID", movie.Movie.ID, "crewCount", len(movie.Credits.Crew))

	for _, crew := range movie.Credits.Crew {
		err = qtx.UpsertPerson(ctx, sqlc.UpsertPersonParams{
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

		err := qtx.UpsertCrew(ctx, sqlc.UpsertCrewParams{
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
func (d *SqliteDB) InsertWatched(ctx context.Context, watched InsertWatched) error {
	log.Debug("inserting watched record", "movieID", watched.MovieID, "date", watched.Date, "inTheaters", watched.InTheaters)

	_, err := d.queries.InsertWatched(ctx, sqlc.InsertWatchedParams{
		UserID:           &watched.UserID,
		MovieID:          watched.MovieID,
		WatchedDate:      date.New(watched.Date),
		WatchedInTheater: watched.InTheaters,
		Rating:           watched.Rating,
	})
	if err != nil {
		log.Error("failed to insert watched record", "movieID", watched.MovieID, "error", err)
		return fmt.Errorf("failed to insert watched record for movie ID %d: %w", watched.MovieID, err)
	}

	log.Debug("successfully inserted watched record", "movieID", watched.MovieID)
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
	defer func() { _ = tx.Rollback() }()

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

	if err := tx.Commit(); err != nil {
		log.Error("failed to commit movie insert transaction", "movieID", movie.Movie.ID, "error", err)
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	log.Info(
		"successfully retrieved complete movie details",
		"movieID",
		id,
		"title",
		movie.Movie.Title,
		"genreCount",
		len(movie.Genres),
		"castCount",
		len(movie.Credits.Cast),
		"crewCount",
		len(movie.Credits.Crew),
		"updatedAt", movie.Movie.UpdatedAt,
	)
	return &movie, nil
}

func (d *SqliteDB) GetWatchedJoinMovie(ctx context.Context, userID int64) ([]models.WatchedMovie, error) {
	log.Debug("retrieving all watched movies with details")

	results, err := d.queries.GetWatchedJoinMovie(ctx, &userID)
	if err != nil {
		log.Error("failed to get watched movies from database", "error", err)
		return nil, fmt.Errorf("failed to get watched movies: %w", err)
	}

	log.Debug("retrieved watched movies from database", "count", len(results))

	watched := make([]models.WatchedMovie, len(results))
	for i, result := range results {
		watched[i] = models.WatchedMovie{
			MovieDetails: toModelsMovieDetails(result.Movie),
			Date:         result.Watched.WatchedDate.Time,
			InTheaters:   result.Watched.WatchedInTheater,
			Rating:       result.Watched.Rating,
		}
	}

	log.Debug("converted watched movies to internal models", "count", len(watched))
	return watched, nil
}

func (d *SqliteDB) GetWatchedJoinMovieByID(ctx context.Context, userID, movieID int64) ([]models.WatchedMovie, error) {
	log.Debug("retrieving watched rows for movie", "movieID", movieID)

	rows, err := d.queries.GetWatchedJoinMovieByID(ctx, sqlc.GetWatchedJoinMovieByIDParams{UserID: &userID, MovieID: movieID})
	if err != nil {
		log.Error("db query failed", "movieID", movieID, "error", err)
		return nil, fmt.Errorf("get watched by id: %w", err)
	}

	watched := make([]models.WatchedMovie, len(rows))
	for i, r := range rows {
		watched[i] = models.WatchedMovie{
			MovieDetails: toModelsMovieDetails(r.Movie),
			Date:         r.Watched.WatchedDate.Time,
			InTheaters:   r.Watched.WatchedInTheater,
			Rating:       r.Watched.Rating,
		}
	}

	return watched, nil
}

func (d *SqliteDB) GetWatchedMoviesByPerson(ctx context.Context, userID, personID int64) ([]models.PersonWatchMovieMatch, error) {
	log.Debug("retrieving watched movies for person", "userID", userID, "personID", personID)

	rows, err := d.queries.GetWatchedMoviesByPerson(ctx, sqlc.GetWatchedMoviesByPersonParams{
		PersonID: personID,
		UserID:   &userID,
	})
	if err != nil {
		log.Error("failed to get watched movies for person", "userID", userID, "personID", personID, "error", err)
		return nil, fmt.Errorf("get watched movies by person: %w", err)
	}

	movies := make([]models.PersonWatchMovieMatch, len(rows))
	for i, row := range rows {
		lastWatchedDate, err := scanDateValue(row.LastWatchedDate)
		if err != nil {
			return nil, fmt.Errorf("failed to parse last watched date for person %d movie %d: %w", personID, row.ID, err)
		}

		roleLabel, err := scanStringValue(row.RoleLabel)
		if err != nil {
			return nil, fmt.Errorf("failed to parse role label for person %d movie %d: %w", personID, row.ID, err)
		}

		movies[i] = models.PersonWatchMovieMatch{
			ID:              row.ID,
			Title:           row.Title,
			PosterPath:      row.PosterPath,
			WatchCount:      row.WatchCount,
			LastWatchedDate: lastWatchedDate,
			Role: models.PersonWatchRole{
				Kind:  models.PersonWatchRoleKind(row.RoleKind),
				Label: roleLabel,
			},
		}
	}

	log.Debug("retrieved watched movies for person", "userID", userID, "personID", personID, "count", len(movies))
	return movies, nil
}

func (d *SqliteDB) GetRecentWatchedMovies(ctx context.Context, userID int64, limit int) ([]models.WatchedMovieInDay, error) {
	log.Debug("retrieving recent watched movies", "limit", limit)

	rows, err := d.queries.GetRecentWatchedMovies(ctx, sqlc.GetRecentWatchedMoviesParams{UserID: &userID, Limit: int64(limit)})
	if err != nil {
		log.Error("failed to fetch recent watched movies from database", "error", err)
		return nil, fmt.Errorf("failed to fetch recent watched movies: %w", err)
	}

	result := make([]models.WatchedMovieInDay, len(rows))
	for i, row := range rows {
		result[i] = models.WatchedMovieInDay{
			MovieDetails: toModelsMovieDetails(row.Movie),
			InTheaters:   row.Watched.WatchedInTheater,
			Rating:       row.Watched.Rating,
		}
	}

	log.Debug("retrieved recent watched movies", "count", len(result))
	return result, nil
}

func (d *SqliteDB) InsertList(ctx context.Context, list InsertList) (int64, error) {
	log.Debug("inserting new list into database", "name", list.Name)

	id, err := d.queries.InsertList(ctx, sqlc.InsertListParams{
		UserID:       &list.UserID,
		Name:         list.Name,
		CreationDate: time.Now().Format("2006-01-02 15:04:05.999999999 -0700 MST"),
		Description:  list.Description,
		IsWatchlist:  list.IsWatchlist,
	})
	if err != nil {
		log.Error("failed to insert list", "name", list.Name, "error", err)
		return 0, fmt.Errorf("failed to insert list %q: %w", list.Name, err)
	}

	log.Info("successfully inserted list", "name", list.Name, "id", id)
	return id, nil
}

func (d *SqliteDB) GetList(ctx context.Context, userID, id int64) (*models.List, error) {
	log.Debug("retrieving list with movies", "listID", id)

	tx, err := d.db.Begin()
	if err != nil {
		log.Error("failed to start database transaction for list retrieval", "listID", id, "error", err)
		return nil, fmt.Errorf("failed to start db transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	qtx := d.queries.WithTx(tx)

	results, err := qtx.GetListJoinMovieByID(ctx, sqlc.GetListJoinMovieByIDParams{UserID: &userID, ID: id})
	if err != nil {
		log.Error("failed to fetch list with movies", "listID", id, "error", err)
		return nil, fmt.Errorf("failed to fetch list with ID %d: %w", id, err)
	}
	if len(results) == 0 {
		log.Debug("list has no movies associated to it", "listID", id)
		// try to search for the list without joining in case it is empty
		list, err := qtx.GetListByID(ctx, sqlc.GetListByIDParams{UserID: &userID, ID: id})
		if err != nil {
			// no list with that id exists, return an error
			return nil, fmt.Errorf("failed to get list by ID %d: %w", id, err)
		}

		// this list exists but it is empty, return it anyway
		creationDate, err := time.Parse("2006-01-02 15:04:05.999999999 -0700 MST", list.CreationDate)
		if err != nil {
			log.Error("failed to parse list creation_date", "listID", id, "error", err)
			return nil, fmt.Errorf("failed to parse creation_date for list %d: %w", id, err)
		}

		return &models.List{
			ID:           list.ID,
			Name:         list.Name,
			CreationDate: creationDate,
			Description:  list.Description,
			IsWatchlist:  list.IsWatchlist,
			Movies:       []models.MovieItem{},
		}, nil
	}

	list := results[0].List
	movies := make([]models.MovieItem, len(results))

	for i, result := range results {
		dateAdded, err := time.Parse("2006-01-02 15:04:05.999999999 -0700 MST", result.ListMovie.DateAdded)
		if err != nil {
			log.Error("failed to parse movie date_added in list", "listID", id, "movieID", result.Movie.ID, "error", err)
			return nil, fmt.Errorf("failed to parse date_added for movie %d: %w", result.Movie.ID, err)
		}

		movies[i] = models.MovieItem{
			MovieDetails: toModelsMovieDetails(result.Movie),
			DateAdded:    dateAdded,
			Position:     result.ListMovie.Position,
			Note:         result.ListMovie.Note,
		}
	}

	if err := tx.Commit(); err != nil {
		log.Error("failed to commit list retrieval transaction", "listID", id, "error", err)
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	creationDate, err := time.Parse("2006-01-02 15:04:05.999999999 -0700 MST", list.CreationDate)
	if err != nil {
		log.Error("failed to parse list creation_date", "listID", id, "error", err)
		return nil, fmt.Errorf("failed to parse creation_date for list %d: %w", id, err)
	}

	log.Info("successfully retrieved list with movies", "listID", id, "movieCount", len(movies))
	return &models.List{
		ID:           list.ID,
		Name:         list.Name,
		CreationDate: creationDate,
		Description:  list.Description,
		IsWatchlist:  list.IsWatchlist,
		Movies:       movies,
	}, nil
}

func (d *SqliteDB) AddMovieToList(ctx context.Context, userID int64, insertMovieList InsertMovieList) error {
	log.Debug("adding movie to list", "movieID", insertMovieList.MovieID, "position", insertMovieList.Position)

	// First verify the list exists and belongs to the user
	_, err := d.queries.GetListByID(ctx, sqlc.GetListByIDParams{UserID: &userID, ID: insertMovieList.ListID})
	if err != nil {
		log.Error("failed to verify list ownership", "listID", insertMovieList.ListID, "userID", userID, "error", err)
		return fmt.Errorf("failed to verify list ownership: %w", err)
	}

	err = d.queries.AddMovieToList(ctx, sqlc.AddMovieToListParams{
		MovieID:   insertMovieList.MovieID,
		ListID:    insertMovieList.ListID,
		DateAdded: insertMovieList.DateAdded.Format("2006-01-02 15:04:05.999999999 -0700 MST"),
		Position:  insertMovieList.Position,
		Note:      insertMovieList.Note,
		ID:        insertMovieList.ListID,
		UserID:    &userID,
	})
	if err != nil {
		log.Error("failed to add movie to list", "movieID", insertMovieList.MovieID, "error", err)
		return fmt.Errorf("failed to add movie %d to list: %w", insertMovieList.MovieID, err)
	}

	log.Info("successfully added movie to list", "movieID", insertMovieList.MovieID)
	return nil
}

func (d *SqliteDB) UpsertMovieInList(ctx context.Context, userID int64, insertMovieList InsertMovieList) error {
	log.Debug("upserting movie in list", "movieID", insertMovieList.MovieID, "listID", insertMovieList.ListID)

	// First verify the list exists and belongs to the user
	_, err := d.queries.GetListByID(ctx, sqlc.GetListByIDParams{UserID: &userID, ID: insertMovieList.ListID})
	if err != nil {
		log.Error("failed to verify list ownership", "listID", insertMovieList.ListID, "userID", userID, "error", err)
		return fmt.Errorf("failed to verify list ownership: %w", err)
	}

	err = d.queries.UpsertMovieInList(ctx, sqlc.UpsertMovieInListParams{
		MovieID:   insertMovieList.MovieID,
		ListID:    insertMovieList.ListID,
		DateAdded: insertMovieList.DateAdded.Format("2006-01-02 15:04:05.999999999 -0700 MST"),
		Position:  insertMovieList.Position,
		Note:      insertMovieList.Note,
		ID:        insertMovieList.ListID,
		UserID:    &userID,
	})
	if err != nil {
		log.Error("failed to upsert movie in list", "movieID", insertMovieList.MovieID, "listID", insertMovieList.ListID, "error", err)
		return fmt.Errorf("failed to upsert movie %d in list %d: %w", insertMovieList.MovieID, insertMovieList.ListID, err)
	}

	log.Info("successfully upserted movie in list", "movieID", insertMovieList.MovieID, "listID", insertMovieList.ListID)
	return nil
}

func (d *SqliteDB) GetAllLists(ctx context.Context, userID int64) ([]InsertList, error) {
	log.Debug("retrieving all lists from database")

	results, err := d.queries.GetAllLists(ctx, &userID)
	if err != nil {
		log.Error("failed to get all lists", "error", err)
		return nil, fmt.Errorf("failed to get all lists: %w", err)
	}

	lists := make([]InsertList, len(results))
	for i, result := range results {
		lists[i] = InsertList{
			ID:          result.ID,
			Name:        result.Name,
			Description: result.Description,
			IsWatchlist: result.IsWatchlist,
		}
	}

	log.Info("successfully retrieved all lists", "count", len(lists))
	return lists, nil
}

func (d *SqliteDB) GetWatchedCount(ctx context.Context, userID int64) (int64, error) {
	log.Debug("getting watched count")

	count, err := d.queries.GetWatchedCount(ctx, &userID)
	if err != nil {
		log.Error("failed to get watched count", "error", err)
		return 0, fmt.Errorf("failed to get watched movie count: %w", err)
	}

	log.Debug("retrieved watched count", "count", count)
	return count, nil
}

func (d *SqliteDB) DeleteListByID(ctx context.Context, userID, id int64) error {
	log.Debug("deleting list by ID", "listID", id)

	err := d.queries.DeleteListByID(ctx, sqlc.DeleteListByIDParams{UserID: &userID, ID: id})
	if err != nil {
		log.Error("failed to delete list", "listID", id, "error", err)
		return fmt.Errorf("failed to delete list with id '%d' in db: %w", id, err)
	}

	log.Debug("successfully deleted list", "listID", id)
	return nil
}

func (d *SqliteDB) DeleteMovieFromList(ctx context.Context, userID, listID, movieID int64) error {
	log.Debug("deleting movie from list", "listID", listID, "movieID", movieID)

	err := d.queries.DeleteMovieFromList(ctx, sqlc.DeleteMovieFromListParams{
		ListID:  listID,
		MovieID: movieID,
		UserID:  &userID,
	})
	if err != nil {
		log.Error("failed to delete movie from list", "listID", listID, "movieID", movieID, "error", err)
		return fmt.Errorf("failed to delete movie '%d' from list '%d' in db: %w", movieID, listID, err)
	}

	log.Debug("successfully deleted movie from list", "listID", listID, "movieID", movieID)
	return nil
}

func (d *SqliteDB) GetWatchlistID(ctx context.Context, userID int64) (int64, error) {
	log.Debug("getting watchlist ID", "userID", userID)

	id, err := d.queries.GetWatchlistID(ctx, &userID)
	if err != nil {
		log.Error("failed to get watchlist ID", "userID", userID, "error", err)
		return 0, fmt.Errorf("failed to get watchlist ID for user %d: %w", userID, err)
	}

	log.Debug("retrieved watchlist ID", "userID", userID, "watchlistID", id)
	return id, nil
}

func (d *SqliteDB) GetWatchedStatsPerMonthLastYear(ctx context.Context, userID int64) ([]models.PeriodStats, error) {
	log.Debug("getting watched stats per month last year")

	data, err := d.queries.GetWatchedStatsPerMonthLastYear(ctx, &userID)
	if err != nil {
		log.Error("failed to get monthly stats", "error", err)
		return nil, fmt.Errorf("failed to get monthly stats: %w", err)
	}

	result := make([]models.PeriodStats, len(data))
	for i, item := range data {
		totalRuntime := 0.0
		if item.TotalRuntime != nil {
			totalRuntime = *item.TotalRuntime
		}
		result[i] = models.PeriodStats{
			Period: item.Month,
			Count:  item.Count,
			Hours:  totalRuntime / 60.0,
		}
	}

	log.Debug("retrieved monthly stats", "periodCount", len(result))
	return result, nil
}

func (d *SqliteDB) GetWatchedPerYear(ctx context.Context, userID int64) ([]models.PeriodCount, error) {
	log.Debug("getting watched per year")

	data, err := d.queries.GetWatchedPerYear(ctx, &userID)
	if err != nil {
		log.Error("failed to get yearly data", "error", err)
		return nil, fmt.Errorf("failed to get yearly data: %w", err)
	}

	result := make([]models.PeriodCount, len(data))
	for i, item := range data {
		result[i] = models.PeriodCount{
			Period: item.Year,
			Count:  item.Count,
		}
	}

	log.Debug("retrieved yearly data", "periodCount", len(result))
	return result, nil
}

func (d *SqliteDB) GetWeekdayDistribution(ctx context.Context, userID int64) ([]models.PeriodCount, error) {
	log.Debug("getting weekday distribution")

	data, err := d.queries.GetWeekdayDistribution(ctx, &userID)
	if err != nil {
		log.Error("failed to get weekday distribution", "error", err)
		return nil, fmt.Errorf("failed to get weekday distribution: %w", err)
	}

	weekdays := []string{"Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"}
	result := make([]models.PeriodCount, len(weekdays))

	// Initialize map for quick lookup
	counts := make(map[int]int64)
	for _, item := range data {
		// item.WeekdayIndex is int64, convert to int for usage
		counts[int(item.WeekdayIndex)] = item.Count
	}

	for i, day := range weekdays {
		result[i] = models.PeriodCount{
			Period: day,
			Count:  counts[i],
		}
	}

	// Shift Sunday to end to match original behavior (Mon-Sun)?
	// Original Go implementation used time.Weekday().String() which relies on standard order.
	// Standard order is Sunday(0) to Saturday(6).
	// The previous implementation constructed:
	// weekdays := []string{"Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday", "Sunday"}
	// result := make([]models.PeriodCount, len(weekdays))
	// for i, day := range weekdays { ... }

	// So I need to match that output structure.

	output := make([]models.PeriodCount, 0, 7)
	// Monday (1) to Saturday (6)
	for i := 1; i <= 6; i++ {
		output = append(output, models.PeriodCount{Period: weekdays[i], Count: counts[i]})
	}
	// Sunday (0)
	output = append(output, models.PeriodCount{Period: weekdays[0], Count: counts[0]})

	log.Debug("retrieved weekday distribution", "dayCount", len(output))
	return output, nil
}

func (d *SqliteDB) GetWatchedByGenre(ctx context.Context, userID int64) ([]models.GenreCount, error) {
	log.Debug("getting watched by genre")

	data, err := d.queries.GetWatchedByGenre(ctx, &userID)
	if err != nil {
		log.Error("failed to get genre data", "error", err)
		return nil, fmt.Errorf("failed to get genre data: %w", err)
	}
	result := make([]models.GenreCount, len(data))
	for i, d := range data {
		result[i] = models.GenreCount{Name: d.Name, Count: d.Count}
	}

	log.Debug("retrieved genre data", "genreCount", len(result))
	return result, nil
}

func (d *SqliteDB) GetTheaterVsHomeCount(ctx context.Context, userID int64) ([]models.TheaterCount, error) {
	log.Debug("getting theater vs home count")

	data, err := d.queries.GetTheaterVsHomeCount(ctx, &userID)
	if err != nil {
		log.Error("failed to get theater data", "error", err)
		return nil, fmt.Errorf("failed to get theater data: %w", err)
	}
	result := make([]models.TheaterCount, len(data))
	for i, d := range data {
		result[i] = models.TheaterCount{InTheater: d.WatchedInTheater, Count: d.Count}
	}

	log.Debug("retrieved theater data", "count", len(result))
	return result, nil
}

func (d *SqliteDB) GetMostWatchedMovies(ctx context.Context, userID int64, limit int) ([]models.TopMovie, error) {
	log.Debug("getting most watched movies", "limit", limit)

	data, err := d.queries.GetMostWatchedMovies(ctx, sqlc.GetMostWatchedMoviesParams{UserID: &userID, Limit: int64(limit)})
	if err != nil {
		log.Error("failed to get most watched movies", "error", err)
		return nil, fmt.Errorf("failed to get most watched movies: %w", err)
	}
	result := make([]models.TopMovie, len(data))
	for i, d := range data {
		result[i] = models.TopMovie{Title: d.Title, ID: d.ID, WatchCount: d.WatchCount, PosterPath: d.PosterPath}
	}

	log.Debug("retrieved most watched movies", "count", len(result))
	return result, nil
}

func (d *SqliteDB) GetMostWatchedDay(ctx context.Context, userID int64) (*models.MostWatchedDay, error) {
	log.Debug("getting most watched day")

	result, err := d.queries.GetMostWatchedDay(ctx, &userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			log.Debug("no watched days found")
			return nil, sql.ErrNoRows
		}
		log.Error("failed to get most watched day", "error", err)
		return nil, fmt.Errorf("failed to get most watched day: %w", err)
	}

	log.Debug("retrieved most watched day", "date", result.WatchedDate.Time, "count", result.Count)
	return &models.MostWatchedDay{Date: result.WatchedDate.Time, Count: result.Count}, nil
}

func (d *SqliteDB) GetMostWatchedActorsByGender(ctx context.Context, userID int64, gender int64, limit int) ([]models.TopActor, error) {
	log.Debug("getting most watched actors by gender", "limit", limit, "gender", gender)

	data, err := d.queries.GetMostWatchedActorsByGender(ctx, sqlc.GetMostWatchedActorsByGenderParams{
		Gender: gender,
		UserID: &userID,
		Limit:  int64(limit),
	})
	if err != nil {
		log.Error("failed to get most watched actors by gender", "error", err, "gender", gender)
		return nil, fmt.Errorf("failed to get most watched actors by gender: %w", err)
	}
	result := make([]models.TopActor, len(data))
	for i, d := range data {
		result[i] = models.TopActor{Name: d.Name, ID: d.ID, WatchCount: d.WatchCount, ProfilePath: d.ProfilePath, Gender: d.Gender}
	}

	log.Debug("retrieved most watched actors by gender", "count", len(result), "gender", gender)
	return result, nil
}

func (d *SqliteDB) GetMostWatchedActorRankByGender(ctx context.Context, userID, personID int64) (*int64, error) {
	log.Debug("getting most watched actor rank by gender", "userID", userID, "personID", personID)

	targetActorStats, err := d.queries.GetWatchedActorStatsByID(ctx, sqlc.GetWatchedActorStatsByIDParams{
		UserID: &userID,
		ID:     personID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			log.Debug("no most watched actor rank found", "userID", userID, "personID", personID)
			return nil, nil
		}

		log.Error("failed to get most watched actor rank by gender", "userID", userID, "personID", personID, "error", err)
		return nil, fmt.Errorf("failed to get most watched actor rank by gender: %w", err)
	}

	higherRankedCount, err := d.queries.CountHigherRankedActorsByGender(ctx, sqlc.CountHigherRankedActorsByGenderParams{
		UserID:  &userID,
		Gender:  targetActorStats.Gender,
		Column3: targetActorStats.WatchCount,
	})
	if err != nil {
		log.Error("failed to count higher ranked actors by gender", "userID", userID, "personID", personID, "gender", targetActorStats.Gender, "error", err)
		return nil, fmt.Errorf("failed to count higher ranked actors by gender: %w", err)
	}

	rank := higherRankedCount + 1
	log.Debug("retrieved most watched actor rank by gender", "userID", userID, "personID", personID, "rank", rank)
	return &rank, nil
}

func (d *SqliteDB) GetWatchedDateRange(ctx context.Context, userID int64) (*models.DateRange, error) {
	log.Debug("getting watched date range")

	data, err := d.queries.GetWatchedDateRange(ctx, &userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			log.Debug("no watched dates found")
			return &models.DateRange{}, nil
		}
		log.Error("failed to get date range", "error", err)
		return nil, fmt.Errorf("failed to get date range: %w", err)
	}
	var min, max *time.Time
	if data.MinDate != nil {
		minStr, ok := data.MinDate.(string)
		if !ok {
			log.Error("unexpected type for MinDate", "type", fmt.Sprintf("%T", data.MinDate))
			return nil, fmt.Errorf("unexpected type for MinDate: %T", data.MinDate)
		}
		parsed, err := time.Parse(date.Layout, minStr)
		if err != nil {
			// fallback for backward compatibility
			parsed, err = time.Parse("2006-01-02 15:04:05 -0700 MST", minStr)
			if err != nil {
				log.Error("failed to parse min date", "date", minStr, "error", err)
				return nil, fmt.Errorf("failed to parse min date %q: %w", minStr, err)
			}
		}
		min = &parsed
	}
	if data.MaxDate != nil {
		maxStr, ok := data.MaxDate.(string)
		if !ok {
			log.Error("unexpected type for MaxDate", "type", fmt.Sprintf("%T", data.MaxDate))
			return nil, fmt.Errorf("unexpected type for MaxDate: %T", data.MaxDate)
		}
		parsed, err := time.Parse(date.Layout, maxStr)
		if err != nil {
			// fallback for backward compatibility
			parsed, err = time.Parse("2006-01-02 15:04:05 -0700 MST", maxStr)
			if err != nil {
				log.Error("failed to parse max date", "date", maxStr, "error", err)
				return nil, fmt.Errorf("failed to parse max date %q: %w", maxStr, err)
			}
		}
		max = &parsed
	}

	log.Debug("retrieved watched date range", "minDate", min, "maxDate", max)
	return &models.DateRange{MinDate: min, MaxDate: max}, nil
}

func (d *SqliteDB) GetWatchedDates(ctx context.Context, userID int64) ([]time.Time, error) {
	log.Debug("getting watched dates")

	data, err := d.queries.GetWatchedDates(ctx, &userID)
	if err != nil {
		log.Error("failed to get watched dates", "error", err)
		return nil, fmt.Errorf("failed to get watched dates: %w", err)
	}

	result := make([]time.Time, len(data))
	for i, watchedDate := range data {
		result[i] = watchedDate.Time
	}

	log.Debug("retrieved watched dates", "count", len(result))
	return result, nil
}

func (d *SqliteDB) GetTotalWatchedStats(ctx context.Context, userID int64) (*models.TotalStats, error) {
	log.Debug("getting total watched stats")

	stats, err := d.queries.GetTotalWatchedStats(ctx, &userID)
	if err != nil {
		log.Error("failed to get total stats", "error", err)
		return nil, fmt.Errorf("failed to get total stats: %w", err)
	}

	totalRuntime := 0.0
	if stats.TotalRuntime != nil {
		totalRuntime = *stats.TotalRuntime
	}

	log.Debug("retrieved total stats", "count", stats.Count, "totalHours", totalRuntime/60.0)
	return &models.TotalStats{
		Count: stats.Count,
		Hours: totalRuntime / 60.0,
	}, nil
}

func (d *SqliteDB) GetRewatchStats(ctx context.Context, userID int64) (*models.RewatchStats, error) {
	log.Debug("getting rewatch stats")

	stats, err := d.queries.GetRewatchStats(ctx, &userID)
	if err != nil {
		log.Error("failed to get rewatch stats", "error", err)
		return nil, fmt.Errorf("failed to get rewatch stats: %w", err)
	}

	result := &models.RewatchStats{
		UniqueMovieCount:    stats.UniqueMovieCount,
		RewatchedMovieCount: stats.RewatchedMovieCount,
		RewatchCount:        stats.RewatchCount,
	}

	log.Debug("retrieved rewatch stats", "uniqueMovieCount", result.UniqueMovieCount, "rewatchedMovieCount", result.RewatchedMovieCount, "rewatchCount", result.RewatchCount)
	return result, nil
}

func (d *SqliteDB) GetDailyWatchCountsLastYear(ctx context.Context, userID int64) ([]models.DailyWatchCount, error) {
	log.Debug("getting daily watch counts for last year")

	data, err := d.queries.GetDailyWatchCountsLastYear(ctx, &userID)
	if err != nil {
		log.Error("failed to get daily watch counts for last year", "error", err)
		return nil, fmt.Errorf("failed to get daily watch counts for last year: %w", err)
	}

	result := make([]models.DailyWatchCount, len(data))
	for i, row := range data {
		result[i] = models.DailyWatchCount{
			Date:  row.WatchedDate.Time,
			Count: row.Count,
		}
	}

	log.Debug("retrieved daily watch counts for last year", "count", len(result))
	return result, nil
}

func (d *SqliteDB) GetTopDirectors(ctx context.Context, userID int64, limit int) ([]models.TopCrewMember, error) {
	log.Debug("getting top directors", "limit", limit)

	data, err := d.queries.GetTopDirectors(ctx, sqlc.GetTopDirectorsParams{UserID: &userID, Limit: int64(limit)})
	if err != nil {
		log.Error("failed to get top directors", "error", err)
		return nil, fmt.Errorf("failed to get top directors: %w", err)
	}

	result := make([]models.TopCrewMember, len(data))
	for i, row := range data {
		result[i] = models.TopCrewMember{
			ID:          row.ID,
			Name:        row.Name,
			ProfilePath: row.ProfilePath,
			WatchCount:  row.WatchCount,
		}
	}

	log.Debug("retrieved top directors", "count", len(result))
	return result, nil
}

func (d *SqliteDB) GetTopWriters(ctx context.Context, userID int64, limit int) ([]models.TopCrewMember, error) {
	log.Debug("getting top writers", "limit", limit)

	data, err := d.queries.GetTopWriters(ctx, sqlc.GetTopWritersParams{UserID: &userID, Limit: int64(limit)})
	if err != nil {
		log.Error("failed to get top writers", "error", err)
		return nil, fmt.Errorf("failed to get top writers: %w", err)
	}

	result := make([]models.TopCrewMember, len(data))
	for i, row := range data {
		result[i] = models.TopCrewMember{
			ID:          row.ID,
			Name:        row.Name,
			ProfilePath: row.ProfilePath,
			WatchCount:  row.WatchCount,
		}
	}

	log.Debug("retrieved top writers", "count", len(result))
	return result, nil
}

func (d *SqliteDB) GetTopComposers(ctx context.Context, userID int64, limit int) ([]models.TopCrewMember, error) {
	log.Debug("getting top composers", "limit", limit)

	data, err := d.queries.GetTopComposers(ctx, sqlc.GetTopComposersParams{UserID: &userID, Limit: int64(limit)})
	if err != nil {
		log.Error("failed to get top composers", "error", err)
		return nil, fmt.Errorf("failed to get top composers: %w", err)
	}

	result := make([]models.TopCrewMember, len(data))
	for i, row := range data {
		result[i] = models.TopCrewMember{
			ID:          row.ID,
			Name:        row.Name,
			ProfilePath: row.ProfilePath,
			WatchCount:  row.WatchCount,
		}
	}

	log.Debug("retrieved top composers", "count", len(result))
	return result, nil
}

func (d *SqliteDB) GetTopCinematographers(ctx context.Context, userID int64, limit int) ([]models.TopCrewMember, error) {
	log.Debug("getting top cinematographers", "limit", limit)

	data, err := d.queries.GetTopCinematographers(ctx, sqlc.GetTopCinematographersParams{UserID: &userID, Limit: int64(limit)})
	if err != nil {
		log.Error("failed to get top cinematographers", "error", err)
		return nil, fmt.Errorf("failed to get top cinematographers: %w", err)
	}

	result := make([]models.TopCrewMember, len(data))
	for i, row := range data {
		result[i] = models.TopCrewMember{
			ID:          row.ID,
			Name:        row.Name,
			ProfilePath: row.ProfilePath,
			WatchCount:  row.WatchCount,
		}
	}

	log.Debug("retrieved top cinematographers", "count", len(result))
	return result, nil
}

func (d *SqliteDB) GetTopLanguages(ctx context.Context, userID int64, limit int) ([]models.LanguageCount, error) {
	log.Debug("getting top languages", "limit", limit)

	data, err := d.queries.GetTopLanguages(ctx, sqlc.GetTopLanguagesParams{UserID: &userID, Limit: int64(limit)})
	if err != nil {
		log.Error("failed to get top languages", "error", err)
		return nil, fmt.Errorf("failed to get top languages: %w", err)
	}

	result := make([]models.LanguageCount, len(data))
	for i, row := range data {
		result[i] = models.LanguageCount{
			Language:   row.Language,
			WatchCount: row.WatchCount,
		}
	}

	log.Debug("retrieved top languages", "count", len(result))
	return result, nil
}

func (d *SqliteDB) GetReleaseYearDistribution(ctx context.Context, userID int64) ([]models.ReleaseYearCount, error) {
	log.Debug("getting release year distribution")

	data, err := d.queries.GetReleaseYearDistribution(ctx, &userID)
	if err != nil {
		log.Error("failed to get release year distribution", "error", err)
		return nil, fmt.Errorf("failed to get release year distribution: %w", err)
	}

	result := make([]models.ReleaseYearCount, len(data))
	for i, row := range data {
		result[i] = models.ReleaseYearCount{
			Year:  int(row.ReleaseYear),
			Count: row.Count,
		}
	}

	log.Debug("retrieved release year distribution", "count", len(result))
	return result, nil
}

func (d *SqliteDB) GetLongestWatchedMovie(ctx context.Context, userID int64) (*models.RuntimeMovie, error) {
	log.Debug("getting longest watched movie")

	row, err := d.queries.GetLongestWatchedMovie(ctx, &userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			log.Debug("no longest watched movie found")
			return nil, sql.ErrNoRows
		}
		log.Error("failed to get longest watched movie", "error", err)
		return nil, fmt.Errorf("failed to get longest watched movie: %w", err)
	}

	result := &models.RuntimeMovie{
		ID:             row.ID,
		Title:          row.Title,
		PosterPath:     row.PosterPath,
		RuntimeMinutes: row.Runtime,
	}

	log.Debug("retrieved longest watched movie", "movieID", result.ID, "runtimeMinutes", result.RuntimeMinutes)
	return result, nil
}

func (d *SqliteDB) GetShortestWatchedMovie(ctx context.Context, userID int64) (*models.RuntimeMovie, error) {
	log.Debug("getting shortest watched movie")

	row, err := d.queries.GetShortestWatchedMovie(ctx, &userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			log.Debug("no shortest watched movie found")
			return nil, sql.ErrNoRows
		}
		log.Error("failed to get shortest watched movie", "error", err)
		return nil, fmt.Errorf("failed to get shortest watched movie: %w", err)
	}

	result := &models.RuntimeMovie{
		ID:             row.ID,
		Title:          row.Title,
		PosterPath:     row.PosterPath,
		RuntimeMinutes: row.Runtime,
	}

	log.Debug("retrieved shortest watched movie", "movieID", result.ID, "runtimeMinutes", result.RuntimeMinutes)
	return result, nil
}

func (d *SqliteDB) GetBudgetTierDistribution(ctx context.Context, userID int64) ([]models.BudgetTierCount, error) {
	log.Debug("getting budget tier distribution")

	data, err := d.queries.GetBudgetTierDistribution(ctx, &userID)
	if err != nil {
		log.Error("failed to get budget tier distribution", "error", err)
		return nil, fmt.Errorf("failed to get budget tier distribution: %w", err)
	}

	result := make([]models.BudgetTierCount, len(data))
	for i, row := range data {
		result[i] = models.BudgetTierCount{
			Tier:  models.BudgetTierFromString(row.Tier),
			Count: row.Count,
		}
	}

	log.Debug("retrieved budget tier distribution", "count", len(result))
	return result, nil
}

func (d *SqliteDB) GetTopReturnOnInvestmentMovies(ctx context.Context, userID int64, limit int) ([]models.MovieFinancial, error) {
	log.Debug("getting top return on investment movies", "limit", limit)

	data, err := d.queries.GetTopReturnOnInvestmentMovies(ctx, sqlc.GetTopReturnOnInvestmentMoviesParams{UserID: &userID, Limit: int64(limit)})
	if err != nil {
		log.Error("failed to get top return on investment movies", "error", err)
		return nil, fmt.Errorf("failed to get top return on investment movies: %w", err)
	}

	result := make([]models.MovieFinancial, len(data))
	for i, row := range data {
		result[i] = models.MovieFinancial{
			ID:         row.ID,
			Title:      row.Title,
			PosterPath: row.PosterPath,
			Budget:     row.Budget,
			Revenue:    row.Revenue,
			ROI:        row.Roi,
		}
	}

	log.Debug("retrieved top return on investment movies", "count", len(result))
	return result, nil
}

func (d *SqliteDB) GetBiggestBudgetMovies(ctx context.Context, userID int64, limit int) ([]models.MovieFinancial, error) {
	log.Debug("getting biggest budget movies", "limit", limit)

	data, err := d.queries.GetBiggestBudgetMovies(ctx, sqlc.GetBiggestBudgetMoviesParams{UserID: &userID, Limit: int64(limit)})
	if err != nil {
		log.Error("failed to get biggest budget movies", "error", err)
		return nil, fmt.Errorf("failed to get biggest budget movies: %w", err)
	}

	result := make([]models.MovieFinancial, len(data))
	for i, row := range data {
		result[i] = models.MovieFinancial{
			ID:         row.ID,
			Title:      row.Title,
			PosterPath: row.PosterPath,
			Budget:     row.Budget,
			Revenue:    row.Revenue,
			ROI:        row.Roi,
		}
	}

	log.Debug("retrieved biggest budget movies", "count", len(result))
	return result, nil
}

func (d *SqliteDB) GetMonthlyGenreBreakdown(ctx context.Context, userID int64) ([]models.MonthlyGenreBreakdown, error) {
	log.Debug("getting monthly genre breakdown")

	rawData, err := d.queries.GetMonthlyGenreBreakdown(ctx, &userID)
	if err != nil {
		log.Error("failed to get monthly genre data", "error", err)
		return nil, fmt.Errorf("failed to get monthly genre data: %w", err)
	}

	monthMap := make(map[string]map[string]int)
	for _, row := range rawData {
		monthStr := row.WatchedDate.Format("2006-01")
		if monthMap[monthStr] == nil {
			monthMap[monthStr] = make(map[string]int)
		}
		monthMap[monthStr][row.GenreName] = int(row.MovieCount)
	}

	// Convert to slice and sort by month
	result := make([]models.MonthlyGenreBreakdown, 0, len(monthMap))
	for month := range monthMap {
		result = append(result, models.MonthlyGenreBreakdown{
			Month:  month,
			Genres: monthMap[month],
		})
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Month < result[j].Month
	})

	log.Debug("retrieved monthly genre breakdown", "months", len(result))
	return result, nil
}

func (d *SqliteDB) GetRatingSummary(ctx context.Context, userID int64) (*models.RatingSummary, error) {
	log.Debug("getting rating summary")

	row, err := d.queries.GetRatingSummary(ctx, &userID)
	if err != nil {
		log.Error("failed to get rating summary", "error", err)
		return nil, fmt.Errorf("failed to get rating summary: %w", err)
	}

	result := &models.RatingSummary{
		AverageRating: row.AverageRating,
		RatedCount:    row.RatedCount,
	}

	log.Debug("retrieved rating summary", "averageRating", result.AverageRating, "ratedCount", result.RatedCount)
	return result, nil
}

func (d *SqliteDB) GetRatingDistribution(ctx context.Context, userID int64) ([]models.RatingBucketCount, error) {
	log.Debug("getting rating distribution")

	data, err := d.queries.GetRatingDistribution(ctx, &userID)
	if err != nil {
		log.Error("failed to get rating distribution", "error", err)
		return nil, fmt.Errorf("failed to get rating distribution: %w", err)
	}

	result := make([]models.RatingBucketCount, len(data))
	for i, row := range data {
		result[i] = models.RatingBucketCount{
			Rating: row.RatingBucket,
			Count:  row.Count,
		}
	}

	log.Debug("retrieved rating distribution", "count", len(result))
	return result, nil
}

func (d *SqliteDB) GetMonthlyAverageRatingLastYear(ctx context.Context, userID int64) ([]models.PeriodRating, error) {
	log.Debug("getting monthly average rating")

	data, err := d.queries.GetMonthlyAverageRatingLastYear(ctx, &userID)
	if err != nil {
		log.Error("failed to get monthly average rating", "error", err)
		return nil, fmt.Errorf("failed to get monthly average rating: %w", err)
	}

	result := make([]models.PeriodRating, len(data))
	for i, row := range data {
		result[i] = models.PeriodRating{
			Period:        row.Month,
			AverageRating: row.AverageRating,
			RatedCount:    row.RatedCount,
		}
	}

	log.Debug("retrieved monthly average rating", "count", len(result))
	return result, nil
}

func (d *SqliteDB) GetTheaterVsHomeAverageRating(ctx context.Context, userID int64) ([]models.TheaterRating, error) {
	log.Debug("getting theater vs home average rating")

	data, err := d.queries.GetTheaterVsHomeAverageRating(ctx, &userID)
	if err != nil {
		log.Error("failed to get theater vs home average rating", "error", err)
		return nil, fmt.Errorf("failed to get theater vs home average rating: %w", err)
	}

	result := make([]models.TheaterRating, len(data))
	for i, row := range data {
		result[i] = models.TheaterRating{
			InTheater:     row.WatchedInTheater,
			AverageRating: row.AverageRating,
			RatedCount:    row.RatedCount,
		}
	}

	log.Debug("retrieved theater vs home average rating", "count", len(result))
	return result, nil
}

func (d *SqliteDB) GetHighestRatedMovies(ctx context.Context, userID int64, limit int) ([]models.RatedMovie, error) {
	log.Debug("getting highest rated movies", "limit", limit)

	data, err := d.queries.GetHighestRatedMovies(ctx, sqlc.GetHighestRatedMoviesParams{UserID: &userID, Limit: int64(limit)})
	if err != nil {
		log.Error("failed to get highest rated movies", "error", err)
		return nil, fmt.Errorf("failed to get highest rated movies: %w", err)
	}

	result := make([]models.RatedMovie, len(data))
	for i, row := range data {
		result[i] = models.RatedMovie{
			ID:              row.ID,
			Title:           row.Title,
			PosterPath:      row.PosterPath,
			AverageRating:   row.AverageRating,
			RatedWatchCount: row.RatedWatchCount,
		}
	}

	log.Debug("retrieved highest rated movies", "count", len(result))
	return result, nil
}

func (d *SqliteDB) GetRatingVsTMDB(ctx context.Context, userID int64, minVoteCount int) (*models.RatingVsTMDB, error) {
	log.Debug("getting rating vs TMDB", "minVoteCount", minVoteCount)

	row, err := d.queries.GetRatingVsTMDB(ctx, sqlc.GetRatingVsTMDBParams{UserID: &userID, VoteCount: int64(minVoteCount)})
	if err != nil {
		log.Error("failed to get rating vs TMDB", "error", err)
		return nil, fmt.Errorf("failed to get rating vs TMDB: %w", err)
	}

	result := &models.RatingVsTMDB{
		AverageUserRating:  row.AverageUserRating,
		AverageTMDBRating:  row.AverageTmdbRating,
		AverageDifference:  row.AverageDifference,
		ComparedMovieCount: row.ComparedMovieCount,
	}

	log.Debug("retrieved rating vs TMDB", "comparedMovieCount", result.ComparedMovieCount, "averageDifference", result.AverageDifference)
	return result, nil
}

func (d *SqliteDB) GetRatingByReleaseDecade(ctx context.Context, userID int64) ([]models.DecadeRating, error) {
	log.Debug("getting rating by release decade")

	data, err := d.queries.GetRatingByReleaseDecade(ctx, &userID)
	if err != nil {
		log.Error("failed to get rating by release decade", "error", err)
		return nil, fmt.Errorf("failed to get rating by release decade: %w", err)
	}

	result := make([]models.DecadeRating, len(data))
	for i, row := range data {
		result[i] = models.DecadeRating{
			Decade:          int(row.Decade),
			AverageRating:   row.AverageRating,
			RatedMovieCount: row.RatedMovieCount,
		}
	}

	log.Debug("retrieved rating by release decade", "count", len(result))
	return result, nil
}

func (d *SqliteDB) GetFavoriteDirectorsByRating(ctx context.Context, userID int64, minRatedMovies, limit int) ([]models.RatedPerson, error) {
	log.Debug("getting favorite directors by rating", "minRatedMovies", minRatedMovies, "limit", limit)

	data, err := d.queries.GetFavoriteDirectorsByRating(ctx, &userID)
	if err != nil {
		log.Error("failed to get favorite directors by rating", "error", err)
		return nil, fmt.Errorf("failed to get favorite directors by rating: %w", err)
	}

	people := make([]models.RatedPerson, len(data))
	for i, row := range data {
		people[i] = models.RatedPerson{
			ID:              row.ID,
			Name:            row.Name,
			ProfilePath:     row.ProfilePath,
			AverageRating:   row.AverageRating,
			RatedMovieCount: row.RatedMovieCount,
		}
	}

	result := filterRatedPeople(people, minRatedMovies, limit)
	log.Debug("retrieved favorite directors by rating", "count", len(result))
	return result, nil
}

func (d *SqliteDB) GetFavoriteActorsByRating(ctx context.Context, userID int64, minRatedMovies, limit int) ([]models.RatedPerson, error) {
	log.Debug("getting favorite actors by rating", "minRatedMovies", minRatedMovies, "limit", limit)

	data, err := d.queries.GetFavoriteActorsByRating(ctx, &userID)
	if err != nil {
		log.Error("failed to get favorite actors by rating", "error", err)
		return nil, fmt.Errorf("failed to get favorite actors by rating: %w", err)
	}

	people := make([]models.RatedPerson, len(data))
	for i, row := range data {
		people[i] = models.RatedPerson{
			ID:              row.ID,
			Name:            row.Name,
			ProfilePath:     row.ProfilePath,
			Gender:          row.Gender,
			AverageRating:   row.AverageRating,
			RatedMovieCount: row.RatedMovieCount,
		}
	}

	result := filterRatedPeople(people, minRatedMovies, limit)
	log.Debug("retrieved favorite actors by rating", "count", len(result))
	return result, nil
}

func (d *SqliteDB) GetRewatchRatingDrift(ctx context.Context, userID int64, minRatedWatches, limit int) ([]models.RewatchRatingDrift, error) {
	log.Debug("getting rewatch rating drift", "minRatedWatches", minRatedWatches, "limit", limit)

	data, err := d.queries.GetRewatchRatingDrift(ctx, sqlc.GetRewatchRatingDriftParams{UserID: &userID, Column2: int64(minRatedWatches), Limit: int64(limit)})
	if err != nil {
		log.Error("failed to get rewatch rating drift", "error", err)
		return nil, fmt.Errorf("failed to get rewatch rating drift: %w", err)
	}

	result := make([]models.RewatchRatingDrift, len(data))
	for i, row := range data {
		firstWatchedDate, err := scanDateValue(row.FirstWatchedDate)
		if err != nil {
			return nil, fmt.Errorf("failed to parse first watched date for movie %d: %w", row.ID, err)
		}

		lastWatchedDate, err := scanDateValue(row.LastWatchedDate)
		if err != nil {
			return nil, fmt.Errorf("failed to parse last watched date for movie %d: %w", row.ID, err)
		}

		result[i] = models.RewatchRatingDrift{
			MovieID:          row.ID,
			Title:            row.Title,
			PosterPath:       row.PosterPath,
			FirstRating:      row.FirstRating,
			LastRating:       row.LastRating,
			RatingChange:     row.RatingChange,
			RatedWatchCount:  row.RatedWatchCount,
			FirstWatchedDate: firstWatchedDate,
			LastWatchedDate:  lastWatchedDate,
		}
	}

	log.Debug("retrieved rewatch rating drift", "count", len(result))
	return result, nil
}

func filterRatedPeople(people []models.RatedPerson, minRatedMovies, limit int) []models.RatedPerson {
	result := make([]models.RatedPerson, 0, len(people))
	for _, person := range people {
		if minRatedMovies > 0 && person.RatedMovieCount < int64(minRatedMovies) {
			continue
		}
		result = append(result, person)
		if limit > 0 && len(result) >= limit {
			break
		}
	}
	return result
}

func scanDateValue(value any) (time.Time, error) {
	var parsed date.Date
	if err := parsed.Scan(value); err != nil {
		return time.Time{}, err
	}
	return parsed.Time, nil
}

func scanStringValue(value any) (string, error) {
	switch v := value.(type) {
	case string:
		return v, nil
	case []byte:
		return string(v), nil
	case nil:
		return "", nil
	default:
		return "", fmt.Errorf("unexpected string value type: %T", value)
	}
}

func (d *SqliteDB) CreateSession(ctx context.Context, id string, userID int64, expiresAt time.Time) error {
	log.Debug("creating session", "sessionID", id, "userID", userID)

	err := d.queries.CreateSession(ctx, sqlc.CreateSessionParams{
		ID:        id,
		UserID:    userID,
		ExpiresAt: expiresAt,
	})
	if err != nil {
		log.Error("failed to create session", "sessionID", id, "error", err)
		return fmt.Errorf("failed to create session with ID %s: %w", id, err)
	}
	log.Debug("successfully created session", "sessionID", id)
	return nil
}

func (d *SqliteDB) GetSession(ctx context.Context, id string) (*models.Session, error) {
	log.Debug("retrieving session", "sessionID", id)

	session, err := d.queries.GetSession(ctx, id)
	if err != nil {
		log.Error("failed to get session", "sessionID", id, "error", err)
		return nil, fmt.Errorf("failed to get session with ID %s: %w", id, err)
	}

	log.Debug("retrieved session", "sessionID", id, "userID", session.UserID)
	return &models.Session{
		UserID:    session.UserID,
		ExpiresAt: session.ExpiresAt,
	}, nil
}

func (d *SqliteDB) DeleteSession(ctx context.Context, id string) error {
	log.Debug("deleting session", "sessionID", id)

	err := d.queries.DeleteSession(ctx, id)
	if err != nil {
		log.Error("failed to delete session", "sessionID", id, "error", err)
		return fmt.Errorf("failed to delete session with ID %s: %w", id, err)
	}

	log.Debug("successfully deleted session", "sessionID", id)
	return nil
}

func (d *SqliteDB) CleanupExpiredSessions(ctx context.Context) error {
	log.Debug("cleaning up expired sessions")

	err := d.queries.DeleteExpiredSessions(ctx)
	if err != nil {
		log.Error("failed to cleanup expired sessions", "error", err)
		return fmt.Errorf("failed to cleanup expired sessions: %w", err)
	}

	log.Debug("successfully cleaned up expired sessions")
	return nil
}

func (d *SqliteDB) CreateUser(ctx context.Context, email, name, passwordHash string) (*models.User, error) {
	log.Debug("creating new user", "email", email)

	user, err := d.queries.CreateUser(ctx, sqlc.CreateUserParams{
		Email:        email,
		Name:         name,
		PasswordHash: passwordHash,
	})
	if err != nil {
		log.Error("failed to create user in database", "email", email, "error", err)
		return nil, fmt.Errorf("failed to create user %s: %w", email, err)
	}

	log.Debug("successfully created user", "user_id", user.ID, "email", email)
	return &models.User{
		ID:                    user.ID,
		Email:                 user.Email,
		Name:                  user.Name,
		PasswordHash:          user.PasswordHash,
		Admin:                 user.Admin,
		CreatedAt:             user.CreatedAt,
		PasswordResetRequired: user.PasswordResetRequired,
	}, nil
}

func (d *SqliteDB) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
	log.Debug("retrieving user by email", "email", email)

	user, err := d.queries.GetUserByEmail(ctx, email)
	if err != nil {
		log.Error("failed to retrieve user from database", "email", email, "error", err)
		return nil, fmt.Errorf("failed to get user %s: %w", email, err)
	}

	log.Debug("successfully retrieved user", "email", email, "user_id", user.ID)
	return &models.User{
		ID:                    user.ID,
		Email:                 user.Email,
		Name:                  user.Name,
		PasswordHash:          user.PasswordHash,
		Admin:                 user.Admin,
		CreatedAt:             user.CreatedAt,
		PasswordResetRequired: user.PasswordResetRequired,
	}, nil
}

func (d *SqliteDB) GetUserByID(ctx context.Context, id int64) (*models.User, error) {
	log.Debug("retrieving user by ID", "userID", id)

	user, err := d.queries.GetUserByID(ctx, id)
	if err != nil {
		log.Error("failed to retrieve user from database", "userID", id, "error", err)
		return nil, fmt.Errorf("failed to get user with ID %d: %w", id, err)
	}

	log.Debug("successfully retrieved user", "userID", id, "email", user.Email)
	return &models.User{
		ID:                    user.ID,
		Email:                 user.Email,
		Name:                  user.Name,
		PasswordHash:          user.PasswordHash,
		Admin:                 user.Admin,
		CreatedAt:             user.CreatedAt,
		PasswordResetRequired: user.PasswordResetRequired,
	}, nil
}

func (d *SqliteDB) CountUsers(ctx context.Context) (int64, error) {
	log.Debug("counting users")

	count, err := d.queries.CountUsers(ctx)
	if err != nil {
		log.Error("failed to count users", "error", err)
		return 0, fmt.Errorf("failed to count users: %w", err)
	}

	log.Debug("retrieved user count", "count", count)
	return count, nil
}

func (d *SqliteDB) AssignNilUserWatched(ctx context.Context, userID *int64) error {
	log.Debug("assigning nil user to watched records", "userID", userID)

	err := d.queries.AssignNilUserWatched(ctx, userID)
	if err != nil {
		log.Error("failed to assign nil user to watched", "userID", userID, "error", err)
		return fmt.Errorf("failed to assign nil user to watched: %w", err)
	}

	log.Debug("assigned nil user to watched records", "userID", userID)
	return nil
}

func (d *SqliteDB) AssignNilUserLists(ctx context.Context, userID *int64) error {
	log.Debug("assigning nil user to list records", "userID", userID)

	err := d.queries.AssignNilUserLists(ctx, userID)
	if err != nil {
		log.Error("failed to assign nil user to lists", "userID", userID, "error", err)
		return fmt.Errorf("failed to assign nil user to lists: %w", err)
	}

	log.Debug("assigned nil user to list records", "userID", userID)
	return nil
}

func (d *SqliteDB) SetAdmin(ctx context.Context, userID int64) error {
	log.Debug("setting user as admin", "userID", userID)

	err := d.queries.SetAdmin(ctx, userID)
	if err != nil {
		log.Error("failed to set user as admin", "userID", userID, "error", err)
		return fmt.Errorf("failed to set user as admin: %w", err)
	}

	log.Debug("set user as admin", "userID", userID)
	return nil
}

func (d *SqliteDB) GetAllUsersWithStats(ctx context.Context) ([]models.UserWithStats, error) {
	log.Debug("retrieving all users with stats")

	rows, err := d.queries.GetAllUsersWithStats(ctx)
	if err != nil {
		log.Error("failed to fetch users with stats", "error", err)
		return nil, fmt.Errorf("failed to fetch users with stats: %w", err)
	}

	users := make([]models.UserWithStats, len(rows))
	for i, r := range rows {
		users[i] = models.UserWithStats{
			User: models.User{
				ID:           r.ID,
				Email:        r.Email,
				Name:         r.Name,
				Admin:        r.Admin,
				CreatedAt:    r.CreatedAt,
				PasswordHash: "", // Don't expose this in stats
			},
			WatchedCount: r.WatchedCount,
			ListCount:    r.ListCount,
		}
	}

	log.Debug("retrieved users with stats", "count", len(users))
	return users, nil
}

func (d *SqliteDB) DeleteUser(ctx context.Context, userID int64) error {
	log.Debug("deleting user", "userID", userID)

	err := d.queries.DeleteUser(ctx, userID)
	if err != nil {
		log.Error("failed to delete user", "userID", userID, "error", err)
		return fmt.Errorf("failed to delete user %d: %w", userID, err)
	}

	log.Debug("successfully deleted user", "userID", userID)
	return nil
}

func (d *SqliteDB) UpdateUserPassword(ctx context.Context, userID int64, passwordHash string) error {
	log.Debug("updating user password", "userID", userID)

	err := d.queries.UpdateUserPassword(ctx, sqlc.UpdateUserPasswordParams{
		PasswordHash: passwordHash,
		ID:           userID,
	})
	if err != nil {
		log.Error("failed to update user password", "userID", userID, "error", err)
		return fmt.Errorf("failed to update password for user %d: %w", userID, err)
	}

	log.Debug("successfully updated user password", "userID", userID)
	return nil
}

func (d *SqliteDB) UpdatePasswordResetRequired(ctx context.Context, userID int64, reset bool) error {
	log.Debug("updating password reset required", "userID", userID)

	err := d.queries.UpdatePasswordResetRequired(ctx, sqlc.UpdatePasswordResetRequiredParams{
		PasswordResetRequired: reset,
		ID:                    userID,
	})
	if err != nil {
		log.Error("failed to update password reset required", "userID", userID, "error", err)
		return fmt.Errorf("failed to update password reset required for user %d: %w", userID, err)
	}

	log.Debug("successfully updated password reset required", "userID", userID)
	return nil
}

func (d *SqliteDB) ExportLists(ctx context.Context, userID int64) ([]models.List, error) {
	log.Debug("exporting all lists with movies", "userID", userID)

	results, err := d.queries.GetAllListsWithMovies(ctx, &userID)
	if err != nil {
		log.Error("failed to fetch lists with movies", "userID", userID, "error", err)
		return nil, fmt.Errorf("failed to fetch lists with movies: %w", err)
	}

	// create the list with a deterministic order
	lists := []models.List{}
	listIndexes := map[int64]int{}

	for _, result := range results {
		listID := result.List.ID

		idx, ok := listIndexes[listID]
		if !ok {
			creationDate, err := time.Parse("2006-01-02 15:04:05.999999999 -0700 MST", result.List.CreationDate)
			if err != nil {
				log.Error("failed to parse list creation_date", "listID", listID, "error", err)
				return nil, fmt.Errorf("failed to parse creation_date for list %d: %w", listID, err)
			}

			lists = append(lists, models.List{
				ID:           result.List.ID,
				Name:         result.List.Name,
				CreationDate: creationDate,
				Description:  result.List.Description,
				IsWatchlist:  result.List.IsWatchlist,
				Movies:       []models.MovieItem{},
			})
			idx = len(lists) - 1
			listIndexes[listID] = idx
		}

		// Only add movie if it exists (LEFT JOIN may have null movies for empty lists)
		if result.Movie.ID != 0 {
			dateAdded, err := time.Parse("2006-01-02 15:04:05.999999999 -0700 MST", result.ListMovie.DateAdded)
			if err != nil {
				log.Error("failed to parse movie date_added in list", "listID", listID, "movieID", result.Movie.ID, "error", err)
				return nil, fmt.Errorf("failed to parse date_added for movie %d: %w", result.Movie.ID, err)
			}

			movieItem := models.MovieItem{
				MovieDetails: toModelsMovieDetails(result.Movie),
				DateAdded:    dateAdded,
				Position:     result.ListMovie.Position,
				Note:         result.ListMovie.Note,
			}

			lists[idx].Movies = append(lists[idx].Movies, movieItem)
		}
	}

	log.Info("successfully exported lists", "userID", userID, "listCount", len(lists))
	return lists, nil
}
