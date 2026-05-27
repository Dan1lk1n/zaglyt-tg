package handlers

import (
	"context"
	"fmt"
	"strconv"
	"zaglyt-tg/modules/helpers"
	"zaglyt-tg/modules/messages"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

func (h *Handler) ClearCommandHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message != nil {
		if update.Message.Chat.Type == "channel" {
			return
		}

		if update.Message.Chat.Type == "private" {
			_, _ = b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: update.Message.Chat.ID,
				Text:   `Команда только для чатов.`,
				ReplyParameters: &models.ReplyParameters{
					MessageID: update.Message.ID,
				},
			})

			return
		}

		isAdmin, err := helpers.IsUserAdmin(ctx, b, update.Message.Chat.ID, update.Message.From.ID, string(update.Message.Chat.Type))
		if err != nil {
			fmt.Println(err)
			return
		}

		if !isAdmin {
			_, _ = b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: update.Message.Chat.ID,
				Text:   `❌ Команда доступна только для администраторов`,
				ReplyParameters: &models.ReplyParameters{
					MessageID: update.Message.ID,
				},
			})

			return
		}

		channel, err := h.app.GetChatByID(ctx, update.Message.Chat.ID)
		if err != nil {
			fmt.Println(err)
			return
		}

		err = messages.Clear(strconv.FormatInt(channel.ChannelID, 10))
		if err != nil {
			fmt.Println(err)
			return
		}

		_, err = b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "🗑 БД сообщений очищена!",
			ReplyParameters: &models.ReplyParameters{
				MessageID: update.Message.ID,
			},
		})
	}
}
