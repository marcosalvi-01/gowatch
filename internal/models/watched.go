package models

import "time"

// TrendDirection represents the direction of a trend
type TrendDirection string

const (
	TrendUp      TrendDirection = "up"
	TrendDown    TrendDirection = "down"
	TrendNeutral TrendDirection = "neutral"
)

// WatchedMovie is a movie and some more info relative for the specific watch
type WatchedMovie struct {
	MovieDetails MovieDetails
	Date         time.Time
	InTheaters   bool
	Rating       *float64
}

// WatchedMoviesInDay represents a day and all the movies watched in that day.
// Used in the watched page to group movies by watched day
type WatchedMoviesInDay struct {
	Date   time.Time
	Movies []WatchedMovieInDay
}

type WatchedMovieInDay struct {
	MovieDetails MovieDetails
	InTheaters   bool
	Rating       *float64
}

type WatchedMovieRecord struct {
	Date       time.Time
	InTheaters bool
	Rating     *float64
}

type WatchedMovieRecords struct {
	MovieDetails MovieDetails
	Records      []WatchedMovieRecord
}

type PersonWatchRoleKind string

const (
	PersonWatchRoleKindActing PersonWatchRoleKind = "acting"
	PersonWatchRoleKindCrew   PersonWatchRoleKind = "crew"
)

type PersonWatchRole struct {
	Kind  PersonWatchRoleKind
	Label string
}

type PersonWatchMovieMatch struct {
	ID              int64
	Title           string
	PosterPath      string
	WatchCount      int64
	LastWatchedDate time.Time
	Role            PersonWatchRole
}

type PersonWatchActivity struct {
	TotalWatchCount  int64
	ActingMovieCount int
	CrewMovieCount   int
	ActorRank        *int64
	Movies           []PersonWatchedMovie
}

type PersonWatchedMovie struct {
	ID              int64
	Title           string
	PosterPath      string
	WatchCount      int64
	LastWatchedDate time.Time
	Roles           []PersonWatchRole
}

// WatchedStats contains all statistics for watched movies
type WatchedStats struct {
	TotalWatched                int64
	RewatchStats                RewatchStats
	LongestStreak               StreakStats
	Ratings                     RatingStats
	TheaterVsHome               []TheaterCount
	MonthlyLastYear             []PeriodCount
	YearlyAllTime               []PeriodCount
	DailyWatchCountsLastYear    []DailyWatchCount
	WeekdayDistribution         []PeriodCount
	Genres                      []GenreCount
	ReleaseYearDistribution     []ReleaseYearCount
	MostWatchedMovies           []TopMovie
	MostWatchedDay              *MostWatchedDay
	MostWatchedActors           []TopActor
	TopDirectors                []TopCrewMember
	TopWriters                  []TopCrewMember
	TopComposers                []TopCrewMember
	TopCinematographers         []TopCrewMember
	TopLanguages                []LanguageCount
	LongestMovieWatched         *RuntimeMovie
	ShortestMovieWatched        *RuntimeMovie
	BudgetTierDistribution      []BudgetTierCount
	TopReturnOnInvestmentMovies []MovieFinancial
	BiggestBudgetMovies         []MovieFinancial
	AvgPerDay                   float64
	AvgPerWeek                  float64
	AvgPerMonth                 float64
	TotalHoursWatched           float64
	AvgHoursPerDay              float64
	AvgHoursPerWeek             float64
	AvgHoursPerMonth            float64
	MonthlyHoursLastYear        []PeriodHours
	MonthlyHoursTrendDirection  TrendDirection
	MonthlyHoursTrendValue      float64
	MonthlyMoviesTrendDirection TrendDirection
	MonthlyMoviesTrendValue     int64
	MonthlyGenreBreakdown       []MonthlyGenreBreakdown
}

