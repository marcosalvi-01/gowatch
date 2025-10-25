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
	"sort"
	"time"
)

const MaxGenresDisplayed = 11

// WatchedService handles user's watched movie tracking
type WatchedService struct {
	db   db.DB
	tmdb *MovieService
	log  *slog.Logger
}

func NewWatchedService(db db.DB, tmdb *MovieService) *WatchedService {
	log := logging.Get("watched service")
	log.Debug("creating new WatchedService instance")
	return &WatchedService{
		db:   db,
		tmdb: tmdb,
		log:  log,
	}
}

func (s *WatchedService) AddWatched(ctx context.Context, movieID int64, date time.Time, inTheaters bool) error {
	s.log.Debug("adding watched movie", "movieID", movieID, "date", date, "inTheaters", inTheaters)

	err := s.db.InsertWatched(ctx, db.InsertWatched{
		MovieID:    movieID,
		Date:       date,
		InTheaters: inTheaters,
	})
	if err != nil {
		s.log.Error("failed to insert watched entry", "movieID", movieID, "error", err)
		return fmt.Errorf("failed to record watched entry: %w", err)
	}

	s.log.Info("successfully added watched movie", "movieID", movieID)
	return nil
}

func (s *WatchedService) GetAllWatchedMoviesInDay(ctx context.Context) ([]models.WatchedMoviesInDay, error) {
	s.log.Debug("retrieving all watched movies grouped by day")

	movies, err := s.db.GetWatchedJoinMovie(ctx)
	if err != nil {
		s.log.Error("failed to fetch watched movies from database", "error", err)
		return nil, fmt.Errorf("failed to fetch watched join movie: %w", err)
	}

	s.log.Debug("fetched watched movies from database", "count", len(movies))

	sort.Slice(movies, func(i, j int) bool {
		return movies[i].Date.After(movies[j].Date)
	})

	var out []models.WatchedMoviesInDay
	for _, m := range movies {
		d := m.Date.Truncate(24 * time.Hour)
		if len(out) == 0 || !d.Equal(out[len(out)-1].Date) {
			out = append(out, models.WatchedMoviesInDay{Date: d})
		}
		out[len(out)-1].Movies = append(out[len(out)-1].Movies, models.WatchedMovieInDay{
			MovieDetails: m.MovieDetails,
			InTheaters:   m.InTheaters,
		})
	}

	s.log.Debug("grouped movies by day", "dayCount", len(out))
	return out, nil
}

func (s *WatchedService) ImportWatched(ctx context.Context, movies models.ImportWatchedMoviesLog) error {
	totalMovies := 0
	for _, importMovie := range movies {
		totalMovies += len(importMovie.Movies)
	}

	s.log.Info("starting watched movies import", "totalDays", len(movies), "totalMovies", totalMovies)

	for _, importMovie := range movies {
		for _, movieRef := range importMovie.Movies {
			err := s.AddWatched(ctx, int64(movieRef.MovieID), importMovie.Date, movieRef.InTheaters)
			if err != nil {
				s.log.Error("failed to import movie", "movieID", movieRef.MovieID, "date", importMovie.Date, "error", err)
				return fmt.Errorf("failed to import movie: %w", err)
			}

			_, err = s.tmdb.GetMovieDetails(ctx, int64(movieRef.MovieID))
			if err != nil {
				s.log.Error("failed to cache movie in db", "movieID", movieRef.MovieID, "date", importMovie.Date, "error", err)
				return fmt.Errorf("failed to cache movie in db: %w", err)
			}
		}
	}

	s.log.Info("successfully imported watched movies", "totalMovies", totalMovies)
	return nil
}

