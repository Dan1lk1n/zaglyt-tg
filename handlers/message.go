package handlers

import (
	"context"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

func (h *Handler) MessageHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message != nil {
		if update.Message.Chat.Type == "group" || update.Message.Chat.Type == "supergroup" {
			channel, err := h.channels.GetByChannelID(ctx, update.Message.Chat.ID)
			if err != nil {
				return
			}
			if channel == nil {
				err := h.channels.Insert(ctx, update.Message.Chat.ID, true, "default")
				if err != nil {
					return
				}

				channel, err = h.channels.GetByChannelID(ctx, update.Message.Chat.ID)
				if err != nil {
					return
				}
			}

			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: update.Message.Chat.ID,
				Text:   "Chat is created",
			})
		}
	}
}
