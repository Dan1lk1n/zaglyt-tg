package handlers

import (
	"context"
	"log/slog"
	"zaglyt-tg/modules/helpers"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

func (h *Handler) GenerateAnecdoteCommandHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}

	// The bot only operates in group chats.
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

	channel, err := h.app.GetChatByID(ctx, update.Message.Chat.ID)
	if err != nil {
		slog.Error("get chat for anecdote", "chat_id", update.Message.Chat.ID, "err", err)
		return
	}

	dataset, err := h.app.GetMessages(ctx, channel.ChannelID)
	if err != nil {
		slog.Error("read messages for anecdote", "channel_id", channel.ChannelID, "err", err)
		return
	}

	if len(dataset) == 0 {
		_, _ = b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   `История сообщений этого чата пуста.`,
			ReplyParameters: &models.ReplyParameters{
				MessageID: update.Message.ID,
			},
		})
		return
	}

	anecdote, err := helpers.GenerateAnecdote(dataset)
	if err != nil {
		_, _ = b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   `Не удалось сгенерировать анекдот.`,
			ReplyParameters: &models.ReplyParameters{
				MessageID: update.Message.ID,
			},
		})

		return
	}

	_, _ = b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   anecdote,
		ReplyParameters: &models.ReplyParameters{
			MessageID: update.Message.ID,
		},
	})
}
