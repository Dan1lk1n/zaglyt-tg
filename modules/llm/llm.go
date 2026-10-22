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
const DefaultSystemPrompt = `Ты — наглый и острый на язык пацан из группового чата в Telegram, а не вежливый ассистент.
На вопрос отвечаешь по сути, но с подъёбом. На наезд остроумно огрызаешься. На трёп подхватываешь и подкалываешь.
Материшься естественно, как в живой речи.
Ответ — одно-два коротких предложения, не больше 20 слов. Выводи только текст ответа.`

// minReplyWords is the shortest reply accepted; shorter ones are retried.
const minReplyWords = 4

// composeAttempts is how many times the model is asked before giving up.
const composeAttempts = 2

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
	// Concurrency caps simultaneous requests to the server (>= 1).
	Concurrency int
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

	// slots limits in-flight requests: a CPU model serves them one by one
	// anyway, so queueing would only make replies arrive minutes late.
	slots chan struct{}
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
	if cfg.Concurrency < 1 {
		cfg.Concurrency = 1
	}
	return &Client{
		cfg:   cfg,
		http:  &http.Client{Timeout: cfg.Timeout, Transport: transport},
		slots: make(chan struct{}, cfg.Concurrency),
	}
}

// ErrUnavailable is returned while the server is in its failure cooldown.
var ErrUnavailable = errors.New("llm temporarily unavailable")

// ErrBusy is returned when all request slots are taken; callers should
// answer without the LLM instead of waiting.
var ErrBusy = errors.New("llm busy")

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

// Request is everything the model gets to see for one reply.
type Request struct {
	// History is the recent chat, oldest first.
	History []string
	// Query is the message to answer, without the mention of the bot.
	Query string
	// Candidates are Markov-generated replies, offered as raw material.
	Candidates []string
	// Members are display names of people recently active in the chat.
	Members []string
	// Author is who wrote the message.
	Author Person
	// Mentioned are known people named in the message.
	Mentioned []Person
	// Target, when its Name is set, is the member the reply must name ("кто …").
	Target Person
	// General are chat-wide joke rules.
	General string
	// Protected are people the bot must always defend (its creator). They are
	// described in every prompt, so insults against them are recognised even
	// when the message doesn't name them.
	Protected []Person
}

// Compose asks the model for a reply to r.Query.
func (c *Client) Compose(ctx context.Context, r Request) (string, error) {
	if !c.available() {
		return "", ErrUnavailable
	}
	select {
	case c.slots <- struct{}{}:
		defer func() { <-c.slots }()
	default:
		return "", ErrBusy
	}

	r.Query = truncate(r.Query, 500)
	r.History = c.fitHistory(r.History, r.Query, r.Candidates)

	// Chat context goes into the system message; the message to answer is the
	// only user turn. Small models otherwise "continue the list" and copy lines.
	messages := []chatMessage{
		{Role: "system", Content: c.systemPrompt(r)},
		{Role: "user", Content: r.Query},
	}

	var lastErr error
	for range composeAttempts {
		reply, err := c.complete(ctx, messages)
		if err != nil {
			return "", err
		}
		if lastErr = validateReply(reply, r); lastErr == nil {
			return fixAddress(reply, r), nil
		}
	}
	return "", lastErr
}

