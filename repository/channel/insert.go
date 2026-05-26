package channel

import (
	"context"
	"fmt"
)

func (r *ChannelRepository) Insert(ctx context.Context, channelID int64, enabled bool, mode string) error {
	query := `INSERT INTO channels (channel_id, enabled, mode) VALUES ($1, $2, $3)`

	_, err := r.db.ExecContext(ctx, query, channelID, enabled, mode)
	if err != nil {
		return fmt.Errorf("an error occurred while adding the channel %d: %w", channelID, err)
	}

	return nil
}
