package world

import (
	"encoding/binary"
	"testing"
)

// TestSpawnMobRange cobre a derivação de Entity.Range no spawn — o port de
// BASE_GetMobAbility(EF_RANGE) (Basedef.cpp:2415/2523): por equipamento, catálogo
// EF_RANGE (Config.ItemRanges) mais os efeitos EF_RANGE da instância, e o MAX
// entre os 16 equipamentos vence. Este é o caminho de produção (main liga
// ItemList.Ranges() no Config): criaturas arqueiras ganham seu alcance pelo
// item modelo de Equip[0], por exemplo Ciclope_Arqueiro 242 → EF_RANGE 4.
func TestSpawnMobRange(t *testing.T) {
	w := New(Config{GridDim: 32, ItemRanges: map[int]int16{242: 4}}, slogDiscard(), nil, nil)

	// Equip[0] = item 242 (alcance 4 do catálogo); Equip[1] = item 99 com um efeito
	// EF_RANGE=2 de instância (uma fonte menor — o máximo precisa vencer, não a soma
	// entre itens).
	tmpl := make([]byte, structMobTemplateSize)
	binary.LittleEndian.PutUint16(tmpl[140:], 242)
	binary.LittleEndian.PutUint16(tmpl[148:], 99)
	tmpl[150] = 27 // Eff[0] = EF_RANGE
	tmpl[151] = 2
	id := w.SpawnMob(tmpl, 5, 5)
	if got := w.Entity(id).Range; got != 4 {
		t.Errorf("Range with catalog gear = %d, want 4", got)
	}

	// Efeitos de catálogo e de instância no MESMO item se somam (BASE_GetItemAbility
	// soma o stEffect do catálogo e o stEffect da instância para o tipo consultado).
	tmpl2 := make([]byte, structMobTemplateSize)
	binary.LittleEndian.PutUint16(tmpl2[140:], 242)
	tmpl2[142] = 27 // EF_RANGE +3 de instância no mesmo item
	tmpl2[143] = 3
	id2 := w.SpawnMob(tmpl2, 6, 5)
	if got := w.Entity(id2).Range; got != 7 {
		t.Errorf("Range with catalog+instance on one item = %d, want 7", got)
	}

	// Sem equipamento (ou sem catálogo): Range fica 0 → a IA cai para o alcance corpo a corpo.
	id3 := w.SpawnMob(make([]byte, structMobTemplateSize), 7, 5)
	if got := w.Entity(id3).Range; got != 0 {
		t.Errorf("Range with no gear = %d, want 0", got)
	}
}
