package handlers

import (
	"math/rand"
	"strings"
	"sync"
	"zaglyt-tg/modules/llm"

	goTelegramModels "github.com/go-telegram/bot/models"
)

// maxMembersPerChat bounds how many recent authors are remembered per chat.
const maxMembersPerChat = 30

// maxDialogLines bounds the per-chat "Name: text" dialog kept for the LLM.
const maxDialogLines = 30

// chatMembers remembers who recently wrote in each chat, most recent first.
// It lives in memory only, so it starts empty after a restart.
type chatMembers struct {
	mu       sync.Mutex
	chats    map[int64][]int64
	personas *llm.Personas
	// names holds Telegram display names of users without a persona.
	names map[int64]string
	// dialogs holds the recent conversation per chat as "Name: text", oldest
	// first, so the LLM knows who said what (the DB keeps text only).
	dialogs map[int64][]string
}

func newChatMembers(personas *llm.Personas) *chatMembers {
	return &chatMembers{
		chats:    make(map[int64][]int64),
		names:    make(map[int64]string),
		dialogs:  make(map[int64][]string),
		personas: personas,
	}
}

// Said appends a line to the chat's recent dialog. name is who said it.
func (m *chatMembers) Said(chatID int64, name, text string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	lines := append(m.dialogs[chatID], name+": "+text)
	if len(lines) > maxDialogLines {
		lines = lines[len(lines)-maxDialogLines:]
	}
	m.dialogs[chatID] = lines
}

// Dialog returns a copy of the chat's recent dialog, oldest first.
func (m *chatMembers) Dialog(chatID int64) []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]string(nil), m.dialogs[chatID]...)
}

// Seen records the author of a message.
func (m *chatMembers) Seen(chatID int64, user *goTelegramModels.User) {
	if user == nil || user.IsBot {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if name := displayName(user); name != "" {
		m.names[user.ID] = name
	}

	ids := m.chats[chatID]
	for i, id := range ids {
		if id == user.ID {
			ids = append(ids[:i], ids[i+1:]...)
			break
		}
	}
	ids = append([]int64{user.ID}, ids...)
	if len(ids) > maxMembersPerChat {
		ids = ids[:maxMembersPerChat]
	}
	m.chats[chatID] = ids
}

// Person returns how the model should see a user: their persona when known,
// otherwise just their Telegram name.
func (m *chatMembers) Person(user *goTelegramModels.User) llm.Person {
	if user == nil {
		return llm.Person{}
	}
	if p, ok := m.personas.Lookup(user.ID); ok {
		return p
	}
	return llm.Person{Name: displayName(user)}
}

// List returns the people recently active in a chat.
func (m *chatMembers) List(chatID int64) []llm.Person {
	m.mu.Lock()
	defer m.mu.Unlock()

	people := make([]llm.Person, 0, len(m.chats[chatID]))
	for _, id := range m.chats[chatID] {
		if p, ok := m.personas.Lookup(id); ok {
			people = append(people, p)
		} else if name := m.names[id]; name != "" {
			people = append(people, llm.Person{Name: name})
		}
	}
	return people
}

func displayName(user *goTelegramModels.User) string {
	if user == nil || user.IsBot {
		return ""
	}
	if name := strings.TrimSpace(user.FirstName); name != "" {
		return name
	}
	if user.Username != "" {
		return "@" + user.Username
	}
	return ""
}

// pickTarget answers "кто …" questions with a random chat member.
// Protected people (the bot's creator) are never picked.
func pickTarget(query string, members []llm.Person) llm.Person {
	q := strings.ToLower(strings.TrimSpace(query))
	if q != "кто" && !strings.HasPrefix(q, "кто ") && !strings.HasPrefix(q, "кто,") {
		return llm.Person{}
	}
	var pool []llm.Person
	for _, p := range members {
		if !p.Protected {
			pool = append(pool, p)
		}
	}
	if len(pool) == 0 {
		return llm.Person{}
	}
	return pool[rand.Intn(len(pool))]
}

func names(people []llm.Person) []string {
	out := make([]string, len(people))
	for i, p := range people {
		out[i] = p.Name
	}
	return out
}
