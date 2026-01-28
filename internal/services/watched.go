package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"time"

	"github.com/marcosalvi-01/gowatch/db"
	"github.com/marcosalvi-01/gowatch/internal/common"
	"github.com/marcosalvi-01/gowatch/internal/models"
	"github.com/marcosalvi-01/gowatch/logging"
	"golang.org/x/sync/errgroup"
)

const MaxGenresDisplayed = 11

// WatchedService handles user's watched movie tracking
type WatchedService struct {
	db          db.DB
	listService *ListService
	tmdb        *MovieService
	log         *slog.Logger
}

func NewWatchedService(db db.DB, listService *ListService, tmdb *MovieService) *WatchedService {
	log := logging.Get("watched service")
	log.Debug("creating new WatchedService instance")
	return &WatchedService{
		db:          db,
		listService: listService,
		tmdb:        tmdb,
		log:         log,
	}
}

func (s *WatchedService) AddWatched(ctx context.Context, movieID int64, date time.Time, inTheaters bool, rating *float64) error {
	if movieID <= 0 {
		return fmt.Errorf("AddWatched: invalid movie ID")
	}
	if rating != nil && (*rating < 0 || *rating > 5) {
		return fmt.Errorf("AddWatched: rating must be between 0 and 5")
	}
	user, err := common.GetUser(ctx)
	if err != nil {
		s.log.Error("AddWatched: failed to get user", "error", err)
		return fmt.Errorf("AddWatched: failed to get user: %w", err)
	}

	s.log.Debug("AddWatched: adding watched movie", "movieID", movieID, "date", date, "inTheaters", inTheaters, "rating", rating, "userID", user.ID)

	err = s.db.InsertWatched(ctx, db.InsertWatched{
		UserID:     user.ID,
		MovieID:    movieID,
		Date:       date,
		InTheaters: inTheaters,
		Rating:     rating,
	})
	if err != nil {
		s.log.Error("AddWatched: failed to insert watched entry", "movieID", movieID, "error", err, "userID", user.ID)
		return fmt.Errorf("AddWatched: failed to record watched entry: %w", err)
	}

	err = s.listService.RemoveMovieFromWatchlist(ctx, movieID)
	if err != nil {
		s.log.Warn("AddWatched: failed to auto-remove movie from watchlist after marking as watched", "movieID", movieID, "error", err)
		// don't stop on fail
	}

	s.log.Info("AddWatched: successfully added watched movie", "movieID", movieID, "userID", user.ID)
	return nil
}

func (s *WatchedService) GetAllWatchedMoviesInDay(ctx context.Context) ([]models.WatchedMoviesInDay, error) {
	s.log.Debug("GetAllWatchedMoviesInDay: retrieving all watched movies grouped by day")

	user, err := common.GetUser(ctx)
	if err != nil {
		s.log.Error("GetAllWatchedMoviesInDay: failed to get user", "error", err)
		return nil, fmt.Errorf("GetAllWatchedMoviesInDay: failed to get user: %w", err)
	}

	movies, err := s.db.GetWatchedJoinMovie(ctx, user.ID)
	if err != nil {
		s.log.Error("GetAllWatchedMoviesInDay: failed to fetch watched movies from database", "error", err)
		return nil, fmt.Errorf("GetAllWatchedMoviesInDay: failed to fetch watched join movie: %w", err)
	}

	s.log.Debug("GetAllWatchedMoviesInDay: fetched watched movies from database", "count", len(movies))

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
			Rating:       m.Rating,
		})
	}

	s.log.Debug("GetAllWatchedMoviesInDay: grouped movies by day", "dayCount", len(out))
	return out, nil
}

func (s *WatchedService) ImportWatched(ctx context.Context, movies models.ImportWatchedMoviesLog) error {
	totalMovies := 0
	for _, importMovie := range movies {
		totalMovies += len(importMovie.Movies)
	}

	s.log.Info("ImportWatched: starting watched movies import", "totalDays", len(movies), "totalMovies", totalMovies)

	for _, importMovie := range movies {
		for _, movieRef := range importMovie.Movies {
			_, err := s.tmdb.GetMovieDetails(ctx, movieRef.MovieID)
			if err != nil {
				s.log.Error("ImportWatched: failed to fetch movie details", "movieID", movieRef.MovieID, "date", importMovie.Date, "error", err)
				return fmt.Errorf("ImportWatched: failed to fetch movie details: %w", err)
			}

			err = s.AddWatched(ctx, movieRef.MovieID, importMovie.Date, movieRef.InTheaters, movieRef.Rating)
			if err != nil {
				s.log.Error("ImportWatched: failed to import movie", "movieID", movieRef.MovieID, "date", importMovie.Date, "error", err)
				return fmt.Errorf("ImportWatched: failed to import movie: %w", err)
			}
		}
	}

	s.log.Info("ImportWatched: successfully imported watched movies", "totalMovies", totalMovies)
	return nil
}

