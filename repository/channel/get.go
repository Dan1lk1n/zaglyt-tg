package channel

import (
	"context"
	"fmt"
	"zaglyt-tg/models"
)

// channelColumns is the full column list for the channels table. It is shared by
// every query that returns a *models.Channel so that adding a column only
// requires touching this one place.
const channelColumns = "id, channel_id, enabled, mode, min_gen_words, max_gen_words, reply_probability"

// GetOrCreate returns the channel row for channelID, creating it with default
// values if it does not exist yet. It is a single, race-safe round-trip: the
// ON CONFLICT clause makes concurrent first-time inserts idempotent, and the
// no-op UPDATE lets RETURNING yield the existing row as well as a freshly
// inserted one.
func (r *ChannelRepository) GetOrCreate(ctx context.Context, channelID int64) (*models.Channel, error) {
	query := fmt.Sprintf(`
		INSERT INTO channels (channel_id) VALUES ($1)
		ON CONFLICT (channel_id) DO UPDATE SET channel_id = EXCLUDED.channel_id
		RETURNING %s`, channelColumns)

	var channel models.Channel
	if err := r.db.GetContext(ctx, &channel, query, channelID); err != nil {
		return nil, fmt.Errorf("get or create channel %d: %w", channelID, err)
	}

	return &channel, nil
}

func (r *ChannelRepository) GetActiveChannelIDs(ctx context.Context) ([]int64, error) {
	query := `SELECT channel_id FROM channels WHERE enabled = true`

	var ids []int64
	err := r.db.SelectContext(ctx, &ids, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get active channel IDs: %w", err)
	}

	return ids, nil
}
