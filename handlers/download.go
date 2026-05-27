package handlers

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"zaglyt-tg/modules/helpers"
	"zaglyt-tg/modules/messages"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

func (h *Handler) DownloadCommandHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
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
	}

	file, err := messages.Open(strconv.FormatInt(update.Message.Chat.ID, 10))
	if err != nil {
		if os.IsNotExist(err) {
			_, _ = b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: update.Message.Chat.ID,
				Text:   `История сообщений этого чата пуста.`,
				ReplyParameters: &models.ReplyParameters{
					MessageID: update.Message.ID,
				},
			})
			return
		}

		fmt.Println(err)
		return
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		fmt.Println(err)
		return
	}

	if fileInfo.Size() == 0 {
		_, _ = b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   `История сообщений пуста.`,
			ReplyParameters: &models.ReplyParameters{
				MessageID: update.Message.ID,
			},
		})
		return
	}

	_, err = b.SendDocument(ctx, &bot.SendDocumentParams{
		ChatID: update.Message.Chat.ID,
		Document: &models.InputFileUpload{
			Filename: fmt.Sprintf("history_%d.txt", update.Message.Chat.ID),
			Data:     file,
		},
		Caption: "📥 Файл с сообщениями чата",
		ReplyParameters: &models.ReplyParameters{
			MessageID: update.Message.ID,
		},
	})
	if err != nil {
		fmt.Println(err)
	}

}
