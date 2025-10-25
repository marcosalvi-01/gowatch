package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"gowatch/db/sqlc"
	"gowatch/internal/models"
	"sort"
	"time"
)

// UpsertMovie adds a new movie to the database
func (d *SqliteDB) UpsertMovie(ctx context.Context, movie *models.MovieDetails) error {
	log.Debug("inserting movie into database", "movieID", movie.Movie.ID, "title", movie.Movie.Title)

	tx, err := d.db.Begin()
	if err != nil {
		log.Error("failed to start database transaction for movie insert", "movieID", movie.Movie.ID, "error", err)
		return fmt.Errorf("failed to start db transaction: %w", err)
	}
	defer tx.Rollback()

	qtx := d.queries.WithTx(tx)

	err = qtx.UpsertMovie(ctx, sqlc.UpsertMovieParams{
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
	}

	log.Debug("processed cast, inserting crew", "movieID", movie.Movie.ID, "crewCount", len(movie.Credits.Crew))

	for _, crew := range movie.Credits.Crew {
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
		MovieID:          watched.MovieID,
		WatchedDate:      watched.Date,
		WatchedInTheater: watched.InTheaters,
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

func (d *SqliteDB) InsertList(ctx context.Context, list InsertList) error {
	log.Debug("inserting new list into database", "name", list.Name)

	err := d.queries.InsertList(ctx, sqlc.InsertListParams{
		Name:         list.Name,
		CreationDate: time.Now().Format("2006-01-02 15:04:05.999999999 -0700 MST"),
		Description:  list.Description,
	})
	if err != nil {
		log.Error("failed to insert list", "name", list.Name, "error", err)
		return fmt.Errorf("failed to insert list %q: %w", list.Name, err)
	}

	log.Info("successfully inserted list", "name", list.Name)
	return nil
}

func (d *SqliteDB) GetList(ctx context.Context, id int64) (*models.List, error) {
	log.Debug("retrieving list with movies", "listID", id)

	tx, err := d.db.Begin()
	if err != nil {
		log.Error("failed to start database transaction for list retrieval", "listID", id, "error", err)
		return nil, fmt.Errorf("failed to start db transaction: %w", err)
	}
	defer tx.Rollback()

	qtx := d.queries.WithTx(tx)

	results, err := qtx.GetListJoinMovieByID(ctx, id)
	if err != nil {
		log.Error("failed to fetch list with movies", "listID", id, "error", err)
		return nil, fmt.Errorf("failed to fetch list with ID %d: %w", id, err)
	}
	if len(results) == 0 {
		log.Debug("list has no movies associated to it", "listID", id)
		// try to search for the list without joining in case it is empty
		list, err := qtx.GetListByID(ctx, id)
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
		Movies:       movies,
	}, nil
}

func (d *SqliteDB) AddMovieToList(ctx context.Context, insertMovieList InsertMovieList) error {
	log.Debug("adding movie to list", "movieID", insertMovieList.MovieID, "position", insertMovieList.Position)

	err := d.queries.AddMovieToList(ctx, sqlc.AddMovieToListParams{
		MovieID:   insertMovieList.MovieID,
		ListID:    insertMovieList.ListID,
		DateAdded: insertMovieList.DateAdded.Format("2006-01-02 15:04:05.999999999 -0700 MST"),
		Position:  insertMovieList.Position,
		Note:      insertMovieList.Note,
	})
	if err != nil {
		log.Error("failed to add movie to list", "movieID", insertMovieList.MovieID, "error", err)
		return fmt.Errorf("failed to add movie %d to list: %w", insertMovieList.MovieID, err)
	}

	log.Info("successfully added movie to list", "movieID", insertMovieList.MovieID)
	return nil
}

func (d *SqliteDB) GetAllLists(ctx context.Context) ([]InsertList, error) {
	log.Debug("retrieving all lists from database")

	results, err := d.queries.GetAllLists(ctx)
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
		}
	}

	log.Info("successfully retrieved all lists", "count", len(lists))
	return lists, nil
}

func (d *SqliteDB) GetWatchedCount(ctx context.Context) (int64, error) {
	log.Debug("getting watched count")

	count, err := d.queries.GetWatchedCount(ctx)
	if err != nil {
		log.Error("failed to get watched count", "error", err)
		return 0, fmt.Errorf("failed to get watched movie count: %w", err)
	}

	log.Debug("retrieved watched count", "count", count)
	return count, nil
}

func (d *SqliteDB) DeleteListByID(ctx context.Context, id int64) error {
	log.Debug("deleting list by ID", "listID", id)

	err := d.queries.DeleteListByID(ctx, id)
	if err != nil {
		log.Error("failed to delete list", "listID", id, "error", err)
		return fmt.Errorf("failed to delete list with id '%d' in db: %w", id, err)
	}

	log.Debug("successfully deleted list", "listID", id)
	return nil
}

func (d *SqliteDB) DeleteMovieFromList(ctx context.Context, listID, movieID int64) error {
	log.Debug("deleting movie from list", "listID", listID, "movieID", movieID)

	err := d.queries.DeleteMovieFromList(ctx, sqlc.DeleteMovieFromListParams{
		ListID:  listID,
		MovieID: movieID,
	})
	if err != nil {
		log.Error("failed to delete movie from list", "listID", listID, "movieID", movieID, "error", err)
		return fmt.Errorf("failed to delete movie '%d' from list '%d' in db: %w", movieID, listID, err)
	}

	log.Debug("successfully deleted movie from list", "listID", listID, "movieID", movieID)
	return nil
}

func (d *SqliteDB) GetWatchedPerMonthLastYear(ctx context.Context) ([]models.PeriodCount, error) {
	log.Debug("getting watched per month last year")

	data, err := d.queries.GetWatchedPerMonthLastYear(ctx)
	if err != nil {
		log.Error("failed to get monthly data", "error", err)
		return nil, fmt.Errorf("failed to get monthly data: %w", err)
	}
	counts := make(map[string]int64)
	for _, t := range data {
		period := t.Format("2006-01")
		counts[period]++
	}
	result := make([]models.PeriodCount, 0, len(counts))
	for period, count := range counts {
		result = append(result, models.PeriodCount{Period: period, Count: count})
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Period < result[j].Period
	})

	log.Debug("retrieved monthly data", "periodCount", len(result))
	return result, nil
}

