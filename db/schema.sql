PRAGMA foreign_keys = ON;

CREATE TABLE IF NOT EXISTS movie (
    id INTEGER PRIMARY KEY,
    imdb_id TEXT,
    title TEXT NOT NULL,
    release_date DATE NOT NULL,
    original_language TEXT NOT NULL,
    overview TEXT NOT NULL,
    poster_path TEXT NOT NULL,
    budget BIGINT NOT NULL,
    revenue BIGINT NOT NULL,
    runtime INTEGER NOT NULL,
    vote_average DECIMAL(3, 1) NOT NULL
);

CREATE TABLE IF NOT EXISTS watched (
    movie_id INTEGER NOT NULL,
    watched_date DATE NOT NULL,
    PRIMARY KEY (movie_id, watched_date),
    FOREIGN KEY (movie_id) REFERENCES movie(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS genre (
    id INTEGER PRIMARY KEY,
    name TEXT NOT NULL UNIQUE
);

CREATE TABLE IF NOT EXISTS genre_movie (
    movie_id INTEGER NOT NULL,
    genre_id INTEGER NOT NULL,
    PRIMARY KEY (movie_id, genre_id),
    FOREIGN KEY (movie_id) REFERENCES movie(id) ON DELETE CASCADE,
    FOREIGN KEY (genre_id) REFERENCES genre(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS actor (
    id INTEGER PRIMARY KEY,
    name TEXT NOT NULL,
    profile_path TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS actor_movie (
    movie_id INTEGER NOT NULL,
    actor_id INTEGER NOT NULL,
    character TEXT NOT NULL,
    cast_order INTEGER NOT NULL,
    PRIMARY KEY (movie_id, actor_id),
    FOREIGN KEY (movie_id) REFERENCES movie(id) ON DELETE CASCADE,
    FOREIGN KEY (actor_id) REFERENCES actor(id) ON DELETE CASCADE
);
