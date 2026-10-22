package handlers

import (
	"os"
	"path/filepath"
	"testing"
	"zaglyt-tg/modules/llm"
)

func TestPickTargetFallsBackToPersonas(t *testing.T) {
	path := filepath.Join(t.TempDir(), "p.json")
	data := `{"people":[{"id":1,"name":"Создатель","protected":true},{"id":2,"name":"Кардак"}]}`
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}
	personas, err := llm.LoadPersonas(path)
	if err != nil {
		t.Fatal(err)
	}

	// Only the protected creator has written since the restart.
	members := []llm.Person{{Name: "Создатель", Protected: true}}
	if got := pickTarget("кто пидор", members, personas); got.Name != "Кардак" {
		t.Fatalf("got %q, want a persona fallback", got.Name)
	}
	if got := pickTarget("как дела", members, personas); got.Name != "" {
		t.Fatalf("non-кто query picked %q", got.Name)
	}
}
