-- Movie catalog and metadata.
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

-- Watched history and activity.
-- name: InsertWatched :one
INSERT INTO
    watched (movie_id, watched_date, watched_in_theater, user_id, rating)
VALUES
    (?, ?, ?, ?, ?)
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
    JOIN movie ON watched.movie_id = movie.id
WHERE
    watched.user_id = ?;

-- People and credits metadata.
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

-- Watched history by movie.
-- name: GetWatchedJoinMovieByID :many
SELECT
    sqlc.embed(movie),
    sqlc.embed(watched)
FROM
    watched
    JOIN movie ON watched.movie_id = movie.id
WHERE
    watched.user_id = ?
    AND watched.movie_id = ?
ORDER BY
    watched.watched_date DESC;

-- name: GetWatchedMoviesByPerson :many
WITH person_roles AS (
    SELECT
        "cast".movie_id,
        'acting' AS role_kind,
        COALESCE(NULLIF(TRIM("cast".character), ''), 'Actor') AS role_label
    FROM
        "cast"
    WHERE
        "cast".person_id = ?1
    UNION
    SELECT
        crew.movie_id,
        'crew' AS role_kind,
        COALESCE(NULLIF(TRIM(crew.job), ''), COALESCE(NULLIF(TRIM(crew.department), ''), 'Crew')) AS role_label
    FROM
        crew
    WHERE
        crew.person_id = ?1
),
watched_movies AS (
    SELECT
        watched.movie_id,
        COUNT(watched.id) AS watch_count,
        MAX(watched.watched_date) AS last_watched_date
    FROM
        watched
    WHERE
        watched.user_id = ?2
    GROUP BY
        watched.movie_id
)
SELECT
    movie.id,
    movie.title,
    movie.poster_path,
    watched_movies.watch_count,
    watched_movies.last_watched_date,
    person_roles.role_kind,
    person_roles.role_label
FROM
    person_roles
    JOIN watched_movies ON watched_movies.movie_id = person_roles.movie_id
    JOIN movie ON watched_movies.movie_id = movie.id
ORDER BY
    watched_movies.watch_count DESC,
    watched_movies.last_watched_date DESC,
    movie.title ASC,
    person_roles.role_kind ASC,
    person_roles.role_label ASC;

-- Lists and watchlist.
-- name: InsertList :one
INSERT INTO
    list (
        name,
        creation_date,
        description,
        user_id,
		is_watchlist
    )
VALUES
    (?, ?, ?, ?, ?)
RETURNING
    id;

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
    list.user_id = ?
    AND list.id = ?;

-- name: AddMovieToList :exec
INSERT INTO
    list_movie (
        movie_id,
        list_id,
        date_added,
        position,
        note
    )
SELECT
    ?,
    ?,
    ?,
    ?,
    ?
FROM
    list
WHERE
    list.id = ?
    AND list.user_id = ?;

-- name: UpsertMovieInList :exec
INSERT INTO
    list_movie (
        movie_id,
        list_id,
        date_added,
        position,
        note
    )
SELECT
    ?,
    ?,
    ?,
    ?,
    ?
FROM
    list
WHERE
    list.id = ?
    AND list.user_id = ?
ON CONFLICT(movie_id, list_id) DO
UPDATE
SET
    date_added = excluded.date_added,
    position = excluded.position,
    note = excluded.note;

-- name: GetAllLists :many
SELECT
    *
FROM
    list
WHERE
    user_id = ?
ORDER BY
    id;

-- name: GetWatchlistID :one
SELECT
    id
FROM
    list
WHERE
    user_id = ?
    AND is_watchlist = TRUE;

-- name: GetWatchedCount :one
SELECT
    COUNT(*) AS count
FROM
    watched
WHERE
    user_id = ?;

-- name: GetListByID :one
SELECT
    *
FROM
    list
WHERE
    user_id = ?
    AND id = ?;

-- name: DeleteListByID :exec
DELETE FROM
    list
WHERE
    user_id = ?
    AND id = ?;

-- name: DeleteMovieFromList :exec
DELETE FROM
    list_movie
WHERE
    list_id = ?
    AND movie_id = ?
    AND EXISTS (
        SELECT
            1
        FROM
            list
        WHERE
    list.id = list_id
    AND list.user_id = ?
);

-- Watched stats.
-- name: GetWatchedStatsPerMonthLastYear :many
SELECT
    CAST(strftime('%Y-%m', watched_date) AS TEXT) AS month,
    COUNT(*) AS count,
    SUM(movie.runtime) AS total_runtime
