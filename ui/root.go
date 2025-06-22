package ui

import (
	"context"
	"embed"
	"gowatch/db"
	"gowatch/logging"
	"gowatch/model"
	"gowatch/ui/stats"
	"net/http"
	"sort"
	"strconv"
	"time"

	"github.com/a-h/templ"
	tmdb "github.com/cyruzin/golang-tmdb"
	"github.com/go-chi/chi"
)

//go:generate go tool templ generate

//go:embed static/css/*
var staticFiles embed.FS

var log = logging.Get("ui")

type App struct {
	query *db.Queries
	tmdb  *tmdb.Client
	stats *stats.Stats
}

func New(query *db.Queries, tmdb *tmdb.Client) *App {
	return &App{
		query: query,
		tmdb:  tmdb,
		stats: stats.NewStats(query),
	}
}

func (a *App) Index() *templ.ComponentHandler {
	return templ.Handler(Index(a))
}

func (a *App) Stats() *templ.ComponentHandler {
	return templ.Handler(StatsPage(a))
}

func (a *App) Static() http.Handler {
	return http.FileServer(http.FS(staticFiles))
}

func (a *App) Routes() chi.Router {
	ui := chi.NewRouter()

	ui.Handle("/watched-movies-html", templ.Handler(WatchedList(a)))
	ui.Handle("/add-watched-movie-html", templ.Handler(AddWatched()))
	ui.Handle("/add-watched-date-input", templ.Handler(AddWatchedDateAndSubmit()))
	ui.Handle("/add-watched-date-input", templ.Handler(AddWatchedDateAndSubmit()))

	ui.Post("/add-watched", a.postWatched)
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

	// reverse the original slice to keep an order of first = last watched movie added
	for i, j := 0, len(movies)-1; i < j; i, j = i+1, j-1 {
		movies[i], movies[j] = movies[j], movies[i]
	}

	// sort keeping the original order for items that are equal
	sort.SliceStable(movies, func(i, j int) bool {
		return movies[i].WatchedDate.After(*movies[j].WatchedDate)
	})

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

func (a *App) postWatched(w http.ResponseWriter, r *http.Request) {
	movie := r.FormValue("selected_movie")
	if movie == "" {
		log.Error("TODO")
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		component := Toast(false, "wrong movie format")
		component.Render(r.Context(), w)
		return
	}
	tmdbID, err := strconv.ParseInt(movie, 10, 64)
	if err != nil {
		log.Error("TODO")
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		component := Toast(false, err.Error())
		component.Render(r.Context(), w)
		return
	}
	date := r.FormValue("date_watched")
	if date == "" {
		log.Error("TODO")
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		component := Toast(false, "wrong date format")
		component.Render(r.Context(), w)
		return
	}

	err = a.query.NewWatched(context.Background(), model.Watched{
		ID:   tmdbID,
		Date: &date,
	}, a.tmdb)
	if err != nil {
		log.Error("TODO", "err", err.Error())
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		component := Toast(false, err.Error())
		component.Render(r.Context(), w)
		return
	}

	w.Header().Set("HX-Trigger", "refreshWatched")

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusCreated)
	component := Toast(true, "Movie added!")
	component.Render(r.Context(), w)
}