// ImportAll imports both watched movies and lists from combined format
func (s *WatchedService) ImportAll(ctx context.Context, data models.ImportAllData) error {
	s.log.Info("ImportAll: starting combined import", "watchedDays", len(data.Watched), "lists", len(data.Lists))

	// Import watched movies first
	if len(data.Watched) > 0 {
		if err := s.ImportWatched(ctx, data.Watched); err != nil {
			s.log.Error("ImportAll: failed to import watched movies", "error", err)
			return fmt.Errorf("ImportAll: failed to import watched movies: %w", err)
		}
	}

	// Import lists
	if len(data.Lists) > 0 {
		if err := s.listService.ImportLists(ctx, data.Lists); err != nil {
			s.log.Error("ImportAll: failed to import lists", "error", err)
			return fmt.Errorf("ImportAll: failed to import lists: %w", err)
		}
	}

	s.log.Info("ImportAll: successfully imported all data")
	return nil
}

func (s *WatchedService) ExportWatched(ctx context.Context) (models.ImportWatchedMoviesLog, error) {
	s.log.Debug("ExportWatched: starting watched movies export")

	watched, err := s.GetAllWatchedMoviesInDay(ctx)
	if err != nil {
		s.log.Error("ExportWatched: failed to get watched movies for export", "error", err)
		return nil, fmt.Errorf("ExportWatched: failed to get all watched movies for export: %w", err)
	}

	s.log.Debug("ExportWatched: retrieved watched movies for export", "dayCount", len(watched))

	export := make(models.ImportWatchedMoviesLog, len(watched))
	totalMovies := 0

	for i, w := range watched {
		ids := make([]models.ImportWatchedMovieRef, len(w.Movies))
		for j, movieDetails := range w.Movies {
			ids[j] = models.ImportWatchedMovieRef{
				MovieID:    movieDetails.MovieDetails.Movie.ID,
				InTheaters: movieDetails.InTheaters,
				Rating:     movieDetails.Rating,
			}
		}
		totalMovies += len(w.Movies)
		export[i] = models.ImportWatchedMoviesEntry{
			Date:   w.Date,
			Movies: ids,
		}
	}

	s.log.Info("ExportWatched: successfully exported watched movies", "dayCount", len(export), "totalMovies", totalMovies)
	return export, nil
}

func (s *WatchedService) GetWatchedMovieRecordsByID(ctx context.Context, movieID int64) (models.WatchedMovieRecords, error) {
	s.log.Debug("GetWatchedMovieRecordsByID: get watch records", "movieID", movieID)

	user, err := common.GetUser(ctx)
	if err != nil {
		s.log.Error("GetWatchedMovieRecordsByID: failed to get user", "error", err)
		return models.WatchedMovieRecords{}, fmt.Errorf("GetWatchedMovieRecordsByID: failed to get user: %w", err)
	}

	rows, err := s.db.GetWatchedJoinMovieByID(ctx, user.ID, movieID)
	if errors.Is(err, sql.ErrNoRows) || len(rows) == 0 {
		return models.WatchedMovieRecords{}, nil
	}
	if err != nil {
		s.log.Error("GetWatchedMovieRecordsByID: db query failed", "movieID", movieID, "error", err)
		return models.WatchedMovieRecords{}, fmt.Errorf("GetWatchedMovieRecordsByID: get watched records: %w", err)
	}

	rec := models.WatchedMovieRecords{
		MovieDetails: rows[0].MovieDetails, // same in every row
		Records:      make([]models.WatchedMovieRecord, 0, len(rows)),
	}
	for _, r := range rows {
		rec.Records = append(rec.Records, models.WatchedMovieRecord{
			Date:       r.Date,
			InTheaters: r.InTheaters,
			Rating:     r.Rating,
		})
	}

	sort.Slice(rec.Records, func(i, j int) bool {
		return rec.Records[i].Date.After(rec.Records[j].Date)
	})

	return rec, nil
}

