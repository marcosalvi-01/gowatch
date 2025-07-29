-- +goose Up
CREATE TABLE movie (
    id INTEGER PRIMARY KEY,
    imdb_id TEXT NOT NULL,
    title TEXT NOT NULL,
    original_title TEXT NOT NULL,
    release_date DATE NOT NULL,
    original_language TEXT NOT NULL,
    overview TEXT NOT NULL,
    poster_path TEXT NOT NULL,
    backdrop_path TEXT NOT NULL,
    budget INTEGER NOT NULL,
    revenue INTEGER NOT NULL,
    runtime INTEGER NOT NULL,
    vote_average DECIMAL(4, 2) NOT NULL,
    vote_count INTEGER NOT NULL,
    popularity REAL NOT NULL,
    homepage TEXT NOT NULL,
    status TEXT NOT NULL,
    tagline TEXT NOT NULL
);

CREATE TABLE watched (
    movie_id INTEGER NOT NULL,
    watched_date DATE NOT NULL,
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

CREATE TABLE actor (
    id INTEGER PRIMARY KEY,
    name TEXT NOT NULL,
    profile_path TEXT NOT NULL,
    imdb_id TEXT NOT NULL
);

CREATE TABLE actor_movie (
    movie_id INTEGER NOT NULL,
    actor_id INTEGER NOT NULL,
    character TEXT NOT NULL,
    cast_order INTEGER NOT NULL,
    PRIMARY KEY (movie_id, actor_id),
    FOREIGN KEY (movie_id) REFERENCES movie(id) ON DELETE CASCADE,
    FOREIGN KEY (actor_id) REFERENCES actor(id) ON DELETE CASCADE
);

-- +goose Down
DROP TABLE actor_movie;

DROP TABLE genre_movie;

DROP TABLE watched;

DROP TABLE actor;

DROP TABLE genre;

DROP TABLE movie;
