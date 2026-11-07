package combine

import (
	"testing"

	"openwyd-nextgen/tmserver/internal/refine"
	"openwyd-nextgen/tmserver/internal/world"
)

// efSanc é EF_SANC (ItemEffect.h:100). O pacote lê níveis através de
// refine.Level; os testes ainda precisam do id cru para plantar pares.
const efSanc = 43

func TestMatchAnctBaseRecipe(t *testing.T) {
	cat := Catalog{
		Unique:     map[int]int{801: 41},
		Extra:      map[int]int{801: 2451},
		AnctChance: [3]int{2, 4, 10},
	}
	// A base carrega um par de sanc, então o +7 do resultado tem slot onde cair.
	items := []world.Item{
		{Index: 801, Effects: [3]world.Effect{{Effect: efSanc, Value: 0}}},
		{Index: 2442},
	}
	if got := MatchAnct(cat, items); got != 1 {
		t.Fatalf("rate = %d, want 1", got)
	}
	got := AnctResult(cat, items)
	if got.Index != 2452 {
		t.Errorf("result index = %d, want 2452 (joia 1 + extra 2451)", got.Index)
	}
	if refine.Level(got) != 7 {
		t.Errorf("result sanc = %d, want 7", refine.Level(got))
	}
}

// BASE_SetItemSanc não aloca nada: item base sem par de sanc produz
// resultado sem sanc nenhum (Basedef.cpp:2312 devolve FALSE). O caminho de combinação
// não cria slot — só o caminho de refino com pó cria.
func TestAnctResultWithoutSancSlot(t *testing.T) {
	cat := Catalog{
		Unique: map[int]int{801: 41},
		Extra:  map[int]int{801: 2451},
	}
	items := []world.Item{
		{Index: 801, Effects: [3]world.Effect{{Effect: efPos, Value: 4}, {Effect: efItemLevel, Value: 2}, {Effect: efMobType, Value: 1}}},
		{Index: 2442},
	}
	got := AnctResult(cat, items)
	if got.Index != 2452 {
		t.Errorf("result index = %d, want 2452", got.Index)
	}
	if lvl := refine.Level(got); lvl != 0 {
		t.Errorf("result sanc = %d, want 0 (no slot to write into)", lvl)
	}
	// E os efeitos existentes precisam sobreviver intocados.
	if got.Effects[2] != (world.Effect{Effect: efMobType, Value: 1}) {
		t.Errorf("Effects[2] = %+v, want the original EF_MOBTYPE pair", got.Effects[2])
	}
}

func TestMatchAnctSacrificeBonus(t *testing.T) {
	cat := Catalog{
		Unique: map[int]int{801: 41},
		Extra:  map[int]int{801: 2451},
		// EF_POS mora em nPos, nunca num par de BaseEffect (ver itemAbility).
		Pos:        map[int]int{900: 4},
		AnctChance: [3]int{2, 4, 10},
	}
	items := []world.Item{
		{Index: 801},
		{Index: 2441},
		{Index: 900, Effects: [3]world.Effect{{Effect: efSanc, Value: 9}}},
	}
	if got := MatchAnct(cat, items); got != 11 {
		t.Fatalf("rate = %d, want 11 (1 + AnctChance[2])", got)
	}
}

func TestMatchAnctRejectsInvalid(t *testing.T) {
	cat := Catalog{Unique: map[int]int{100: 10}, Extra: map[int]int{100: 1}}
	if got := MatchAnct(cat, []world.Item{{Index: 100}, {Index: 2441}}); got != 0 {
		t.Errorf("bad nUnique rate = %d, want 0", got)
	}
}
