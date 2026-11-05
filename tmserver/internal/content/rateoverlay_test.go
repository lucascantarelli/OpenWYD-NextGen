package content

import (
	"testing"

	"openwyd-nextgen/internal/ratetables"
)

func TestApplyRateOverridesSanc(t *testing.T) {
	sanc := DefaultSancRate()
	applied, ignored := ApplyRateOverrides(sanc, nil, nil, []ratetables.Override{
		{File: ratetables.FileSancRate, Family: "PO", Name: "3", Value: 55},
		{File: ratetables.FileSancRate, Family: "AMAGO", Name: "0", Value: 90},
		{File: ratetables.FileSancRate, Family: "PO", Name: "99", Value: 1}, // fora da tabela
		{File: ratetables.FileSancRate, Family: "XX", Name: "0", Value: 1},  // âmbar inexistente
	})
	if applied != 2 || ignored != 2 {
		t.Fatalf("applied=%d ignored=%d; quer 2/2", applied, ignored)
	}
	if got := sanc.Rate(sancAnvilOri, 3); got != 55 {
		t.Errorf("PO índice 3 = %d; quer 55", got)
	}
	if got := sanc.Rate(sancAnvilAma, 0); got != 90 {
		t.Errorf("âmago índice 0 = %d; quer 90", got)
	}
	// Células não editadas continuam com o padrão compilado.
	if got := sanc.Rate(sancAnvilOri, 0); got != 100 {
		t.Errorf("PO índice 0 = %d; quer 100 compilado", got)
	}
}

func TestApplyRateOverridesCompKeepsSiblings(t *testing.T) {
	comp := &CompRate{rates: map[string]map[string]int{"Compositor": {"Item_+7": 2, "Item_+8": 4}}}
	applied, _ := ApplyRateOverrides(nil, comp, nil, []ratetables.Override{
		{File: ratetables.FileCompRate, Family: "Compositor", Name: "Item_+7", Value: 6},
	})
	if applied != 1 {
		t.Fatalf("applied = %d; quer 1", applied)
	}
	if got, _ := comp.Rate("Compositor", "Item_+7"); got != 6 {
		t.Errorf("Item_+7 = %d; quer 6", got)
	}
	if got, _ := comp.Rate("Compositor", "Item_+8"); got != 4 {
		t.Errorf("Item_+8 = %d; quer 4 (chave vizinha intacta)", got)
	}
}

func TestApplyRateOverridesQuests(t *testing.T) {
	quests := DefaultQuestRates()
	applied, ignored := ApplyRateOverrides(nil, nil, quests, []ratetables.Override{
		{File: ratetables.FileQuestsRate, Family: "EXP", Name: "0.0", Value: 45000},
		{File: ratetables.FileQuestsRate, Family: "COIN", Name: "4.0", Value: 900000},
		{File: ratetables.FileQuestsRate, Family: "LEVEL", Name: "1.2", Value: 120},
		{File: ratetables.FileQuestsRate, Family: "EXP", Name: "9.0", Value: 1}, // tier inexistente
		{File: ratetables.FileQuestsRate, Family: "EXP", Name: "0", Value: 1},   // nome sem coluna
	})
	if applied != 3 || ignored != 2 {
		t.Fatalf("applied=%d ignored=%d; quer 3/2", applied, ignored)
	}
	tier0, _ := quests.Tier(0)
	if tier0.MortalExp != 45000 {
		t.Errorf("tier 0 EXP mortal = %d; quer 45000", tier0.MortalExp)
	}
	tier4, _ := quests.Tier(4)
	if tier4.Coin != 900000 {
		t.Errorf("tier 4 moedas = %d; quer 900000", tier4.Coin)
	}
	tier1, _ := quests.Tier(1)
	if tier1.ArchMin != 120 {
		t.Errorf("tier 1 arch min = %d; quer 120", tier1.ArchMin)
	}
}
