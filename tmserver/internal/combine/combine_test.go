package combine

import (
	"testing"

	"openwyd-nextgen/tmserver/internal/rng"
)

type fixedRand struct{ v int }

func (f fixedRand) Intn(int) int { return f.v }

func TestRollExact(t *testing.T) {
	tests := []struct {
		roll, rate int
		wantValue  int
		wantOK     bool
	}{
		{41, 50, 41, true},   // 41 <= 50
		{41, 30, 41, false},  // 41 > 30
		{99, 99, 99, true},   // sem achatamento
		{100, 85, 85, true},  // achatado para 85
		{114, 99, 99, true},  // achatado para 99
		{114, 98, 99, false}, // achatado 99 > 98
	}
	for _, tt := range tests {
		v, ok := Roll(fixedRand{tt.roll}, tt.rate)
		if v != tt.wantValue || ok != tt.wantOK {
			t.Errorf("Roll(%d, rate %d) = (%d,%v), want (%d,%v)", tt.roll, tt.rate, v, ok, tt.wantValue, tt.wantOK)
		}
	}
}

// TestRollWithLCG amarra ao fluxo real: primeiro rand 41 ⇒ valor 41.
func TestRollWithLCG(t *testing.T) {
	if v, ok := Roll(rng.New(), 50); v != 41 || !ok {
		t.Errorf("Roll(LCG, 50) = (%d,%v), want (41,true)", v, ok)
	}
}

// TestRollDistribution valida o achatamento: em muitas amostras, valores 85..99
// ocorrem ~2x mais vezes por valor que 0..84, e nada cai em 100..114.
func TestRollDistribution(t *testing.T) {
	const n = 300000
	r := rng.New()
	var counts [RollModulo]int
	for i := 0; i < n; i++ {
		v, _ := Roll(r, 0) // taxa irrelevante; só inspecionamos o valor
		if v >= 100 {
			t.Fatalf("value %d landed in the flattened range [100,114]", v)
		}
		counts[v]++
	}
	low := 0
	for v := 0; v < 85; v++ {
		low += counts[v]
	}
	high := 0
	for v := 85; v < 100; v++ {
		high += counts[v]
	}
	lowPer := float64(low) / 85
	highPer := float64(high) / 15
	ratio := highPer / lowPer
	if ratio < 1.8 || ratio > 2.2 {
		t.Errorf("per-value ratio high/low = %.3f, want ~2.0", ratio)
	}
}
