# bugs:

- update the add to list button after creating a list
- the top header (with the search) disappears if scrolling too much
- don't save the movie in the db if the status is not released by at least some time

# feats:

- add section in a movie details to see in which lists it already is
- support people in the search
- multiple users

# optimizations:

- the watched call takes around 120 ms for 1200 movies, use pagination?
- do not do a full reload but just load the page contents with htmx
- invalidate the cache after some time?
