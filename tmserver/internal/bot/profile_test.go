package bot

import "testing"

func TestDefaultProfilesAreValid(t *testing.T) {
	for name, p := range DefaultProfiles() {
		if p.Name != name {
			t.Fatalf("perfil %s com Name %s", name, p.Name)
		}
		if err := p.Validate(); err != nil {
			t.Fatalf("perfil %s inválido: %v", name, err)
		}
	}
}

func TestProfileRejectsFleeAfterHeal(t *testing.T) {
	p := Profile{Name: ProfileFarmer, HealBelow: 0.2, FleeBelow: 0.8}
	if err := p.Validate(); err == nil {
		t.Fatal("perfil que recua antes de curar foi aceito")
	}
}

func TestRNGIsReproducibleForSameSeed(t *testing.T) {
	a, b := NewRNG(42), NewRNG(42)
	for i := 0; i < 100; i++ {
		if a.Float64() != b.Float64() {
			t.Fatalf("sequência divergiu no passo %d", i)
		}
	}
	c := NewRNG(43)
	same := true
	x, y := NewRNG(7), c
	for i := 0; i < 10; i++ {
		if x.Float64() != y.Float64() {
			same = false
			break
		}
	}
	if same {
		t.Fatal("sementes diferentes produziram a mesma sequência")
	}
}

func TestRNGChanceBounds(t *testing.T) {
	g := NewRNG(1)
	for i := 0; i < 100; i++ {
		if g.Chance(0) {
			t.Fatal("Chance(0) devolveu true")
		}
		if !g.Chance(1) {
			t.Fatal("Chance(1) devolveu false")
		}
	}
}

func TestMixPopulationSumsToRequested(t *testing.T) {
	weights := map[ProfileName]float64{
		ProfileFarmer:      0.3,
		ProfileCombatente:  0.2,
		ProfileComerciante: 0.15,
		ProfileCasual:      0.15,
		ProfileExplorador:  0.1,
		ProfileSocial:      0.1,
	}
	for _, n := range []int{1, 7, 20, 137} {
		got, err := MixPopulation(n, weights)
		if err != nil {
			t.Fatalf("MixPopulation(%d): %v", n, err)
		}
		if len(got) != n {
			t.Fatalf("MixPopulation(%d) devolveu %d perfis", n, len(got))
		}
	}
}

func TestMixPopulationIsDeterministic(t *testing.T) {
	weights := map[ProfileName]float64{ProfileFarmer: 1, ProfileSocial: 1, ProfileQuest: 1}
	a, _ := MixPopulation(10, weights)
	b, _ := MixPopulation(10, weights)
	for i := range a {
		if a[i] != b[i] {
			t.Fatalf("posição %d divergiu: %s != %s", i, a[i], b[i])
		}
	}
}

func TestMixPopulationRejectsBadInput(t *testing.T) {
	if _, err := MixPopulation(0, map[ProfileName]float64{ProfileCasual: 1}); err == nil {
		t.Fatal("população zero foi aceita")
	}
	if _, err := MixPopulation(5, map[ProfileName]float64{ProfileCasual: 0}); err == nil {
		t.Fatal("população com peso zero foi aceita")
	}
	if _, err := MixPopulation(5, map[ProfileName]float64{"inexistente": 1}); err == nil {
		t.Fatal("perfil desconhecido foi aceito")
	}
	if _, err := MixPopulation(5, map[ProfileName]float64{ProfileCasual: -1}); err == nil {
		t.Fatal("peso negativo foi aceito")
	}
}