FROM
    watched
    JOIN movie ON watched.movie_id = movie.id
WHERE
    watched_date >= date('now', 'start of month', '-12 months')
    AND watched.user_id = ?
GROUP BY
    month
ORDER BY
    month;

-- name: GetWatchedPerYear :many
SELECT
    CAST(strftime('%Y', watched_date) AS TEXT) AS year,
    COUNT(*) AS count
FROM
    watched
WHERE
    user_id = ?
GROUP BY
    year
ORDER BY
    year;

-- name: GetWeekdayDistribution :many
SELECT
    CAST(strftime('%w', watched_date) AS INTEGER) AS weekday_index,
    COUNT(*) AS count
FROM
    watched
WHERE
    user_id = ?
GROUP BY
    weekday_index
ORDER BY
    weekday_index;

-- name: GetWatchedByGenre :many
SELECT
    genre.name,
    COUNT(*) AS count
FROM
    watched
    JOIN movie ON watched.movie_id = movie.id
    JOIN genre_movie ON movie.id = genre_movie.movie_id
    JOIN genre ON genre_movie.genre_id = genre.id
WHERE
    watched.user_id = ?
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
WHERE
    user_id = ?
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
WHERE
    watched.user_id = ?
GROUP BY
    movie.id,
    movie.title,
    movie.poster_path
ORDER BY
    watch_count DESC
LIMIT
    ?;

-- name: GetMostWatchedDay :one
SELECT
    watched_date,
    COUNT(*) AS count
FROM
    watched
WHERE
    user_id = ?
GROUP BY
    watched_date
ORDER BY
    count DESC
LIMIT 1;

-- name: GetWatchedActors :many
WITH watched_actors AS (
    SELECT DISTINCT
        watched.id AS watched_id,
        person.id,
        person.name,
        person.profile_path,
        person.gender
    FROM
        watched
        JOIN "cast" ON watched.movie_id = "cast".movie_id
        JOIN person ON "cast".person_id = person.id
    WHERE
        watched.user_id = ?
        AND person.gender IN (1, 2)
)
SELECT
    watched_actors.name,
    watched_actors.id,
    watched_actors.profile_path,
    watched_actors.gender,
    COUNT(*) AS watch_count
FROM
    watched_actors
GROUP BY
    watched_actors.id,
    watched_actors.name,
    watched_actors.profile_path,
    watched_actors.gender
ORDER BY
    watched_actors.gender ASC,
    watch_count DESC,
    watched_actors.name ASC;

-- name: GetWatchedDateRange :one
SELECT
    MIN(watched_date) AS min_date,
    MAX(watched_date) AS max_date
FROM
    watched
WHERE
    watched_date IS NOT NULL
    AND user_id = ?;

-- name: GetWatchedDates :many
SELECT
    watched_date
FROM
    watched
WHERE
    user_id = ?
ORDER BY
    watched_date;

-- name: GetRecentWatchedMovies :many
SELECT
    sqlc.embed(movie),
    sqlc.embed(watched)
FROM
    watched
    JOIN movie ON watched.movie_id = movie.id
WHERE
    watched.user_id = ?
ORDER BY
    watched.watched_date DESC
LIMIT
    ?;

-- name: GetTotalWatchedStats :one
SELECT
    COUNT(*) AS count,
    SUM(movie.runtime) AS total_runtime
FROM
    watched
    JOIN movie ON watched.movie_id = movie.id
WHERE
    watched.user_id = ?;

-- name: GetRewatchStats :one
WITH movie_watch_counts AS (
    SELECT
        watched.movie_id,
        COUNT(*) AS watch_count
    FROM
        watched
    WHERE
        watched.user_id = ?
    GROUP BY
        watched.movie_id
)
SELECT
    COUNT(*) AS unique_movie_count,
    CAST(
        COALESCE(
            SUM(
                CASE
                    WHEN watch_count > 1 THEN 1
                    ELSE 0
                END
            ),
            0
        ) AS INTEGER
    ) AS rewatched_movie_count,
    CAST(
        COALESCE(
            SUM(
                CASE
                    WHEN watch_count > 1 THEN watch_count - 1
                    ELSE 0
                END
            ),
            0
        ) AS INTEGER
    ) AS rewatch_count
FROM
    movie_watch_counts;

-- name: GetDailyWatchCountsLastYear :many
SELECT
    watched.watched_date,
    COUNT(*) AS count
FROM
    watched
WHERE
    watched.user_id = ?
    AND watched.watched_date >= date('now', '-364 days')
