-- +goose Up
-- Per-chat generation settings. Defaults mirror the previously hardcoded values
-- (2..10 words, 10% random-reply chance) so existing chats keep their behavior.
ALTER TABLE channels
    ADD COLUMN IF NOT EXISTS min_gen_words     INT NOT NULL DEFAULT 2,
    ADD COLUMN IF NOT EXISTS max_gen_words     INT NOT NULL DEFAULT 10,
    ADD COLUMN IF NOT EXISTS reply_probability INT NOT NULL DEFAULT 10;

-- +goose Down
ALTER TABLE channels
    DROP COLUMN IF EXISTS reply_probability,
    DROP COLUMN IF EXISTS max_gen_words,
    DROP COLUMN IF EXISTS min_gen_words;
