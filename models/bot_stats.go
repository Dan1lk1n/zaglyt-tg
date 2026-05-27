package models

type BotStats struct {
	TotalChats   int64 `db:"total_chats"`
	EnabledChats int64 `db:"enabled_chats"`
}
