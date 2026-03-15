package models

import "time"

type PersonDetailsPage struct {
	ID                 int64
	Name               string
	Biography          string
	ProfilePath        string
	KnownForDepartment string
	Popularity         float32

	Birthday     *time.Time
	Deathday     *time.Time
	PlaceOfBirth string
	IMDbID       string
	Homepage     string

	KnownFor      []PersonCredit
	ActingCredits []PersonCredit
	CrewCredits   []PersonCredit
}

type PersonCredit struct {
	ID           int64
	CreditID     string
	MediaType    string
	Title        string
	Role         string
	Department   string
	EpisodeCount int
	ReleaseDate  *time.Time
	BackdropPath string
	PosterPath   string
	VoteAverage  float32
	VoteCount    int64
	Popularity   float32
}
