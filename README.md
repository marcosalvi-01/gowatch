# Gowatch

[![Build](https://img.shields.io/github/actions/workflow/status/marcosalvi-01/gowatch/docker-build.yaml)](https://github.com/marcosalvi-01/gowatch/actions)
[![Docker Image](https://img.shields.io/badge/docker-ghcr.io%2Fmarcosalvi--01%2Fgowatch-blue)](https://ghcr.io/marcosalvi-01/gowatch)
[![Go Version](https://img.shields.io/badge/Go-1.25.3+-00ADD8?style=flat&logo=go)](https://golang.org/)
[![License: GPL v3](https://img.shields.io/badge/License-GPLv3-blue.svg)](https://www.gnu.org/licenses/gpl-3.0)

A self-hosted web application for tracking movies you've watched, creating custom watchlists, and viewing detailed statistics. Built with Go and integrated with The Movie Database (TMDB) API.

## Features

- **Movie Tracking**: Log movies you've watched with dates, theater/home viewing options, and ratings
- **Custom Lists**: Create and manage personalized movie lists (watchlists, favorites, etc.) with notes
- **Movie Search**: Search for movies using TMDB's extensive database with cast, crew, and genre info
- **Detailed Movie Pages**: View comprehensive movie information including cast, genres, ratings, and watch history
- **Statistics Dashboard**: Analyze your watching habits with charts, trends, and metrics (genre preferences, viewing times)
- **User Management**: Multi-user support with admin panel, password reset, and session management
- **Import/Export**: JSON-based data portability for watched movies and lists
- **Responsive Design**: Modern, mobile-friendly interface built with Tailwind CSS
- **HTMX Integration**: Fast, dynamic interactions without complex JavaScript

## Architecture

Gowatch follows a clean service-oriented architecture:

- **Backend Services**: Movie, Watched, List, Auth, and Home services handle business logic with TMDB integration
- **Handler Organization**: Separate handlers for pages (full HTML), HTMX (dynamic updates), and API (JSON)
- **Database**: SQLite with migrations, using sqlc for type-safe queries and caching
- **Frontend**: Server-side rendering with Templ, enhanced by HTMX for interactivity
- **Security**: Session-based authentication with bcrypt password hashing
- **Self-Hosted Focus**: Designed for personal deployment with Docker and comprehensive configuration options

The app integrates deeply with TMDB for movie data and provides rich statistics on viewing habits.

## Screenshots

### Search

![Search](screenshots/search.png)

### Movie Details

![Movie Details](screenshots/movie.png)
![Mark As Watched](screenshots/mark_as_watched.png)

### Lists

![Lists](screenshots/lists.png)

### Statistics

![Statistics](screenshots/stats.png)

## Installation

### Docker (Recommended)

1. Create a `.env` file with your TMDB API key:

   ```env
   TMDB_API_KEY=your_tmdb_api_key_here
   ```

2. Run with Docker Compose:
   ```bash
   docker-compose up -d
   ```

The application will be available at `http://localhost:8080`.

### Local Development

1. Ensure you have Go 1.25.3+ and Node.js/npm installed.

2. Clone the repository and install dependencies:

   ```bash
   git clone https://github.com/yourusername/gowatch.git
   cd gowatch
   make setup
   ```

3. Create a `.env` file with your TMDB API key:

   ```env
   TMDB_API_KEY=your_tmdb_api_key_here
   ```

4. Start the development server:
   ```bash
   make dev
   ```

The application will be available at `http://localhost:8080`.

### CLI Installation

For CLI usage or direct binary installation:

1. Install the binary:
   ```bash
   go install github.com/marcosalvi-01/gowatch@latest
   ```

2. Run the server:
   ```bash
   gowatch start
   ```

The CLI supports YAML config files, environment variables, and command-line flags.

## Usage

### Web Interface

- **Home**: Overview dashboard with recent activity and quick stats
- **Watched**: View movies grouped by watch date with theater/home indicators
- **Search**: Find movies to add to your lists using TMDB database
- **Movie Details**: Click any movie for full information, cast/crew, and personal watch history
- **Lists**: Create and manage custom movie collections with notes
- **Stats**: View comprehensive watching statistics including genre distribution, viewing trends, and actor/actress frequency

### CLI Commands

- `gowatch start [--port=8080] [--config=config.yaml]`: Start the web server
- `gowatch version`: Display version information

## Configuration

The application supports multiple configuration sources (in order of precedence):
1. Command-line flags (for CLI usage)
2. Environment variables
3. YAML config file (default: `./config.yaml`)
4. Default values

### Example YAML Config

```yaml
port: "8080"
db_path: "/var/lib/gowatch"
db_name: "db.db"
tmdb_api_key: "your_key_here"
cache_ttl: "168h"
session_expiry: "24h"
shutdown_timeout: "30s"
admin_default_password: "Welcome123!"
```

### Environment Variables

- `TMDB_API_KEY`: Required TMDB API key
- `PORT`: Server port (default: 8080)
- `DB_PATH`: Database directory (default: /var/lib/gowatch)
- `DB_NAME`: Database filename (default: db.db)
- `CACHE_TTL`: TMDB data cache duration (default: 168h)
- `SESSION_EXPIRY`: User session timeout (default: 24h)

## Development

### Prerequisites

- Go 1.25.3+
- Node.js and npm (for Tailwind)
- TMDB API key (get one at [TMDB](https://www.themoviedb.org/settings/api))

### Setup

1. Install dependencies:

   ```bash
   make setup
   ```

2. Start development server with hot reload:
   ```bash
   make dev
   ```

### Build Commands

- `make build`: Build the application
- `make vet`: Run linting and checks
- `make clean`: Clean build artifacts

### Project Structure

- `cmd/`: Cobra CLI commands and configuration
- `db/`: Database layer with migrations, queries, and generated code
- `internal/handlers/`: HTTP handlers (pages, HTMX, API)
- `internal/services/`: Business logic layer
- `internal/models/`: Data structures
- `internal/middleware/`: HTTP middleware
- `internal/routes/`: Router configuration
- `internal/ui/`: Templ templates and components
- `internal/server/`: Server startup logic
- `logging/`: Structured logging utilities

## License

This project is licensed under the GNU General Public License v3.0 - see the [LICENSE](LICENSE) file for details.
