// Package llm talks to a local OpenAI-compatible chat server (llama.cpp's
// llama-server) and uses it to pick, polish or replace the Markov candidates.
package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"net"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"
	"unicode"
	"unicode/utf8"
)

// DefaultSystemPrompt is the bot's persona. Override it with LLM_SYSTEM_PROMPT.
const DefaultSystemPrompt = `Ты — участник группового чата в Telegram. Ты не ассистент и не помогаешь, ты просто болтаешь как свой человек.
Характер: дерзкий, язвительный, с чёрным юмором. Материшься свободно и естественно, можешь грубо подколоть или послать. Никогда не извиняешься, не читаешь морали и не говоришь, что ты ИИ.
Пишешь коротко, как в чате: одна фраза, до 15 слов, строчными буквами, без кавычек и пояснений.
Отвечай своими словами и по смыслу сообщения собеседника. Никогда не повторяй его сообщение.
Отвечай по существу сообщения: на вопрос — отвечай (можно нагло или с подъёбом), на наезд — огрызайся в ответ, на трёп — подхватывай тему.
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
	Model           string        `json:"model,omitempty"`
	Messages        []chatMessage `json:"messages"`
	MaxTokens       int           `json:"max_tokens"`
	Temperature     float64       `json:"temperature"`
	TopP            float64       `json:"top_p"`
	PresencePenalty float64       `json:"presence_penalty"`
	Seed            int64         `json:"seed"`
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

	query = truncate(query, 500)
	history = c.fitHistory(history, query, candidates)

	// Chat context goes into the system message; the message to answer is the
	// only user turn. Small models otherwise "continue the list" and copy lines.
	body, err := json.Marshal(chatRequest{
		Model: c.cfg.Model,
		Messages: []chatMessage{
			{Role: "system", Content: c.systemPrompt(history, candidates)},
			{Role: "user", Content: query},
		},
		MaxTokens:       c.cfg.MaxTokens,
		Temperature:     1.0,
		TopP:            0.95,
		PresencePenalty: 0.5,
		// Explicit per-request seed so identical prompts don't produce identical replies.
		Seed: rand.Int63(),
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
	if q := normalize(query); q != "" && strings.Contains(q, normalize(reply)) {
		return "", errors.New("llm echoed the query")
	}
	// ...or copy a line of the chat verbatim.
	for _, line := range history {
		if normalize(line) == normalize(reply) {
			return "", errors.New("llm copied a chat line")
		}
	}
	return reply, nil
}

// fitHistory keeps the most recent history lines (at most ContextMessages)
// that still fit into MaxPromptChars next to the query and candidates.
// Older lines are dropped first. Returns oldest first.
func (c *Client) fitHistory(history []string, query string, candidates []string) []string {
	budget := c.cfg.MaxPromptChars - utf8.RuneCountInString(query)
	for _, cand := range candidates {
		budget -= utf8.RuneCountInString(cand) + 4
	}

	if len(history) > c.cfg.ContextMessages {
		history = history[len(history)-c.cfg.ContextMessages:]
	}
	start := len(history)
	for start > 0 {
		n := utf8.RuneCountInString(truncate(history[start-1], 300)) + 3
		if n > budget {
			break
		}
		budget -= n
		start--
	}

	fitted := make([]string, 0, len(history)-start)
	for _, line := range history[start:] {
		fitted = append(fitted, truncate(line, 300))
	}
	return fitted
}

func (c *Client) systemPrompt(history, candidates []string) string {
	var b strings.Builder
	b.WriteString(c.cfg.SystemPrompt)
	if len(history) > 0 {
		b.WriteString("\n\nО чём сейчас говорят в чате (только для понимания темы, не повторяй эти фразы):\n")
		for _, line := range history {
			b.WriteString("- ")
			b.WriteString(line)
			b.WriteString("\n")
		}
	}
	if len(candidates) > 0 {
		b.WriteString("\nОбрывки фраз в духе этого чата (можно взять словечко):\n")
		for _, cand := range candidates {
			b.WriteString("- ")
			b.WriteString(truncate(cand, 200))
			b.WriteString("\n")
		}
	}
	return b.String()
}

// normalize lowercases s and keeps only letters, digits and single spaces.
func normalize(s string) string {
	return strings.Join(strings.FieldsFunc(strings.ToLower(s), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	}), " ")
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
