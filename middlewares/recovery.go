package middlewares

import (
	"context"
	"fmt"
	"runtime/debug"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

func RecoveryMiddleware(next bot.HandlerFunc) bot.HandlerFunc {
	return func(ctx context.Context, b *bot.Bot, update *models.Update) {
		defer func() {
			if r := recover(); r != nil {
				fmt.Printf("Panic: %v\nStack:\n%s", r, debug.Stack())
			}
		}()

		next(ctx, b, update)
	}
}
