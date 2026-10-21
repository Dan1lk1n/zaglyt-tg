package handlers

import (
	"context"
	"errors"
	"log/slog"
	"time"
	"zaglyt-tg/modules/helpers"
	"zaglyt-tg/modules/llm"
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

	// Show "typing…" up front: an LLM call can take a few seconds.
	startedAt := time.Now()
	_, err = b.SendChatAction(ctx, &bot.SendChatActionParams{
		ChatID: update.Message.Chat.ID,
		Action: goTelegramModels.ChatActionTyping,
	})
	if err != nil {
		slog.Error("send chat action", "chat_id", update.Message.Chat.ID, "err", err)
	}

	response := h.composeResponse(ctx, update.Message.Text, history, genCfg)
	if response == "" {
		slog.Debug("no response generated", "chat_id", update.Message.Chat.ID)
		return
	}

	typingDuration := time.Duration(len(response)) * 15 * time.Millisecond

	if typingDuration > 5*time.Second {
		typingDuration = 5 * time.Second
	}
	if typingDuration < 500*time.Millisecond {
		typingDuration = 500 * time.Millisecond
	}

	select {
	case <-time.After(typingDuration - time.Since(startedAt)):
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

// llmCandidates is how many Markov variants the LLM gets to choose from.
const llmCandidates = 3

// composeResponse builds a reply: Markov candidates first, then the LLM picks,
// polishes or replaces them. Any LLM failure falls back to the Markov reply.
func (h *Handler) composeResponse(ctx context.Context, query string, history []string, genCfg z3abp.Config) string {
	attempts := 1
	if h.llm != nil {
		attempts = llmCandidates
	}

	var candidates []string
	seen := make(map[string]bool)
	for range attempts {
		cand, err := z3abp.GenerateBestResponse(query, history, genCfg)
		if err != nil || seen[cand] {
			continue
		}
		seen[cand] = true
		candidates = append(candidates, cand)
	}

	if h.llm == nil {
		if len(candidates) == 0 {
			return ""
		}
		return candidates[0]
	}

	// The triggering message was just appended; it's passed separately as query.
	recent := history
	if n := len(recent); n > 0 && recent[n-1] == query {
		recent = recent[:n-1]
	}

	reply, err := h.llm.Compose(ctx, recent, query, candidates)
	if err != nil {
		if !errors.Is(err, llm.ErrUnavailable) {
			slog.Warn("llm compose failed, using markov reply", "err", err)
		}
		if len(candidates) == 0 {
			return ""
		}
		return candidates[0]
	}
	return reply
}
