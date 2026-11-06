package itemeffects

import "testing"

// TestLabelCoversRealAdds fixa os códigos que aparecem em saves reais: os
// atributos de score e o marcador de refino, com a ordem de resistência do
// conteúdo.
func TestLabelCoversRealAdds(t *testing.T) {
	cases := map[int32]string{
		2:  "Dano",
		3:  "Defesa",
		43: "Refino",
		54: "Resist. todos",
		71: "Crítico (2)",
		30: "Skill",
	}
	for code, want := range cases {
		if got := Label(code); got != want {
			t.Errorf("Label(%d) = %q, want %q", code, got, want)
		}
	}
	if got := Label(49); got != "Resist. fogo" {
		t.Errorf("Label(49) = %q, want Resist. fogo (EF_RESIST1)", got)
	}
}

// TestLabelFallsBackToConstant: um código sem rótulo curado ainda recebe o nome
// do vocabulário em vez de um número cru.
func TestLabelFallsBackToConstant(t *testing.T) {
	if code, known := int32(1), "LEVEL"; Name(code) != known {
		t.Fatalf("Name(%d) = %q, want %q", code, Name(code), known)
	}
	if got := Label(1); got != "LEVEL" {
		t.Errorf("Label(1) = %q, want the header constant LEVEL", got)
	}
}

// TestLabelUnknownIsEmpty: um código fora do vocabulário devolve "", então a UI
// mostra o número cru em vez de inventar um significado.
func TestLabelUnknownIsEmpty(t *testing.T) {
	if got := Label(98); got != "" {
		t.Errorf("Label(98) = %q, want empty (not in ItemEffect.h)", got)
	}
}
