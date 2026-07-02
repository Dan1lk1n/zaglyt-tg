package handlers

import (
	"context"
	"log/slog"
	"time"
	"zaglyt-tg/modules/helpers"
	"zaglyt-tg/modules/z3abp"

	"github.com/go-telegram/bot"
	goTelegramModels "github.com/go-telegram/bot/models"
)

func (h *Handler) MessageHandler(ctx context.Context, b *bot.Bot, update *goTelegramModels.Update) {
	if update.Message == nil {
		return
	}

	// The bot only operates in group chats: channels and private messages are
	// ignored entirely (no storing, no responding).
	if update.Message.Chat.Type == "channel" || update.Message.Chat.Type == "private" {
		return
	}

	if update.Message.Text == "" {
		return
	}

	channel, err := h.app.GetChatByID(ctx, update.Message.Chat.ID)
	if err != nil {
		slog.Error("get chat for message", "chat_id", update.Message.Chat.ID, "err", err)
		return
	}

	if !channel.Enabled {
		return
	}

	if err := h.app.AppendMessage(ctx, channel.ChannelID, update.Message.Text); err != nil {
		slog.Error("append message", "channel_id", channel.ChannelID, "err", err)
		return
	}

	if !helpers.ShouldRespond(ctx, b, update, h.bot.Username, h.cfg.BotName, channel.ReplyProbability) {
		return
	}

	history, err := h.app.GetMessages(ctx, channel.ChannelID)
	if err != nil {
		slog.Error("read messages", "channel_id", channel.ChannelID, "err", err)
		return
	}

	genCfg := z3abp.DefaultConfig()
	genCfg.MinGenWords = channel.MinGenWords
	genCfg.MaxGenWords = channel.MaxGenWords

	response, err := z3abp.GenerateBestResponse(update.Message.Text, history, genCfg)
	if err != nil {
		slog.Debug("no response generated", "chat_id", update.Message.Chat.ID, "err", err)
		return
	}

	typingDuration := time.Duration(len(response)) * 15 * time.Millisecond

	if typingDuration > 5*time.Second {
		typingDuration = 5 * time.Second
	}
	if typingDuration < 500*time.Millisecond {
		typingDuration = 500 * time.Millisecond
	}

	_, err = b.SendChatAction(ctx, &bot.SendChatActionParams{
		ChatID: update.Message.Chat.ID,
		Action: goTelegramModels.ChatActionTyping,
	})
	if err != nil {
		slog.Error("send chat action", "chat_id", update.Message.Chat.ID, "err", err)
	}

	select {
	case <-time.After(typingDuration):
	case <-ctx.Done():
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
		slog.Error("send message", "chat_id", update.Message.Chat.ID, "err", err)
		return
	}
}
