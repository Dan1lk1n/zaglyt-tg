package message

import (
	"context"
	"fmt"
)

// Clear removes all stored messages for the chat.
func (r *MessageRepository) Clear(ctx context.Context, channelID int64) error {
	query := `DELETE FROM messages WHERE channel_id = $1`

	if _, err := r.db.ExecContext(ctx, query, channelID); err != nil {
		return fmt.Errorf("clear messages for channel %d: %w", channelID, err)
	}

	return nil
}