type RatingStats struct {
	Summary            RatingSummary
	Distribution       []RatingBucketCount
	MonthlyAverage     []PeriodRating
	TheaterVsHome      []TheaterRating
	HighestRatedMovies []RatedMovie
	VsTMDB             RatingVsTMDB
	ReleaseDecades     []DecadeRating
	FavoriteDirectors  []RatedPerson
	FavoriteActors     []RatedPerson
	RewatchDrift       []RewatchRatingDrift
}

type RatingSummary struct {
	AverageRating float64
	RatedCount    int64
	UnratedCount  int64
	Coverage      float64
}

type RatingBucketCount struct {
	Rating float64
	Count  int64
}

type PeriodRating struct {
	Period        string
	AverageRating float64
	RatedCount    int64
}

type TheaterRating struct {
	InTheater     bool
	AverageRating float64
	RatedCount    int64
}

type RatedMovie struct {
	ID              int64
	Title           string
	PosterPath      string
	AverageRating   float64
	RatedWatchCount int64
}

type RatingVsTMDB struct {
	AverageUserRating  float64
	AverageTMDBRating  float64
	AverageDifference  float64
	ComparedMovieCount int64
}

type DecadeRating struct {
	Decade          int
	AverageRating   float64
	RatedMovieCount int64
}

type RatedPerson struct {
	ID              int64
	Name            string
	ProfilePath     string
	Gender          int64
	AverageRating   float64
	RatedMovieCount int64
}

type RewatchRatingDrift struct {
	MovieID          int64
	Title            string
	PosterPath       string
	FirstRating      float64
	LastRating       float64
	RatingChange     float64
	RatedWatchCount  int64
	FirstWatchedDate time.Time
	LastWatchedDate  time.Time
}

type MonthlyGenreBreakdown struct {
	Month  string
	Genres map[string]int
}

type PeriodCount struct {
	Period string
	Count  int64
}

type PeriodHours struct {
	Period string
	Hours  float64
}

type PeriodStats struct {
	Period string
	Count  int64
	Hours  float64
}

type TotalStats struct {
	Count int64
	Hours float64
}

type GenreCount struct {
	Name  string
	Count int64
}

type TheaterCount struct {
	InTheater bool
	Count     int64
}

type TopMovie struct {
	Title      string
	ID         int64
	PosterPath string
	WatchCount int64
}

type RewatchStats struct {
	UniqueMovieCount    int64
	RewatchedMovieCount int64
	RewatchCount        int64
}

type StreakStats struct {
	CurrentDays  int64
	LongestDays  int64
	LongestStart *time.Time
	LongestEnd   *time.Time
}

type DailyWatchCount struct {
	Date  time.Time
	Count int64
}

type ReleaseYearCount struct {
	Year  int
	Count int64
}

type TopCrewMember struct {
	ID          int64
	Name        string
	ProfilePath string
	WatchCount  int64
}

type LanguageCount struct {
	Language   string
	WatchCount int64
}

type RuntimeMovie struct {
	ID             int64
	Title          string
	PosterPath     string
	RuntimeMinutes int64
}

type BudgetTier string

const (
	BudgetTierIndie       BudgetTier = "indie"
	BudgetTierMid         BudgetTier = "mid"
	BudgetTierBlockbuster BudgetTier = "blockbuster"
	BudgetTierUnknown     BudgetTier = "unknown"
)

func BudgetTierFromString(value string) BudgetTier {
	switch BudgetTier(value) {
	case BudgetTierIndie, BudgetTierMid, BudgetTierBlockbuster, BudgetTierUnknown:
		return BudgetTier(value)
	default:
		return BudgetTierUnknown
	}
}

type BudgetTierCount struct {
	Tier  BudgetTier
	Count int64
}

type MovieFinancial struct {
	ID         int64
	Title      string
	PosterPath string
	Budget     int64
	Revenue    int64
	ROI        float64
}

type TopActor struct {
	Name        string
	ID          int64
	ProfilePath string
	WatchCount  int64
	Gender      int64
}

type MostWatchedDay struct {
	Date  time.Time
	Count int64
}

type DateRange struct {
	MinDate *time.Time
	MaxDate *time.Time
}
