// bot membership guard: only developers may add the bot to chats
package handlers

import (
	"context"
	"log/slog"
	"zaglyt-tg/modules/helpers"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

// MyChatMemberHandler enforces that only authorized users (the DEVELOPERS list)
// may add the bot to a group. When someone else adds it, the bot leaves the chat
// immediately. Chats the bot is already in are unaffected — this reacts only to
// new "add" events (my_chat_member updates).
func (h *Handler) MyChatMemberHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	cm := update.MyChatMember
	if cm == nil {
		return
	}

	// Only groups/supergroups can be joined and left.
	if cm.Chat.Type != "group" && cm.Chat.Type != "supergroup" {
		return
	}

	// React only to the transition "not in chat" -> "in chat" (an add), not to
	// promotions/demotions or removals.
	if isPresentMember(cm.OldChatMember.Type) || !isPresentMember(cm.NewChatMember.Type) {
		return
	}

	if helpers.IsUserDeveloper(cm.From.ID, h.cfg.Developers) {
		return
	}

	slog.Info("leaving unauthorized chat", "chat_id", cm.Chat.ID, "added_by", cm.From.ID)
	if _, err := b.LeaveChat(ctx, &bot.LeaveChatParams{ChatID: cm.Chat.ID}); err != nil {
		slog.Error("leave unauthorized chat", "chat_id", cm.Chat.ID, "err", err)
	}
}

// isPresentMember reports whether a chat-member type means the bot is currently
// in the chat (as opposed to having left or been banned).
func isPresentMember(t models.ChatMemberType) bool {
	switch t {
	case models.ChatMemberTypeOwner,
		models.ChatMemberTypeAdministrator,
		models.ChatMemberTypeMember,
		models.ChatMemberTypeRestricted:
		return true
	default:
		return false
	}
}