GROUP BY
    watched.watched_date
ORDER BY
    watched.watched_date;

-- name: GetWatchedCrewMembers :many
WITH normalized_crew AS (
    SELECT DISTINCT
        watched.id AS watched_id,
        person.id,
        person.name,
        person.profile_path,
        CASE
            WHEN crew.job = 'Director' THEN 'director'
            WHEN crew.job IN (
                'Writer',
                'Screenplay',
                'Story',
                'Novel',
                'Original Story',
                'Characters'
            ) THEN 'writer'
            WHEN crew.job IN ('Original Music Composer', 'Composer', 'Music') THEN 'composer'
            WHEN crew.job IN ('Director of Photography', 'Cinematography') THEN 'cinematographer'
            ELSE NULL
        END AS role_key
    FROM
        watched
        JOIN crew ON watched.movie_id = crew.movie_id
        JOIN person ON crew.person_id = person.id
    WHERE
        watched.user_id = ?
)
SELECT
    normalized_crew.role_key,
    normalized_crew.id,
    normalized_crew.name,
    normalized_crew.profile_path,
    COUNT(*) AS watch_count
FROM
    normalized_crew
WHERE
    normalized_crew.role_key IS NOT NULL
GROUP BY
    normalized_crew.role_key,
    normalized_crew.id,
    normalized_crew.name,
    normalized_crew.profile_path
ORDER BY
    normalized_crew.role_key ASC,
    watch_count DESC,
    normalized_crew.name ASC;

-- name: GetTopLanguages :many
SELECT
    movie.original_language AS language,
    COUNT(*) AS watch_count
FROM
    watched
    JOIN movie ON watched.movie_id = movie.id
WHERE
    watched.user_id = ?
    AND movie.original_language <> ''
GROUP BY
    movie.original_language
ORDER BY
    watch_count DESC,
    language ASC
LIMIT
    ?;

-- name: GetReleaseYearDistribution :many
SELECT
    CAST(strftime('%Y', movie.release_date) AS INTEGER) AS release_year,
    COUNT(DISTINCT watched.movie_id) AS count
FROM
    watched
    JOIN movie ON watched.movie_id = movie.id
WHERE
    watched.user_id = ?
    AND movie.release_date IS NOT NULL
GROUP BY
    release_year
ORDER BY
    release_year;

-- name: GetLongestWatchedMovie :one
SELECT
    movie.id,
    movie.title,
    movie.poster_path,
    movie.runtime
FROM
    watched
    JOIN movie ON watched.movie_id = movie.id
WHERE
    watched.user_id = ?
    AND movie.runtime > 0
GROUP BY
    movie.id,
    movie.title,
    movie.poster_path,
    movie.runtime
ORDER BY
    movie.runtime DESC,
    movie.title ASC
LIMIT
    1;

-- name: GetShortestWatchedMovie :one
SELECT
    movie.id,
    movie.title,
    movie.poster_path,
    movie.runtime
FROM
    watched
    JOIN movie ON watched.movie_id = movie.id
WHERE
    watched.user_id = ?
    AND movie.runtime > 0
GROUP BY
    movie.id,
    movie.title,
    movie.poster_path,
    movie.runtime
ORDER BY
    movie.runtime ASC,
    movie.title ASC
LIMIT
    1;

-- name: GetBudgetTierDistribution :many
WITH watched_movies AS (
    SELECT DISTINCT
        watched.movie_id,
        movie.budget
    FROM
        watched
        JOIN movie ON watched.movie_id = movie.id
    WHERE
        watched.user_id = ?
)
SELECT
    CASE
        WHEN watched_movies.budget <= 0 THEN 'unknown'
        WHEN watched_movies.budget < 10000000 THEN 'indie'
        WHEN watched_movies.budget <= 100000000 THEN 'mid'
        ELSE 'blockbuster'
    END AS tier,
    COUNT(*) AS count
FROM
    watched_movies
GROUP BY
    tier
ORDER BY
    CASE
        WHEN tier = 'indie' THEN 1
        WHEN tier = 'mid' THEN 2
        WHEN tier = 'blockbuster' THEN 3
        ELSE 4
    END;

-- name: GetTopReturnOnInvestmentMovies :many
WITH watched_movies AS (
    SELECT DISTINCT
        watched.movie_id
    FROM
        watched
    WHERE
        watched.user_id = ?
)
SELECT
    movie.id,
    movie.title,
    movie.poster_path,
    movie.budget,
    movie.revenue,
    CAST(
        CAST(movie.revenue - movie.budget AS REAL) / CAST(movie.budget AS REAL) AS REAL
    ) AS roi
