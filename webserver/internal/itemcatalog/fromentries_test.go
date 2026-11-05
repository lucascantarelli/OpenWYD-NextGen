package itemcatalog

import (
	"testing"

	"internal/domain"
)

// TestFromEntriesMapsDatabaseRows guards the database → web shape conversion that
// replaced the ItemList.csv scan (issue 62): mesh/texture still come from the raw
// CSV field, nPos keeps its 16-bit masking (capes arrive negative) and the base
// effects come pre-resolved from item_catalog.
func TestFromEntriesMapsDatabaseRows(t *testing.T) {
	rows := []domain.ItemCatalogEntry{
		{
			Index:  100,
			Name:   "Poção_Vermelha",
			Grade:  1,
			Pos:    64,
			SHA256: "aa",
			Fields: []string{"100", "Poção_Vermelha", "12.3", "0", "0", "0", "64", "0", "1"},
			Effects: []domain.ItemCatalogEffect{
				{Eff: 5, Val: 30},
			},
		},
		{Index: 200, Name: "Capa_Rara", Grade: 4, Pos: -32768, SHA256: "bb"},
	}

	cat := FromEntries(rows)
	if len(cat.Items) != 2 {
		t.Fatalf("itens = %d, quer 2", len(cat.Items))
	}
	byIndex := make(map[int32]Entry, len(cat.Items))
	for _, it := range cat.Items {
		byIndex[it.Index] = it
	}

	potion := byIndex[100]
	if potion.Mesh != 12 || potion.Texture != 3 {
		t.Errorf("mesh.texture = %d.%d, quer 12.3", potion.Mesh, potion.Texture)
	}
	if potion.DisplayName != "Poção Vermelha" {
		t.Errorf("display = %q", potion.DisplayName)
	}
	if potion.SlotMask != 64 || len(potion.Slots) != 1 || potion.Slots[0] != "weapon" {
		t.Errorf("slot = %#x %v, quer 0x40 [weapon]", potion.SlotMask, potion.Slots)
	}
	if potion.Grade != 1 {
		t.Errorf("grade = %d, quer 1", potion.Grade)
	}
	if len(potion.Effects) != 1 || potion.Effects[0].Code != 5 || potion.Effects[0].Value != 30 {
		t.Errorf("effects = %+v", potion.Effects)
	}

	cape := byIndex[200]
	if cape.SlotMask != 0x8000 || len(cape.Slots) != 1 || cape.Slots[0] != "cape" {
		t.Errorf("capa: slot = %#x %v, quer 0x8000 [cape]", cape.SlotMask, cape.Slots)
	}

	if !cat.Has(100) || cat.Has(999) {
		t.Errorf("Has: 100=%v 999=%v", cat.Has(100), cat.Has(999))
	}
	if cat.Version == "" {
		t.Error("Version vazia")
	}
	if cat.Version != FromEntries(rows).Version {
		t.Error("Version não determinística")
	}
}

// TestFromEntriesSkipsBrokenRows keeps the tolerance the CSV scanner had: a row
// without an index or name is dropped instead of poisoning the catalog.
func TestFromEntriesSkipsBrokenRows(t *testing.T) {
	cat := FromEntries([]domain.ItemCatalogEntry{
		{Index: -1, Name: "negativo"},
		{Index: 7, Name: ""},
		{Index: 8, Name: "ok"},
	})
	if len(cat.Items) != 1 || cat.Items[0].Index != 8 {
		t.Fatalf("itens = %+v, quer só o 8", cat.Items)
	}
}
