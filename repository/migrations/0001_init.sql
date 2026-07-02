-- +goose Up
-- Initial schema: channels the bot operates in.
CREATE TABLE IF NOT EXISTS channels (
    id         SERIAL PRIMARY KEY,
    channel_id BIGINT      NOT NULL UNIQUE,
    enabled    BOOLEAN     NOT NULL DEFAULT TRUE,
    mode       TEXT                 DEFAULT 'default'
);

-- +goose Down
DROP TABLE IF EXISTS channels;
