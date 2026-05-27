// admin commands
package handlers

import (
	"context"
	"fmt"
	"zaglyt-tg/modules/helpers"

	"github.com/go-telegram/bot"
	goTelegramModels "github.com/go-telegram/bot/models"
)

func (h *Handler) WhoAmICommandHandler(ctx context.Context, b *bot.Bot, update *goTelegramModels.Update) {
	if update.Message != nil {
		if helpers.IsUserDeveloper(update.Message.From.ID) {
			_, _ = b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: update.Message.Chat.ID,
				Text:   "Разработчик.",
				ReplyParameters: &goTelegramModels.ReplyParameters{
					MessageID: update.Message.ID,
				},
			})
		}
	}
}

func (h *Handler) GetBotStatsCommandHandler(ctx context.Context, b *bot.Bot, update *goTelegramModels.Update) {
	if update.Message != nil {
		if helpers.IsUserDeveloper(update.Message.From.ID) {
			stats, err := h.app.GetBotStats(ctx)
			if err != nil {
				fmt.Println(err)
				return
			}

			_, _ = b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: update.Message.Chat.ID,
				Text:   fmt.Sprintf("1. Чатов в БД: %d\n2. Чатов в БД, где бот включен: %d\n3. Чатов в БД, где бот выключен: %d", stats.TotalChats, stats.EnabledChats, stats.TotalChats-stats.EnabledChats),
				ReplyParameters: &goTelegramModels.ReplyParameters{
					MessageID: update.Message.ID,
				},
			})
		}
	}
}
