package message

import (
	"context"
	"fmt"
)

// List returns every stored message text for the chat, oldest first.
func (r *MessageRepository) List(ctx context.Context, channelID int64) ([]string, error) {
	query := `SELECT text FROM messages WHERE channel_id = $1 ORDER BY id`

	var texts []string
	if err := r.db.SelectContext(ctx, &texts, query, channelID); err != nil {
		return nil, fmt.Errorf("list messages for channel %d: %w", channelID, err)
	}

	return texts, nil
}
