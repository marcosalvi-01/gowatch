-- name: GetAllMovies :many
SELECT
    *
FROM
    movie;

-- name: GetAllWatched :many
SELECT
    *
FROM
    watched;

-- name: InsertMovie :one
INSERT INTO
    movie (
        id,
        imdb_id,
        title,
        original_language,
        overview,
        poster_path,
        release_date,
        budget,
        revenue,
        runtime,
        vote_average
    )
VALUES
    (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
RETURNING
    *;

-- name: InsertWatched :one
INSERT INTO
    watched (movie_id, watched_date)
VALUES
    (?, ?)
RETURNING
    *;

-- name: GetMovieFromReference :one
SELECT
    *
FROM
    movie
WHERE
    id = ?;

-- name: GetMovieFromName :one
SELECT
    *
FROM
    movie
WHERE
    title = ?;

-- name: GetWatchedJoinMovie :many
SELECT
    *
FROM
    watched
    JOIN movie ON watched.movie_id = movie.id
