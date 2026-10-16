-- +goose Up
ALTER TABLE person DROP COLUMN original_name;

ALTER TABLE person DROP COLUMN known_for_department;

ALTER TABLE person DROP COLUMN popularity;

ALTER TABLE person DROP COLUMN adult;

-- +goose Down
ALTER TABLE person ADD COLUMN original_name TEXT NOT NULL DEFAULT '';

ALTER TABLE person ADD COLUMN known_for_department TEXT NOT NULL DEFAULT '';

ALTER TABLE person ADD COLUMN popularity REAL NOT NULL DEFAULT 0;

ALTER TABLE person ADD COLUMN adult BOOLEAN NOT NULL DEFAULT FALSE;
