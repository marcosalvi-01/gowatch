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
    movie.id = ?
GROUP BY
    movie.id;

-- name: InsertGenre :one
INSERT INTO
    genre (id, name)
VALUES
    (?, ?)
RETURNING
    *;

-- name: InsertGenreMovie :one
INSERT INTO
    genre_movie (movie_id, genre_id)
VALUES
    (?, ?)
RETURNING
    *;

-- name: InsertPerson :one
INSERT INTO
    person (
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
    (?, ?, ?, ?, ?, ?, ?, ?)
RETURNING
    *;

-- name: InsertCast :one
INSERT INTO
    cast (
        movie_id,
        person_id,
        cast_id,
        credit_id,
        character,
        cast_order
    )
VALUES
    (?, ?, ?, ?, ?, ?)
RETURNING
    *;

-- name: InsertCrew :one
INSERT INTO
    crew (
        movie_id,
        person_id,
        credit_id,
        job,
        department
    )
VALUES
    (?, ?, ?, ?, ?)
RETURNING
    *;

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
