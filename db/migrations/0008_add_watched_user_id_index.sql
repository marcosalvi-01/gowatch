-- +goose Up
CREATE INDEX IF NOT EXISTS idx_watched_user_id ON watched(user_id);

-- +goose Down
DROP INDEX IF EXISTS idx_watched_user_id;
