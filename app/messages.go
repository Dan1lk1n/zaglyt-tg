package app

import "context"

// AppendMessage stores a chat message in the history the bot learns from.
func (a *App) AppendMessage(ctx context.Context, channelID int64, text string) error {
	return a.messages.Append(ctx, channelID, text)
}

// GetMessages returns the stored message history for a chat, oldest first.
func (a *App) GetMessages(ctx context.Context, channelID int64) ([]string, error) {
	return a.messages.List(ctx, channelID)
}

// ClearMessages wipes the stored message history for a chat.
func (a *App) ClearMessages(ctx context.Context, channelID int64) error {
	return a.messages.Clear(ctx, channelID)
}
