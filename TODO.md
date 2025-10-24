# bugs:

- fix addtolistdialog to return full dialog content with buttons and handle empty lists
- fix stats page 500 error when no watched movies exist
- add input sanitization for search queries and user inputs

# feats:

- show lists containing the movie in movie details
- add people search support
- implement multiple user support
- improve home page with useful content (recent activity, stats overview, quick actions)
- enhance stats page with customizable data limits and additional metrics (weekday distribution, configurable top lists)
- add comprehensive test suite
- implement proper error pages (404, 500, etc.)
- create comprehensive README documentation
- add health check endpoint

# optimizations:

- add pagination to watched lists
- load movie activity section asynchronously with HTMX
- implement graceful server shutdown
- add slog middleware for API routes
- add test target to makefile

# security:

- implement CSRF protection for forms

# other:

- include inTheaters field in watched export
- add loading indicator to add-to-list form
- better htmx-indicator (different one for different pages?)
