package models

import "time"

type ListMovieSort string

const (
	ListMovieSortCustom          ListMovieSort = "custom"
	ListMovieSortDateAddedDesc   ListMovieSort = "date_added_desc"
	ListMovieSortDateAddedAsc    ListMovieSort = "date_added_asc"
	ListMovieSortTitleAsc        ListMovieSort = "title_asc"
	ListMovieSortRatingDesc      ListMovieSort = "rating_desc"
	ListMovieSortReleaseDateDesc ListMovieSort = "release_date_desc"
	ListMovieSortReleaseStatus   ListMovieSort = "release_status"
)

// List is a list of movies
type List struct {
	ID           int64
	Name         string
	CreationDate time.Time
	Description  *string
	IsWatchlist  bool
	DisplaySort  ListMovieSort

	Movies []MovieItem
}

// MovieItem represents a Movie inside a list
type MovieItem struct {
	MovieDetails MovieDetails
	DateAdded    time.Time
	Position     *int64
	Note         *string
}

type ListEntry struct {
	ID   int64
	Name string
}

type ListViewData struct {
	List             *List
	Sort             ListMovieSort
	IsEditing        bool
	GridURL          string
	ToggleEditURL    string
	HasAverageRating bool
	AverageRating    float32
	UpcomingMovies   []MovieItem
	ReleasedMovies   []MovieItem
}
