package models

type Channel struct {
	ID        int    `db:"id" json:"id"`
	ChannelID int64  `db:"channel_id" json:"channel_id"`
	Enabled   bool   `db:"enabled" json:"enabled"`
	Mode      string `db:"mode" json:"mode"`
}
