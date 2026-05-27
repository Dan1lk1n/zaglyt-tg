package handlers

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"zaglyt-tg/models"
	"zaglyt-tg/modules/helpers"
	"zaglyt-tg/modules/messages"
	"zaglyt-tg/modules/z3abp"

	"github.com/go-telegram/bot"
	goTelegramModels "github.com/go-telegram/bot/models"
)

func (h *Handler) MessageHandler(ctx context.Context, b *bot.Bot, update *goTelegramModels.Update) {
	if update.Message != nil {
		if update.Message.Chat.Type == "channel" {
			return
		}

		var channel *models.Channel
		if update.Message.Chat.Type != "private" {
			var err error
			channel, err = h.app.GetChatByID(ctx, update.Message.Chat.ID)
			if err != nil {
				fmt.Println(err)
				return
			}
		}

		if update.Message.Chat.Type == "private" || channel.Enabled {
			if update.Message.Text != "" {
				var fileName string
				if update.Message.Chat.Type == "private" {
					fileName = "dm"
				} else {
					fileName = strconv.FormatInt(channel.ChannelID, 10)
				}

				err := messages.Append(fileName, update.Message.Text)
				if err != nil {
					fmt.Println(err)
					return
				}

				if !helpers.ShouldRespond(ctx, b, update, h.bot.Username) {
					return
				}

				db, err := messages.Read(fileName)
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
					ReplyParameters: &goTelegramModels.ReplyParameters{
						MessageID: update.Message.ID,
					},
				})
				if err != nil {
					fmt.Println(err)
					return
				}
			}
		}
	}
}
