package channel

import (
	"context"
	"fmt"
	"zaglyt-tg/models"
)

func (r *ChannelRepository) GetBotStats(ctx context.Context) (*models.BotStats, error) {
	query := `
		SELECT
			COUNT(*)                         AS total_chats,
			COUNT(*) FILTER (WHERE enabled)  AS enabled_chats
		FROM channels
	`

	var stats models.BotStats
	err := r.db.GetContext(ctx, &stats, query)
	if err != nil {
		return nil, fmt.Errorf("an error occurred while getting bot stats: %w", err)
	}

	return &stats, nil
}
