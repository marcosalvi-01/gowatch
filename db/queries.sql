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

-- name: GetMovieByID :one
SELECT
    *
FROM
    movie
WHERE
    id = ?;

-- name: GetWatchedJoinMovie :many
SELECT
    sqlc.embed(movie),
    sqlc.embed(watched)
FROM
    watched
    JOIN movie ON watched.movie_id = movie.id;

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

-- name: InsertList :exec
INSERT INTO
    list (
        name,
        creation_date,
        description
    )
VALUES
    (?, ?, ?);

-- name: GetListJoinMovieByID :many
SELECT
    sqlc.embed(movie),
    sqlc.embed(list_movie),
    sqlc.embed(list)
FROM
    list
    JOIN list_movie ON list_movie.list_id = list.id
    JOIN movie ON movie.id = list_movie.movie_id
WHERE
    list.id = ?;

-- name: AddMovieToList :exec
INSERT INTO
    list_movie (
        movie_id,
        list_id,
        date_added,
        position,
        note
    )
VALUES
    (?, ?, ?, ?, ?);

-- name: GetAllLists :many
SELECT
    *
FROM
    list;
