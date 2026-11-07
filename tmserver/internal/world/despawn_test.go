package world

import "testing"

// TestDespawnMob cobre a limpeza de morte de criatura (usada por handler.mobKilled): o
// slot da entidade é liberado e sua célula de grade limpa para a criatura não poder ser realvo
// e o slot ser recuperável. Roda num World não iniciado (DespawnMob é um
// auxiliar só-no-loop que aqui toca apenas entities/grade — sem sessões em visão).
func TestDespawnMob(t *testing.T) {
	w := New(Config{GridDim: 16}, slogDiscard(), nil, nil)

	id := w.SpawnMob(make([]byte, structMobTemplateSize), 5, 6)
	if id < MaxUser {
		t.Fatalf("SpawnMob = %d, want a mob id >= %d", id, MaxUser)
	}
	if w.Entity(id) == nil {
		t.Fatal("mob entity missing after spawn")
	}
	if got, ok := w.grid.MobAt(5, 6); !ok || int(got) != id {
		t.Fatalf("grid cell after spawn = (%d,%v), want (%d,true)", got, ok, id)
	}

	w.DespawnMob(id, 1) // 1 = morte

	if w.Entity(id) != nil {
		t.Error("entity slot not freed after DespawnMob")
	}
	if _, ok := w.grid.MobAt(5, 6); ok {
		t.Error("grid cell not cleared after DespawnMob")
	}

	// No-ops: um slot já liberado e um id na faixa de jogador não podem dar panic.
	w.DespawnMob(id, 1)
	w.DespawnMob(1, 1)
}

// structMobTemplateSize é o comprimento bruto de STRUCT_MOB que SpawnMob interpreta (mob.go).
const structMobTemplateSize = 816
