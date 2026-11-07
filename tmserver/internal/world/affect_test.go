package world

import "testing"

// TestSetAffect e TestSetTick passam AffectDuration ZERO de propósito: esse é
// o modo fiel ao legado, então estes continuam testes de paridade da fórmula
// original de Server.cpp:9209/9256. O ajuste da issue #229 é coberto separadamente por
// TestAffectDurationPolicy.
func TestSetAffect(t *testing.T) {
	player := &Entity{ID: 1}
	// (AffectTime+1)*time/100: um buff de 150 ticks lançado com Delay 120 → 181 ticks.
	if !player.SetAffect(11, 5, 150, 0, 120, 33, AffectDuration{}) {
		t.Fatal("SetAffect refused a valid player buff")
	}
	got := player.Affect[0]
	if got.Type != 11 || got.Value != 5 || got.Level != 33 || got.Time != 181 {
		t.Errorf("Affect[0] = %+v, want type 11 value 5 level 33 time 181", got)
	}
	// Reaplicar o mesmo tipo reutiliza o slot (semântica de GetEmptyAffect).
	if !player.SetAffect(11, 5, 150, 0, 200, 40, AffectDuration{}) {
		t.Fatal("re-apply refused")
	}
	if player.Affect[0].Level != 40 || player.Affect[1].Type != 0 {
		t.Errorf("re-apply did not reuse slot 0: %+v / %+v", player.Affect[0], player.Affect[1])
	}

	// Criaturas nunca recebem SetAffect (Server.cpp:9211 `conn > MAX_USER`).
	mob := &Entity{ID: MaxUser + 5}
	if mob.SetAffect(11, 5, 150, 0, 120, 33, AffectDuration{}) {
		t.Error("SetAffect must refuse mobs")
	}
	// RSV_BLOCK barra só casts agressivos.
	blocked := &Entity{ID: 2, Rsv: RsvBlock}
	if blocked.SetAffect(20, 5, 150, 1, 120, 33, AffectDuration{}) {
		t.Error("aggressive affect must bounce off RSV_BLOCK")
	}
	if !blocked.SetAffect(11, 5, 150, 0, 120, 33, AffectDuration{}) {
		t.Error("friendly affect must pass RSV_BLOCK")
	}
}

func TestSetTick(t *testing.T) {
	// Criaturas RECEBEM ticks (veneno) — só NPCs mercadores ficam de fora.
	mob := &Entity{ID: MaxUser + 5}
	if !mob.SetTick(20, 75, 150, 1, 120, 33, AffectDuration{}) {
		t.Fatal("SetTick must allow mobs")
	}
	if got := mob.Affect[0]; got.Type != 20 || got.Value != 75 || got.Time != 181 {
		t.Errorf("tick slot = %+v, want type 20 value 75 time 181", got)
	}
	merchant := &Entity{ID: MaxUser + 6, Merchant: 1}
	if merchant.SetTick(20, 75, 150, 1, 120, 33, AffectDuration{}) {
		t.Error("SetTick must refuse merchant NPCs")
	}
	// Tipos 1/3/10 limitam timers longos a 2 ticks.
	p := &Entity{ID: 3}
	if !p.SetTick(3, 10, 150, 0, 120, 5, AffectDuration{}) {
		t.Fatal("SetTick refused")
	}
	if p.Affect[0].Time != 2 {
		t.Errorf("type-3 tick time = %d, want clamp 2", p.Affect[0].Time)
	}
	if !p.SetTick(10, 10, 150, 0, 120, 5, AffectDuration{}) {
		t.Fatal("SetTick refused type 10")
	}
	if p.Affect[1].Time != 2 {
		t.Errorf("type-10 tick time = %d, want clamp 2", p.Affect[1].Time)
	}
	// Um HoT (tipo 17) NÃO é tipo de clamp 1/3/10 → mantém o timer
	// (affectTime+1)*delay/100 completo (um tick = 8s de tempo real).
	hoT := &Entity{ID: 4}
	if !hoT.SetTick(17, 40, 150, 0, 120, 100, AffectDuration{}) {
		t.Fatal("SetTick refused a HoT")
	}
	if got := hoT.Affect[0]; got.Type != 17 || got.Time != 181 {
		t.Errorf("HoT tick = %+v, want type 17 time 181 (unclamped)", got)
	}
}

