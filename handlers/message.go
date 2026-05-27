package handlers

import (
	"context"
	"fmt"
	"strings"
	"zaglyt-tg/modules/helpers"
	"zaglyt-tg/modules/messages"
	"zaglyt-tg/modules/z3abp"

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

					if !helpers.ShouldRespond(ctx, b, update, h.bot.Username) {
						return
					}

					db, err := messages.Read(channel.ChannelID)
					if err != nil {
						fmt.Println(err)
						return
					}

					response, err := z3abp.GenerateBestResponse(update.Message.Text, strings.Split(db, "\n"), z3abp.DefaultConfig())
					if err != nil {
						fmt.Println(err)
						return
					}

					_, err = b.SendMessage(ctx, &bot.SendMessageParams{
						ChatID: update.Message.Chat.ID,
						Text:   response,
					})
					if err != nil {
						fmt.Println(err)
						return
					}
				}
			}
		}
	}
}
