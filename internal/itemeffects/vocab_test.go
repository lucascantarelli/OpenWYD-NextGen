package itemeffects

import "testing"

// TestCodeRoundTrip: o token de catálogo resolve para o código, token desconhecido
// não resolve, e o vocabulário inteiro é não vazio e ordenado.
func TestCodeRoundTrip(t *testing.T) {
	code, ok := Code("EF_CRITICAL")
	if !ok || code != 42 {
		t.Fatalf("Code(EF_CRITICAL) = %d, %v; want 42, true", code, ok)
	}
	if _, ok := Code("EF_NAO_EXISTE"); ok {
		t.Fatal("Code(EF_NAO_EXISTE) devolveu ok, want false")
	}

	all := All()
	if len(all) < 100 {
		t.Fatalf("All() = %d tipos, want >= 100", len(all))
	}
	for i, typ := range all {
		if typ.Name == "" || typ.Label == "" {
			t.Fatalf("All()[%d] = %+v, want nome e rótulo", i, typ)
		}
		if i > 0 && all[i-1].Code > typ.Code {
			t.Fatalf("All() fora de ordem em %d (%d > %d)", i, all[i-1].Code, typ.Code)
		}
	}
}

// TestKnown protege a validação de concessão/edição: 43 (refino) existe, 98/99
// não — o vocabulário salta de 91 para 100.
func TestKnown(t *testing.T) {
	if !Known(43) {
		t.Error("Known(43) = false, want true (EF_SANC)")
	}
	for _, code := range []int32{98, 99, 200} {
		if Known(code) {
			t.Errorf("Known(%d) = true, want false", code)
		}
	}
}
