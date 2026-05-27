package app

import (
	"context"
	"zaglyt-tg/models"
)

func (a *App) GetBotStats(ctx context.Context) (*models.BotStats, error) {
	return a.channels.GetBotStats(ctx)
}
