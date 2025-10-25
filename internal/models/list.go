package models

import "time"

// List is a list of movies
type List struct {
	ID           int64
	Name         string
	CreationDate time.Time
	Description  *string

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
