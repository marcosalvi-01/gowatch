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

func (d *SqliteDB) GetWatchedPerMonthLastYear(ctx context.Context, userID int64) ([]models.PeriodCount, error) {
	log.Debug("getting watched per month last year")

	data, err := d.queries.GetWatchedPerMonthLastYear(ctx, &userID)
	if err != nil {
		log.Error("failed to get monthly data", "error", err)
		return nil, fmt.Errorf("failed to get monthly data: %w", err)
	}

	result := make([]models.PeriodCount, len(data))
	for i, item := range data {
		result[i] = models.PeriodCount{
			Period: item.Month,
			Count:  item.Count,
		}
	}

	log.Debug("retrieved monthly data", "periodCount", len(result))
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

func (d *SqliteDB) GetMostWatchedMaleActors(ctx context.Context, userID int64, limit int) ([]models.TopActor, error) {
	log.Debug("getting most watched male actors", "limit", limit)

	data, err := d.queries.GetMostWatchedMaleActors(ctx, sqlc.GetMostWatchedMaleActorsParams{UserID: &userID, Limit: int64(limit)})
	if err != nil {
		log.Error("failed to get most watched male actors", "error", err)
		return nil, fmt.Errorf("failed to get most watched male actors: %w", err)
	}
	result := make([]models.TopActor, len(data))
	for i, d := range data {
		result[i] = models.TopActor{Name: d.Name, ID: d.ID, WatchCount: d.WatchCount, ProfilePath: d.ProfilePath, Gender: d.Gender}
	}

	log.Debug("retrieved most watched male actors", "count", len(result))
	return result, nil
}

func (d *SqliteDB) GetMostWatchedFemaleActors(ctx context.Context, userID int64, limit int) ([]models.TopActor, error) {
	log.Debug("getting most watched female actors", "limit", limit)

	data, err := d.queries.GetMostWatchedFemaleActors(ctx, sqlc.GetMostWatchedFemaleActorsParams{UserID: &userID, Limit: int64(limit)})
	if err != nil {
		log.Error("failed to get most watched female actors", "error", err)
		return nil, fmt.Errorf("failed to get most watched female actors: %w", err)
	}
	result := make([]models.TopActor, len(data))
	for i, d := range data {
		result[i] = models.TopActor{Name: d.Name, ID: d.ID, WatchCount: d.WatchCount, ProfilePath: d.ProfilePath, Gender: d.Gender}
	}

	log.Debug("retrieved most watched female actors", "count", len(result))
	return result, nil
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

func (d *SqliteDB) aggregateWatchedByPeriodHours(data []sqlc.GetWatchedRuntimesLastYearRow, format string) []models.PeriodHours {
	hours := make(map[string]float64)
	for _, item := range data {
		period := item.WatchedDate.Format(format)
		hours[period] += float64(item.Runtime) / 60.0 // Convert minutes to hours
	}
	result := make([]models.PeriodHours, 0, len(hours))
	for period, hourCount := range hours {
		result = append(result, models.PeriodHours{Period: period, Hours: hourCount})
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Period < result[j].Period
	})
	return result
}

func (d *SqliteDB) GetWatchedHoursPerMonthLastYear(ctx context.Context, userID int64) ([]models.PeriodHours, error) {
	log.Debug("getting watched hours per month last year")

	data, err := d.queries.GetWatchedRuntimesLastYear(ctx, &userID)
	if err != nil {
		log.Error("failed to get monthly hours data", "error", err)
		return nil, fmt.Errorf("failed to get monthly hours data: %w", err)
	}
	result := d.aggregateWatchedByPeriodHours(data, "2006-01")

	log.Debug("retrieved monthly hours data", "periodCount", len(result))
	return result, nil
}

func (d *SqliteDB) GetTotalHoursWatched(ctx context.Context, userID int64) (float64, error) {
	log.Debug("getting total hours watched")

	totalMinutes, err := d.queries.GetTotalHoursWatched(ctx, &userID)
	if err != nil {
		log.Error("failed to get total minutes", "error", err)
		return 0, fmt.Errorf("failed to get total minutes: %w", err)
	}

	if totalMinutes == nil {
		log.Debug("found 0 minutes in the db")
		return 0, nil
	}
	hours := *totalMinutes / 60.0
	log.Debug("retrieved total hours", "hours", hours)
	return hours, nil
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
