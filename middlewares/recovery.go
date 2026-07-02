package middlewares

import (
	"context"
	"log/slog"
	"runtime/debug"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

func RecoveryMiddleware(next bot.HandlerFunc) bot.HandlerFunc {
	return func(ctx context.Context, b *bot.Bot, update *models.Update) {
		defer func() {
			if r := recover(); r != nil {
				slog.Error("recovered from panic in handler",
					"panic", r,
					"stack", string(debug.Stack()),
				)
			}
		}()

		next(ctx, b, update)
	}
}
