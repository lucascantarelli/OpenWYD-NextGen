// Package mobcarry applies moderator-edited Carry[] tables onto raw STRUCT_MOB
// template bytes (issue 34, fatia 3) — the drop half of the same editing
// surface mobstat covers for stats.
//
// The Carry[] table is a mob's drop table: the SLOT decides the item's rarity
// (DropRate[slot] is the divisor) and the item in it is what falls. Editing it
// from the web is an overlay row in mob_carry_override; the file in Release/ is
// never written. The overlay is applied at template load, once at boot, behind
// -mob-carry-editing (OFF = files exactly as they are).
//
// Semantics are authoritative per template, like the Equip[] overlay: a slot
// not present in the override is empty, not "keep the file's item". The web-api
// always sends the whole table, so "not present" only happens when a moderator
// cleared the slot.
//
// RNG parity: this package only changes item values. It adds, removes and
// reorders no rand() call, so the drop sequence still matches the legacy server
// for the same table.
package mobcarry

import (
	"github.com/jeanluca/w2pp-openwyd/internal/savefmt"
)

// Slot is one Carry[] position override (0..63, savefmt.MaxCarry).
type Slot struct {
	Slot      int
	ItemIndex uint16
	Eff       [3][2]uint8
}

// Override is a template's whole Carry[] table.
type Override struct {
	Slots []Slot
}

// Apply returns template with its Carry[] replaced by ov, encoded as the
// canonical 816-byte layout. A corrupt template is returned unchanged; a slot
// out of range is ignored rather than panicking (the web validates first, so an
// ignored slot means the table was edited out of band).
func Apply(template []byte, ov Override) []byte {
	mob, _, err := savefmt.DecodeMobAny(template)
	if err != nil {
		return template
	}

	mob.Carry = [savefmt.MaxCarry]savefmt.Item{}
	for _, s := range ov.Slots {
		if s.Slot < 0 || s.Slot >= savefmt.MaxCarry || s.ItemIndex == 0 {
			continue
		}
		mob.Carry[s.Slot] = savefmt.Item{
			Index: int16(s.ItemIndex),
			Effects: [3]savefmt.Effect{
				{Effect: s.Eff[0][0], Value: s.Eff[0][1]},
				{Effect: s.Eff[1][0], Value: s.Eff[1][1]},
				{Effect: s.Eff[2][0], Value: s.Eff[2][1]},
			},
		}
	}
	return savefmt.EncodeMob(mob)
}

// ApplyOverride applies overrides[name] to template if present, returning the
// template unchanged otherwise. Called from both template-resolution points in
// the tmServer (NPCGener spawn and the DB-managed NPC config loader), same as
// mobstat.ApplyOverride.
func ApplyOverride(template []byte, name string, overrides map[string]Override) []byte {
	if ov, ok := overrides[name]; ok {
		return Apply(template, ov)
	}
	return template
}
