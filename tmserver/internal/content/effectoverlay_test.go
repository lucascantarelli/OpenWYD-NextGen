package content

import "testing"

func baseEffectOf(effs []BaseEffect, code uint8) (int16, bool) {
	for _, e := range effs {
		if e.Eff == code {
			return e.Val, true
		}
	}
	return 0, false
}

func TestApplyItemEffectOverrides(t *testing.T) {
	base := map[int][]BaseEffect{
		1188: {{Eff: 3, Val: 47}, {Eff: 29, Val: 2}}, // EF_AC, EF_RUNSPEED
		165:  {{Eff: 42, Val: 50}},                   // EF_CRITICAL, untouched
	}
	rows := []ItemEffectOverride{
		{ItemIndex: 1188, Effect: 3, Value: 60},  // replaces EF_AC
		{ItemIndex: 1188, Effect: 42, Value: 10}, // EF_CRITICAL, fora da linha: acrescentado
		{ItemIndex: 1188, Effect: 1, Value: 3},   // EF_RANGE: não é código de score, recusado
		{ItemIndex: 900, Effect: 4, Value: 250},  // item ausente do catálogo
	}

	out, stats := ApplyItemEffectOverrides(base, rows)

	if got, _ := baseEffectOf(out[1188], 3); got != 60 {
		t.Errorf("EF_AC = %d; quer 60 do overlay", got)
	}
	if got, ok := baseEffectOf(out[1188], 42); !ok || got != 10 {
		t.Errorf("EF_CRITICAL = %d (found=%v); quer 10 anexado", got, ok)
	}
	if _, ok := baseEffectOf(out[1188], 1); ok {
		t.Error("EF_RANGE entrou no catálogo — o score nunca o lê, seria bônus fantasma")
	}
	if got := out[1188][1]; got.Eff != 29 || got.Val != 2 {
		t.Errorf("EF_RUNSPEED = %+v; quer o valor do conteúdo preservado", got)
	}
	if got, _ := baseEffectOf(out[165], 42); got != 50 {
		t.Errorf("item intocado mudou para %d", got)
	}
	if got, _ := baseEffectOf(out[900], 4); got != 250 {
		t.Errorf("efeito novo em item fora do catálogo = %d; quer 250", got)
	}
	if stats.Replaced != 1 || stats.Added != 2 || stats.Ignored != 1 {
		t.Errorf("stats = %+v; quer 1/2/1", stats)
	}
}

func TestApplyItemEffectOverridesLeavesCatalogAlone(t *testing.T) {
	base := map[int][]BaseEffect{1188: {{Eff: 3, Val: 47}}}

	if out, stats := ApplyItemEffectOverrides(base, nil); len(out) != 1 || stats != (EffectOverrideStats{}) {
		t.Fatalf("overlay vazio = %v/%+v; quer o catálogo sem alteração", out, stats)
	}

	ApplyItemEffectOverrides(base, []ItemEffectOverride{{ItemIndex: 1188, Effect: 3, Value: 60}})
	if got, _ := baseEffectOf(base[1188], 3); got != 47 {
		t.Fatalf("catálogo de origem mutado: EF_AC = %d; quer 47", got)
	}
}