FROM
    watched_movies
    JOIN movie ON watched_movies.movie_id = movie.id
WHERE
    movie.budget > 0
    AND movie.revenue > 0
ORDER BY
    roi DESC,
    movie.revenue DESC,
    movie.title ASC
LIMIT
    ?;

-- name: GetBiggestBudgetMovies :many
WITH watched_movies AS (
    SELECT DISTINCT
        watched.movie_id
    FROM
        watched
    WHERE
        watched.user_id = ?
)
SELECT
    movie.id,
    movie.title,
    movie.poster_path,
    movie.budget,
    movie.revenue,
    CAST(
        CASE
            WHEN movie.budget > 0 THEN CAST(movie.revenue - movie.budget AS REAL) / CAST(movie.budget AS REAL)
            ELSE 0.0
        END AS REAL
    ) AS roi
FROM
    watched_movies
    JOIN movie ON watched_movies.movie_id = movie.id
WHERE
    movie.budget > 0
ORDER BY
    movie.budget DESC,
    movie.revenue DESC,
    movie.title ASC
LIMIT
    ?;

-- name: GetMonthlyGenreBreakdown :many
SELECT
    watched.watched_date,
    genre.name AS genre_name,
    COUNT(*) AS movie_count
FROM
    watched
    JOIN movie ON watched.movie_id = movie.id
    JOIN genre_movie ON movie.id = genre_movie.movie_id
    JOIN genre ON genre_movie.genre_id = genre.id
WHERE
    watched.watched_date >= date('now', 'start of month', '-12 months')
    AND watched.user_id = ?
GROUP BY
    watched.watched_date,
    genre_name
ORDER BY
    watched.watched_date,
    movie_count DESC;

-- Rating stats.
-- name: GetRatingSummary :one
SELECT
    CAST(COALESCE(AVG(CAST(watched.rating AS REAL)), 0.0) AS REAL) AS average_rating,
    COUNT(watched.rating) AS rated_count
FROM
    watched
WHERE
    watched.user_id = ?
    AND watched.rating IS NOT NULL;

-- name: GetRatingDistribution :many
SELECT
    CAST(ROUND(CAST(watched.rating AS REAL) * 2.0, 0) / 2.0 AS REAL) AS rating_bucket,
    COUNT(*) AS count
FROM
    watched
WHERE
    watched.user_id = ?
    AND watched.rating IS NOT NULL
GROUP BY
    rating_bucket
ORDER BY
    rating_bucket;

-- name: GetMonthlyAverageRatingLastYear :many
SELECT
    CAST(strftime('%Y-%m', watched.watched_date) AS TEXT) AS month,
    CAST(COALESCE(AVG(CAST(watched.rating AS REAL)), 0.0) AS REAL) AS average_rating,
    COUNT(watched.rating) AS rated_count
FROM
    watched
WHERE
    watched.user_id = ?
    AND watched.rating IS NOT NULL
    AND watched.watched_date >= date('now', 'start of month', '-11 months')
GROUP BY
    month
ORDER BY
    month;

-- name: GetTheaterVsHomeAverageRating :many
SELECT
    watched.watched_in_theater,
    CAST(COALESCE(AVG(CAST(watched.rating AS REAL)), 0.0) AS REAL) AS average_rating,
    COUNT(watched.rating) AS rated_count
FROM
    watched
WHERE
    watched.user_id = ?
    AND watched.rating IS NOT NULL
GROUP BY
    watched.watched_in_theater;

-- name: GetHighestRatedMovies :many
WITH rated_movies AS (
    SELECT
        watched.movie_id,
        CAST(COALESCE(AVG(CAST(watched.rating AS REAL)), 0.0) AS REAL) AS average_rating,
        COUNT(watched.rating) AS rated_watch_count
    FROM
        watched
    WHERE
        watched.user_id = ?1
        AND watched.rating IS NOT NULL
    GROUP BY
        watched.movie_id
)
SELECT
    movie.id,
    movie.title,
    movie.poster_path,
    rated_movies.average_rating,
    rated_movies.rated_watch_count
FROM
    rated_movies
    JOIN movie ON rated_movies.movie_id = movie.id
ORDER BY
    rated_movies.average_rating DESC,
    rated_movies.rated_watch_count DESC,
    movie.title ASC
LIMIT
    ?2;

