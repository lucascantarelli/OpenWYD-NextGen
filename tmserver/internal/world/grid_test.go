package world

import "testing"

func TestGrid(t *testing.T) {
	g := newGrid(8)
	if g.Dim() != 8 {
		t.Fatalf("Dim = %d, want 8", g.Dim())
	}
	if _, ok := g.MobAt(3, 4); ok {
		t.Errorf("fresh cell should be empty")
	}
	g.SetMob(3, 4, 0) // o índice 0 é um jogador válido; precisa voltar como ocupado
	if v, ok := g.MobAt(3, 4); !ok || v != 0 {
		t.Errorf("MobAt(3,4) = %d,%v, want 0,true", v, ok)
	}
	g.ClearMob(3, 4)
	if _, ok := g.MobAt(3, 4); ok {
		t.Errorf("cell should be empty after ClearMob")
	}
	// Fora dos limites é seguro.
	g.SetMob(-1, 0, 5)
	g.SetMob(8, 8, 5)
	if _, ok := g.MobAt(8, 8); ok {
		t.Errorf("out-of-bounds cell should report empty")
	}
}
