-- name: UpsertMovie :exec
INSERT INTO
    movie (
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
        tagline,
        updated_at
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
        ?,
        CURRENT_TIMESTAMP
    ) ON CONFLICT(id) DO
UPDATE
SET
    title = excluded.title,
    original_title = excluded.original_title,
    original_language = excluded.original_language,
    overview = excluded.overview,
    release_date = excluded.release_date,
    poster_path = excluded.poster_path,
    backdrop_path = excluded.backdrop_path,
    popularity = excluded.popularity,
    vote_count = excluded.vote_count,
    vote_average = excluded.vote_average,
    budget = excluded.budget,
    homepage = excluded.homepage,
    imdb_id = excluded.imdb_id,
    revenue = excluded.revenue,
    runtime = excluded.runtime,
    STATUS = excluded.STATUS,
    tagline = excluded.tagline,
    updated_at = CURRENT_TIMESTAMP;

-- name: UpsertGenre :exec
INSERT INTO
    genre (id, name, updated_at)
VALUES
    (?, ?, CURRENT_TIMESTAMP) ON CONFLICT(id) DO
UPDATE
SET
    name = excluded.name,
    updated_at = CURRENT_TIMESTAMP;

-- name: UpsertGenreMovie :exec
INSERT INTO
    genre_movie (movie_id, genre_id, updated_at)
VALUES
    (?, ?, CURRENT_TIMESTAMP) ON CONFLICT(movie_id, genre_id) DO
UPDATE
SET
    updated_at = CURRENT_TIMESTAMP;

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

-- name: UpsertPerson :exec
INSERT INTO
    person (
        id,
        name,
        original_name,
        profile_path,
        known_for_department,
        popularity,
        gender,
        adult,
        updated_at
    )
VALUES
    (?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP) ON CONFLICT(id) DO
UPDATE
SET
    name = excluded.name,
    original_name = excluded.original_name,
    profile_path = excluded.profile_path,
    known_for_department = excluded.known_for_department,
    popularity = excluded.popularity,
    gender = excluded.gender,
    adult = excluded.adult,
    updated_at = CURRENT_TIMESTAMP;

-- name: UpsertCast :exec
INSERT INTO
    cast (
        movie_id,
        person_id,
        cast_id,
        credit_id,
        character,
        cast_order,
        updated_at
    )
VALUES
    (?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP) ON CONFLICT(movie_id, person_id, cast_id) DO
UPDATE
SET
    credit_id = excluded.credit_id,
    character = excluded.character,
    cast_order = excluded.cast_order,
    updated_at = CURRENT_TIMESTAMP;

-- name: UpsertCrew :exec
INSERT INTO
    crew (
        movie_id,
        person_id,
        credit_id,
        job,
        department,
        updated_at
    )
VALUES
    (?, ?, ?, ?, ?, CURRENT_TIMESTAMP) ON CONFLICT(movie_id, person_id, credit_id) DO
UPDATE
SET
    job = excluded.job,
    department = excluded.department,
    updated_at = CURRENT_TIMESTAMP;

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

-- name: GetWatchedCount :one
SELECT
    COUNT(*) AS count
FROM
    watched;

-- name: GetListByID :one
SELECT
    *
FROM
    list
WHERE
    id = ?;

-- name: DeleteListByID :exec
DELETE FROM
    list
WHERE
    id = ?;

-- name: DeleteMovieFromList :exec
DELETE FROM
    list_movie
WHERE
    list_id = ?
    AND movie_id = ?;

-- name: GetWatchedPerMonthLastYear :many
SELECT
    watched_date
FROM
    watched
WHERE
    watched_date >= date('now', 'start of month', '-12 months')
    AND watched_date < date('now', 'start of month')
ORDER BY
    watched_date;

-- name: GetWatchedPerYear :many
SELECT
    watched_date
FROM
    watched
ORDER BY
    watched_date;

-- name: GetWatchedByGenre :many
SELECT
    genre.name,
    COUNT(*) AS count
FROM
    watched
    JOIN movie ON watched.movie_id = movie.id
    JOIN genre_movie ON movie.id = genre_movie.movie_id
    JOIN genre ON genre_movie.genre_id = genre.id
GROUP BY
    genre.id,
    genre.name
ORDER BY
    count DESC;

-- name: GetTheaterVsHomeCount :many
SELECT
    watched_in_theater,
    COUNT(*) AS count
FROM
    watched
GROUP BY
    watched_in_theater;

-- name: GetMostWatchedMovies :many
SELECT
    movie.title,
    movie.id,
    movie.poster_path,
    COUNT(*) AS watch_count
FROM
    watched
    JOIN movie ON watched.movie_id = movie.id
GROUP BY
    movie.id,
    movie.title,
    movie.poster_path
ORDER BY
    watch_count DESC
LIMIT
    10;

-- name: GetMostWatchedDay :many
SELECT
    watched_date
FROM
    watched;

-- name: GetMostWatchedActors :many
SELECT
    person.name,
    person.id,
    person.profile_path,
    COUNT(DISTINCT watched.movie_id) AS movie_count
FROM
    watched
    JOIN "cast" ON watched.movie_id = "cast".movie_id
    JOIN person ON "cast".person_id = person.id
GROUP BY
    person.id,
    person.name,
    person.profile_path
ORDER BY
    movie_count DESC
LIMIT
    10;

-- name: GetWatchedDateRange :one
SELECT
    MIN(watched_date) AS min_date,
    MAX(watched_date) AS max_date
FROM
    watched
WHERE
    watched_date IS NOT NULL;
