package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"time"

	"gowatch/db"
	"gowatch/internal/models"
	"gowatch/logging"
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
	if movieID <= 0 {
		return fmt.Errorf("invalid movie ID")
	}
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
			_, err := s.tmdb.GetMovieDetails(ctx, int64(movieRef.MovieID))
			if err != nil {
				s.log.Error("failed to fetch movie details", "movieID", movieRef.MovieID, "date", importMovie.Date, "error", err)
				return fmt.Errorf("failed to fetch movie details: %w", err)
			}

			err = s.AddWatched(ctx, int64(movieRef.MovieID), importMovie.Date, movieRef.InTheaters)
			if err != nil {
				s.log.Error("failed to import movie", "movieID", movieRef.MovieID, "date", importMovie.Date, "error", err)
				return fmt.Errorf("failed to import movie: %w", err)
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

func (s *WatchedService) GetRecentWatchedMovies(ctx context.Context, limit int) ([]models.WatchedMovieInDay, error) {
	s.log.Debug("retrieving recent watched movies", "limit", limit)

	result, err := s.db.GetRecentWatchedMovies(ctx, limit)
	if err != nil {
		s.log.Error("failed to fetch recent watched movies from database", "error", err)
		return nil, fmt.Errorf("failed to fetch recent watched movies: %w", err)
	}

	s.log.Debug("retrieved recent watched movies", "count", len(result))
	return result, nil
}

func (s *WatchedService) GetWatchedStats(ctx context.Context, limit int) (*models.WatchedStats, error) {
	stats := &models.WatchedStats{}

	var err error
	stats.TotalWatched, err = s.getTotalWatched(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get total watched: %w", err)
	}

	stats.TheaterVsHome, err = s.getTheaterVsHome(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get theater vs home data: %w", err)
	}

	stats.MonthlyLastYear, err = s.getMonthlyLastYear(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get monthly data: %w", err)
	}

	stats.YearlyAllTime, err = s.getYearlyAllTime(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get yearly data: %w", err)
	}

	stats.WeekdayDistribution, err = s.getWeekdayDistribution(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get weekday distribution: %w", err)
	}

	stats.Genres, err = s.getGenres(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get genre data: %w", err)
	}

	stats.MostWatchedMovies, err = s.getMostWatchedMovies(ctx, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get most watched movies: %w", err)
	}

	stats.MostWatchedDay, err = s.getMostWatchedDay(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get most watched day: %w", err)
	}

	stats.MostWatchedActors, err = s.getMostWatchedActors(ctx, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get most watched actors: %w", err)
	}

	stats.AvgPerDay, stats.AvgPerWeek, stats.AvgPerMonth, err = s.getAverages(ctx, stats.TotalWatched)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate averages: %w", err)
	}

	stats.TotalHoursWatched, err = s.getTotalHoursWatched(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get total hours watched: %w", err)
	}

	stats.MonthlyHoursLastYear, err = s.getMonthlyHoursLastYear(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get monthly hours data: %w", err)
	}

	stats.MonthlyHoursTrendDirection, stats.MonthlyHoursTrendValue = s.calculateMonthlyHoursTrend(stats.MonthlyHoursLastYear)

	stats.MonthlyGenreBreakdown, err = s.getMonthlyGenreBreakdown(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get monthly genre breakdown: %w", err)
	}

	stats.MonthlyMoviesTrendDirection, stats.MonthlyMoviesTrendValue = s.calculateMonthlyMoviesTrend(stats.MonthlyLastYear)

	stats.AvgHoursPerDay, stats.AvgHoursPerWeek, stats.AvgHoursPerMonth, err = s.getHoursAverages(ctx, stats.TotalHoursWatched)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate hours averages: %w", err)
	}

	return stats, nil
}
