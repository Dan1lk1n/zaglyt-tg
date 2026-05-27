package app

import (
	"context"
)

func (a *App) GetActiveChats(ctx context.Context) ([]int64, error) {
	return a.channels.GetActiveChannelIDs(ctx)
}
