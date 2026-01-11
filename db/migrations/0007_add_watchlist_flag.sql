-- +goose Up
ALTER TABLE
    list
ADD
    COLUMN is_watchlist BOOLEAN DEFAULT FALSE NOT NULL;

-- ensure only one watchlist per user
CREATE UNIQUE INDEX idx_user_watchlist ON list(user_id)
WHERE
    is_watchlist = TRUE;

-- +goose Down
DROP INDEX idx_user_watchlist;

ALTER TABLE
    list DROP COLUMN is_watchlist;

