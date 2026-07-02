package helpers

import (
	"context"
	"math/rand"
	"strings"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

// ShouldRespond decides whether the bot should react to a group message.
// botName and botUsername are injected by the caller (loaded once from config),
// so this function performs no I/O on the hot path. replyProbabilityPercent is
// the per-chat chance (0..100) of replying to an otherwise-unaddressed message.
func ShouldRespond(ctx context.Context, b *bot.Bot, update *models.Update, botUsername, botName string, replyProbabilityPercent int) bool {
	if update.Message == nil {
		return false
	}

	if update.Message.Chat.Type == "private" {
		return true
	}

	msg := update.Message
	text := msg.Text

	isReplyToBot := msg.ReplyToMessage != nil &&
		msg.ReplyToMessage.From != nil &&
		msg.ReplyToMessage.From.ID == b.ID()

	botName = strings.ToLower(botName)
	startsWithBotName := botName != "" && strings.HasPrefix(strings.ToLower(text), botName)

	includesBotUsername := botUsername != "" && strings.Contains(text, botUsername)

	isRandomChance := replyProbabilityPercent > 0 && rand.Intn(100) < replyProbabilityPercent

	return isReplyToBot || startsWithBotName || includesBotUsername || isRandomChance
}
