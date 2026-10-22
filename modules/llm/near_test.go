package llm

import "testing"

func TestInsultsNear(t *testing.T) {
	creator := Person{Name: "Данил Петров", Nicknames: []string{"Пенис", "создатель"}, Protected: true}
	cases := map[string]bool{
		"Петров просто дурак, лох конченый":                  true,
		"создатель твой лох":                                 true,
		"Кардак, рот закрой, Петров тут главный, а ты мудак": false,
		"Залупин, иди нахуй, Петров меня создал":             false,
		"Глеб, кальян норм":                                  false,
	}
	for text, want := range cases {
		if got := insultsNear(text, creator); got != want {
			t.Errorf("insultsNear(%q) = %v, want %v", text, got, want)
		}
	}
}
