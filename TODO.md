# bugs:

- [ ] fix addtolistdialog to return full dialog content with buttons and handle empty lists and add loading indicator
- [ ] we need to keep track of the selected item on the sidebar in a cookie or something so that the backend can render the sidebar with the correct state (should also simplify the sidebar script?). this is to fix removing a movie from the watchlist removing the selection on the watchlist item in the sidebar when refreshing it. also we should keep track of the state of the lists collapsible item in a cookie for the same reason
- [ ] use a correct sqlite supported date type in the db instead of just the go time as a string

# feats:

- [ ] show lists containing the movie in movie details
- [ ] different switchable view for the watched timeline
- [ ] add people search support
- [ ] add tests
- [ ] create README documentation
- [ ] better search handling with automatic search and debounce
- [ ] shortcut to search
- [ ] package it as a cli so that it can be installed with `go install`
- [ ] more stats
  - [ ] rewatches
  - [ ] longest streak
  - [ ] most in one day
  - [ ] calendar heatmap
  - [ ] watches by year
  - [ ] top directors
  - [ ] distribution of years of the movies
  - [ ] longest/shortest movie watched
  - [ ] Top countries/languages/studios/writers/composers/cinematographers
  - [ ] budget of watched movies distribution (indie/mid/blockbuster)
  - [ ] return of investment movies/biggest budget movies
  - [ ] franchises
- [ ] import from letterboxd
- [ ] jellyfin watched integration
- [ ] change/modify watched data (in case of errors when adding or jellyfin is wrong?)
- [x] add rating capabilities to watched movies
- [x] custom ad-hoc watchlist (separate from lists)
- [ ] sharable lists

# optimizations:

- [ ] add pagination to watched lists
- [ ] load movie activity section asynchronously with HTMX
- [ ] add proxy and cache for poster images
- [ ] better mobile ui (change the hover stuff to click?)
- [ ] have an enum with all the possible htmx events instead of using raw strings
- [ ] faster stats generation
- [ ] faster import of movies

# security:

- [ ] implement CSRF protection for forms
