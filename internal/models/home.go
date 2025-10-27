package models

// HomeData contains all data needed for the home page
type HomeData struct {
	RecentMovies []WatchedMovieInDay
	Stats        HomeStatsSummary
}

// HomeStatsSummary contains key stats for home page overview
type HomeStatsSummary struct {
	TotalWatched int64
	AvgPerWeek   float64
	TopGenre     *GenreCount
}
