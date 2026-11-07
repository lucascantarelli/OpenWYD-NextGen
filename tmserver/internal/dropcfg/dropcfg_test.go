package dropcfg

import (
	"testing"

	"openwyd-nextgen/tmserver/internal/loot"
)

// TestApplyChangesOnlyTheEditedSlot é a regra de ouro deste recurso: sem
// sobreposições a tabela é exatamente os defaults de código do loot, e uma
// sobreposição muda um divisor e nada mais.
func TestApplyChangesOnlyTheEditedSlot(t *testing.T) {
	before := Snapshot()
	t.Cleanup(func() { loot.DropRate = before })

	if applied, ignored := Apply(nil); applied != 0 || ignored != 0 {
		t.Fatalf("Apply(nil) = %d applied, %d ignored, want 0,0", applied, ignored)
	}
	if got := Snapshot(); got != before {
		t.Fatalf("empty overlay changed the table: %v", got)
	}

	applied, ignored := Apply([]Override{{Slot: 12, Value: 450}})
	if applied != 1 || ignored != 0 {
		t.Fatalf("Apply = %d applied, %d ignored, want 1,0", applied, ignored)
	}
	if loot.DropRate[12] != 450 {
		t.Errorf("DropRate[12] = %d, want 450", loot.DropRate[12])
	}
	for slot, value := range before {
		if slot == 12 {
			continue
		}
		if loot.DropRate[slot] != value {
			t.Fatalf("slot %d changed to %d, want %d", slot, loot.DropRate[slot], value)
		}
	}
}

// TestApplyRejectsOutOfRange torna a validação explícita: a faixa do portal e
// esta precisam concordar, e uma linha ruim não pode virar divisor em silêncio.
func TestApplyRejectsOutOfRange(t *testing.T) {
	before := Snapshot()
	t.Cleanup(func() { loot.DropRate = before })

	cases := []Override{
		{Slot: -1, Value: 900},
		{Slot: 64, Value: 900},
		{Slot: 3, Value: 0},
		{Slot: 3, Value: MaxRate + 1},
	}
	applied, ignored := Apply(cases)
	if applied != 0 || ignored != len(cases) {
		t.Fatalf("Apply(bad rows) = %d applied, %d ignored, want 0,%d", applied, ignored, len(cases))
	}
	if got := Snapshot(); got != before {
		t.Fatalf("invalid overlay changed the table: %v", got)
	}
}

// TestEffectiveRateFollowsTheOverride prova que a mudança alcança a fórmula que o
// mundo usa (EffectiveDropRate), que é o motivo de a sobreposição ser útil.
func TestEffectiveRateFollowsTheOverride(t *testing.T) {
	before := Snapshot()
	t.Cleanup(func() { loot.DropRate = before })

	base := loot.EffectiveDropRate(12, 0, 50)
	Apply([]Override{{Slot: 12, Value: int64(loot.DropRate[12] / 2)}})
	half := loot.EffectiveDropRate(12, 0, 50)

	if half >= base {
		t.Fatalf("EffectiveDropRate(12) after halving the divisor = %d, want < %d", half, base)
	}
	// O slot 11 é sobreposição dura (sempre dropa) e não pode ser afetado pelo slot 12.
	if got := loot.EffectiveDropRate(11, 0, 50); got != 1 {
		t.Errorf("EffectiveDropRate(11) = %d, want 1 (hard override)", got)
	}
}
