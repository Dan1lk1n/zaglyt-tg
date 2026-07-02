package app

import (
	"zaglyt-tg/repository/channel"
	"zaglyt-tg/repository/message"
)

type App struct {
	channels channel.ChannelRepository
	messages message.MessageRepository
}

func NewApp(channels channel.ChannelRepository, messages message.MessageRepository) App {
	return App{
		channels: channels,
		messages: messages,
	}
}
