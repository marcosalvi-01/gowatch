-- +goose Up
CREATE TABLE movie (
    id INTEGER PRIMARY KEY,
    title TEXT NOT NULL,
    original_title TEXT NOT NULL,
    original_language TEXT NOT NULL,
    overview TEXT NOT NULL,
    release_date DATE NOT NULL,
    poster_path TEXT NOT NULL,
    backdrop_path TEXT NOT NULL,
    popularity REAL NOT NULL,
    vote_count INTEGER NOT NULL,
    vote_average DECIMAL(4, 2) NOT NULL,
    budget INTEGER NOT NULL,
    homepage TEXT NOT NULL,
    imdb_id TEXT NOT NULL,
    -- origin_country []string
    revenue INTEGER NOT NULL,
    runtime INTEGER NOT NULL,
    status TEXT NOT NULL,
    tagline TEXT NOT NULL
);

CREATE TABLE watched (
    movie_id INTEGER NOT NULL,
    watched_date DATE NOT NULL,
    watched_in_theater BOOLEAN DEFAULT false NOT NULL,
    PRIMARY KEY (movie_id, watched_date),
    FOREIGN KEY (movie_id) REFERENCES movie(id) ON DELETE CASCADE
);

CREATE TABLE genre (
    id INTEGER PRIMARY KEY,
    name TEXT NOT NULL UNIQUE
);

CREATE TABLE genre_movie (
    movie_id INTEGER NOT NULL,
    genre_id INTEGER NOT NULL,
    PRIMARY KEY (movie_id, genre_id),
    FOREIGN KEY (movie_id) REFERENCES movie(id) ON DELETE CASCADE,
    FOREIGN KEY (genre_id) REFERENCES genre(id) ON DELETE CASCADE
);

CREATE TABLE person (
    id INTEGER PRIMARY KEY,
    name TEXT NOT NULL,
    original_name TEXT NOT NULL,
    profile_path TEXT NOT NULL,
    known_for_department TEXT NOT NULL,
    popularity REAL NOT NULL,
    gender INTEGER NOT NULL,
    adult BOOLEAN NOT NULL DEFAULT FALSE
);

CREATE TABLE cast (
    movie_id INTEGER NOT NULL,
    person_id INTEGER NOT NULL,
    cast_id INTEGER NOT NULL,
    credit_id TEXT NOT NULL,
    character TEXT NOT NULL,
    cast_order INTEGER NOT NULL,
    PRIMARY KEY (movie_id, person_id, cast_id),
    FOREIGN KEY (movie_id) REFERENCES movie(id) ON DELETE CASCADE,
    FOREIGN KEY (person_id) REFERENCES person(id) ON DELETE CASCADE
);

CREATE TABLE crew (
    movie_id INTEGER NOT NULL,
    person_id INTEGER NOT NULL,
    credit_id TEXT NOT NULL,
    job TEXT NOT NULL,
    department TEXT NOT NULL,
    PRIMARY KEY (movie_id, person_id, credit_id),
    FOREIGN KEY (movie_id) REFERENCES movie(id) ON DELETE CASCADE,
    FOREIGN KEY (person_id) REFERENCES person(id) ON DELETE CASCADE
);

-- +goose Down
DROP TABLE crew;

DROP TABLE cast;

DROP TABLE genre_movie;

DROP TABLE watched;

DROP TABLE person;

DROP TABLE genre;

DROP TABLE movie;
