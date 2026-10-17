-- +goose Up
CREATE TABLE production_company (
    id INTEGER PRIMARY KEY,
    logo_path TEXT NOT NULL DEFAULT '',
    origin_country TEXT NOT NULL DEFAULT '',
    name TEXT NOT NULL
);

CREATE TABLE movie_production_company (
    movie_id INTEGER NOT NULL,
    production_company_id INTEGER NOT NULL,
    PRIMARY KEY (movie_id, production_company_id),
    FOREIGN KEY (movie_id) REFERENCES movie(id) ON DELETE CASCADE,
    FOREIGN KEY (production_company_id) REFERENCES production_company(id) ON DELETE CASCADE
);

CREATE TABLE production_country (
    iso_code TEXT PRIMARY KEY,
    name TEXT NOT NULL
);

CREATE TABLE movie_production_country (
    movie_id INTEGER NOT NULL,
    country_iso_code TEXT NOT NULL,
    PRIMARY KEY (movie_id, country_iso_code),
    FOREIGN KEY (movie_id) REFERENCES movie(id) ON DELETE CASCADE,
    FOREIGN KEY (country_iso_code) REFERENCES production_country(iso_code) ON DELETE CASCADE
);

CREATE TABLE movie_related_video (
    movie_id INTEGER NOT NULL,
    position INTEGER NOT NULL,
    url TEXT NOT NULL,
    video_key TEXT NOT NULL,
    name TEXT NOT NULL,
    size INTEGER NOT NULL,
    type TEXT NOT NULL,
    site TEXT NOT NULL,
    PRIMARY KEY (movie_id, position),
    FOREIGN KEY (movie_id) REFERENCES movie(id) ON DELETE CASCADE
);

UPDATE movie SET updated_at = '1970-01-01 00:00:00';

-- +goose Down
DROP TABLE movie_related_video;
DROP TABLE movie_production_country;
DROP TABLE production_country;
DROP TABLE movie_production_company;
DROP TABLE production_company;
