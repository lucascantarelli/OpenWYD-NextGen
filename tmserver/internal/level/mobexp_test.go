package level

import "testing"

// TestMobExpPacing alimenta a curva de projeto de volta pelo pipeline real de
// recompensa (flags padrão de evento, mortal solo, abate de mesmo nível) e confere
// que os abates-por-nível caem no alvo de ritmo clássico. A tolerância absorve
// os truncamentos inteiros do pipeline.
func TestMobExpPacing(t *testing.T) {
	for _, lvl := range []int32{1, 5, 10, 20, 35, 50, 75, 99, 100, 150, 199, 200, 250, 299, 300, 350, 380, 399} {
		e := MobExpForLevel(lvl)
		if e <= 0 {
			t.Fatalf("MobExpForLevel(%d) = %d, want > 0", lvl, e)
		}
		gain := SoloExpReward(e, lvl, lvl, classMortal, 0, ExpEvents{})
		if gain <= 0 {
			t.Fatalf("level %d: reward pipeline returned %d for template exp %d", lvl, gain, e)
		}
		delta := float64(NextLevelExp(lvl) - NextLevelExp(lvl-1))
		kills := delta / float64(gain)
		target := killsPerLevel(lvl)
		if kills < target*0.85 || kills > target*1.2 {
			t.Errorf("level %d: %.1f kills per level, want ≈%.1f (template exp %d, gain %d)",
				lvl, kills, target, e, gain)
		}
	}
}

// TestMobExpForLevelClamps: níveis de mob fora da faixa usam os extremos da curva.
func TestMobExpForLevelClamps(t *testing.T) {
	if got, want := MobExpForLevel(0), MobExpForLevel(1); got != want {
		t.Errorf("MobExpForLevel(0) = %d, want %d", got, want)
	}
	if got, want := MobExpForLevel(1000), MobExpForLevel(MaxLevel); got != want {
		t.Errorf("MobExpForLevel(1000) = %d, want %d", got, want)
	}
}

// O gate de 10M nunca pode engolir um abate de mesmo nível em ponto algum da curva —
// isso recriaria o sintoma de "sem EXP" que a curva existe para corrigir.
func TestMobExpNeverGated(t *testing.T) {
	for lvl := int32(1); lvl <= MaxLevel; lvl++ {
		if gain := SoloExpReward(MobExpForLevel(lvl), lvl, lvl, classMortal, 0, ExpEvents{}); gain <= 0 {
			t.Fatalf("level %d: same-level kill gained %d exp", lvl, gain)
		}
	}
}
