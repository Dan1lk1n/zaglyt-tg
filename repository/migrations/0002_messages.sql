-- +goose Up
-- Message history the bot learns from, one row per stored chat message.
-- Replaces the previous file-based storage under database/messages/*.txt.
CREATE TABLE IF NOT EXISTS messages (
    id         BIGSERIAL   PRIMARY KEY,
    channel_id BIGINT      NOT NULL,
    text       TEXT        NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_messages_channel_id ON messages (channel_id);

-- +goose Down
DROP INDEX IF EXISTS idx_messages_channel_id;
DROP TABLE IF EXISTS messages;
