package message

import "github.com/jmoiron/sqlx"

// MessageRepository persists per-chat message history in Postgres. It replaces
// the previous file-based store, which was not safe under the concurrent update
// handling of the bot.
type MessageRepository struct {
	db *sqlx.DB
}

func NewMessageRepository(db *sqlx.DB) MessageRepository {
	return MessageRepository{db: db}
}