func (d *SqliteDB) GetWatchedPerYear(ctx context.Context) ([]models.PeriodCount, error) {
	log.Debug("getting watched per year")

	data, err := d.queries.GetWatchedPerYear(ctx)
	if err != nil {
		log.Error("failed to get yearly data", "error", err)
		return nil, fmt.Errorf("failed to get yearly data: %w", err)
	}
	counts := make(map[string]int64)
	for _, t := range data {
		period := t.Format("2006")
		counts[period]++
	}
	result := make([]models.PeriodCount, 0, len(counts))
	for period, count := range counts {
		result = append(result, models.PeriodCount{Period: period, Count: count})
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Period < result[j].Period
	})

	log.Debug("retrieved yearly data", "periodCount", len(result))
	return result, nil
}

func (d *SqliteDB) GetWatchedByGenre(ctx context.Context) ([]models.GenreCount, error) {
	log.Debug("getting watched by genre")

	data, err := d.queries.GetWatchedByGenre(ctx)
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

func (d *SqliteDB) GetTheaterVsHomeCount(ctx context.Context) ([]models.TheaterCount, error) {
	log.Debug("getting theater vs home count")

	data, err := d.queries.GetTheaterVsHomeCount(ctx)
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

func (d *SqliteDB) GetMostWatchedMovies(ctx context.Context) ([]models.TopMovie, error) {
	log.Debug("getting most watched movies")

	data, err := d.queries.GetMostWatchedMovies(ctx)
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

func (d *SqliteDB) GetMostWatchedDay(ctx context.Context) (*models.MostWatchedDay, error) {
	log.Debug("getting most watched day")

	data, err := d.queries.GetMostWatchedDay(ctx)
	if err != nil {
		log.Error("failed to get most watched day", "error", err)
		return nil, fmt.Errorf("failed to get most watched day: %w", err)
	}
	if len(data) == 0 {
		log.Debug("no watched days found")
		return nil, sql.ErrNoRows
	}
	counts := make(map[string]int64)
	for _, t := range data {
		day := t.Format("2006-01-02")
		counts[day]++
	}
	var maxDay string
	var maxCount int64
	for day, count := range counts {
		if count > maxCount {
			maxCount = count
			maxDay = day
		}
	}

	t, err := time.Parse("2006-01-02", maxDay)
	if err != nil {
		log.Error("failed to parse most watched day", "error", err, "day", maxDay)
		return nil, fmt.Errorf("failed to parse most watched day: %w", err)
	}

	log.Debug("retrieved most watched day", "date", t, "count", maxCount)
	return &models.MostWatchedDay{Date: t, Count: maxCount}, nil
}

func (d *SqliteDB) GetMostWatchedActors(ctx context.Context) ([]models.TopActor, error) {
	log.Debug("getting most watched actors")

	data, err := d.queries.GetMostWatchedActors(ctx)
	if err != nil {
		log.Error("failed to get most watched actors", "error", err)
		return nil, fmt.Errorf("failed to get most watched actors: %w", err)
	}
	result := make([]models.TopActor, len(data))
	for i, d := range data {
		result[i] = models.TopActor{Name: d.Name, ID: d.ID, MovieCount: d.MovieCount, ProfilePath: d.ProfilePath}
	}

	log.Debug("retrieved most watched actors", "count", len(result))
	return result, nil
}

func (d *SqliteDB) GetWatchedDateRange(ctx context.Context) (*models.DateRange, error) {
	log.Debug("getting watched date range")

	data, err := d.queries.GetWatchedDateRange(ctx)
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
		parsed, err := time.Parse("2006-01-02 15:04:05 -0700 MST", minStr)
		if err != nil {
			log.Error("failed to parse min date", "date", minStr, "error", err)
			return nil, fmt.Errorf("failed to parse min date %q: %w", minStr, err)
		}
		min = &parsed
	}
	if data.MaxDate != nil {
		maxStr, ok := data.MaxDate.(string)
		if !ok {
			log.Error("unexpected type for MaxDate", "type", fmt.Sprintf("%T", data.MaxDate))
			return nil, fmt.Errorf("unexpected type for MaxDate: %T", data.MaxDate)
		}
		parsed, err := time.Parse("2006-01-02 15:04:05 -0700 MST", maxStr)
		if err != nil {
			log.Error("failed to parse max date", "date", maxStr, "error", err)
			return nil, fmt.Errorf("failed to parse max date %q: %w", maxStr, err)
		}
		max = &parsed
	}

	log.Debug("retrieved watched date range", "minDate", min, "maxDate", max)
	return &models.DateRange{MinDate: min, MaxDate: max}, nil
}