func (s *WatchedService) GetWatchedCount(ctx context.Context) (int64, error) {
	user, err := common.GetUser(ctx)
	if err != nil {
		s.log.Error("GetWatchedCount: failed to get user", "error", err)
		return 0, fmt.Errorf("GetWatchedCount: failed to get user: %w", err)
	}

	count, err := s.db.GetWatchedCount(ctx, user.ID)
	if err != nil {
		return 0, fmt.Errorf("GetWatchedCount: failed to get watched count from db: %w", err)
	}

	s.log.Debug("GetWatchedCount: retrieved watched count", "count", count)

	return count, nil
}

func (s *WatchedService) GetRecentWatchedMovies(ctx context.Context, limit int) ([]models.WatchedMovieInDay, error) {
	s.log.Debug("GetRecentWatchedMovies: retrieving recent watched movies", "limit", limit)

	user, err := common.GetUser(ctx)
	if err != nil {
		s.log.Error("GetRecentWatchedMovies: failed to get user", "error", err)
		return nil, fmt.Errorf("GetRecentWatchedMovies: failed to get user: %w", err)
	}

	result, err := s.db.GetRecentWatchedMovies(ctx, user.ID, limit)
	if err != nil {
		s.log.Error("GetRecentWatchedMovies: failed to fetch recent watched movies from database", "error", err)
		return nil, fmt.Errorf("GetRecentWatchedMovies: failed to fetch recent watched movies: %w", err)
	}

	s.log.Debug("GetRecentWatchedMovies: retrieved recent watched movies", "count", len(result))
	return result, nil
}

