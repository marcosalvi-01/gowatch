package models

type SearchResultType string

const (
	SearchResultTypeMovie  SearchResultType = "movie"
	SearchResultTypePerson SearchResultType = "person"
)

type SearchResult struct {
	ID            int64
	Title         string
	ImagePath     string
	SecondaryText string
	VoteAverage   float32
	Type          SearchResultType
}