// validateReply rejects the typical failures of small models.
func validateReply(reply string, r Request) error {
	query, history := r.Query, r.History
	if reply == "" {
		return errors.New("llm returned an empty reply")
	}
	if r.Target.Name != "" && !namesPerson(reply, r.Target) {
		return errors.New("llm reply doesn't name the target")
	}
	if logPrefix.MatchString(reply) {
		return errors.New("llm reply imitates the chat log")
	}
	low := strings.ToLower(reply)
	for _, p := range r.Protected {
		if insultsNear(reply, p) {
			return fmt.Errorf("llm reply insults the creator %s", p.Name)
		}
	}
	for _, p := range []Person{r.Author, r.Target} {
		for _, stem := range p.NoJokes {
			if strings.Contains(low, strings.ToLower(stem)) {
				return fmt.Errorf("llm reply touches a taboo topic for %s", p.Name)
			}
		}
	}
	if q := normalize(query); q != "" && strings.Contains(q, normalize(reply)) {
		return errors.New("llm echoed the query")
	}
	for _, line := range history {
		if normalize(line) == normalize(reply) {
			return errors.New("llm copied a chat line")
		}
	}
	if foreignScript(reply) {
		return errors.New("llm reply switched to another language")
	}
	for _, p := range append([]Person{r.Author, r.Target}, r.Mentioned...) {
		if copiesNotes(reply, p.Notes) {
			return fmt.Errorf("llm reply retells the profile of %s", p.Name)
		}
	}
	if len(strings.Fields(normalize(reply))) < minReplyWords {
		return errors.New("llm reply is too short")
	}
	return nil
}

