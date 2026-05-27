package helpers

import (
	"context"
	"math/rand"
	"strings"
	"zaglyt-tg/configs"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

func ShouldRespond(ctx context.Context, b *bot.Bot, update *models.Update, botUsername string) bool {
	if update.Message == nil {
		return false
	}

	if update.Message.Chat.Type == "private" {
		return true
	}

	config, err := configs.LoadConfig()
	if err != nil {
		return false
	}

	msg := update.Message
	text := msg.Text

	isReplyToBot := msg.ReplyToMessage != nil &&
		msg.ReplyToMessage.From != nil &&
		msg.ReplyToMessage.From.ID == b.ID()

	botName := strings.ToLower(config.BotName)
	startsWithBotName := botName != "" && strings.HasPrefix(strings.ToLower(text), botName)

	includesBotUsername := botUsername != "" && strings.Contains(text, botUsername)

	isRandomChance := rand.Intn(10) == 9

	return isReplyToBot || startsWithBotName || includesBotUsername || isRandomChance
}
