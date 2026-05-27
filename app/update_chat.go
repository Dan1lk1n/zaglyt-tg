package app

import (
	"context"
	"zaglyt-tg/models"
)

func (a *App) UpdateChat(ctx context.Context, channelID int64, enabled bool, mode string) (*models.Channel, error) {
	channel, err := a.channels.GetByChannelID(ctx, channelID)
	if err != nil {
		return nil, err
	}

	channel, err = a.channels.Update(ctx, channelID, enabled, mode)
	if err != nil {
		return nil, err
	}

	return channel, nil
}
