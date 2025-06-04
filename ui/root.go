package ui

import (
	"context"
	"gowatch/db"
	"gowatch/logging"
	"gowatch/model"
	"net/http"
	"time"

	"github.com/a-h/templ"
	tmdb "github.com/cyruzin/golang-tmdb"
	"github.com/go-chi/chi"
)

//go:generate go tool templ generate

var log = logging.Get("ui")

type App struct {
	query *db.Queries
	tmdb  *tmdb.Client
}

func New(query *db.Queries, tmdb *tmdb.Client) *App {
	return &App{
		query: query,
		tmdb:  tmdb,
	}
}

func (a *App) Index() *templ.ComponentHandler {
	return templ.Handler(Index())
}

func (a *App) Routes() chi.Router {
	ui := chi.NewRouter()

	ui.Handle("/watched-movies-html", templ.Handler(WatchedList(a)))
	ui.Handle("/add-watched-movie-html", templ.Handler(AddWatched()))
	ui.Handle("/add-watched-date-input", templ.Handler(AddWatchedDateAndSubmit()))

	ui.Post("/search", a.searchMovie)

	return ui
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

func (a *App) searchMovie(w http.ResponseWriter, r *http.Request) {
	query := r.FormValue("query")
	log.Debug("search movie", "query", query)
	search, err := a.tmdb.GetSearchMovies(query, nil)
	if err != nil {
		log.Error("Failed to search movies via TMDB", "query", query, "error", err)
		http.Error(w, "Failed to search movies", http.StatusInternalServerError)
		return
	}

	log.Debug("search movies found results", "results", search.Results)
	movies := make([]model.Movie, len(search.Results))
	for i, movie := range search.Results {
		var release time.Time
		if movie.ReleaseDate != "" {
			release, err = time.Parse("2006-01-02", movie.ReleaseDate)
			if err != nil {
				log.Error("Failed to parse release date", "release date", release)
				continue
			}
		}
		movies[i] = model.Movie{
			ID:               movie.ID,
			Title:            movie.Title,
			ReleaseDate:      release,
			OriginalLanguage: movie.OriginalLanguage,
			Overview:         movie.Overview,
			PosterPath:       movie.PosterPath,
			VoteAverage:      float64(movie.VoteAverage),
		}
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	component := SearchMovieList(movies)
	component.Render(r.Context(), w)
}
