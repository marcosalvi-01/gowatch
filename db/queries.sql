-- name: InsertMovie :exec
INSERT
    OR IGNORE INTO movie (
        id,
        title,
        original_title,
        original_language,
        overview,
        release_date,
        poster_path,
        backdrop_path,
        popularity,
        vote_count,
        vote_average,
        budget,
        homepage,
        imdb_id,
        revenue,
        runtime,
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
    );

-- name: InsertGenre :exec
INSERT
    OR IGNORE INTO genre (id, name)
VALUES
    (?, ?);

-- name: InsertGenreMovie :exec
INSERT
    OR IGNORE INTO genre_movie (movie_id, genre_id)
VALUES
    (?, ?);

-- name: InsertWatched :one
INSERT INTO
    watched (movie_id, watched_date, watched_in_theater)
VALUES
    (?, ?, ?)
RETURNING
    *;

-- name: GetAllWatchedForExport :many
SELECT
    sqlc.embed(movie),
    sqlc.embed(watched)
FROM
    watched
    JOIN movie ON watched.movie_id = movie.id;

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
    movie.id = ?
GROUP BY
    movie.id;

-- name: InsertPerson :exec
INSERT
    OR IGNORE INTO person (
        id,
        name,
        original_name,
        profile_path,
        known_for_department,
        popularity,
        gender,
        adult
    )
VALUES
    (?, ?, ?, ?, ?, ?, ?, ?);

-- name: InsertCast :exec
INSERT
    OR IGNORE INTO cast (
        movie_id,
        person_id,
        cast_id,
        credit_id,
        character,
        cast_order
    )
VALUES
    (?, ?, ?, ?, ?, ?);

-- name: InsertCrew :exec
INSERT
    OR IGNORE INTO crew (
        movie_id,
        person_id,
        credit_id,
        job,
        department
    )
VALUES
    (?, ?, ?, ?, ?);

-- name: GetPerson :one
SELECT
    *
FROM
    person
WHERE
    id = ?;

-- name: GetCrewByMovieID :many
SELECT
    *
FROM
    crew
WHERE
    movie_id = ?;

-- name: GetCastByMovieID :many
SELECT
    *
FROM
    cast
WHERE
    movie_id = ?;

-- name: GetMovieGenre :many
SELECT
    *
FROM
    genre
    JOIN genre_movie ON genre.id = genre_movie.genre_id
WHERE
    genre_movie.movie_id = ?;

-- name: GetWatchedJoinMovieByID :many
SELECT
    sqlc.embed(movie),
    sqlc.embed(watched)
FROM
    watched
    JOIN movie ON watched.movie_id = movie.id
WHERE
    watched.movie_id = ?
ORDER BY
    watched.watched_date DESC;
