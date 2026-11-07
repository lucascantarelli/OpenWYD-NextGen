package level

import "testing"

// TestNextLevel2Placeholder protege o substituto sintético de g_pNextLevel_2: ele
// precisa ser estritamente crescente (para o laço de level-up da tier terminar e os
// gates se comportarem), cobrir 0..MaxCLevel+1 e começar em 0. NÃO é checagem de
// paridade — os valores reais capturados substituem este vetor (e este teste ganha
func TestNextLevel2Placeholder(t *testing.T) {
	if len(nextLevel2) != int(MaxCLevel)+2 {
		t.Fatalf("nextLevel2 has %d entries, want %d (0..MaxCLevel+1)", len(nextLevel2), MaxCLevel+2)
	}
	if nextLevel2[0] != 0 {
		t.Errorf("nextLevel2[0] = %d, want 0", nextLevel2[0])
	}
	for i := 1; i < len(nextLevel2); i++ {
		if nextLevel2[i] <= nextLevel2[i-1] {
			t.Fatalf("nextLevel2 not strictly increasing at %d: %d <= %d", i, nextLevel2[i], nextLevel2[i-1])
		}
	}
}

func TestMaxLevelForTier(t *testing.T) {
	tests := []struct {
		classMaster uint8
		want        int32
	}{
		{classMortal, MaxLevel},
		{classArch, MaxLevel},
		{classCelestial, MaxCLevel},
		{classCelestialCS, MaxCLevel},
		{classSCelestial, MaxCLevel},
	}
	for _, tt := range tests {
		if got := MaxLevelForTier(tt.classMaster); got != tt.want {
			t.Errorf("MaxLevelForTier(%d) = %d, want %d", tt.classMaster, got, tt.want)
		}
	}
}

// TestNextLevelExpTier confere a escolha da curva: Mortal/Arch andam na g_pNextLevel,
// as tiers celestiais andam em g_pNextLevel_2, e cada uma trava no próprio teto.
func TestNextLevelExpTier(t *testing.T) {
	// O caminho Mortal bate exatamente com a curva Mortal.
	if got, want := NextLevelExpTier(1, classMortal), NextLevelExp(1); got != want {
		t.Errorf("NextLevelExpTier(1, mortal) = %d, want %d (mortal curve)", got, want)
	}
	// Arch também anda na curva Mortal (CMob.cpp:1082).
	if got, want := NextLevelExpTier(50, classArch), NextLevelExp(50); got != want {
		t.Errorf("NextLevelExpTier(50, arch) = %d, want %d (mortal curve)", got, want)
	}
	// Celestial anda na segunda curva.
	if got, want := NextLevelExpTier(1, classCelestial), nextLevel2[2]; got != want {
		t.Errorf("NextLevelExpTier(1, celestial) = %d, want %d (curve2[2])", got, want)
	}
	// No teto celestial ou acima dele devolve o topo da curva (sem nível seguinte).
	if got, want := NextLevelExpTier(MaxCLevel, classCelestial), nextLevel2[len(nextLevel2)-1]; got != want {
		t.Errorf("NextLevelExpTier(MaxCLevel, celestial) = %d, want ceiling %d", got, want)
	}
}
