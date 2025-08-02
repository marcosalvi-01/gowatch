package models

type Cast struct {
	MovieID   int64
	PersonID  int64
	CastID    int64
	CreditID  string
	Character string
	CastOrder int64
	Person    Person
}

type Crew struct {
	MovieID    int64
	PersonID   int64
	CreditID   string
	Job        string
	Department string
	Person     Person
}

type Person struct {
	ID                 int64
	Name               string
	OriginalName       string
	ProfilePath        string
	KnownForDepartment string
	Popularity         float64
	Gender             int64
	Adult              bool
}

type MovieCredits struct {
	Crew []Crew
	Cast []Cast
}
