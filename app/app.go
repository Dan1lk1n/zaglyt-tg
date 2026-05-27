package app

import "zaglyt-tg/repository/channel"

type App struct {
	channels channel.ChannelRepository
}

func NewApp(channels channel.ChannelRepository) App {
	return App{
		channels: channels,
	}
}
