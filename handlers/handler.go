package handlers

import (
	"zaglyt-tg/repository/channel"
)

type Handler struct {
	channels channel.ChannelRepository
}

func NewHandler(channels channel.ChannelRepository) Handler {
	return Handler{
		channels: channels,
	}
}
