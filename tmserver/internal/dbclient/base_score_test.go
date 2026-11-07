package dbclient

import (
	"testing"

	dbv1 "openwyd-nextgen/api/db/v1"
	"openwyd-nextgen/tmserver/internal/world"
)

// A base sem equipamento (migration 0027) atravessa o wire nos dois sentidos; um save
// sempre a carrega, e uma linha sem ela carrega com HasBase falso.
func TestBaseScoreMapping(t *testing.T) {
	c := characterSaveToProto(world.CharacterSave{Str: 212, BaseStr: 12, BaseInt: 13, BaseDex: 2512, BaseCon: 860,
		BaseMaxHP: 2000, BaseMaxMP: 500})
	if !c.GetHasBaseScore() || c.GetBaseStr() != 12 || c.GetBaseInt() != 13 || c.GetBaseDex() != 2512 ||
		c.GetBaseCon() != 860 || c.GetBaseMaxHp() != 2000 || c.GetBaseMaxMp() != 500 || c.GetStr() != 212 {
		t.Fatalf("save mapping %+v", c)
	}
	st, err := characterStateFromProto(c)
	if err != nil {
		t.Fatal(err)
	}
	if !st.HasBase || st.BaseStr != 12 || st.BaseInt != 13 || st.BaseDex != 2512 || st.BaseCon != 860 ||
		st.BaseMaxHP != 2000 || st.BaseMaxMP != 500 || st.Str != 212 {
		t.Fatalf("load mapping %+v", st)
	}
	if old, err := characterStateFromProto(&dbv1.Character{Str: 112}); err != nil || old.HasBase {
		t.Fatal("a row without the base must load with HasBase false")
	}
}

// Um save_celestial indecifrável precisa falhar o load (issue #410): carregar
// fichas guardadas em branco deixaria o próximo save sobrescrever as reais.
func TestLoadRejectsBadSaveCelestial(t *testing.T) {
	if _, err := characterStateFromProto(&dbv1.Character{SaveCelestial: []byte{9, 9, 9}}); err == nil {
		t.Fatal("a malformed save_celestial blob loaded")
	}
}
