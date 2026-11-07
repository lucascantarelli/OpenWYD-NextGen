package world

import "testing"

func TestVillage(t *testing.T) {
	cases := []struct {
		x, y int16
		want int
	}{
		{2096, 2096, 0}, // Armia (área de spawn padrão do jogador)
		{2086, 2093, 0}, // ponto de spawn de Armia
		{2494, 1707, 1}, // Azran
		{2453, 2000, 2}, // Erion
		{3652, 3122, 3}, // Nippleheim
		{1050, 1706, 4}, // Noatum
		{0, 0, -1},      // ermos
		{3000, 3000, -1},
	}
	for _, c := range cases {
		if got := Village(c.x, c.y); got != c.want {
			t.Errorf("Village(%d,%d) = %d, want %d", c.x, c.y, got, c.want)
		}
	}
}

func TestPvPZone(t *testing.T) {
	cases := []struct {
		name      string
		x, y      int16
		wantArena int
		wantZone  bool
	}{
		{"armia pvp arena", 2135, 2147, -1, true}, // dentro dos limites da cidade de Armia
		{"armia square", 2110, 2095, -1, true},
		{"field east of armia", 2185, 2100, -1, false},
		{"armia guild war area", 220, 220, 0, true},
		{"noatum war area", 4005, 4005, 4, true},
		{"duel arena", 160, 4044, -1, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := Arena(c.x, c.y); got != c.wantArena {
				t.Errorf("Arena(%d,%d) = %d, want %d", c.x, c.y, got, c.wantArena)
			}
			if got := PvPZone(c.x, c.y); got != c.wantZone {
				t.Errorf("PvPZone(%d,%d) = %v, want %v", c.x, c.y, got, c.wantZone)
			}
		})
	}
}

func TestCitySpawn(t *testing.T) {
	// O spawn é a base CitySpawn + rand%15 → precisa cair na cidade e em Armia para
	// ids fora da faixa (a última cidade salva só ocupa 2 bits; Noatum=4 → Armia).
	for city := 0; city < 4; city++ {
		for i := 0; i < 50; i++ {
			x, y := CitySpawn(city)
			base := cities[city]
			if x < base.spawnX || x >= base.spawnX+15 || y < base.spawnY || y >= base.spawnY+15 {
				t.Fatalf("CitySpawn(%d) = %d,%d out of [%d,%d)+15", city, x, y, base.spawnX, base.spawnY)
			}
			if Village(x, y) != city {
				t.Fatalf("CitySpawn(%d) landed in village %d", city, Village(x, y))
			}
		}
	}
	x, y := CitySpawn(4) // Noatum não salvavel → Armia
	if Village(x, y) != 0 {
		t.Errorf("CitySpawn(4) should fall back to Armia, got village %d", Village(x, y))
	}
}

func TestSpawnMobClassifiesTownNPCsAsNonCombat(t *testing.T) {
	w := New(Config{GridDim: 4096}, slogDiscard(), nil, nil)

	neutral := w.SpawnMob(genMobTemplate(3), 2086, 2093) // ator amistoso de cidade
	if neutral < 0 {
		t.Fatal("neutral town NPC did not spawn")
	}
	if e := w.Entity(neutral); e == nil || !e.NonCombatNPC {
		t.Fatalf("neutral town NPC NonCombatNPC = %v, want true", e)
	}

	hostile := w.SpawnMob(genMobTemplate(1), 2087, 2093) // criatura hostil dentro de um retângulo de cidade
	if hostile < 0 {
		t.Fatal("hostile city mob did not spawn")
	}
	if e := w.Entity(hostile); e == nil || e.NonCombatNPC {
		t.Fatalf("hostile city mob NonCombatNPC = %v, want false", e)
	}

	merchant := w.SpawnMob(genMerchantTemplate(1), 0, 0) // mercadores não combatem em lugar nenhum
	if merchant < 0 {
		t.Fatal("merchant NPC did not spawn")
	}
	if e := w.Entity(merchant); e == nil || !e.NonCombatNPC {
		t.Fatalf("merchant NonCombatNPC = %v, want true", e)
	}
}
