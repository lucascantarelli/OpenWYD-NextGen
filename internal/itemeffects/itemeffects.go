// Package itemeffects names the STRUCT_EFFECT.cEffect codes carried in a
// STRUCT_ITEM's Eff1..Eff3 fields, so an admin screen can say "Dano +70"
// instead of "2:70".
//
// The vocabulary is ItemEffect.h (Source/Code/ItemEffect.h), the same header the
// legacy server and the client compile against; names[] mirrors its
// `#define EF_*` constants one-to-one. labels[] is the short Portuguese label
// the portal shows, limited on purpose to the codes that appear as adds on real
// saves — the rest fall back to the constant name (Name) and, when the code has
// no constant at all, to "" so the caller can render the raw number.
//
// This is a leaf package: no imports beyond the standard library, so both the
// website (webserver) and the future editors of issue 44 can share it.
package itemeffects

import (
	"sort"
	"sync"
)

// names maps a code to its ItemEffect.h constant, without the EF_ prefix.
var names = map[int32]string{
	0: "NONE", 1: "LEVEL", 2: "DAMAGE", 3: "AC", 4: "HP", 5: "MP", 6: "EXP",
	7: "STR", 8: "INT", 9: "DEX", 10: "CON",
	11: "SPECIAL1", 12: "SPECIAL2", 13: "SPECIAL3", 14: "SPECIAL4",
	15: "SCORE14", 16: "SCORE15", 17: "POS", 18: "CLASS", 19: "R1SIDC", 20: "R2SIDC",
	21: "WTYPE", 22: "REQ_STR", 23: "REQ_INT", 24: "REQ_DEX", 25: "REQ_CON",
	26: "ATTSPEED", 27: "RANGE", 28: "PRICE", 29: "RUNSPEED", 30: "SPELL",
	31: "DURATION", 32: "PARM2", 33: "GRID", 34: "GROUND", 35: "CLAN",
	36: "HWORDCOIN", 37: "LWORDCOIN", 38: "VOLATILE", 39: "KEYID",
	40: "PARRY", 41: "HITRATE", 42: "CRITICAL", 43: "SANC", 44: "SAVEMANA",
	45: "HPADD", 46: "MPADD", 47: "REGENHP", 48: "REGENMP",
	49: "RESIST1", 50: "RESIST2", 51: "RESIST3", 52: "RESIST4",
	53: "ACADD", 54: "RESISTALL", 55: "BONUS", 56: "HWORDGUILD", 57: "LWORDGUILD",
	58: "QUEST", 59: "UNIQUE", 60: "MAGIC", 61: "AMOUNT", 62: "HWORDINDEX", 63: "LWORDINDEX",
	64: "INIT1", 65: "INIT2", 66: "INIT3", 67: "DAMAGEADD", 68: "MAGICADD",
	69: "HPADD2", 70: "MPADD2", 71: "CRITICAL2", 72: "ACADD2", 73: "DAMAGE2",
	74: "SPECIALALL", 75: "CURKILL", 76: "LTOTKILL", 77: "HTOTKILL", 78: "INCUBATE",
	79: "MOUNTLIFE", 80: "MOUNTHP", 81: "MOUNTSANC", 82: "MOUNTFEED", 83: "MOUNTKILL",
	84: "INCUDELAY", 85: "SUBGUILD", 87: "ITEMLEVEL", 91: "DONATE",
	100: "GRADE0", 101: "GRADE1", 102: "GRADE2", 103: "GRADE3", 104: "GRADE4", 105: "GRADE5",
	106: "WDAY", 107: "HOUR", 108: "MIN", 109: "YEAR", 110: "WMONTH",
	112: "MOBTYPE", 113: "ITEMTYPE", 126: "NOSANC", 127: "NOTRADE",
}

// labels are the portal's short names for the codes that actually show up as
// adds. Resist order follows the content (game-rules.md §refino): EF_RESIST1 is
// Fogo, 2 Gelo, 3 Sagrado, 4 Relâmpago — the same order Orb de Fogo/Gelo/Sagrada/
// Relâmpago rely on.
var labels = map[int32]string{
	2: "Dano", 3: "Defesa", 4: "HP", 5: "MP", 6: "EXP",
	7: "Força", 8: "Inteligência", 9: "Destreza", 10: "Constituição",
	11: "Especial 1", 12: "Especial 2", 13: "Especial 3", 14: "Especial 4",
	17: "Posição", 18: "Classe", 21: "Tipo de arma",
	26: "Vel. de ataque", 27: "Alcance", 29: "Vel. de movimento", 30: "Skill",
	33: "Espaço no cargo", 38: "Consumível", 40: "Aparo", 41: "Precisão",
	42: "Crítico", 43: "Refino", 44: "Economia de mana",
	45: "HP %", 46: "MP %", 47: "Regen. HP", 48: "Regen. MP",
	49: "Resist. fogo", 50: "Resist. gelo", 51: "Resist. sagrado", 52: "Resist. trovão",
	53: "Defesa add", 54: "Resist. todos", 58: "Quest", 59: "Único",
	60: "Ataque mágico", 61: "Quantidade", 67: "Dano add", 68: "Ataque mágico add",
	69: "HP % (2)", 70: "MP % (2)", 71: "Crítico (2)", 72: "Defesa add (2)", 73: "Dano (2)",
	74: "Todos os especiais", 78: "Choco", 87: "Nível do item", 91: "Donate",
	100: "Grade A", 101: "Grade B", 102: "Grade C", 103: "Grade D", 104: "Grade E",
	105: "Grade Arch", 112: "Tipo de mob", 113: "Tipo de item",
	126: "Sem refino", 127: "Intransferível",
}

// Type is one row of the vocabulary, for UIs that need the whole table.
type Type struct {
	Code  int32
	Name  string
	Label string
}

var (
	codesOnce sync.Once
	codes     map[string]int32
)

func codeIndex() map[string]int32 {
	codesOnce.Do(func() {
		codes = make(map[string]int32, len(names))
		for code, name := range names {
			codes["EF_"+name] = code
		}
	})
	return codes
}

// Code resolves a catalog token such as "EF_CRITICAL" to its numeric code.
func Code(token string) (int32, bool) {
	code, ok := codeIndex()[token]
	return code, ok
}

// Known reports whether the code exists in ItemEffect.h. A grant or an edit
// must never store a code the client cannot interpret.
func Known(code int32) bool {
	_, ok := names[code]
	return ok
}

// All returns the vocabulary sorted by code, for the portal's effect pickers.
func All() []Type {
	out := make([]Type, 0, len(names))
	for code, name := range names {
		out = append(out, Type{Code: code, Name: name, Label: Label(code)})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Code < out[j].Code })
	return out
}

// Name returns the ItemEffect.h constant for a code (without the EF_ prefix).
// Empty means the code is not in the header.
func Name(code int32) string { return names[code] }

// Label returns the portal label for a code: the curated Portuguese name when
// there is one, otherwise the raw constant. Empty means the code is unknown and
// the caller should show the number.
func Label(code int32) string {
	if label, ok := labels[code]; ok {
		return label
	}
	return names[code]
}
