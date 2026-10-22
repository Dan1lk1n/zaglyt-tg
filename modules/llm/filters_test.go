package llm

import "testing"

func TestForeignScript(t *testing.T) {
	cases := map[string]bool{
		"Кардак, иди нахуй, сука":                false,
		"Кардак, твой iPhone в кредит":           false,
		"Кардак, это 你好 значит":                  true,
		"Kardak, you are so stupid, иди":         true,
		"Глеб, ставь на Зенит, odds хорошие тут": false,
	}
	for in, want := range cases {
		if got := foreignScript(in); got != want {
			t.Errorf("foreignScript(%q) = %v, want %v", in, got, want)
		}
	}
}

func TestCopiesNotes(t *testing.T) {
	notes := "Заводчанин — можно слать его на завод. Ещё он пропеллер."
	if !copiesNotes("Виталя, можно слать его на завод, ага", notes) {
		t.Error("verbatim chunk not detected")
	}
	if copiesNotes("Виталя, вали на свой завод крутиться", notes) {
		t.Error("paraphrase flagged")
	}
}
