package channel

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"zaglyt-tg/models"
)

func (r *ChannelRepository) Update(ctx context.Context, channelID int64, enabled bool, mode *string) (*models.Channel, error) {
	query := `
		UPDATE channels 
		SET enabled = $2, mode = $3 
		WHERE channel_id = $1 
		RETURNING id, channel_id, enabled, mode`

	var updatedChannel models.Channel
	err := r.db.GetContext(ctx, &updatedChannel, query, channelID, enabled, mode)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("channel %d is not found", channelID)
		}
		return nil, fmt.Errorf("an error occurred while updating the channel %d: %w", channelID, err)
	}

	return &updatedChannel, nil
}
