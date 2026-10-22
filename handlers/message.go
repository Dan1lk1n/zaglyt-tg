package handlers

import (
	"context"
	"errors"
	"log/slog"
	"strings"
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

	h.members.Seen(update.Message.Chat.ID, update.Message.From)
	h.members.Said(update.Message.Chat.ID, h.members.Person(update.Message.From).Name, update.Message.Text)

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

	response := h.composeResponse(ctx, update.Message, history, genCfg)
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

	// The bot's own lines are part of the dialog too.
	h.members.Said(update.Message.Chat.ID, "Ты", response)
}

// stripBotMention removes the leading bot name and the @username from the
// message, so the LLM sees what was said rather than who it was said to.
func (h *Handler) stripBotMention(text string) string {
	if h.bot != nil && h.bot.Username != "" {
		text = strings.ReplaceAll(text, "@"+h.bot.Username, "")
	}
	text = strings.TrimSpace(text)
	if name := h.cfg.BotName; name != "" && strings.HasPrefix(strings.ToLower(text), strings.ToLower(name)) {
		text = strings.TrimLeft(text[len(strings.ToLower(name)):], " ,.!:")
	}
	if text == "" {
		return "?"
	}
	return text
}

// llmCandidates is how many Markov variants the LLM gets to choose from.
const llmCandidates = 3

// composeResponse builds a reply: Markov candidates first, then the LLM picks,
// polishes or replaces them. Any LLM failure falls back to the Markov reply.
func (h *Handler) composeResponse(ctx context.Context, msg *goTelegramModels.Message, history []string, genCfg z3abp.Config) string {
	query := msg.Text
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

	// Prefer the in-memory dialog with author names; right after a restart it
	// is empty, so fall back to the stored text-only history. Either way the
	// triggering message is last and is passed separately as the query.
	recent := h.members.Dialog(msg.Chat.ID)
	if len(recent) <= 1 {
		recent = history
	}
	if n := len(recent); n > 0 {
		recent = recent[:n-1]
	}

	stripped := h.stripBotMention(query)
	members := h.members.List(msg.Chat.ID)
	reply, err := h.llm.Compose(ctx, llm.Request{
		History:    recent,
		Query:      stripped,
		Candidates: candidates,
		Members:    names(members),
		Author:     h.members.Person(msg.From),
		Mentioned:  h.personas.Mentioned(stripped),
		Target:     pickTarget(stripped, members, h.personas),
		General:    h.personas.General,
		Protected:  h.personas.Protected(),
	})
	if err != nil {
		if !errors.Is(err, llm.ErrUnavailable) && !errors.Is(err, llm.ErrBusy) {
			slog.Warn("llm compose failed, using markov reply", "err", err)
		}
		if len(candidates) == 0 {
			return ""
		}
		return candidates[0]
	}
	return reply
}
