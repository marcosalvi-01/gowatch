-- +goose Up
CREATE TABLE list (
    id INTEGER PRIMARY KEY autoincrement,
    name TEXT NOT NULL,
    creation_date TEXT NOT NULL,
    description TEXT
);

CREATE TABLE list_movie (
    movie_id INTEGER NOT NULL,
    list_id INTEGER NOT NULL,
    date_added TEXT NOT NULL,
    position INTEGER,
    note TEXT,
    PRIMARY KEY (movie_id, list_id),
    FOREIGN KEY (movie_id) REFERENCES movie(id) ON DELETE CASCADE,
    FOREIGN KEY (list_id) REFERENCES list(id) ON DELETE CASCADE
);

-- +goose Down
DROP TABLE list;

DROP TABLE list_movie;
