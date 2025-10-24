# bugs:

- fix addtolistdialog to return full dialog content with buttons and handle empty lists
- fix stats page layout on no watched movies
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
- better search handling with automatic search and debounce

# optimizations:

- add pagination to watched lists
- load movie activity section asynchronously with HTMX
- implement graceful server shutdown

# security:

- implement CSRF protection for forms

# other:

- add loading indicator to add-to-list form
