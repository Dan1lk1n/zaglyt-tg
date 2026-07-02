package channel

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"zaglyt-tg/models"
)

// SetEnabled toggles only the enabled flag for the channel, leaving mode
// untouched, and returns the updated row. Callers no longer need to read the
// channel first just to preserve mode.
func (r *ChannelRepository) SetEnabled(ctx context.Context, channelID int64, enabled bool) (*models.Channel, error) {
	query := fmt.Sprintf(`
		UPDATE channels
		SET enabled = $2
		WHERE channel_id = $1
		RETURNING %s`, channelColumns)

	var updatedChannel models.Channel
	err := r.db.GetContext(ctx, &updatedChannel, query, channelID, enabled)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("channel %d is not found", channelID)
		}
		return nil, fmt.Errorf("an error occurred while updating the channel %d: %w", channelID, err)
	}

	return &updatedChannel, nil
}

// SetWords updates the generated-response word range for the channel and returns
// the updated row. Validation of min/max is the caller's responsibility.
func (r *ChannelRepository) SetWords(ctx context.Context, channelID int64, minWords, maxWords int) (*models.Channel, error) {
	query := fmt.Sprintf(`
		UPDATE channels
		SET min_gen_words = $2, max_gen_words = $3
		WHERE channel_id = $1
		RETURNING %s`, channelColumns)

	var updatedChannel models.Channel
	err := r.db.GetContext(ctx, &updatedChannel, query, channelID, minWords, maxWords)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("channel %d is not found", channelID)
		}
		return nil, fmt.Errorf("an error occurred while updating the channel %d: %w", channelID, err)
	}

	return &updatedChannel, nil
}

// SetReplyProbability updates the random-reply chance (percent, 0..100) for the
// channel and returns the updated row.
func (r *ChannelRepository) SetReplyProbability(ctx context.Context, channelID int64, pct int) (*models.Channel, error) {
	query := fmt.Sprintf(`
		UPDATE channels
		SET reply_probability = $2
		WHERE channel_id = $1
		RETURNING %s`, channelColumns)

	var updatedChannel models.Channel
	err := r.db.GetContext(ctx, &updatedChannel, query, channelID, pct)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("channel %d is not found", channelID)
		}
		return nil, fmt.Errorf("an error occurred while updating the channel %d: %w", channelID, err)
	}

	return &updatedChannel, nil
}
