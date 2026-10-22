package llm

import "testing"

func TestFixAddress(t *testing.T) {
	r := Request{
		Author:  Person{Name: "Ярослав Кобелев", Nicknames: []string{"Ярик"}},
		Members: []string{"Данил Петров", "Ярослав Кобелев", "Глеб"},
	}
	cases := map[string]string{
		"Данил, это ж Ярик, а ты чего": "Ярик, это ж Ярик, а ты чего",
		"Глеб, иди на завод":           "Ярик, иди на завод",
		"Ярик, иди на завод":           "Ярик, иди на завод",
		"Ярослав, иди на завод":        "Ярослав, иди на завод",
		"Слушай, ты чего такой":        "Слушай, ты чего такой",
		"Короче, иди нахуй":            "Короче, иди нахуй",
	}
	for in, want := range cases {
		if got := fixAddress(in, r); got != want {
			t.Errorf("fixAddress(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestRelatedLinesSkipsOwnReplies(t *testing.T) {
	history := []string{"Данил Петров: кто пидор", "Ты: Данил, вот тебе цитата про пидоров"}
	got := relatedLines("кто пидор?", history)
	if len(got) != 1 || got[0] != history[0] {
		t.Fatalf("got %q", got)
	}
}
