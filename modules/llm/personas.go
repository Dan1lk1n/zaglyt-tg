package llm

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// Person is a chat member as the model sees it.
type Person struct {
	Name string `json:"name"`
	// Nicknames are alternative names people use in the chat.
	Nicknames []string `json:"nicknames,omitempty"`
	// Notes tell the model how to treat this person (what to joke about, what not).
	Notes string `json:"notes,omitempty"`
	// Protected people are never picked for "кто …" and never insulted.
	Protected bool `json:"protected,omitempty"`
	// NoJokes are word stems that must not appear in replies to or about this
	// person. Enforced in code: naming a taboo in the prompt suggests it.
	NoJokes []string `json:"no_jokes,omitempty"`
}

// Personas maps Telegram user IDs to people the bot knows.
type Personas struct {
	// General are chat-wide joke rules, added to the system prompt.
	General string `json:"general,omitempty"`
	People  []struct {
		ID int64 `json:"id"`
		Person
	} `json:"people"`

	byID map[int64]Person
}

// LoadPersonas reads personas from a JSON file. An empty path means none.
func LoadPersonas(path string) (*Personas, error) {
	p := &Personas{byID: map[int64]Person{}}
	if strings.TrimSpace(path) == "" {
		return p, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read personas: %w", err)
	}
	if err := json.Unmarshal(data, p); err != nil {
		return nil, fmt.Errorf("parse personas %s: %w", path, err)
	}
	for _, person := range p.People {
		p.byID[person.ID] = person.Person
	}
	return p, nil
}

// Protected returns the people the bot must always defend.
func (p *Personas) Protected() []Person {
	if p == nil {
		return nil
	}
	var out []Person
	for _, person := range p.People {
		if person.Protected {
			out = append(out, person.Person)
		}
	}
	return out
}

// Lookup returns the known person for a Telegram user ID.
func (p *Personas) Lookup(id int64) (Person, bool) {
	if p == nil {
		return Person{}, false
	}
	person, ok := p.byID[id]
	return person, ok
}

// Mentioned returns known people whose name or nickname appears in text.
func (p *Personas) Mentioned(text string) []Person {
	if p == nil {
		return nil
	}
	text = strings.ToLower(text)
	var found []Person
	for _, person := range p.People {
		for _, name := range append([]string{person.Name}, person.Nicknames...) {
			// Match on the first word stem so "лёхи"/"лёхе" still hit "Лёха".
			stem := []rune(strings.ToLower(strings.Fields(name + " x")[0]))
			if len(stem) > 4 {
				stem = stem[:len(stem)-1]
			}
			if len(stem) >= 3 && strings.Contains(text, string(stem)) {
				found = append(found, person.Person)
				break
			}
		}
	}
	return found
}
