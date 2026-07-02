package app

import (
	"context"
	"zaglyt-tg/models"
)

// GetChatByID returns the channel for chatID, creating it with default values
// on first sight. Creation is a single race-safe upsert, so concurrent updates
// from the same new chat can no longer collide.
func (a *App) GetChatByID(ctx context.Context, channelID int64) (*models.Channel, error) {
	return a.channels.GetOrCreate(ctx, channelID)
}
