# bugs:

- [ ] fix addtolistdialog to return full dialog content with buttons and handle empty lists and add loading indicator
- [x] we need to keep track of the selected item on the sidebar in a cookie or something so that the backend can render the sidebar with the correct state (should also simplify the sidebar script?). this is to fix removing a movie from the watchlist removing the selection on the watchlist item in the sidebar when refreshing it.
- [x] use a correct sqlite supported date type in the db instead of just the go time as a string

# feats:

- [x] export also the watchlist and all lists with the watched movies export
- [ ] show lists containing the movie in movie details
- [ ] different switchable view for the watched timeline
- [ ] add people search support
- [ ] add tests
- [ ] create README documentation
- [ ] better search handling with automatic search and debounce
- [ ] shortcut to search
- [x] package it as a cli so that it can be installed with `go install`
- [ ] more stats
  - [x] rewatches
  - [x] longest streak
  - [x] most in one day
  - [x] calendar heatmap
  - [x] watches by year
  - [x] top directors
  - [x] distribution of years of the movies
  - [x] longest/shortest movie watched
  - [x] Top languages/writers/composers/cinematographers
  - [ ] Top countries/studios
  - [x] budget of watched movies distribution (indie/mid/blockbuster)
  - [x] return of investment movies/biggest budget movies
  - [ ] franchises
- [ ] import from letterboxd
- [ ] jellyfin watched integration
- [ ] change/modify watched data (in case of errors when adding or jellyfin is wrong?)
- [x] add rating capabilities to watched movies
- [x] custom ad-hoc watchlist (separate from lists)
- [ ] sharable lists
- [ ] seer integration
- [ ] improve home
  - [ ] add a watchlist section in the home page with the movies to watch next
  - [ ] add calendar heatmap to home
  - [ ] generic improvements now that we have a better stats page

# optimizations:

- [ ] add pagination (or htmx progressive loading) to watched movies
- [ ] load movie activity section asynchronously with HTMX
- [ ] add proxy and cache for poster images
- [ ] better mobile ui (change the hover stuff to click?)
- [ ] have an enum with all the possible htmx events instead of using raw strings
- [x] faster stats generation
- [ ] faster import of movies

# security:

- [ ] implement CSRF protection for forms
