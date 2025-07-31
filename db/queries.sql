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
        original_title,
        release_date,
        original_language,
        overview,
        poster_path,
        backdrop_path,
        budget,
        revenue,
        runtime,
        vote_average,
        vote_count,
        popularity,
        homepage,
        STATUS,
        tagline
    )
VALUES
    (
        ?,
        ?,
        ?,
        ?,
        ?,
        ?,
        ?,
        ?,
        ?,
        ?,
        ?,
        ?,
        ?,
        ?,
        ?,
        ?,
        ?,
        ?
    )
RETURNING
    *;

-- name: InsertWatched :one
INSERT INTO
    watched (movie_id, watched_date)
VALUES
    (?, ?)
RETURNING
    *;

-- name: GetMovieByID :one
SELECT
    *
FROM
    movie
WHERE
    id = ?;

-- name: GetMovieByName :one
SELECT
    *
FROM
    movie
WHERE
    title = ?;

-- name: GetWatchedJoinMovie :many
SELECT
    sqlc.embed(movie),
    sqlc.embed(watched)
FROM
    watched
    JOIN movie ON watched.movie_id = movie.id;

-- name: GetMostWatchedMovies :many
SELECT
    sqlc.embed(movie),
    COUNT(*) AS view_count
FROM
    watched
    JOIN movie ON watched.movie_id = movie.id
GROUP BY
    movie.id
ORDER BY
    view_count DESC;

-- name: GetWatchedMovieDetails :one
SELECT
    sqlc.embed(movie),
    COUNT(*) AS view_count
FROM
    watched
    JOIN movie ON watched.movie_id = movie.id
WHERE
    movie.id = :movie_id
GROUP BY
    movie.id;
