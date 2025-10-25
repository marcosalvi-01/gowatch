# bugs:

- fix addtolistdialog to return full dialog content with buttons and handle empty lists and add loading indicator
- fix stats page layout on no watched movies
- add input sanitization for search queries and user inputs

# feats:

- show lists containing the movie in movie details
- add people search support
- implement multiple user support
- improve home page with useful content (recent activity, stats overview, quick actions)
- enhance stats page with customizable data limits and additional metrics (weekday distribution, configurable top lists)
- add comprehensive test suite
- create comprehensive README documentation
- better search handling with automatic search and debounce
- shortcut to search
- improve the watched page when empty

# optimizations:

- add pagination to watched lists
- load movie activity section asynchronously with HTMX
- implement graceful server shutdown
- add proxy and cache for poster images

# security:

- implement CSRF protection for forms