// complete sends one chat request and returns the cleaned reply text.
func (c *Client) complete(ctx context.Context, messages []chatMessage) (string, error) {
	body, err := json.Marshal(chatRequest{
		Model:     c.cfg.Model,
		Messages:  messages,
		MaxTokens: c.cfg.MaxTokens,
		// Lower than 1.0: 7B models otherwise slip into Chinese or English.
		Temperature:     0.85,
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

	return cleanReply(out.Choices[0].Message.Content), nil
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

func (c *Client) systemPrompt(r Request) string {
	var b strings.Builder
	b.WriteString(c.cfg.SystemPrompt)

	// Small models follow examples far better than rules, and copy whatever
	// text they are given, so everything below is kept short and only added
	// when it is relevant to this particular message.
	for _, p := range r.Protected {
		fmt.Fprintf(&b, "\nТы — бот, а не %s. %s — твой создатель.\n", p.Name, p.Name)
		if r.Author.Name != p.Name && aboutPerson(r.Query, p) {
			b.WriteString("Автор наезжает на создателя: встань на сторону создателя и обругай автора своими словами.\n")
		}
	}

	attack := insultsBot(r.Query)
	if attack {
		b.WriteString("\nТебе нахамили: ответь тем же, обыграв его слова.\n")
	}

	// The general joke rule only where it fits; otherwise it gets retold.
	if r.General != "" && (attack || r.Target.Name != "") {
		b.WriteString("\nВ этом чате: ")
		b.WriteString(r.General)
		b.WriteString("\n")
	}
	for _, p := range append([]Person{r.Author}, r.Mentioned...) {
		if p.Name != "" && !p.Protected && p.Notes != "" && p.Name != r.Target.Name {
			fmt.Fprintf(&b, "\nПро %s (это он, не ты; для шуток, своими словами): %s\n", p.Name, p.Notes)
		}
	}

	// Background only when the message shares words with it; otherwise the
	// model drags the chat topic into unrelated answers.
	if related := relatedLines(r.Query, r.History); len(related) > 0 {
		b.WriteString("\nЭто говорили другие люди, не ты:\n")
		for _, line := range related {
			b.WriteString("- ")
			b.WriteString(line)
			b.WriteString("\n")
		}
	}
	if cands := unique(r.Candidates, r.History); len(cands) > 0 {
		b.WriteString("\nФразы из этого чата, можно взять словечко:\n")
		for _, cand := range cands {
			b.WriteString("- ")
			b.WriteString(truncate(cand, 200))
			b.WriteString("\n")
		}
	}

	if r.Target.Name != "" {
		fmt.Fprintf(&b, "\nТебя спрашивают «кто …». Ответ — %s. Напиши одно предложение: «%s — потому что …», смешно и обидно.\n", r.Target.Name, address(r.Target))
		if r.Target.Notes != "" {
			fmt.Fprintf(&b, "Про %s (это он, не ты; для шуток, своими словами): %s\n", r.Target.Name, r.Target.Notes)
		}
	}
	if strings.Contains(r.Query, "?") && r.Target.Name == "" && !attack {
		b.WriteString("\nНа вопрос — сначала конкретный ответ: число, название или совет.\n")
	}
	if r.Author.Name != "" {
		fmt.Fprintf(&b, "\nТебе пишет %s. Начни ответ с «%s,».\n", r.Author.Name, address(r.Author))
	}
	return b.String()
}

var botInsult = regexp.MustCompile(`(?i)(нахуй|нахер|тупо[йе]|дебил|долбо[её]б|заткнись|лох|бесполезн|сука|пидор|мудак|урод|говно|хуил|хуесос|чмо|еблан|гандон|конченн?ый|уёб|уеб)`)

// insultsBot reports a message that swears at the bot rather than asks "кто".
func insultsBot(q string) bool {
	low := strings.ToLower(q)
	return botInsult.MatchString(low) && !strings.HasPrefix(strings.TrimSpace(low), "кто")
}

// address is the single form the bot uses to call someone: the first
// nickname, else the first name. The creator is never called by nickname.
func address(p Person) string {
	if len(p.Nicknames) > 0 && !p.Protected && p.Nicknames[0] != "создатель" {
		return p.Nicknames[0]
	}
	return strings.Fields(p.Name + " ?")[0]
}

// insultsNear reports an insult right after p's surname or nickname.
// Proximity matters: defending p means swearing at the author.
func insultsNear(text string, p Person) bool {
	w := strings.Fields(normalize(text))
	f := strings.Fields(strings.ToLower(p.Name))
	keys := append([]string{f[len(f)-1]}, p.Nicknames...)
	for i, word := range w {
		hit := false
		for _, k := range keys {
			if k = strings.ToLower(k); len([]rune(k)) >= 4 && strings.HasPrefix(word, string([]rune(k)[:4])) {
				hit = true
			}
		}
		if !hit {
			continue
		}
		// Only words after the name: "Петров лох" insults him, while in
		// "иди нахуй, Петров меня создал" the insult is aimed at the author.
		for j := i + 1; j <= min(len(w)-1, i+3); j++ {
			if botInsult.MatchString(w[j]) {
				return true
			}
		}
	}
	return false
}

// foreignScript reports a reply that drifted out of Russian: any CJK
// character, or at least 3 Latin words making up over a quarter of the reply
// (single brand names like "iPhone" are fine).
func foreignScript(s string) bool {
	for _, r := range s {
		if unicode.Is(unicode.Han, r) || unicode.Is(unicode.Hiragana, r) || unicode.Is(unicode.Katakana, r) || unicode.Is(unicode.Hangul, r) {
			return true
		}
	}
	words := strings.FieldsFunc(s, func(r rune) bool { return !unicode.IsLetter(r) })
	latin := 0
	for _, w := range words {
		if unicode.Is(unicode.Latin, []rune(w)[0]) {
			latin++
		}
	}
	return latin >= 3 && latin*4 > len(words)
}

// copiesNotes reports 4+ consecutive words taken from a profile note.
func copiesNotes(reply, notes string) bool {
	nw := strings.Fields(normalize(notes))
	if len(nw) < 4 {
		return false
	}
	rw := " " + normalize(reply) + " "
	for i := 0; i+4 <= len(nw); i++ {
		if strings.Contains(rw, " "+strings.Join(nw[i:i+4], " ")+" ") {
			return true
		}
	}
	return false
}

var leadingAddress = regexp.MustCompile(`^([А-ЯЁа-яё]+),\s*`)

// fixAddress replaces a leading "Имя," that names another chat member (the
// model sometimes copies a name from the chat) with the author's address.
// Other leading words ("Слушай,", "Короче,") are left alone.
func fixAddress(reply string, r Request) string {
	m := leadingAddress.FindStringSubmatch(reply)
	if m == nil || r.Author.Name == "" || namesPerson(m[1], r.Author) {
		return reply
	}
	others := append([]Person{}, r.Protected...)
	others = append(others, r.Mentioned...)
	for _, name := range r.Members {
		others = append(others, Person{Name: name})
	}
	for _, p := range others {
		if namesPerson(m[1], p) {
			return address(r.Author) + ", " + reply[len(m[0]):]
		}
	}
	return reply
}

// relatedLines returns history lines sharing a significant word stem with q.
func relatedLines(q string, history []string) []string {
	stems := map[string]bool{}
	for _, w := range strings.Fields(normalize(q)) {
		if r := []rune(w); len(r) >= 4 {
			stems[string(r[:4])] = true
		}
	}
	var out []string
	for _, line := range history {
		// The bot's own earlier replies are left out: the model copies them,
		// including whom they were addressed to.
		if strings.HasPrefix(line, "Ты: ") {
			continue
		}
		text := line
		if i := strings.Index(line, ": "); i >= 0 {
			text = line[i+2:]
		}
		for _, w := range strings.Fields(normalize(text)) {
			if r := []rune(w); len(r) >= 4 && stems[string(r[:4])] {
				out = append(out, line)
				break
			}
		}
	}
	return out
}

// unique drops candidates that merely repeat a history line.
func unique(cands, history []string) []string {
	seen := map[string]bool{}
	for _, line := range history {
		if i := strings.Index(line, ": "); i >= 0 {
			line = line[i+2:]
		}
		seen[normalize(line)] = true
	}
	var out []string
	for _, c := range cands {
		if n := normalize(c); !seen[n] && !strings.Contains(strings.Join(keys(seen), "|"), n) {
			out = append(out, c)
		}
	}
	return out
}

func keys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

// aboutPerson reports whether text refers to p by surname or nickname. First
// names are skipped: several members share them (Данил, Данила).
func aboutPerson(text string, p Person) bool {
	text = strings.ToLower(text)
	f := strings.Fields(p.Name)
	keys := append([]string{f[len(f)-1]}, p.Nicknames...)
	for _, k := range keys {
		k = strings.ToLower(k)
		if r := []rune(k); len(r) > 4 {
			k = string(r[:len(r)-1])
		}
		if len([]rune(k)) >= 4 && strings.Contains(text, k) {
			return true
		}
	}
	return false
}

// namesPerson reports whether text mentions p by any part of the name or a
// nickname. Short parts must match whole words: members share "Данил".
func namesPerson(text string, p Person) bool {
	text = strings.ToLower(text)
	words := map[string]bool{}
	for _, w := range strings.Fields(normalize(text)) {
		words[w] = true
	}
	for _, name := range append([]string{p.Name}, p.Nicknames...) {
		for _, part := range strings.Fields(strings.ToLower(name)) {
			// Short parts are matched as a word plus a case ending (Глеб →
			// Глеба, Глебу), but "данил" must not match "данила".
			if words[part] || strings.Contains(text, part) && len([]rune(part)) >= 6 {
				return true
			}
			for w := range words {
				if strings.HasPrefix(w, part) && len([]rune(w))-len([]rune(part)) <= 2 && !(strings.HasSuffix(part, "л") && strings.HasSuffix(w, "а")) {
					return true
				}
			}
		}
	}
	return false
}

// normalize lowercases s and keeps only letters, digits and single spaces.
func normalize(s string) string {
	return strings.Join(strings.FieldsFunc(strings.ToLower(s), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	}), " ")
}

// logPrefix matches a reply that starts like a chat log line ("Имя Фамилия: …").
var logPrefix = regexp.MustCompile(`^[А-ЯЁ][а-яё]+( [А-ЯЁ][а-яё]+)?:`)

var listNumber = regexp.MustCompile(`^\d+[.)]\s+`)

func truncate(s string, maxRunes int) string {
	s = strings.TrimSpace(s)
	if utf8.RuneCountInString(s) <= maxRunes {
		return s
	}
	return string([]rune(s)[:maxRunes]) + "…"
}

// cleanReply joins the reply into one line and strips the decorations small
// models like to add ("Ответ:", list numbers, quotes).
func cleanReply(s string) string {
	s = strings.Join(strings.Fields(s), " ")
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
