# Gowatch

A **simple** movie tracking web app.

Built for personal use and learning modern Go web technologies.

## Tech Stack

- **Backend**: Go with [net/http](https://pkg.go.dev/net/http) + [Chi router](https://github.com/go-chi/chi)
- **Database**: [SQLite](https://sqlite.org/) with [sqlc](https://sqlc.dev/) for type-safe queries
- **Frontend**: [HTMX](https://htmx.org/) + [Templ](https://templ.guide/) templates
- **Styling**: [Tailwind CSS](https://tailwindcss.com/)
- **Data Source**: [The Movie Database (TMDB) API](https://www.themoviedb.org/)

## Development

### Prerequisites

- Go 1.24.3+
- npm + npx (for Tailwind CLI)
- TMDB API key (get one at [themoviedb.org](https://www.themoviedb.org/settings/api))

### Setup

1. Copy `example.env` to `.env` and add your TMDB API key
2. Run `make setup` to install dependencies and tools
3. Run `make serve` (or simply `make`) to start the development server with hot-reload on port 8090

> [!NOTE]
> Docker setup is also available (Dockerfile and docker-compose - WIP).

## Roadmap

Eventually add some basic analytics and pretty graphs. But keeping it simple.

## Contributing

This is primarily a personal learning project, but feel free to open issues or submit pull requests if you find bugs or have suggestions!
