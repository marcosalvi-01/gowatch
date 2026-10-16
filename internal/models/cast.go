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
	ID          int64
	Name        string
	ProfilePath string
	Gender      int64
}

type MovieCredits struct {
	Crew []Crew
	Cast []Cast
}
