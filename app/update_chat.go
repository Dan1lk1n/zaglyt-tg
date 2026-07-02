package app

import (
	"context"
	"zaglyt-tg/models"
)

// SetChatEnabled flips the bot on/off for a chat in a single UPDATE, without
// the redundant read the previous implementation performed.
func (a *App) SetChatEnabled(ctx context.Context, channelID int64, enabled bool) (*models.Channel, error) {
	return a.channels.SetEnabled(ctx, channelID, enabled)
}

// SetChatWords updates the per-chat generated-response word range.
func (a *App) SetChatWords(ctx context.Context, channelID int64, minWords, maxWords int) (*models.Channel, error) {
	return a.channels.SetWords(ctx, channelID, minWords, maxWords)
}

// SetChatReplyProbability updates the per-chat random-reply chance (percent, 0..100).
func (a *App) SetChatReplyProbability(ctx context.Context, channelID int64, pct int) (*models.Channel, error) {
	return a.channels.SetReplyProbability(ctx, channelID, pct)
}
