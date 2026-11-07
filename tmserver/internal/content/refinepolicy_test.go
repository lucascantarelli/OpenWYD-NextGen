package content

import (
	"strings"
	"testing"

	"openwyd-nextgen/internal/ratetables"
)

// Os tetos de refino saíram do código no QA 05/11: o arquivo os define, uma
// linha fora da faixa é pulada e um teto de pó nunca excede o limite absoluto.
func TestParseRefinePolicy(t *testing.T) {
	p, err := ParseRefinePolicy(strings.NewReader(
		"# comentario\nPOLICY PO_MAX 4\nPOLICY HARD_CAP 12\nlixo\nPOLICY PL_MAX 99\nPOLICY GEM_LEVEL 8\n"))
	if err != nil {
		t.Fatalf("ParseRefinePolicy: %v", err)
	}
	if p.OriMax != 4 || p.HardCap != 12 || p.GemLevel != 8 {
		t.Errorf("policy = %+v, want PO_MAX 4 / HARD_CAP 12 / GEM_LEVEL 8", p)
	}
	if p.PLMax != DefaultRefinePolicy().PLMax {
		t.Errorf("PL_MAX = %d, want the default %d (99 is out of range)", p.PLMax, DefaultRefinePolicy().PLMax)
	}
}

func TestParseRefinePolicyClampsDustToHardCap(t *testing.T) {
	p, err := ParseRefinePolicy(strings.NewReader("POLICY HARD_CAP 8\nPOLICY PL_MAX 11\n"))
	if err != nil {
		t.Fatalf("ParseRefinePolicy: %v", err)
	}
	if p.PLMax != 8 {
		t.Errorf("PL_MAX = %d, want 8 (clamped to HARD_CAP)", p.PLMax)
	}
}

func TestApplyRefineOverrides(t *testing.T) {
	p, applied, ignored := ApplyRefineOverrides(DefaultRefinePolicy(), []ratetables.Override{
		{File: ratetables.FileRefineRate, Family: "POLICY", Name: "PO_MAX", Value: 9},
		{File: ratetables.FileRefineRate, Family: "POLICY", Name: "NOPE", Value: 1},
		{File: ratetables.FileSancRate, Family: "PO", Name: "3", Value: 50},
	})
	if applied != 1 || ignored != 1 {
		t.Errorf("applied=%d ignored=%d, want 1/1 (the SancRate row is not ours)", applied, ignored)
	}
	if p.OriMax != 9 {
		t.Errorf("OriMax = %d, want 9 from the overlay", p.OriMax)
	}
}
