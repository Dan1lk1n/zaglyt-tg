package helpers

import (
	"context"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

func IsUserAdmin(ctx context.Context, b *bot.Bot, chatID int64, userID int64, chatType string) (bool, error) {
	if chatType == "private" {
		return true, nil
	}

	member, err := b.GetChatMember(ctx, &bot.GetChatMemberParams{
		ChatID: chatID,
		UserID: userID,
	})
	if err != nil {
		return false, err
	}

	return isAdminMember(member.Type), nil
}

// isAdminMember reports whether a chat-member type grants admin rights.
// ChatMember is a discriminated union — only the pointer matching member.Type
// is non-nil — so the decision must be made from Type, not by reading a variant.
func isAdminMember(t models.ChatMemberType) bool {
	switch t {
	case models.ChatMemberTypeOwner, models.ChatMemberTypeAdministrator:
		return true
	default:
		return false
	}
}
