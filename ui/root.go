package ui

import (
	"context"
	"gowatch/db"
	"gowatch/logging"
	"gowatch/model"
	"net/http"

	"github.com/a-h/templ"
)

//go:generate go tool templ generate

var log = logging.Get("ui")

type App struct {
	query *db.Queries
}

func New(query *db.Queries) *App {
	return &App{
		query: query,
	}
}

func (a *App) Routes() *http.ServeMux {
	mux := http.NewServeMux()
	mux.Handle("/", templ.Handler(Index()))
	mux.Handle("/ui/watched-movies-html", templ.Handler(WatchedList(a)))
	return mux
}

func (a *App) getWatchedMovies() ([]model.Movie, error) {
	moviesRows, err := a.query.GetWatchedJoinMovie(context.Background())
	if err != nil {
		return nil, err
	}

	movies := make([]model.Movie, len(moviesRows))
	for i, movie := range moviesRows {
		movies[i] = model.Movie{
			ID:               movie.ID,
			IMDbID:           movie.ImdbID,
			Title:            movie.Title,
			ReleaseDate:      movie.ReleaseDate,
			OriginalLanguage: movie.OriginalLanguage,
			Overview:         movie.Overview,
			PosterPath:       movie.PosterPath,
			Budget:           movie.Budget,
			Revenue:          movie.Revenue,
			Runtime:          movie.Runtime,
			VoteAverage:      movie.VoteAverage,
			WatchedDate:      &movie.WatchedDate,
		}
	}
	return movies, nil
}
