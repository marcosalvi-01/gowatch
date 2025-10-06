## Recommended Directory Structure

```
gowatch/
├── internal/
│   ├── handlers/
│   │   ├── pages.go          # Full page handlers
│   │   ├── movies.go         # Movie-related HTMX endpoints
│   │   ├── lists.go          # List-related HTMX endpoints
│   │   └── components.go     # Reusable component endpoints
│   ├── ui/
│   │   ├── pages/
│   │   │   ├── home.templ
│   │   │   ├── watched.templ
│   │   │   └── list_detail.templ
│   │   ├── components/
│   │   │   ├── sidebar/
│   │   │   │   └── sidebar.templ
│   │   │   ├── movie_card/
│   │   │   │   └── movie_card.templ
│   │   │   ├── button/
│   │   │   └── dialog/
│   │   ├── fragments/         # HTMX-specific fragments
│   │   │   ├── movie_list.templ
│   │   │   ├── watched_count.templ
│   │   │   └── list_items.templ
│   │   └── layout.templ       # Base layout
│   └── services/
│       ├── watched_service.go
│       └── list_service.go
```

## Key Principles

### 1. **Separate Full Pages from Fragments**

```templ
// ui/pages/watched.templ - Full page (for direct navigation)
templ WatchedPage(movies []Movie, currentPage string) {
    @Layout(currentPage) {
        @fragments.MovieList(movies)
    }
}

// ui/fragments/movie_list.templ - Just the content (for HTMX swaps)
templ MovieList(movies []Movie) {
    <div class="movie-grid">
        for _, movie := range movies {
            @components.MovieCard(movie)
        }
    </div>
}
```

### 2. **Handler Organization by Feature**

Instead of one giant `htmx.go`, split by domain:

```go
// handlers/movies.go
type MovieHandlers struct {
    watchedService *services.WatchedService
}

func (h *MovieHandlers) RegisterRoutes(r chi.Router) {
    r.Post("/api/movies/watched", h.AddWatched)
    r.Get("/api/movies/watched/count", h.GetWatchedCount)
    r.Get("/api/movies/{id}", h.GetMovie)
}

func (h *MovieHandlers) AddWatched(w http.ResponseWriter, r *http.Request) {
    // ... your logic

    // Return a fragment
    fragments.WatchedCount(newCount).Render(r.Context(), w)
}

// handlers/pages.go
type PageHandlers struct {
    watchedService *services.WatchedService
    listService    *services.ListService
}

func (h *PageHandlers) RegisterRoutes(r chi.Router) {
    r.Get("/", h.Home)
    r.Get("/watched", h.Watched)
    r.Get("/lists/{id}", h.ListDetail)
}

func (h *PageHandlers) Watched(w http.ResponseWriter, r *http.Request) {
    movies, _ := h.watchedService.GetAll(r.Context())

    // Check if it's an HTMX request
    if r.Header.Get("HX-Request") == "true" {
        // Return just the content fragment
        fragments.MovieList(movies).Render(r.Context(), w)
    } else {
        // Return full page
        pages.WatchedPage(movies, "watched").Render(r.Context(), w)
    }
}
```

### 3. **URL Convention**

```
/                          → Full page (home)
/watched                   → Full page (watched movies)
/lists/123                 → Full page (list detail)

/api/movies/watched        → API endpoint (POST to add)
/api/movies/watched/count  → API endpoint (GET count)
/api/lists                 → API endpoint (CRUD operations)
```

This way:

- **Clean URLs** for pages (good for bookmarking, SEO, sharing)
- **/api/** prefix for data operations
- Same handler can serve both full page and HTMX fragment based on headers

### 4. **Component Co-location Pattern**

For complex components, keep handler logic close:

```
ui/components/movie_card/
├── movie_card.templ       # The component template
├── movie_card_handlers.go # HTMX endpoints specific to this component
└── types.go               # Component-specific types
```

### 5. **Shared Layout Pattern**

```templ
// ui/layout.templ
templ Layout(currentPage string) {
    <!DOCTYPE html>
    <html>
        <head>...</head>
        <body>
            @sidebar.Sidebar(currentPage)
            <main id="main-content">
                { children... }
            </main>
        </body>
    </html>
}

// Then in your handler
func (h *PageHandlers) Watched(w http.ResponseWriter, r *http.Request) {
    movies, _ := h.watchedService.GetAll(r.Context())

    if r.Header.Get("HX-Request") == "true" {
        // Return content + updated sidebar via OOB swap
        WatchedContent(movies).Render(r.Context(), w)
    } else {
        Layout("watched") {
            WatchedContent(movies)
        }.Render(r.Context(), w)
    }
}
```

### 6. **Helper for HTMX Detection**

```go
// handlers/helpers.go
func IsHTMX(r *http.Request) bool {
    return r.Header.Get("HX-Request") == "true"
}

func RenderPage(w http.ResponseWriter, r *http.Request, fragment, fullPage templ.Component) {
    if IsHTMX(r) {
        fragment.Render(r.Context(), w)
    } else {
        fullPage.Render(r.Context(), w)
    }
}

// Usage
func (h *PageHandlers) Watched(w http.ResponseWriter, r *http.Request) {
    movies, _ := h.watchedService.GetAll(r.Context())

    RenderPage(w, r,
        fragments.MovieList(movies),           // For HTMX
        pages.WatchedPage(movies, "watched"),  // For direct navigation
    )
}
```

This structure scales well because:

- **Features are isolated** (easy to find related code)
- **URLs are clean and RESTful**
- **Same endpoints work for HTMX and direct navigation**
- **Components are reusable** across pages
- **Easy to test** (handlers are separated by concern)