func (s *WatchedService) GetWatchedStats(ctx context.Context, limit int) (*models.WatchedStats, error) {
	start := time.Now()
	s.log.Debug("GetWatchedStats: starting stats calculation", "limit", limit)
	stats := &models.WatchedStats{}
	g, ctx := errgroup.WithContext(ctx)

	var totalStats *models.TotalStats
	var statsPerMonth []models.PeriodStats
	var allActors []models.TopActor
	var dateRange *models.DateRange

	// Total Stats (Count & Hours)
	g.Go(func() error {
		var err error
		s.log.Debug("GetWatchedStats: fetching total stats")
		t := time.Now()
		totalStats, err = s.getTotalStats(ctx)
		s.log.Debug("GetWatchedStats: total stats fetched", "count", totalStats.Count, "hours", totalStats.Hours, "duration", time.Since(t).String())
		return err
	})

	// Theater vs Home
	g.Go(func() error {
		var err error
		s.log.Debug("GetWatchedStats: fetching theater vs home")
		t := time.Now()
		stats.TheaterVsHome, err = s.getTheaterVsHome(ctx)
		s.log.Debug("GetWatchedStats: theater vs home fetched", "count", len(stats.TheaterVsHome), "duration", time.Since(t).String())
		return err
	})

	// Monthly Stats (Count & Hours)
	g.Go(func() error {
		var err error
		s.log.Debug("GetWatchedStats: fetching monthly stats")
		t := time.Now()
		statsPerMonth, err = s.getMonthlyStats(ctx)
		s.log.Debug("GetWatchedStats: monthly stats fetched", "count", len(statsPerMonth), "duration", time.Since(t).String())
		return err
	})

	// Yearly Stats
	g.Go(func() error {
		var err error
		s.log.Debug("GetWatchedStats: fetching yearly stats")
		t := time.Now()
		stats.YearlyAllTime, err = s.getYearlyAllTime(ctx)
		s.log.Debug("GetWatchedStats: yearly stats fetched", "count", len(stats.YearlyAllTime), "duration", time.Since(t).String())
		return err
	})

	// Weekday Distribution
	g.Go(func() error {
		var err error
		s.log.Debug("GetWatchedStats: fetching weekday distribution")
		t := time.Now()
		stats.WeekdayDistribution, err = s.getWeekdayDistribution(ctx)
		s.log.Debug("GetWatchedStats: weekday distribution fetched", "count", len(stats.WeekdayDistribution), "duration", time.Since(t).String())
		return err
	})

	// Genres
	g.Go(func() error {
		var err error
		s.log.Debug("GetWatchedStats: fetching genres")
		t := time.Now()
		stats.Genres, err = s.getGenres(ctx)
		s.log.Debug("GetWatchedStats: genres fetched", "count", len(stats.Genres), "duration", time.Since(t).String())
		return err
	})

	// Most Watched Movies
	g.Go(func() error {
		var err error
		s.log.Debug("GetWatchedStats: fetching most watched movies")
		t := time.Now()
		stats.MostWatchedMovies, err = s.getMostWatchedMovies(ctx, limit)
		s.log.Debug("GetWatchedStats: most watched movies fetched", "count", len(stats.MostWatchedMovies), "duration", time.Since(t).String())
		return err
	})

	// Most Watched Day
	g.Go(func() error {
		var err error
		s.log.Debug("GetWatchedStats: fetching most watched day")
		t := time.Now()
		stats.MostWatchedDay, err = s.getMostWatchedDay(ctx)
		s.log.Debug("GetWatchedStats: most watched day fetched", "duration", time.Since(t).String())
		return err
	})

	// Most Watched Actors
	g.Go(func() error {
		var err error
		s.log.Debug("GetWatchedStats: fetching most watched actors")
		t := time.Now()
		allActors, err = s.getMostWatchedActors(ctx, limit)
		s.log.Debug("GetWatchedStats: most watched actors fetched", "count", len(allActors), "duration", time.Since(t).String())
		return err
	})

	// Monthly Genre Breakdown
	g.Go(func() error {
		var err error
		s.log.Debug("GetWatchedStats: fetching monthly genre breakdown")
		t := time.Now()
		stats.MonthlyGenreBreakdown, err = s.getMonthlyGenreBreakdown(ctx)
		s.log.Debug("GetWatchedStats: monthly genre breakdown fetched", "count", len(stats.MonthlyGenreBreakdown), "duration", time.Since(t).String())
		return err
	})

	// Date Range for Averages
	g.Go(func() error {
		var err error
		s.log.Debug("GetWatchedStats: fetching date range")
		t := time.Now()
		dateRange, err = s.getDateRange(ctx)
		s.log.Debug("GetWatchedStats: date range fetched", "duration", time.Since(t).String())
		return err
	})

	if err := g.Wait(); err != nil {
		s.log.Error("GetWatchedStats: failed to fetch data concurrently", "error", err, "duration", time.Since(start).String())
		return nil, err
	}

	fetchDuration := time.Since(start)
	s.log.Debug("GetWatchedStats: all data fetched concurrently", "duration", fetchDuration.String())

	// Process Amalgamated Results
	stats.TotalWatched = totalStats.Count
	stats.TotalHoursWatched = totalStats.Hours
	stats.MostWatchedActors = allActors

	// Split monthly stats
	stats.MonthlyLastYear = make([]models.PeriodCount, len(statsPerMonth))
	stats.MonthlyHoursLastYear = make([]models.PeriodHours, len(statsPerMonth))
	for i, item := range statsPerMonth {
		stats.MonthlyLastYear[i] = models.PeriodCount{Period: item.Period, Count: item.Count}
		stats.MonthlyHoursLastYear[i] = models.PeriodHours{Period: item.Period, Hours: item.Hours}
	}

	// Calculate Averages and Trends
	now := time.Now()
	stats.AvgPerDay, stats.AvgPerWeek, stats.AvgPerMonth = s.calculateAverages(stats.TotalWatched, dateRange, now)
	stats.AvgHoursPerDay, stats.AvgHoursPerWeek, stats.AvgHoursPerMonth = s.calculateHoursAverages(stats.TotalHoursWatched, dateRange, now)

	stats.MonthlyHoursTrendDirection, stats.MonthlyHoursTrendValue = s.calculateMonthlyHoursTrend(stats.MonthlyHoursLastYear)
	stats.MonthlyMoviesTrendDirection, stats.MonthlyMoviesTrendValue = s.calculateMonthlyMoviesTrend(stats.MonthlyLastYear)

	totalDuration := time.Since(start)
	s.log.Info("GetWatchedStats: stats calculation completed", "totalWatched", stats.TotalWatched, "totalHours", stats.TotalHoursWatched, "fetchDuration", fetchDuration.String(), "processDuration", (totalDuration - fetchDuration).String(), "totalDuration", totalDuration.String())

	return stats, nil
}