func (s *WatchedService) ExportWatched(ctx context.Context) (models.ImportWatchedMoviesLog, error) {
	s.log.Debug("starting watched movies export")

	watched, err := s.GetAllWatchedMoviesInDay(ctx)
	if err != nil {
		s.log.Error("failed to get watched movies for export", "error", err)
		return nil, fmt.Errorf("failed to get all watched movies for export: %w", err)
	}

	s.log.Debug("retrieved watched movies for export", "dayCount", len(watched))

	export := make(models.ImportWatchedMoviesLog, len(watched))
	totalMovies := 0

	for i, w := range watched {
		ids := make([]models.ImportWatchedMovieRef, len(w.Movies))
		for j, movieDetails := range w.Movies {
			ids[j] = models.ImportWatchedMovieRef{
				MovieID:    int(movieDetails.MovieDetails.Movie.ID),
				InTheaters: movieDetails.InTheaters,
			}
		}
		totalMovies += len(w.Movies)
		export[i] = models.ImportWatchedMoviesEntry{
			Date:   w.Date,
			Movies: ids,
		}
	}

	s.log.Info("successfully exported watched movies", "dayCount", len(export), "totalMovies", totalMovies)
	return export, nil
}

func (s *WatchedService) GetWatchedMovieRecordsByID(ctx context.Context, movieID int64) (models.WatchedMovieRecords, error) {
	s.log.Debug("get watch records", "movieID", movieID)

	rows, err := s.db.GetWatchedJoinMovieByID(ctx, movieID)
	if errors.Is(err, sql.ErrNoRows) || len(rows) == 0 {
		return models.WatchedMovieRecords{}, nil
	}
	if err != nil {
		s.log.Error("db query failed", "movieID", movieID, "error", err)
		return models.WatchedMovieRecords{}, fmt.Errorf("get watched records: %w", err)
	}

	rec := models.WatchedMovieRecords{
		MovieDetails: rows[0].MovieDetails, // same in every row
		Records:      make([]models.WatchedMovieRecord, 0, len(rows)),
	}
	for _, r := range rows {
		rec.Records = append(rec.Records, models.WatchedMovieRecord{
			Date:       r.Date,
			InTheaters: r.InTheaters,
		})
	}

	sort.Slice(rec.Records, func(i, j int) bool {
		return rec.Records[i].Date.After(rec.Records[j].Date)
	})

	return rec, nil
}

func (s *WatchedService) GetWatchedCount(ctx context.Context) (int64, error) {
	count, err := s.db.GetWatchedCount(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to get watched count from db: %w", err)
	}

	s.log.Debug("retrieved watched count", "count", count)

	return count, nil
}

