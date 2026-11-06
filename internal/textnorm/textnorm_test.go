package textnorm

import "testing"

func TestNFCCompoeDecomposto(t *testing.T) {
	t.Parallel()

	decomposto := "Jose\u0301" // e + combining acute (NFD)
	if got := Clean(decomposto); got != "José" {
		t.Fatalf("Clean(%q) = %q, quer \"José\"", decomposto, got)
	}
}

func TestCleanColapsaEspacoETrim(t *testing.T) {
	t.Parallel()

	casos := []struct{ in, want string }{
		{"  Espada  Longa  ", "Espada Longa"},
		{"\tEspada\nLonga\r", "Espada Longa"},
		{"Espada\u00a0Longa", "Espada Longa"},        // NBSP
		{"Espada \u200bLonga", "Espada \u200bLonga"}, // ZWSP não é espaço; preservado de propósito
	}
	for _, c := range casos {
		if got := Clean(c.in); got != c.want {
			t.Errorf("Clean(%q) = %q, quer %q", c.in, got, c.want)
		}
	}
}

func TestCleanRemoveControle(t *testing.T) {
	t.Parallel()

	if got := Clean("Espa\x00da\x07"); got != "Espada" {
		t.Fatalf("Clean com NUL/BEL = %q, quer \"Espada\"", got)
	}
}

func TestRepairMojibake(t *testing.T) {
	t.Parallel()

	casos := []struct{ in, want string }{
		{"Cajado de AmÃ©m", "Cajado de Amém"},
		{"AÃ§Ãºcar", "Açúcar"},
		{"CoraÃ§Ã£o", "Coração"},
		{"Moeda de Â£", "Moeda de £"},
		// Euro e travessão vêm dos bytes CP1252 0x80/0x94 (não Latin-1 puro).
		{"PreÃ§o: â‚¬ 10", "Preço: € 10"},
		{"Intervalo 10â€“20", "Intervalo 10–20"},
	}
	for _, c := range casos {
		if got := Clean(c.in); got != c.want {
			t.Errorf("Clean(%q) = %q, quer %q", c.in, got, c.want)
		}
	}
}

func TestNaoEstragaPortuguesLegitimo(t *testing.T) {
	t.Parallel()

	// "Âmbar" e "Âmago" são português correto: o "Â" não é marcador de
	// duplo encoding aqui, porque os bytes resultantes não são UTF-8 válido.
	for _, s := range []string{"Cajado de Âmbar", "Baú do Âmago", "Ã", "Ângulo"} {
		if got := Clean(s); got != s {
			t.Errorf("Clean(%q) = %q, deveria ficar intacto", s, got)
		}
	}
}

func TestCleanIdempotente(t *testing.T) {
	t.Parallel()

	entradas := []string{
		"  Cajado de AmÃ©m\u00a0 ",
		"Jose\u0301   da   Silva",
		"Espa\x00da",
		"Âmbar",
	}
	for _, in := range entradas {
		uma := Clean(in)
		duas := Clean(uma)
		if uma != duas {
			t.Errorf("Clean não é idempotente para %q: %q → %q", in, uma, duas)
		}
	}
}

func TestSlice(t *testing.T) {
	t.Parallel()

	got := Slice([]string{" AmÃ©m ", "x\x01y"})
	if got[0] != "Amém" || got[1] != "xy" {
		t.Fatalf("Slice = %q", got)
	}
}
