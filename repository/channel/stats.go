package channel

import (
	"context"
	"fmt"
	"zaglyt-tg/models"
)

func (r *ChannelRepository) GetBotStats(ctx context.Context) (*models.BotStats, error) {
	query := `
		SELECT 
			COUNT(*) as total_chats,
			COALESCE(SUM(CASE WHEN enabled = true THEN 1 ELSE 0 END), 0) as enabled_chats
		FROM channels
	`

	var stats models.BotStats
	err := r.db.GetContext(ctx, &stats, query)
	if err != nil {
		return nil, fmt.Errorf("an error occurred while getting bot stats: %w", err)
	}

	return &stats, nil
}
