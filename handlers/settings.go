// per-chat generation settings commands
package handlers

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"zaglyt-tg/modules/helpers"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

// maxGenWordsCap bounds the configurable response length. It protects the
// generator (a very large max would make random walks slow and rambling) while
// leaving plenty of room for normal use.
const maxGenWordsCap = 50

// reply is a small helper for the common "answer as a reply to the triggering
// message" pattern used across the settings commands.
func reply(ctx context.Context, b *bot.Bot, update *models.Update, text string) {
	_, _ = b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   text,
		ReplyParameters: &models.ReplyParameters{
			MessageID: update.Message.ID,
		},
	})
}

// ensureGroupAdmin enforces the same access rules as the switcher/clear
// commands: group chats only, and only chat administrators. It sends the
// appropriate rejection message and returns false when the caller must stop.
func (h *Handler) ensureGroupAdmin(ctx context.Context, b *bot.Bot, update *models.Update) bool {
	if update.Message == nil {
		return false
	}

	if update.Message.Chat.Type == "channel" {
		return false
	}

	if update.Message.Chat.Type == "private" {
		reply(ctx, b, update, "Команда только для чатов.")
		return false
	}

	isAdmin, err := helpers.IsUserAdmin(ctx, b, update.Message.Chat.ID, update.Message.From.ID, string(update.Message.Chat.Type))
	if err != nil {
		slog.Error("check admin", "chat_id", update.Message.Chat.ID, "user_id", update.Message.From.ID, "err", err)
		return false
	}

	if !isAdmin {
		reply(ctx, b, update, "❌ Команда доступна только для администраторов")
		return false
	}

	return true
}

// SettingsCommandHandler shows the current per-chat generation settings.
func (h *Handler) SettingsCommandHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	if !h.ensureGroupAdmin(ctx, b, update) {
		return
	}

	channel, err := h.app.GetChatByID(ctx, update.Message.Chat.ID)
	if err != nil {
		slog.Error("get chat for settings", "chat_id", update.Message.Chat.ID, "err", err)
		return
	}

	text := fmt.Sprintf(
		"Текущие настройки чата:\n\n"+
			"• Слов в ответе: %d–%d\n"+
			"• Шанс случайного ответа: %d%%\n\n"+
			"Изменить:\n"+
			"/setwords <мин> <макс>\n"+
			"/setprob <процент>",
		channel.MinGenWords, channel.MaxGenWords, channel.ReplyProbability,
	)
	reply(ctx, b, update, text)
}

// SetWordsCommandHandler sets the min/max word count for generated responses.
func (h *Handler) SetWordsCommandHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	if !h.ensureGroupAdmin(ctx, b, update) {
		return
	}

	fields := strings.Fields(update.Message.Text)
	if len(fields) != 3 {
		reply(ctx, b, update, "Использование: /setwords <мин> <макс>\nНапример: /setwords 3 15")
		return
	}

	minWords, errMin := strconv.Atoi(fields[1])
	maxWords, errMax := strconv.Atoi(fields[2])
	if errMin != nil || errMax != nil {
		reply(ctx, b, update, "Мин и макс должны быть числами. Например: /setwords 3 15")
		return
	}

	if minWords < 1 || minWords > maxWords || maxWords > maxGenWordsCap {
		reply(ctx, b, update, fmt.Sprintf("Некорректный диапазон. Условия: 1 ≤ мин ≤ макс ≤ %d.", maxGenWordsCap))
		return
	}

	channel, err := h.app.SetChatWords(ctx, update.Message.Chat.ID, minWords, maxWords)
	if err != nil {
		slog.Error("set chat words", "chat_id", update.Message.Chat.ID, "err", err)
		reply(ctx, b, update, "Не удалось сохранить настройку.")
		return
	}

	reply(ctx, b, update, fmt.Sprintf("✅ Слов в ответе: %d–%d", channel.MinGenWords, channel.MaxGenWords))
}

// SetProbabilityCommandHandler sets the random-reply chance (percent, 0..100).
func (h *Handler) SetProbabilityCommandHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	if !h.ensureGroupAdmin(ctx, b, update) {
		return
	}

	fields := strings.Fields(update.Message.Text)
	if len(fields) != 2 {
		reply(ctx, b, update, "Использование: /setprob <процент>\nНапример: /setprob 25")
		return
	}

	pct, err := strconv.Atoi(fields[1])
	if err != nil {
		reply(ctx, b, update, "Процент должен быть числом. Например: /setprob 25")
		return
	}

	if pct < 0 || pct > 100 {
		reply(ctx, b, update, "Процент должен быть в диапазоне 0–100.")
		return
	}

	channel, err := h.app.SetChatReplyProbability(ctx, update.Message.Chat.ID, pct)
	if err != nil {
		slog.Error("set chat reply probability", "chat_id", update.Message.Chat.ID, "err", err)
		reply(ctx, b, update, "Не удалось сохранить настройку.")
		return
	}

	reply(ctx, b, update, fmt.Sprintf("✅ Шанс случайного ответа: %d%%", channel.ReplyProbability))
}