-- name: GetRatingVsTMDB :one
WITH rated_movies AS (
    SELECT
        watched.movie_id,
        CAST(COALESCE(AVG(CAST(watched.rating AS REAL)), 0.0) AS REAL) AS average_rating
    FROM
        watched
    WHERE
        watched.user_id = ?1
        AND watched.rating IS NOT NULL
    GROUP BY
        watched.movie_id
)
SELECT
    CAST(COALESCE(AVG(rated_movies.average_rating), 0.0) AS REAL) AS average_user_rating,
    CAST(COALESCE(AVG(CAST(movie.vote_average AS REAL) / 2.0), 0.0) AS REAL) AS average_tmdb_rating,
    CAST(COALESCE(AVG(rated_movies.average_rating - (CAST(movie.vote_average AS REAL) / 2.0)), 0.0) AS REAL) AS average_difference,
    COUNT(*) AS compared_movie_count
FROM
    rated_movies
    JOIN movie ON rated_movies.movie_id = movie.id
WHERE
    movie.vote_average > 0
    AND movie.vote_count >= ?2;

-- name: GetRatingByReleaseDecade :many
WITH rated_movies AS (
    SELECT
        watched.movie_id,
        CAST(COALESCE(AVG(CAST(watched.rating AS REAL)), 0.0) AS REAL) AS average_rating
    FROM
        watched
    WHERE
        watched.user_id = ?1
        AND watched.rating IS NOT NULL
    GROUP BY
        watched.movie_id
)
SELECT
    (CAST(strftime('%Y', movie.release_date) AS INTEGER) / 10) * 10 AS decade,
    CAST(COALESCE(AVG(rated_movies.average_rating), 0.0) AS REAL) AS average_rating,
    COUNT(*) AS rated_movie_count
FROM
    rated_movies
    JOIN movie ON rated_movies.movie_id = movie.id
WHERE
    movie.release_date IS NOT NULL
GROUP BY
    decade
ORDER BY
    decade;

-- name: GetFavoriteDirectorsByRating :many
WITH rated_movies AS (
    SELECT
        watched.movie_id,
        CAST(COALESCE(AVG(CAST(watched.rating AS REAL)), 0.0) AS REAL) AS average_rating
    FROM
        watched
    WHERE
        watched.user_id = ?1
        AND watched.rating IS NOT NULL
    GROUP BY
        watched.movie_id
),
directors AS (
    SELECT DISTINCT
        crew.movie_id,
        crew.person_id
    FROM
        crew
    WHERE
        crew.job = 'Director'
)
SELECT
    person.id,
    person.name,
    person.profile_path,
    CAST(COALESCE(AVG(rated_movies.average_rating), 0.0) AS REAL) AS average_rating,
    COUNT(*) AS rated_movie_count
FROM
    rated_movies
    JOIN directors ON rated_movies.movie_id = directors.movie_id
    JOIN person ON directors.person_id = person.id
GROUP BY
    person.id,
    person.name,
    person.profile_path
ORDER BY
    average_rating DESC,
    rated_movie_count DESC,
    person.name ASC;

-- name: GetFavoriteActorsByRating :many
WITH rated_movies AS (
    SELECT
        watched.movie_id,
        CAST(COALESCE(AVG(CAST(watched.rating AS REAL)), 0.0) AS REAL) AS average_rating
    FROM
        watched
    WHERE
        watched.user_id = ?1
        AND watched.rating IS NOT NULL
    GROUP BY
        watched.movie_id
),
cast_members AS (
    SELECT DISTINCT
        "cast".movie_id,
        "cast".person_id
    FROM
        "cast"
)
SELECT
    person.id,
    person.name,
    person.profile_path,
    person.gender,
    CAST(COALESCE(AVG(rated_movies.average_rating), 0.0) AS REAL) AS average_rating,
    COUNT(*) AS rated_movie_count
FROM
    rated_movies
    JOIN cast_members ON rated_movies.movie_id = cast_members.movie_id
    JOIN person ON cast_members.person_id = person.id
GROUP BY
    person.id,
    person.name,
    person.profile_path,
    person.gender
ORDER BY
    average_rating DESC,
    rated_movie_count DESC,
    person.name ASC;

