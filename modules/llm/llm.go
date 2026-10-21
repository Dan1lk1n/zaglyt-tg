// Package llm talks to a local OpenAI-compatible chat server (llama.cpp's
// llama-server) and uses it to pick, polish or replace the Markov candidates.
package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

// DefaultSystemPrompt is the bot's persona. Override it with LLM_SYSTEM_PROMPT.
const DefaultSystemPrompt = `Ты — участник группового чата в Telegram. Ты не ассистент и не помогаешь, ты просто болтаешь как свой человек.
Характер: дерзкий, язвительный, с чёрным юмором. Материшься свободно и естественно, можешь грубо подколоть или послать. Никогда не извиняешься, не читаешь морали и не говоришь, что ты ИИ.
Пишешь коротко, как в чате: одна фраза, до 15 слов, строчными буквами, без кавычек и пояснений.
Тебе дают последние сообщения чата, сообщение, на которое надо ответить, и варианты ответа. Если какой-то вариант смешной и в тему — выдай его, можно чуть подправить. Если варианты не подходят — ответь сам. Никогда не повторяй сообщение собеседника.
Примеры:
сообщение: бот ты тупой -> ответ: на себя посмотри, гений хуев
сообщение: кто пойдёт пиво пить -> ответ: я бы пошёл, да ты опять без денег
сообщение: как дела -> ответ: пока тебя не увидел было нормально
Выводи только текст ответа.`

// Config controls the client and how much context is sent per request.
type Config struct {
	URL          string
	Model        string
	SystemPrompt string
	Timeout      time.Duration
	MaxTokens    int
	// MaxPromptChars caps the user prompt. Older chat messages are dropped
	// first so the request always fits the server's fixed context window.
	MaxPromptChars int
	// ContextMessages is how many recent chat messages are offered as context.
	ContextMessages int
}

// unavailableCooldown is how long the LLM is skipped after it was unreachable,
// so a stopped llm container doesn't slow every reply down.
const unavailableCooldown = time.Minute

// Client is safe for concurrent use. A nil *Client means the LLM is disabled.
type Client struct {
	cfg  Config
	http *http.Client

	mu        sync.Mutex
	skipUntil time.Time
}

// New returns nil when cfg.URL is empty, so callers can treat the LLM as optional.
func New(cfg Config) *Client {
	if strings.TrimSpace(cfg.URL) == "" {
		return nil
	}
	if cfg.SystemPrompt == "" {
		cfg.SystemPrompt = DefaultSystemPrompt
	}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	// A stopped container should fail fast, not after the full timeout.
	transport.DialContext = (&net.Dialer{Timeout: 2 * time.Second}).DialContext
	return &Client{cfg: cfg, http: &http.Client{Timeout: cfg.Timeout, Transport: transport}}
}

// ErrUnavailable is returned while the server is in its failure cooldown.
var ErrUnavailable = errors.New("llm temporarily unavailable")

func (c *Client) available() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return time.Now().After(c.skipUntil)
}

func (c *Client) markUnavailable() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.skipUntil = time.Now().Add(unavailableCooldown)
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatRequest struct {
	Model       string        `json:"model,omitempty"`
	Messages    []chatMessage `json:"messages"`
	MaxTokens   int           `json:"max_tokens"`
	Temperature float64       `json:"temperature"`
	TopP        float64       `json:"top_p"`
}

type chatResponse struct {
	Choices []struct {
		Message chatMessage `json:"message"`
	} `json:"choices"`
}

// Compose asks the model for a reply to query given recent chat history and
// the Markov candidates. history is oldest first.
func (c *Client) Compose(ctx context.Context, history []string, query string, candidates []string) (string, error) {
	if !c.available() {
		return "", ErrUnavailable
	}

	prompt := c.buildPrompt(history, query, candidates)

	body, err := json.Marshal(chatRequest{
		Model: c.cfg.Model,
		Messages: []chatMessage{
			{Role: "system", Content: c.cfg.SystemPrompt},
			{Role: "user", Content: prompt},
		},
		MaxTokens:   c.cfg.MaxTokens,
		Temperature: 0.9,
		TopP:        0.95,
	})
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(c.cfg.URL, "/")+"/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		if ctx.Err() == nil {
			c.markUnavailable()
		}
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// 503 means the model is still loading.
		if resp.StatusCode == http.StatusServiceUnavailable {
			c.markUnavailable()
		}
		return "", fmt.Errorf("llm server returned %s", resp.Status)
	}

	var out chatResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", err
	}
	if len(out.Choices) == 0 {
		return "", errors.New("llm returned no choices")
	}

	reply := cleanReply(out.Choices[0].Message.Content)
	if reply == "" {
		return "", errors.New("llm returned an empty reply")
	}
	// Tiny models sometimes just parrot the message back.
	if strings.EqualFold(reply, strings.TrimSpace(query)) {
		return "", errors.New("llm echoed the query")
	}
	return reply, nil
}

// buildPrompt fits the request into MaxPromptChars: the query and candidates
// always go in, then as many of the most recent history lines as still fit.
func (c *Client) buildPrompt(history []string, query string, candidates []string) string {
	var tail strings.Builder
	tail.WriteString("\nОтветь на сообщение: ")
	tail.WriteString(truncate(query, 500))
	tail.WriteString("\n")
	if len(candidates) > 0 {
		tail.WriteString("Варианты:\n")
		for i, cand := range candidates {
			fmt.Fprintf(&tail, "%d. %s\n", i+1, truncate(cand, 200))
		}
	}
	tail.WriteString("Твой ответ:")

	budget := c.cfg.MaxPromptChars - utf8.RuneCountInString(tail.String())

	if len(history) > c.cfg.ContextMessages {
		history = history[len(history)-c.cfg.ContextMessages:]
	}
	// Walk from newest to oldest and stop once the budget is spent.
	var lines []string
	for i := len(history) - 1; i >= 0; i-- {
		line := "- " + truncate(history[i], 300)
		n := utf8.RuneCountInString(line) + 1
		if n > budget {
			break
		}
		budget -= n
		lines = append(lines, line)
	}

	var b strings.Builder
	if len(lines) > 0 {
		b.WriteString("Последние сообщения чата:\n")
		for i := len(lines) - 1; i >= 0; i-- {
			b.WriteString(lines[i])
			b.WriteString("\n")
		}
	}
	b.WriteString(tail.String())
	return b.String()
}

var listNumber = regexp.MustCompile(`^\d+[.)]\s+`)

func truncate(s string, maxRunes int) string {
	s = strings.TrimSpace(s)
	if utf8.RuneCountInString(s) <= maxRunes {
		return s
	}
	return string([]rune(s)[:maxRunes]) + "…"
}

// cleanReply keeps only the first line and strips the decorations small
// models like to add ("Ответ:", list numbers, quotes).
func cleanReply(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		s = s[:i]
	}
	if i := strings.Index(s, "ответ:"); i >= 0 {
		s = s[i+len("ответ:"):]
	}
	for _, prefix := range []string{"Твой ответ:", "Ответ:"} {
		s = strings.TrimPrefix(s, prefix)
	}
	s = strings.TrimSpace(s)
	s = listNumber.ReplaceAllString(s, "")
	s = strings.Trim(s, "\"«»'` ")
	return s
}
