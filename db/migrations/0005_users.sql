-- +goose Up
-- Add user authentication and multi-user support
-- Note: user_id is nullable in watched/list tables to handle existing data
-- Application code assigns data to first registered user
CREATE TABLE user (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    email TEXT UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    name TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    admin BOOLEAN DEFAULT false NOT NULL,
    password_reset_required BOOLEAN DEFAULT false NOT NULL
);

CREATE TABLE session (
    id TEXT PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES user(id) ON DELETE CASCADE,
    expires_at DATETIME NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Recreate watched table to fix issue: multiple users couldn't watch same movie on same day
-- Old PK: (movie_id, watched_date) -> New: auto-increment ID with unique constraint
CREATE TABLE watched_new (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    movie_id INTEGER NOT NULL,
    user_id INTEGER REFERENCES user(id) ON DELETE CASCADE,  -- Nullable for existing data
    watched_date DATE NOT NULL,
    watched_in_theater BOOLEAN DEFAULT false NOT NULL,
    FOREIGN KEY (movie_id) REFERENCES movie(id) ON DELETE CASCADE,
    UNIQUE(movie_id, user_id, watched_date) -- Prevents same user watching same movie twice on same day
);

-- Copy existing data (user_id will be NULL initially)
INSERT INTO
    watched_new (
        movie_id,
        user_id,
        watched_date,
        watched_in_theater
    )
SELECT
    movie_id,
    NULL,
    watched_date,
    watched_in_theater
FROM
    watched;

DROP TABLE watched;

ALTER TABLE
    watched_new RENAME TO watched;

-- Add user_id to list table (list_movie doesn't need it - inherits from list)
ALTER TABLE
    list
ADD
    COLUMN user_id INTEGER REFERENCES user(id) ON DELETE CASCADE;

-- +goose Down
ALTER TABLE
    list DROP COLUMN user_id;

-- Restore original watched table structure (loses user_id and watch history per user)
CREATE TABLE watched_old (
    movie_id INTEGER NOT NULL,
    watched_date DATE NOT NULL,
    watched_in_theater BOOLEAN DEFAULT false NOT NULL,
    PRIMARY KEY (movie_id, watched_date),
    FOREIGN KEY (movie_id) REFERENCES movie(id) ON DELETE CASCADE
);

INSERT INTO
    watched_old (movie_id, watched_date, watched_in_theater)
SELECT
    movie_id,
    watched_date,
    watched_in_theater
FROM
    watched;

DROP TABLE watched;

ALTER TABLE
    watched_old RENAME TO watched;

DROP TABLE IF EXISTS session;

DROP TABLE IF EXISTS user;
