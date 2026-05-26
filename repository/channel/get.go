package channel

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"zaglyt-tg/models"
)

func (r *ChannelRepository) GetByChannelID(ctx context.Context, channelID int64) (*models.Channel, error) {
	query := `SELECT id, channel_id, enabled, mode FROM channels WHERE channel_id = $1`

	var channel models.Channel
	err := r.db.GetContext(ctx, &channel, query, channelID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("an error occurred while getting the channel %d: %w", channelID, err)
	}

	return &channel, nil
}
