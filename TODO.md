# bugs:

- [ ] fix addtolistdialog to return full dialog content with buttons and handle empty lists and add loading indicator

# feats:

- [ ] notification when movie in watchlist gets published
- [ ] show lists containing the movie in movie details
- [ ] different switchable view for the watched timeline
- [ ] add tests
- [ ] create README documentation
- [ ] better search handling with automatic search and debounce
- [ ] shortcut to search
- [ ] import from letterboxd
- [ ] jellyfin watched integration
- [ ] change/modify watched data (in case of errors when adding or jellyfin is wrong?)
- [ ] sharable lists
- [ ] seer integration

# optimizations:

- [ ] add pagination (or htmx progressive loading) to watched movies
- [ ] load movie activity section asynchronously with HTMX
- [ ] add proxy and cache for poster images
- [ ] better mobile ui (change the hover stuff to click?)
- [ ] have an enum with all the possible htmx events instead of using raw strings (? maybe it's overkill)
- [ ] faster import of movies

# security:

- [ ] implement CSRF protection for forms
