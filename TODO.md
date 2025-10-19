# bugs:

- the addtolistdialog should return the whole dialog content including the buttons, not just the list of lists, so that if there are no lists it can be different to create a list by saying something like no lists exist, create a new one before adding a movie to it

# feats:

- add section in a movie details to see in which lists it already is
- support people in the search
- multiple users
- stats page

# optimizations:

- the watched call takes around 120 ms for 1200 movies, use pagination?
- the your activity section in the movie page should be loaded asynchronously (htmx)
