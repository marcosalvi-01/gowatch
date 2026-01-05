# bugs:

- [ ] fix addtolistdialog to return full dialog content with buttons and handle empty lists and add loading indicator

# feats:

- [ ] show lists containing the movie in movie details
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
- [ ] custom ad- [ ]hoc watchlist (separate from lists)
- [ ] sharable lists

# optimizations:

- [ ] add pagination to watched lists
- [ ] load movie activity section asynchronously with HTMX
- [ ] add proxy and cache for poster images
- [ ] better mobile ui (change the hover stuff to click?)

# security:

- [ ] implement CSRF protection for forms
