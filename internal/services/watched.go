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

const (
	MaxGenresDisplayed        = 11
	minFavoriteDirectorMovies = 2
	minFavoriteActorMovies    = 3
	minRewatchRatedWatches    = 2
	minTMDBVoteCount          = 100
	ratingBucketSize          = 0.5
	maxMovieRating            = 5.0
)

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
				continue
			}

			err = s.AddWatched(ctx, movieRef.MovieID, importMovie.Date, movieRef.InTheaters, movieRef.Rating)
			if err != nil {
				s.log.Error("ImportWatched: failed to import movie", "movieID", movieRef.MovieID, "date", importMovie.Date, "error", err)
				continue
			}
		}
	}

	s.log.Info("ImportWatched: completed watched movies import", "totalMovies", totalMovies)
	return nil
}

// ImportAll imports both watched movies and lists from combined format
func (s *WatchedService) ImportAll(ctx context.Context, data models.ImportAllData) error {
	s.log.Info("ImportAll: starting combined import", "watchedDays", len(data.Watched), "lists", len(data.Lists))

	// Import watched movies first
	if len(data.Watched) > 0 {
		if err := s.ImportWatched(ctx, data.Watched); err != nil {
			s.log.Error("ImportAll: failed to import watched movies", "error", err)
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

func (s *WatchedService) GetPersonWatchActivity(ctx context.Context, personID int64) (models.PersonWatchActivity, error) {
	s.log.Debug("GetPersonWatchActivity: get person watch activity", "personID", personID)

	if personID <= 0 {
		return models.PersonWatchActivity{}, fmt.Errorf("GetPersonWatchActivity: invalid person ID")
	}

	user, err := common.GetUser(ctx)
	if err != nil {
		s.log.Error("GetPersonWatchActivity: failed to get user", "error", err)
		return models.PersonWatchActivity{}, fmt.Errorf("GetPersonWatchActivity: failed to get user: %w", err)
	}

	movies, err := s.db.GetWatchedMoviesByPerson(ctx, user.ID, personID)
	if err != nil {
		s.log.Error("GetPersonWatchActivity: db query failed", "personID", personID, "error", err)
		return models.PersonWatchActivity{}, fmt.Errorf("GetPersonWatchActivity: get watched movies by person: %w", err)
	}

	activity := models.PersonWatchActivity{Movies: []models.PersonWatchedMovie{}}
	movieIndexByID := make(map[int64]int, len(movies))

	for _, movie := range movies {
		index, exists := movieIndexByID[movie.ID]
		if !exists {
			movieIndexByID[movie.ID] = len(activity.Movies)
			activity.Movies = append(activity.Movies, models.PersonWatchedMovie{
				ID:              movie.ID,
				Title:           movie.Title,
				PosterPath:      movie.PosterPath,
				WatchCount:      movie.WatchCount,
				LastWatchedDate: movie.LastWatchedDate,
				Roles:           []models.PersonWatchRole{},
			})
			index = len(activity.Movies) - 1
		}

		activity.Movies[index].Roles = appendUniquePersonWatchRole(activity.Movies[index].Roles, movie.Role)
	}

	for _, movie := range activity.Movies {
		activity.TotalWatchCount += movie.WatchCount
		if personWatchedMovieHasRoleKind(movie, models.PersonWatchRoleKindActing) {
			activity.ActingMovieCount++
		}
		if personWatchedMovieHasRoleKind(movie, models.PersonWatchRoleKindCrew) {
			activity.CrewMovieCount++
		}
	}

	if activity.ActingMovieCount > 0 {
		actors, err := s.db.GetWatchedActors(ctx, user.ID)
		if err != nil {
			s.log.Error("GetPersonWatchActivity: failed to get watched actors", "personID", personID, "error", err)
			return models.PersonWatchActivity{}, fmt.Errorf("GetPersonWatchActivity: get watched actors: %w", err)
		}

		activity.ActorRank = watchedActorRankByGender(actors, personID)
	}

	return activity, nil
}

func appendUniquePersonWatchRole(existing []models.PersonWatchRole, role models.PersonWatchRole) []models.PersonWatchRole {
	for _, existingRole := range existing {
		if existingRole.Kind == role.Kind && existingRole.Label == role.Label {
			return existing
		}
	}

	return append(existing, role)
}

func personWatchedMovieHasRoleKind(movie models.PersonWatchedMovie, kind models.PersonWatchRoleKind) bool {
	for _, role := range movie.Roles {
		if role.Kind == kind {
			return true
		}
	}

	return false
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

func (s *WatchedService) GetDailyWatchCountsLastYear(ctx context.Context) ([]models.DailyWatchCount, error) {
	s.log.Debug("GetDailyWatchCountsLastYear: retrieving daily watch counts")

	data, err := s.getDailyWatchCountsLastYear(ctx)
	if err != nil {
		s.log.Error("GetDailyWatchCountsLastYear: failed to retrieve daily watch counts", "error", err)
		return nil, err
	}

	s.log.Debug("GetDailyWatchCountsLastYear: retrieved daily watch counts", "count", len(data))
	return data, nil
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
	var rewatchStats *models.RewatchStats
	var watchedDates []time.Time
	var ratingSummary *models.RatingSummary
	var ratingVsTMDB *models.RatingVsTMDB
	var ratingDistribution []models.RatingBucketCount
	var monthlyAverageRating []models.PeriodRating
	var theaterVsHomeAverageRating []models.TheaterRating
	var highestRatedMovies []models.RatedMovie
	var ratingByReleaseDecade []models.DecadeRating
	var favoriteDirectorsByRating []models.RatedPerson
	var favoriteActorsByRating []models.RatedPerson
	var rewatchRatingDrift []models.RewatchRatingDrift

	// Total Stats (Count & Hours)
	g.Go(func() error {
		var err error
		s.log.Debug("GetWatchedStats: fetching total stats")
		t := time.Now()
		totalStats, err = s.getTotalStats(ctx)
		s.log.Debug("GetWatchedStats: total stats fetched", "count", totalStats.Count, "hours", totalStats.Hours, "duration", time.Since(t).String())
		return err
	})

	// Rewatch Stats
	g.Go(func() error {
		var err error
		s.log.Debug("GetWatchedStats: fetching rewatch stats")
		t := time.Now()
		rewatchStats, err = s.getRewatchStats(ctx)
		s.log.Debug("GetWatchedStats: rewatch stats fetched", "duration", time.Since(t).String())
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

	// Daily Watch Counts (Last Year) - used for calendar heatmap
	g.Go(func() error {
		var err error
		s.log.Debug("GetWatchedStats: fetching daily watch counts for last year")
		t := time.Now()
		stats.DailyWatchCountsLastYear, err = s.getDailyWatchCountsLastYear(ctx)
		s.log.Debug("GetWatchedStats: daily watch counts for last year fetched", "count", len(stats.DailyWatchCountsLastYear), "duration", time.Since(t).String())
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

	// Top Crew
	g.Go(func() error {
		var err error
		s.log.Debug("GetWatchedStats: fetching top crew members")
		t := time.Now()
		crewMembers, err := s.getWatchedCrewMembers(ctx)
		if err != nil {
			return err
		}

		stats.TopDirectors = filterTopCrewMembersByRole(crewMembers, models.TopCrewRoleDirector, limit)
		stats.TopWriters = filterTopCrewMembersByRole(crewMembers, models.TopCrewRoleWriter, limit)
		stats.TopComposers = filterTopCrewMembersByRole(crewMembers, models.TopCrewRoleComposer, limit)
		stats.TopCinematographers = filterTopCrewMembersByRole(crewMembers, models.TopCrewRoleCinematographer, limit)
		s.log.Debug(
			"GetWatchedStats: top crew members fetched",
			"directors", len(stats.TopDirectors),
			"writers", len(stats.TopWriters),
			"composers", len(stats.TopComposers),
			"cinematographers", len(stats.TopCinematographers),
			"duration", time.Since(t).String(),
		)
		return err
	})

	// Top Languages
	g.Go(func() error {
		var err error
		s.log.Debug("GetWatchedStats: fetching top languages")
		t := time.Now()
		stats.TopLanguages, err = s.getTopLanguages(ctx, limit)
		s.log.Debug("GetWatchedStats: top languages fetched", "count", len(stats.TopLanguages), "duration", time.Since(t).String())
		return err
	})

	// Release Year Distribution
	g.Go(func() error {
		var err error
		s.log.Debug("GetWatchedStats: fetching release year distribution")
		t := time.Now()
		stats.ReleaseYearDistribution, err = s.getReleaseYearDistribution(ctx)
		s.log.Debug("GetWatchedStats: release year distribution fetched", "count", len(stats.ReleaseYearDistribution), "duration", time.Since(t).String())
		return err
	})

	// Longest Watched Movie
	g.Go(func() error {
		var err error
		s.log.Debug("GetWatchedStats: fetching longest watched movie")
		t := time.Now()
		stats.LongestMovieWatched, err = s.getLongestWatchedMovie(ctx)
		s.log.Debug("GetWatchedStats: longest watched movie fetched", "duration", time.Since(t).String())
		return err
	})

	// Shortest Watched Movie
	g.Go(func() error {
		var err error
		s.log.Debug("GetWatchedStats: fetching shortest watched movie")
		t := time.Now()
		stats.ShortestMovieWatched, err = s.getShortestWatchedMovie(ctx)
		s.log.Debug("GetWatchedStats: shortest watched movie fetched", "duration", time.Since(t).String())
		return err
	})

	// Budget Tier Distribution
	g.Go(func() error {
		var err error
		s.log.Debug("GetWatchedStats: fetching budget tier distribution")
		t := time.Now()
		stats.BudgetTierDistribution, err = s.getBudgetTierDistribution(ctx)
		s.log.Debug("GetWatchedStats: budget tier distribution fetched", "count", len(stats.BudgetTierDistribution), "duration", time.Since(t).String())
		return err
	})

	// Top Return on Investment Movies
	g.Go(func() error {
		var err error
		s.log.Debug("GetWatchedStats: fetching top return on investment movies")
		t := time.Now()
		stats.TopReturnOnInvestmentMovies, err = s.getTopReturnOnInvestmentMovies(ctx, limit)
		s.log.Debug("GetWatchedStats: top return on investment movies fetched", "count", len(stats.TopReturnOnInvestmentMovies), "duration", time.Since(t).String())
		return err
	})

	// Biggest Budget Movies
	g.Go(func() error {
		var err error
		s.log.Debug("GetWatchedStats: fetching biggest budget movies")
		t := time.Now()
		stats.BiggestBudgetMovies, err = s.getBiggestBudgetMovies(ctx, limit)
		s.log.Debug("GetWatchedStats: biggest budget movies fetched", "count", len(stats.BiggestBudgetMovies), "duration", time.Since(t).String())
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

	// Rating Summary
	g.Go(func() error {
		var err error
		s.log.Debug("GetWatchedStats: fetching rating summary")
		t := time.Now()
		ratingSummary, err = s.getRatingSummary(ctx)
		s.log.Debug("GetWatchedStats: rating summary fetched", "duration", time.Since(t).String())
		return err
	})

	// Rating Distribution
	g.Go(func() error {
		var err error
		s.log.Debug("GetWatchedStats: fetching rating distribution")
		t := time.Now()
		ratingDistribution, err = s.getRatingDistribution(ctx)
		s.log.Debug("GetWatchedStats: rating distribution fetched", "count", len(ratingDistribution), "duration", time.Since(t).String())
		return err
	})

	// Monthly Average Rating
	g.Go(func() error {
		var err error
		s.log.Debug("GetWatchedStats: fetching monthly average rating")
		t := time.Now()
		monthlyAverageRating, err = s.getMonthlyAverageRatingLastYear(ctx)
		s.log.Debug("GetWatchedStats: monthly average rating fetched", "count", len(monthlyAverageRating), "duration", time.Since(t).String())
		return err
	})

	// Theater vs Home Average Rating
	g.Go(func() error {
		var err error
		s.log.Debug("GetWatchedStats: fetching theater vs home average rating")
		t := time.Now()
		theaterVsHomeAverageRating, err = s.getTheaterVsHomeAverageRating(ctx)
		s.log.Debug("GetWatchedStats: theater vs home average rating fetched", "count", len(theaterVsHomeAverageRating), "duration", time.Since(t).String())
		return err
	})

	// Highest Rated Movies
	g.Go(func() error {
		var err error
		s.log.Debug("GetWatchedStats: fetching highest rated movies")
		t := time.Now()
		highestRatedMovies, err = s.getHighestRatedMovies(ctx, limit)
		s.log.Debug("GetWatchedStats: highest rated movies fetched", "count", len(highestRatedMovies), "duration", time.Since(t).String())
		return err
	})

	// Rating vs TMDB
	g.Go(func() error {
		var err error
		s.log.Debug("GetWatchedStats: fetching rating vs TMDB")
		t := time.Now()
		ratingVsTMDB, err = s.getRatingVsTMDB(ctx)
		s.log.Debug("GetWatchedStats: rating vs TMDB fetched", "duration", time.Since(t).String())
		return err
	})

	// Rating by Release Decade
	g.Go(func() error {
		var err error
		s.log.Debug("GetWatchedStats: fetching rating by release decade")
		t := time.Now()
		ratingByReleaseDecade, err = s.getRatingByReleaseDecade(ctx)
		s.log.Debug("GetWatchedStats: rating by release decade fetched", "count", len(ratingByReleaseDecade), "duration", time.Since(t).String())
		return err
	})

	// Favorite Directors by Rating
	g.Go(func() error {
		var err error
		s.log.Debug("GetWatchedStats: fetching favorite directors by rating")
		t := time.Now()
		favoriteDirectorsByRating, err = s.getFavoriteDirectorsByRating(ctx, limit)
		s.log.Debug("GetWatchedStats: favorite directors by rating fetched", "count", len(favoriteDirectorsByRating), "duration", time.Since(t).String())
		return err
	})

	// Favorite Actors by Rating
	g.Go(func() error {
		var err error
		s.log.Debug("GetWatchedStats: fetching favorite actors by rating")
		t := time.Now()
		favoriteActorsByRating, err = s.getFavoriteActorsByRating(ctx, limit)
		s.log.Debug("GetWatchedStats: favorite actors by rating fetched", "count", len(favoriteActorsByRating), "duration", time.Since(t).String())
		return err
	})

	// Rewatch Rating Drift
	g.Go(func() error {
		var err error
		s.log.Debug("GetWatchedStats: fetching rewatch rating drift")
		t := time.Now()
		rewatchRatingDrift, err = s.getRewatchRatingDrift(ctx, limit)
		s.log.Debug("GetWatchedStats: rewatch rating drift fetched", "count", len(rewatchRatingDrift), "duration", time.Since(t).String())
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

	// Watched Dates for streak calculations
	g.Go(func() error {
		var err error
		s.log.Debug("GetWatchedStats: fetching watched dates")
		t := time.Now()
		watchedDates, err = s.getWatchedDates(ctx)
		s.log.Debug("GetWatchedStats: watched dates fetched", "count", len(watchedDates), "duration", time.Since(t).String())
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
	if rewatchStats != nil {
		stats.RewatchStats = *rewatchStats
	}
	stats.Ratings.Summary = s.finalizeRatingSummary(ratingSummary, stats.TotalWatched)
	stats.Ratings.Distribution = ratingDistribution
	stats.Ratings.MonthlyAverage = monthlyAverageRating
	stats.Ratings.TheaterVsHome = theaterVsHomeAverageRating
	stats.Ratings.HighestRatedMovies = highestRatedMovies
	stats.Ratings.ReleaseDecades = ratingByReleaseDecade
	stats.Ratings.FavoriteDirectors = favoriteDirectorsByRating
	stats.Ratings.FavoriteActors = favoriteActorsByRating
	stats.Ratings.RewatchDrift = rewatchRatingDrift
	if ratingVsTMDB != nil {
		stats.Ratings.VsTMDB = *ratingVsTMDB
	}

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
	stats.LongestStreak = s.calculateStreakStats(watchedDates, now)

	stats.MonthlyHoursTrendDirection, stats.MonthlyHoursTrendValue = s.calculateMonthlyHoursTrend(stats.MonthlyHoursLastYear)
	stats.MonthlyMoviesTrendDirection, stats.MonthlyMoviesTrendValue = s.calculateMonthlyMoviesTrend(stats.MonthlyLastYear)

	totalDuration := time.Since(start)
	s.log.Info("GetWatchedStats: stats calculation completed", "totalWatched", stats.TotalWatched, "totalHours", stats.TotalHoursWatched, "fetchDuration", fetchDuration.String(), "processDuration", (totalDuration - fetchDuration).String(), "totalDuration", totalDuration.String())

	return stats, nil
}