// TestAffectDurationPolicy fixa o ajuste da issue #229 no nível unitário: a
// escala, o teto de maestria, o piso só-amistoso, e o fato de os casos
// especiais legados (os clamps 1/3/10 e o sentinela "infinito") NÃO serem ajustados.
func TestAffectDurationPolicy(t *testing.T) {
	// Padrões de produção (cmd/tmserver/main.go): 15%, piso 60s, teto 10min.
	prod := AffectDuration{ScalePct: 15, MinTicks: 8, MaxTicks: 75}
	tests := []struct {
		name       string
		dur        AffectDuration
		affectTime int
		aggressive int
		delay      int
		want       uint32
	}{
		// Escudo Mágico (43) e similares: AffectTime bruto 600 → 150 depois da
		// ÷4 legada. O Delay é 100+Special, então estes percorrem a curva de maestria da
		// captura da #229 (Special 226 media ≈ 33 min antes).
		{"long buff, no mastery", prod, 150, 0, 100, 22},  // 2m56
		{"long buff, special 100", prod, 150, 0, 200, 45}, // 6m00
		{"long buff, special 226", prod, 150, 0, 326, 73}, // 9m44
		{"long buff, special 255", prod, 150, 0, 355, 75}, // limitado a 10m
		{"long buff, special 400", prod, 150, 0, 500, 75}, // limitado a 10m
		// Escudo Dourado (85): bruto 30 → 7. Sua duração legada já cabe sob
		// o teto, então mantém a curva de maestria legada em vez de ser escalado
		// para o piso — a segunda metade da issue #236.
		{"short buff, no mastery", prod, 7, 0, 100, 8},   // 64s (no piso)
		{"short buff, special 100", prod, 7, 0, 200, 16}, // 2m08, escalando de novo
		{"short buff, special 400", prod, 7, 0, 500, 40}, // 5m20
		// Enfraquecer (51): bruto 5 → 1, agressivo. O piso NÃO pode valer, senão
		// o ajuste ALONGARIA um debuff.
		{"short debuff is not floored", prod, 1, 1, 100, 1},
		// …e também não pode o bypass da banda alvo: um affect hostil fica na
		// escala fixa mesmo que sua duração legada fique bem abaixo do teto.
		// Perseguição (16) no teto de Special é 5 ticks legados, mantido em 1.
		{"short debuff is still scaled", prod, 0, 1, 500, 1},
		// Valor zero = fórmula legada, intocada.
		{"zero value is legacy", AffectDuration{}, 150, 0, 500, 755},
		{"scale 100 is legacy", AffectDuration{ScalePct: 100}, 150, 0, 500, 755},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &Entity{ID: 1}
			if !e.SetAffect(11, 5, tt.affectTime, tt.aggressive, tt.delay, 0, tt.dur) {
				t.Fatal("SetAffect refused")
			}
			if got := e.Affect[0].Time; got != tt.want {
				t.Errorf("Time = %d ticks (%ds), want %d (%ds)",
					got, got*8, tt.want, tt.want*8)
			}
		})
	}
}

// TestAffectDurationSkipsLegacyClamps: o ajuste só toca o
// ramo normal. Os clamps 1/3/10 e o sentinela "infinito" mantêm seus valores
// legados exatos, então uma escala de 15% não pode virar um clamp de 4 ticks em 0.
func TestAffectDurationSkipsLegacyClamps(t *testing.T) {
	dur := AffectDuration{ScalePct: 15, MinTicks: 8, MaxTicks: 75}

	// SetAffect: um slot cujo tipo ANTERIOR é 1/3/10 é limitado a 4 ticks.
	e := &Entity{ID: 1}
	e.Affect[0] = Affect{Type: 3, Time: 99}
	if !e.SetAffect(3, 5, 150, 0, 500, 0, dur) {
		t.Fatal("SetAffect refused")
	}
	if got := e.Affect[0].Time; got != 4 {
		t.Errorf("sType clamp = %d, want the legacy 4 (untuned)", got)
	}

	// SetAffect: o sentinela "infinito" sobrevive ao teto.
	inf := &Entity{ID: 2}
	if !inf.SetAffect(34, 0, 150, 0, 2139062143, 1, dur) {
		t.Fatal("SetAffect refused the sentinel")
	}
	if got := inf.Affect[0].Time; got != 2139062143 {
		t.Errorf("infinite sentinel = %d, want 2139062143 (untuned)", got)
	}

	// SetTick: tipos 1/3/10 são limitados a 2 ticks.
	p := &Entity{ID: 3}
	if !p.SetTick(3, 10, 150, 0, 500, 5, dur) {
		t.Fatal("SetTick refused")
	}
	if got := p.Affect[0].Time; got != 2 {
		t.Errorf("tick clamp = %d, want the legacy 2 (untuned)", got)
	}
}
