package savefmt

import (
	"os"
	"path/filepath"
	"testing"
)

// Templates reais tirados de tmserver/content/TMsrv/run/npc/. O par com prefixo @
// são os registros legados de 756 bytes de mobs que também são entregues no layout
// canônico, e é isso que torna possível o teste de paridade entre layouts abaixo.
const (
	fxLegacy756Gargula = "mob_legacy756_gargula.bin"      // @Gargula, monster
	fxLegacy756Tauron  = "mob_legacy756_tauron.bin"       // @Tauron, monster
	fxLegacy756Shop    = "mob_legacy756_shop_armas_d.bin" // Armas[D], weapon shop
	fxLegacy920NPC1    = "mob_legacy920_npc_1.bin"        // NPC_1, 756 record + junk
	fxCurrent816Gargul = "mob_current816_gargula.bin"     // Gargula, canonical
	fxCurrent816Tauron = "mob_current816_tauron.bin"      // Tauron, canonical
)

func fixture(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	return b
}

func TestDetectMobVersion(t *testing.T) {
	tests := []struct {
		size int
		want MobVersion
	}{
		{MobSize, MobVersionCurrent},
		{MobSizeLegacy756, MobVersionLegacy756},
		{MobSizeLegacy756Padded, MobVersionLegacy756Padded},
		{0, MobVersionUnknown},
		{3, MobVersionUnknown},
		{MobSize - 1, MobVersionUnknown},
		{AccountFileSize, MobVersionUnknown},
	}
	for _, tt := range tests {
		if got := DetectMobVersion(tt.size); got != tt.want {
			t.Errorf("DetectMobVersion(%d) = %v, want %v", tt.size, got, tt.want)
		}
	}
}

// TestDetectMobVersionMatchesFixtures fixa o roteamento tamanho→versão contra os
// três layouts que de fato acompanham a árvore de conteúdo.
func TestDetectMobVersionMatchesFixtures(t *testing.T) {
	tests := []struct {
		file string
		want MobVersion
	}{
		{fxCurrent816Gargul, MobVersionCurrent},
		{fxLegacy756Gargula, MobVersionLegacy756},
		{fxLegacy920NPC1, MobVersionLegacy756Padded},
	}
	for _, tt := range tests {
		b := fixture(t, tt.file)
		if got := DetectMobVersion(len(b)); got != tt.want {
			t.Errorf("%s (%d bytes): got %v, want %v", tt.file, len(b), got, tt.want)
		}
	}
}

func TestMobVersionString(t *testing.T) {
	tests := []struct {
		v    MobVersion
		want string
	}{
		{MobVersionCurrent, "current-816"},
		{MobVersionLegacy756, "legacy-756"},
		{MobVersionLegacy756Padded, "legacy-756+pad920"},
		{MobVersionUnknown, "unknown"},
		{MobVersion(99), "unknown"},
	}
	for _, tt := range tests {
		if got := tt.v.String(); got != tt.want {
			t.Errorf("MobVersion(%d).String() = %q, want %q", tt.v, got, tt.want)
		}
	}
}