-- name: GetRewatchRatingDrift :many
WITH rated_movies AS (
    SELECT
        movie.id,
        movie.title,
        movie.poster_path,
        COUNT(*) AS rated_watch_count,
        MIN(watched.watched_date) AS first_watched_date,
        MAX(watched.watched_date) AS last_watched_date,
        (
            SELECT
                CAST(first_watch.rating AS REAL)
            FROM
                watched AS first_watch
            WHERE
                first_watch.user_id = ?1
                AND first_watch.movie_id = movie.id
                AND first_watch.rating IS NOT NULL
            ORDER BY
                first_watch.watched_date ASC,
                first_watch.id ASC
            LIMIT
                1
        ) AS first_rating,
        (
            SELECT
                CAST(last_watch.rating AS REAL)
            FROM
                watched AS last_watch
            WHERE
                last_watch.user_id = ?1
                AND last_watch.movie_id = movie.id
                AND last_watch.rating IS NOT NULL
            ORDER BY
                last_watch.watched_date DESC,
                last_watch.id DESC
            LIMIT
                1
        ) AS last_rating
    FROM
        watched
        JOIN movie ON watched.movie_id = movie.id
    WHERE
        watched.user_id = ?1
        AND watched.rating IS NOT NULL
    GROUP BY
        movie.id,
        movie.title,
        movie.poster_path
    HAVING
        COUNT(*) >= CAST(?2 AS INTEGER)
)
SELECT
    rated_movies.id,
    rated_movies.title,
    rated_movies.poster_path,
    rated_movies.first_rating,
    rated_movies.last_rating,
    CAST(rated_movies.last_rating - rated_movies.first_rating AS REAL) AS rating_change,
    rated_movies.rated_watch_count,
    rated_movies.first_watched_date,
    rated_movies.last_watched_date
FROM
    rated_movies
WHERE
    ABS(rated_movies.last_rating - rated_movies.first_rating) > 0
ORDER BY
    ABS(rated_movies.last_rating - rated_movies.first_rating) DESC,
    rated_movies.rated_watch_count DESC,
    rated_movies.title ASC
LIMIT
    ?3;

-- Sessions.
-- name: CreateSession :exec
INSERT INTO
    session (id, user_id, expires_at)
VALUES
    (?, ?, ?);

-- name: GetSession :one
SELECT
    user_id,
    expires_at
FROM
    session
WHERE
    id = ?
    AND expires_at > datetime('now');

-- name: DeleteSession :exec
DELETE FROM
    session
WHERE
    id = ?;

-- name: DeleteExpiredSessions :exec
DELETE FROM
    session
WHERE
    expires_at <= datetime('now');

-- Users and authentication.
-- name: CreateUser :one
INSERT INTO
    user (email, name, password_hash, created_at)
VALUES
    (?, ?, ?, datetime('now'))
RETURNING
    *;

-- name: GetUserByEmail :one
SELECT
    *
FROM
    user
WHERE
    email = ?;

-- name: GetUserByID :one
SELECT
    *
FROM
    user
WHERE
    id = ?;

-- name: CountUsers :one
SELECT
    COUNT(*)
FROM
    user;

-- Admin and maintenance.
-- name: AssignNilUserWatched :exec
UPDATE
    watched
SET
    user_id = ?
WHERE
    user_id IS NULL;

-- name: AssignNilUserLists :exec
UPDATE
    list
SET
    user_id = ?
WHERE
    user_id IS NULL;

-- name: SetAdmin :exec
UPDATE
    user
SET
    admin = TRUE
WHERE
    id = ?;

-- name: GetAllUsersWithStats :many
SELECT
    u.id,
    u.email,
    u.name,
    u.created_at,
    u.admin,
    (
        SELECT
            COUNT(*)
        FROM
            watched w
        WHERE
            w.user_id = u.id
    ) AS watched_count,
    (
        SELECT
            COUNT(*)
        FROM
            list l
        WHERE
            l.user_id = u.id
    ) AS list_count
FROM
    user u
ORDER BY
    u.created_at DESC;

-- name: DeleteUser :exec
DELETE FROM
    user
WHERE
    id = ?;

-- name: UpdateUserPassword :exec
UPDATE
    user
SET
    password_hash = ?
WHERE
    id = ?;

-- name: UpdatePasswordResetRequired :exec
UPDATE
    user
SET
    password_reset_required = ?
WHERE
    id = ?;

-- name: GetAllListsWithMovies :many
SELECT
    sqlc.embed(list),
    sqlc.embed(movie),
    sqlc.embed(list_movie)
FROM
    list
    LEFT JOIN list_movie ON list_movie.list_id = list.id
    LEFT JOIN movie ON movie.id = list_movie.movie_id
WHERE
    list.user_id = ?
ORDER BY
    list.id,
    list_movie.date_added,
    list_movie.movie_id;
