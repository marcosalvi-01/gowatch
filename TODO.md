# bugs:

- fix addtolistdialog to return full dialog content with buttons and handle empty lists and add loading indicator
- fix avg per week and avg per month in the stats showing wrong values when having only small data

# feats:

- show lists containing the movie in movie details
- add people search support
- add comprehensive test suite
- create comprehensive README documentation
- better search handling with automatic search and debounce
- shortcut to search
- package it as a cli so that it can be installed with `go install`
- add admin stuff

# optimizations:

- add pagination to watched lists
- load movie activity section asynchronously with HTMX
- add proxy and cache for poster images

# security:

- implement CSRF protection for forms
