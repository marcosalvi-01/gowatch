package models

import "time"

type Watched struct {
	MovieID     int64
	WatchedDate time.Time
}
