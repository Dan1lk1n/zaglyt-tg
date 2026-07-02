package models

type Channel struct {
	ID               int     `db:"id" json:"id"`
	ChannelID        int64   `db:"channel_id" json:"channel_id"`
	Enabled          bool    `db:"enabled" json:"enabled"`
	Mode             *string `db:"mode" json:"mode"`
	MinGenWords      int     `db:"min_gen_words" json:"min_gen_words"`
	MaxGenWords      int     `db:"max_gen_words" json:"max_gen_words"`
	ReplyProbability int     `db:"reply_probability" json:"reply_probability"`
}
