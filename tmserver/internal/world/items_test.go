package world

import "testing"

// TestAddToCargo coloca itens no primeiro slot livre e reporta -1 quando cheio —
// o destino de entrega da loja de doação (issue #34): cargo cheio significa item perdido.
func TestAddToCargo(t *testing.T) {
	w := New(Config{GridDim: 16}, slogDiscard(), nil, nil)

	if got := w.AddToCargo(nil, Item{Index: 1}); got != -1 {
		t.Errorf("AddToCargo(nil) = %d, want -1", got)
	}

	st := &CargoState{AccountID: 1}
	// Ocupar previamente o slot 0 para a próxima adição cair no slot 1.
	st.Items[0] = Item{Index: 42}
	if got := w.AddToCargo(st, Item{Index: 100}); got != 1 {
		t.Fatalf("AddToCargo into first free slot = %d, want 1", got)
	}
	if st.Items[1].Index != 100 {
		t.Errorf("slot 1 index = %d, want 100", st.Items[1].Index)
	}

	// Preencher o resto; o cargo agora está cheio e adições posteriores são perdidas.
	for i := 2; i < MaxCargo; i++ {
		if got := w.AddToCargo(st, Item{Index: int16(i)}); got != i {
			t.Fatalf("fill slot %d = %d", i, got)
		}
	}
	if got := w.AddToCargo(st, Item{Index: 999}); got != -1 {
		t.Errorf("AddToCargo into full cargo = %d, want -1 (lost)", got)
	}
}

// TestSeedWorldItem semeia objetos estáticos do mundo (portões/portas) no espaço de
// ids de chão na ordem das chamadas, mantendo-os fora da grade espacial e fora do
// caminho de coleta (eles carregam State de portão e o flag Static).
func TestSeedWorldItem(t *testing.T) {
	w := New(Config{GridDim: 16}, slogDiscard(), nil, nil)

	id1 := w.SeedWorldItem(Item{Index: 471}, 5, 6, 1)
	id2 := w.SeedWorldItem(Item{Index: 472}, 7, 8, 3)
	if id1 != 1 || id2 != 2 {
		t.Fatalf("seed ids = %d,%d, want 1,2 (file order)", id1, id2)
	}

	g := w.GroundItem(id2)
	if g == nil || !g.Static || g.State != 3 || g.Item.Index != 472 || g.X != 7 || g.Y != 8 {
		t.Fatalf("seeded gate = %+v, want Static gate 472 state 3 at (7,8)", g)
	}
	// Portões não são indexados na grade de itens (buscados por id, não por posição).
	if got := w.GroundItemAt(7, 8); got != nil {
		t.Errorf("GroundItemAt(7,8) = %+v, want nil (gate not on grid)", got)
	}
}
