# bugs:

- the addtolistdialog should return the whole model content, not just the list of lists, so that if there are no lists it can be different to create a list by saying something like no lists exist, create a new one before adding a movie to it

# feats:

- add section in a movie details to see in which lists it already is
- support people in the search
- multiple users
- stats page

# optimizations:

- the watched call takes around 120 ms for 1200 movies, use pagination?
- the your activity section in the movie page should be loaded asynchronously (htmx)
- when deleting a list that i am watching it should show a toast and change the main content to another page (like the home) by using htmx, not just a redirect
