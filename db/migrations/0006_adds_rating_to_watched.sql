-- +goose Up
ALTER TABLE
    watched
ADD
    COLUMN rating DECIMAL(1, 1);

-- +goose Down
ALTER TABLE
    watched DROP COLUMN rating;
