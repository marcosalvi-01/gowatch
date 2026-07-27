-- +goose Up
ALTER TABLE
    list
ADD
    COLUMN display_sort TEXT NOT NULL DEFAULT 'custom';

UPDATE list
SET
    display_sort = 'release_status'
WHERE
    is_watchlist = TRUE;

-- +goose Down
ALTER TABLE
    list DROP COLUMN display_sort;
