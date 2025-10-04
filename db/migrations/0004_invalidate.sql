-- +goose Up
-- Add columns as nullable first (no default constraint during ALTER)
ALTER TABLE
    movie
ADD
    COLUMN updated_at TIMESTAMP;

ALTER TABLE
    genre
ADD
    COLUMN updated_at TIMESTAMP;

ALTER TABLE
    genre_movie
ADD
    COLUMN updated_at TIMESTAMP;

ALTER TABLE
    person
ADD
    COLUMN updated_at TIMESTAMP;

ALTER TABLE
    cast
ADD
    COLUMN updated_at TIMESTAMP;

ALTER TABLE
    crew
ADD
    COLUMN updated_at TIMESTAMP;

-- Then update existing rows to set the timestamp
UPDATE
    movie
SET
    updated_at = CURRENT_TIMESTAMP
WHERE
    updated_at IS NULL;

UPDATE
    genre
SET
    updated_at = CURRENT_TIMESTAMP
WHERE
    updated_at IS NULL;

UPDATE
    genre_movie
SET
    updated_at = CURRENT_TIMESTAMP
WHERE
    updated_at IS NULL;

UPDATE
    person
SET
    updated_at = CURRENT_TIMESTAMP
WHERE
    updated_at IS NULL;

UPDATE
    cast
SET
    updated_at = CURRENT_TIMESTAMP
WHERE
    updated_at IS NULL;

UPDATE
    crew
SET
    updated_at = CURRENT_TIMESTAMP
WHERE
    updated_at IS NULL;

-- +goose Down
ALTER TABLE
    movie DROP COLUMN updated_at;

ALTER TABLE
    genre DROP COLUMN updated_at;

ALTER TABLE
    genre_movie DROP COLUMN updated_at;

ALTER TABLE
    person DROP COLUMN updated_at;

ALTER TABLE
    cast DROP COLUMN updated_at;

ALTER TABLE
    crew DROP COLUMN updated_at;
