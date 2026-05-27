package handlers

import (
	"context"
	"fmt"
	"zaglyt-tg/modules/messages"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

func (h *Handler) MessageHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message != nil {
		if update.Message.Chat.Type == "group" || update.Message.Chat.Type == "supergroup" {
			channel, err := h.app.GetChatByID(ctx, update.Message.Chat.ID)
			if err != nil {
				fmt.Println(err)
				return
			}

			if channel.Enabled {
				if update.Message.Text != "" {
					err := messages.Append(channel.ChannelID, update.Message.Text)
					if err != nil {
						fmt.Println(err)
						return
					}
				}
			}
		}
	}
}
