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

		keyboard := helpers.GetClearKeyboard()

		_, err = b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:      update.Message.Chat.ID,
			Text:        "Если вы нажмёте на кнопку, то память бота сбросится. Это безвозвратное действие.\n\nВы уверены?",
			ReplyMarkup: keyboard,
			ReplyParameters: &models.ReplyParameters{
				MessageID: update.Message.ID,
			},
		})
	}
}

func (h *Handler) CallbackClear(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.CallbackQuery.Message.Message == nil {
		return
	}

	chatID := update.CallbackQuery.Message.Message.Chat.ID
	messageID := update.CallbackQuery.Message.Message.ID
	chatType := string(update.CallbackQuery.Message.Message.Chat.Type)
	userID := update.CallbackQuery.From.ID

	isAdmin, err := helpers.IsUserAdmin(ctx, b, chatID, userID, chatType)
	if err != nil {
		fmt.Println(err)
		return
	}

	if !isAdmin {
		_, _ = b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
			CallbackQueryID: update.CallbackQuery.ID,
			Text:            "❌ Действие доступно только для администраторов чата",
			ShowAlert:       true,
		})
		return
	}

	channel, err := h.app.GetChatByID(ctx, chatID)
	if err != nil {
		fmt.Println(err)
		return
	}

	err = messages.Clear(strconv.FormatInt(channel.ChannelID, 10))
	if err != nil {
		fmt.Println(err)
		return
	}

	_, _ = b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
		CallbackQueryID: update.CallbackQuery.ID,
		Text:            "🗑 БД сообщений очищена!",
	})

	_, _ = b.EditMessageText(ctx, &bot.EditMessageTextParams{
		Text:      "🗑 БД сообщений очищена!",
		ChatID:    chatID,
		MessageID: messageID,
	})
}
