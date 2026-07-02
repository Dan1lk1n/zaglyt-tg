package message

import (
	"context"
	"fmt"
)

// Append stores a single message for the given chat.
func (r *MessageRepository) Append(ctx context.Context, channelID int64, text string) error {
	query := `INSERT INTO messages (channel_id, text) VALUES ($1, $2)`

	if _, err := r.db.ExecContext(ctx, query, channelID, text); err != nil {
		return fmt.Errorf("append message for channel %d: %w", channelID, err)
	}

	return nil
}
