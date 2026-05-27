package app

import (
	"context"
	"zaglyt-tg/models"
)

func (a *App) GetChatByID(ctx context.Context, channelID int64) (*models.Channel, error) {
	channel, err := a.channels.GetByChannelID(ctx, channelID)
	if err != nil {
		return nil, err
	}
	if channel == nil {
		err := a.channels.Insert(ctx, channelID, true, "default")
		if err != nil {
			return nil, err
		}

		defaultMode := "default"

		channel = &models.Channel{
			ChannelID: channelID,
			Enabled:   true,
			Mode:      &defaultMode,
		}
	}

	return channel, nil
}
