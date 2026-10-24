-- +goose Up
ALTER TABLE person ADD COLUMN biography TEXT NOT NULL DEFAULT '';
ALTER TABLE person ADD COLUMN birthday DATE;
ALTER TABLE person ADD COLUMN deathday DATE;
ALTER TABLE person ADD COLUMN place_of_birth TEXT NOT NULL DEFAULT '';
ALTER TABLE person ADD COLUMN imdb_id TEXT NOT NULL DEFAULT '';
ALTER TABLE person ADD COLUMN homepage TEXT NOT NULL DEFAULT '';
ALTER TABLE person ADD COLUMN details_updated_at TIMESTAMP;

ALTER TABLE movie ADD COLUMN imdb_rating_present BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE movie ADD COLUMN imdb_score REAL;
ALTER TABLE movie ADD COLUMN imdb_rating_url TEXT NOT NULL DEFAULT '';
ALTER TABLE movie ADD COLUMN rt_rating_present BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE movie ADD COLUMN rt_critics_score REAL;
ALTER TABLE movie ADD COLUMN rt_critics_rating TEXT NOT NULL DEFAULT '';
ALTER TABLE movie ADD COLUMN rt_audience_score REAL;
ALTER TABLE movie ADD COLUMN rt_audience_rating TEXT NOT NULL DEFAULT '';
ALTER TABLE movie ADD COLUMN rt_rating_url TEXT NOT NULL DEFAULT '';
ALTER TABLE movie ADD COLUMN ratings_updated_at TIMESTAMP;
ALTER TABLE movie ADD COLUMN recommendations_updated_at TIMESTAMP;
ALTER TABLE movie ADD COLUMN similar_updated_at TIMESTAMP;

CREATE TABLE tv (
    id INTEGER PRIMARY KEY,
    title TEXT NOT NULL,
    release_date DATE,
    poster_path TEXT NOT NULL,
    backdrop_path TEXT NOT NULL,
    popularity REAL NOT NULL,
    vote_count INTEGER NOT NULL,
    vote_average REAL NOT NULL
);

CREATE TABLE person_movie_credit (
    person_id INTEGER NOT NULL REFERENCES person(id) ON DELETE CASCADE,
    movie_id INTEGER NOT NULL REFERENCES movie(id) ON DELETE CASCADE,
    role_kind TEXT NOT NULL CHECK (role_kind IN ('acting', 'crew')),
    position INTEGER NOT NULL,
    credit_id TEXT NOT NULL,
    role TEXT NOT NULL,
    department TEXT NOT NULL,
    episode_count INTEGER NOT NULL,
    PRIMARY KEY (person_id, role_kind, position)
);

CREATE TABLE person_tv_credit (
    person_id INTEGER NOT NULL REFERENCES person(id) ON DELETE CASCADE,
    tv_id INTEGER NOT NULL REFERENCES tv(id) ON DELETE CASCADE,
    role_kind TEXT NOT NULL CHECK (role_kind IN ('acting', 'crew')),
    position INTEGER NOT NULL,
    credit_id TEXT NOT NULL,
    role TEXT NOT NULL,
    department TEXT NOT NULL,
    episode_count INTEGER NOT NULL,
    PRIMARY KEY (person_id, role_kind, position)
);

CREATE TABLE movie_relationship (
    movie_id INTEGER NOT NULL REFERENCES movie(id) ON DELETE CASCADE,
    related_movie_id INTEGER NOT NULL REFERENCES movie(id) ON DELETE CASCADE,
    kind TEXT NOT NULL CHECK (kind IN ('recommendations', 'similar')),
    position INTEGER NOT NULL,
    PRIMARY KEY (movie_id, kind, position)
);

-- +goose Down
DROP TABLE movie_relationship;
DROP TABLE person_tv_credit;
DROP TABLE person_movie_credit;
DROP TABLE tv;

ALTER TABLE movie DROP COLUMN similar_updated_at;
ALTER TABLE movie DROP COLUMN recommendations_updated_at;
ALTER TABLE movie DROP COLUMN ratings_updated_at;
ALTER TABLE movie DROP COLUMN rt_rating_url;
ALTER TABLE movie DROP COLUMN rt_audience_rating;
ALTER TABLE movie DROP COLUMN rt_audience_score;
ALTER TABLE movie DROP COLUMN rt_critics_rating;
ALTER TABLE movie DROP COLUMN rt_critics_score;
ALTER TABLE movie DROP COLUMN rt_rating_present;
ALTER TABLE movie DROP COLUMN imdb_rating_url;
ALTER TABLE movie DROP COLUMN imdb_score;
ALTER TABLE movie DROP COLUMN imdb_rating_present;

ALTER TABLE person DROP COLUMN details_updated_at;
ALTER TABLE person DROP COLUMN homepage;
ALTER TABLE person DROP COLUMN imdb_id;
ALTER TABLE person DROP COLUMN place_of_birth;
ALTER TABLE person DROP COLUMN deathday;
ALTER TABLE person DROP COLUMN birthday;
ALTER TABLE person DROP COLUMN biography;
