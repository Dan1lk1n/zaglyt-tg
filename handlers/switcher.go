package handlers

import (
	"context"
	"log/slog"
	"zaglyt-tg/modules/helpers"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

func (h *Handler) SwitcherCommandHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
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
			slog.Error("check admin", "chat_id", update.Message.Chat.ID, "user_id", update.Message.From.ID, "err", err)
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
			slog.Error("get chat for switcher", "chat_id", update.Message.Chat.ID, "err", err)
			return
		}

		keyboard := helpers.GetSwitcherKeyboard(channel.Enabled)

		_, _ = b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text: `Включите или выключите бота.

Если вы его выключите, то он перестанет писать в чат.
Если включите, то он будет писать в чат.`,
			ReplyMarkup: keyboard,
			ReplyParameters: &models.ReplyParameters{
				MessageID: update.Message.ID,
			},
		})
	}
}

func (h *Handler) CallbackBotSwitcher(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.CallbackQuery.Message.Message == nil {
		return
	}

	chatID := update.CallbackQuery.Message.Message.Chat.ID
	messageID := update.CallbackQuery.Message.Message.ID
	chatType := string(update.CallbackQuery.Message.Message.Chat.Type)
	userID := update.CallbackQuery.From.ID

	isAdmin, err := helpers.IsUserAdmin(ctx, b, chatID, userID, chatType)
	if err != nil {
		slog.Error("check admin", "chat_id", chatID, "user_id", userID, "err", err)
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

	data := update.CallbackQuery.Data
	enable := (data == "bot_enable")

	// A toggle only changes the enabled flag, so a single UPDATE suffices — no
	// preliminary read of the channel is needed. The row already exists because
	// the switcher keyboard is only produced after GetChatByID created it.
	channel, err := h.app.SetChatEnabled(ctx, chatID, enable)
	if err != nil {
		slog.Error("update chat enabled state", "chat_id", chatID, "err", err)
		return
	}

	alertText := "Бот выключен."
	if channel.Enabled {
		alertText = "Бот успешно включен!"
	}

	_, _ = b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
		CallbackQueryID: update.CallbackQuery.ID,
		Text:            alertText,
	})

	newKeyboard := helpers.GetSwitcherKeyboard(enable)

	_, _ = b.EditMessageReplyMarkup(ctx, &bot.EditMessageReplyMarkupParams{
		ChatID:      chatID,
		MessageID:   messageID,
		ReplyMarkup: newKeyboard,
	})
}