func (s *WatchedService) GetWatchedStats(ctx context.Context) (*models.WatchedStats, error) {
	stats := &models.WatchedStats{}

	// Total watched
	s.log.Debug("retrieving total watched count")
	total, err := s.db.GetWatchedCount(ctx)
	if err != nil {
		s.log.Error("failed to retrieve total watched count", "error", err)
		return nil, fmt.Errorf("failed to get total watched: %w", err)
	}
	stats.TotalWatched = total

	// Theater vs home
	s.log.Debug("retrieving theater vs home counts")
	theaterData, err := s.db.GetTheaterVsHomeCount(ctx)
	if err != nil {
		s.log.Error("failed to retrieve theater vs home counts", "error", err)
		return nil, fmt.Errorf("failed to get theater vs home: %w", err)
	}
	stats.TheaterVsHome = make([]models.TheaterCount, len(theaterData))
	for i, d := range theaterData {
		stats.TheaterVsHome[i] = models.TheaterCount{
			InTheater: d.InTheater,
			Count:     d.Count,
		}
	}

	// Monthly last year
	s.log.Debug("retrieving monthly watched data")
	monthlyData, err := s.db.GetWatchedPerMonthLastYear(ctx)
	if err != nil {
		s.log.Error("failed to retrieve monthly watched data", "error", err)
		return nil, fmt.Errorf("failed to get monthly data: %w", err)
	}
	stats.MonthlyLastYear = monthlyData

	// Yearly all time
	s.log.Debug("retrieving yearly watched data")
	yearlyData, err := s.db.GetWatchedPerYear(ctx)
	if err != nil {
		s.log.Error("failed to retrieve yearly watched data", "error", err)
		return nil, fmt.Errorf("failed to get yearly data: %w", err)
	}
	stats.YearlyAllTime = yearlyData

	// Genres
	s.log.Debug("retrieving watched by genre data")
	genreData, err := s.db.GetWatchedByGenre(ctx)
	if err != nil {
		s.log.Error("failed to retrieve watched by genre data", "error", err)
		return nil, fmt.Errorf("failed to get genre data: %w", err)
	}
	if len(genreData) <= MaxGenresDisplayed {
		stats.Genres = make([]models.GenreCount, len(genreData))
		for i, d := range genreData {
			stats.Genres[i] = models.GenreCount{
				Name:  d.Name,
				Count: d.Count,
			}
		}
	} else {
		stats.Genres = make([]models.GenreCount, MaxGenresDisplayed+1)
		for i := 0; i < MaxGenresDisplayed; i++ {
			stats.Genres[i] = models.GenreCount{
				Name:  genreData[i].Name,
				Count: genreData[i].Count,
			}
		}
		var othersCount int64
		for i := MaxGenresDisplayed; i < len(genreData); i++ {
			othersCount += genreData[i].Count
		}
		stats.Genres[MaxGenresDisplayed] = models.GenreCount{
			Name:  "Others",
			Count: othersCount,
		}
	}

	// Most watched movies
	s.log.Debug("retrieving most watched movies")
	movieData, err := s.db.GetMostWatchedMovies(ctx)
	if err != nil {
		s.log.Error("failed to retrieve most watched movies", "error", err)
		return nil, fmt.Errorf("failed to get most watched movies: %w", err)
	}
	stats.MostWatchedMovies = make([]models.TopMovie, len(movieData))
	for i, d := range movieData {
		stats.MostWatchedMovies[i] = models.TopMovie{
			Title:      d.Title,
			ID:         d.ID,
			PosterPath: d.PosterPath,
			WatchCount: d.WatchCount,
		}
	}

	// Most watched day
	s.log.Debug("retrieving most watched day")
	dayData, err := s.db.GetMostWatchedDay(ctx)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		s.log.Error("failed to retrieve most watched day", "error", err)
		return nil, fmt.Errorf("failed to get most watched day: %w", err)
	}
	switch err {
	case nil:
		stats.MostWatchedDay = &models.MostWatchedDay{
			Date:  dayData.Date,
			Count: dayData.Count,
		}
	case sql.ErrNoRows:
		s.log.Debug("no watched days found")
	}

	// Most watched actors
	s.log.Debug("retrieving most watched actors")
	actorData, err := s.db.GetMostWatchedActors(ctx)
	if err != nil {
		s.log.Error("failed to retrieve most watched actors", "error", err)
		return nil, fmt.Errorf("failed to get most watched actors: %w", err)
	}
	stats.MostWatchedActors = make([]models.TopActor, len(actorData))
	for i, d := range actorData {
		stats.MostWatchedActors[i] = models.TopActor{
			Name:        d.Name,
			ID:          d.ID,
			ProfilePath: d.ProfilePath,
			MovieCount:  d.MovieCount,
		}
	}

	// Averages
	s.log.Debug("retrieving watched date range for average calculations")
	dateRange, err := s.db.GetWatchedDateRange(ctx)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		s.log.Error("failed to retrieve watched date range", "error", err)
		return nil, fmt.Errorf("failed to get date range: %w", err)
	}
	if err == sql.ErrNoRows {
		s.log.Debug("no valid watched dates found, skipping average calculations")
	}
	if err == nil && dateRange.MinDate != nil && dateRange.MaxDate != nil {
		days := dateRange.MaxDate.Sub(*dateRange.MinDate).Hours()/24 + 1
		stats.AvgPerDay = float64(total) / days
		stats.AvgPerWeek = float64(total) / (days / 7)
		stats.AvgPerMonth = float64(total) / (days / 30)
		s.log.Debug("calculated averages", "avgPerDay", stats.AvgPerDay, "avgPerWeek", stats.AvgPerWeek, "avgPerMonth", stats.AvgPerMonth)
	}

	return stats, nil
}
