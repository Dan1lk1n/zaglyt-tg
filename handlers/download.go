package handlers

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"strings"
	"zaglyt-tg/modules/helpers"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

func (h *Handler) DownloadCommandHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}

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

	history, err := h.app.GetMessages(ctx, update.Message.Chat.ID)
	if err != nil {
		slog.Error("read messages for download", "chat_id", update.Message.Chat.ID, "err", err)
		return
	}

	if len(history) == 0 {
		_, _ = b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   `История сообщений этого чата пуста.`,
			ReplyParameters: &models.ReplyParameters{
				MessageID: update.Message.ID,
			},
		})
		return
	}

	content := strings.Join(history, "\n") + "\n"

	_, err = b.SendDocument(ctx, &bot.SendDocumentParams{
		ChatID: update.Message.Chat.ID,
		Document: &models.InputFileUpload{
			Filename: fmt.Sprintf("history_%d.txt", update.Message.Chat.ID),
			Data:     bytes.NewReader([]byte(content)),
		},
		Caption: "📥 Файл с сообщениями чата",
		ReplyParameters: &models.ReplyParameters{
			MessageID: update.Message.ID,
		},
	})
	if err != nil {
		slog.Error("send messages document", "chat_id", update.Message.Chat.ID, "err", err)
	}
}
