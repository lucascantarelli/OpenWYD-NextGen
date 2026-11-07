package level

import "testing"

// Os valores esperados são derivados à mão do ramo de campo geral de
// MobKilled.cpp:1272-1425 (ver SoloExpReward). kefraLive é escrito caso a caso
// porque o padrão (false) reduz pela metade toda recompensa.
func TestSoloExpReward(t *testing.T) {
	tests := []struct {
		name        string
		mobExp      int64
		killerLevel int32
		mobLevel    int32
		classMaster uint8
		expBonus    int32
		ev          ExpEvents
		want        int64
	}{
		{
			// 450*1000/80=5625 → ÷1 → ×0.6=3375 → eMob cap 1000 → −15% = 850.
			name: "same level low band hits eMob cap", mobExp: 1000,
			killerLevel: 50, mobLevel: 50, classMaster: classMortal,
			ev: ExpEvents{KefraLive: true}, want: 850,
		},
		{
			// Como acima, então queda de Kefra ÷2 = 500 → −15% = 425 (o padrão 0.425·E).
			name: "kefra down halves", mobExp: 1000,
			killerLevel: 50, mobLevel: 50, classMaster: classMortal,
			ev: ExpEvents{}, want: 425,
		},
		{
			// 450*100000/380=118421 → ÷1.25f=94736 → ×0.6=56841 → abaixo do
			// teto de 100000 → −15% = 48315: o fator 450/(30+L) continua visível.
			name: "level 350 uncapped with 1.25f divisor", mobExp: 100_000,
			killerLevel: 350, mobLevel: 350, classMaster: classMortal,
			ev: ExpEvents{KefraLive: true}, want: 48_315,
		},
		{
			// 450*60000/280=96428 → ÷1.07f=90119 → ×0.6=54071 → −15% = 45961.
			name: "level 250 band divisor 1.07f", mobExp: 60_000,
			killerLevel: 250, mobLevel: 250, classMaster: classMortal,
			ev: ExpEvents{KefraLive: true}, want: 45_961,
		},
		{
			// Limitado em eMob=1000 → +100% de bônus de item → 2000 → −15% = 1700.
			name: "exp bonus applies after the cap", mobExp: 1000,
			killerLevel: 50, mobLevel: 50, classMaster: classMortal, expBonus: 100,
			ev: ExpEvents{KefraLive: true}, want: 1700,
		},
		{
			// 450*2M/80 = 11.25M > gate de 10M → o legado pula a concessão por inteiro.
			name: "10M gate skips instead of clamping", mobExp: 2_000_000,
			killerLevel: 50, mobLevel: 50, classMaster: classMortal,
			ev: ExpEvents{KefraLive: true}, want: 0,
		},
		{
			// myLevel=50+400=450: 450*100000/480=93750 → ÷320=292 → ×0.6=175 →
			// Kefra abatido 87 → −15% = 74. A tier quase zero que a issue #43 atingiu.
			name: "celestial tier offset and divisors", mobExp: 100_000,
			killerLevel: 50, mobLevel: 50, classMaster: classCelestial,
			ev: ExpEvents{}, want: 74,
		},
		{
			// Cap 1000 → newbie +25% = 1250 → +15% (não −15%) = 1437.
			name: "newbie event", mobExp: 1000,
			killerLevel: 50, mobLevel: 50, classMaster: classMortal,
			ev: ExpEvents{NewbieEvent: true, KefraLive: true}, want: 1437,
		},
		{
			// Cap 1000 → ×2 = 2000 → −15% = 1700.
			name: "double mode", mobExp: 1000,
			killerLevel: 50, mobLevel: 50, classMaster: classMortal,
			ev: ExpEvents{DoubleMode: true, KefraLive: true}, want: 1700,
		},
		{
			// ExpApply penalty: mult=31*100/101=30 → 30*2−100<0 → 0 exp. Um
			// matador nível 100 farmando mobs nível 30 não ganha nada (paridade).
			name: "overleveled killer gets zero", mobExp: 1000,
			killerLevel: 100, mobLevel: 30, classMaster: classMortal,
			ev: ExpEvents{KefraLive: true}, want: 0,
		},
		{
			name: "zero base", mobExp: 0,
			killerLevel: 50, mobLevel: 50, classMaster: classMortal, expBonus: 100,
			ev: ExpEvents{KefraLive: true}, want: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SoloExpReward(tt.mobExp, tt.killerLevel, tt.mobLevel, tt.classMaster, tt.expBonus, tt.ev)
			if got != tt.want {
				t.Errorf("SoloExpReward(%d, k%d, m%d, cm%d, b%d, %+v) = %d, want %d",
					tt.mobExp, tt.killerLevel, tt.mobLevel, tt.classMaster, tt.expBonus, tt.ev, got, tt.want)
			}
		})
	}
}

// Um personagem padrão (mortal) precisa render mais que o roteamento celestial
// em que ClassMaster=0 sem valor caía (issue #43: EXP quase zero).
func TestSoloExpReward_mortalNotOnCelestialPath(t *testing.T) {
	ev := ExpEvents{}
	mortal := SoloExpReward(1000, 20, 20, classMortal, 0, ev)
	unset := SoloExpReward(1000, 20, 20, 0, 0, ev)
	if mortal <= 0 {
		t.Fatalf("mortal reward = %d, want > 0", mortal)
	}
	if unset >= mortal {
		t.Errorf("ClassMaster 0 reward %d not below mortal %d — celestial routing regressed", unset, mortal)
	}
}

// A recompensa do Campo de Treino é a base sem eventos vezes o multiplicador
// (issue #383); os eventos globais não podem alterá-la.
func TestTrainingFieldExpReward(t *testing.T) {
	tests := []struct {
		name        string
		mobExp      int64
		killerLevel int32
		mobLevel    int32
		expBonus    int32
		mult        int64
		want        int64
	}{
		// Mesmo nível, teto eMob 1000 → ×6, sem o ÷2 do Kefra e sem −15%.
		{name: "base times multiplier", mobExp: 1000, killerLevel: 20, mobLevel: 20, mult: 6, want: 6000},
		// Cap 1000 → +50% item bonus = 1500 → ×6.
		{name: "item bonus before multiplier", mobExp: 1000, killerLevel: 20, mobLevel: 20, expBonus: 50, mult: 6, want: 9000},
		{name: "multiplier one is the raw base", mobExp: 1000, killerLevel: 20, mobLevel: 20, mult: 1, want: 1000},
		{name: "multiplier zero gives nothing", mobExp: 1000, killerLevel: 20, mobLevel: 20, mult: 0, want: 0},
		// A penalidade de ExpApply zera a base, então o multiplicador não a reanima.
		{name: "overleveled killer gets zero", mobExp: 1000, killerLevel: 100, mobLevel: 30, mult: 6, want: 0},
		// Base 1M (o teto morde) ×20 = 20M → limitado pelo gate de 10M.
		{name: "capped at award gate", mobExp: 1_000_000, killerLevel: 200, mobLevel: 200, mult: 20, want: soloExpGate},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := TrainingFieldExpReward(tt.mobExp, tt.killerLevel, tt.mobLevel, classMortal, tt.expBonus, tt.mult)
			if got != tt.want {
				t.Errorf("TrainingFieldExpReward(%d, k%d, m%d, b%d, x%d) = %d, want %d",
					tt.mobExp, tt.killerLevel, tt.mobLevel, tt.expBonus, tt.mult, got, tt.want)
			}
		})
	}
}
