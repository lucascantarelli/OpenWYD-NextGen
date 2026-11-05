package content

import (
	"strconv"
	"strings"

	"openwyd-nextgen/internal/ratetables"
)

// ApplyRateOverrides folds the moderator overlay (issue 36) into the tables
// loaded from Release/Common/Settings.
//
// The identity of a cell is (file, family, name) — the same key the web editor
// and content_rate_override use — never a numeric position: CompRate has several
// keys per family, and a positional key would make them collide. Rows that do not
// fit the typed tables are counted as ignored instead of being invented: the
// world only ever runs a table shape the client understands.
func ApplyRateOverrides(sanc *SancRate, comp *CompRate, quests *QuestRates, overrides []ratetables.Override) (applied, ignored int) {
	for _, o := range overrides {
		switch o.File {
		case ratetables.FileSancRate:
			anvil, idx, ok := sancCell(o.Family, o.Name, sanc)
			if !ok {
				ignored++
				continue
			}
			sanc.rates[anvil][idx] = int(o.Value)
			applied++

		case ratetables.FileCompRate:
			if comp == nil || o.Family == "" || o.Name == "" {
				ignored++
				continue
			}
			if comp.rates == nil {
				comp.rates = make(map[string]map[string]int)
			}
			if comp.rates[o.Family] == nil {
				comp.rates[o.Family] = make(map[string]int)
			}
			comp.rates[o.Family][o.Name] = int(o.Value)
			applied++

		case ratetables.FileQuestsRate:
			if quests == nil {
				ignored++
				continue
			}
			if !applyQuestOverride(quests, o.Family, o.Name, int32(o.Value)) {
				ignored++
				continue
			}
			applied++

		default:
			ignored++
		}
	}
	return applied, ignored
}

// sancCell resolves "<anvil>/<index>" against the table the file produced. The
// family names are the canonical ones ratetables emits (PO, PL, AMAGO).
func sancCell(family, name string, sanc *SancRate) (int, int, bool) {
	if sanc == nil {
		return 0, 0, false
	}
	var anvil int
	switch strings.ToUpper(family) {
	case "PO":
		anvil = sancAnvilOri
	case "PL":
		anvil = sancAnvilLac
	case "AMAGO":
		anvil = sancAnvilAma
	default:
		return 0, 0, false
	}
	idx, err := strconv.Atoi(name)
	if err != nil || idx < 0 || idx >= len(sanc.rates[anvil]) {
		return 0, 0, false
	}
	return anvil, idx, true
}

// applyQuestOverride writes one QuestsRate cell: EXP "<tier>.0|1" (mortal/arch),
// COIN "<tier>.0" and LEVEL "<tier>.<0..3>" (mortal min/max, arch min/max).
func applyQuestOverride(quests *QuestRates, family, name string, value int32) bool {
	if value < 0 {
		return false
	}
	tierPart, columnPart, ok := strings.Cut(name, ".")
	if !ok {
		return false
	}
	tier, err := strconv.Atoi(tierPart)
	if err != nil || tier < 0 || tier >= len(quests.tiers) {
		return false
	}
	column, err := strconv.Atoi(columnPart)
	if err != nil {
		return false
	}
	r := &quests.tiers[tier]

	switch strings.ToUpper(family) {
	case "EXP":
		switch column {
		case 0:
			r.MortalExp = int64(value)
		case 1:
			r.ArchExp = int64(value)
		default:
			return false
		}
		return true
	case "COIN":
		if column != 0 {
			return false
		}
		r.Coin = value
		return true
	case "LEVEL":
		switch column {
		case 0:
			r.MortalMin = value
		case 1:
			r.MortalMax = value
		case 2:
			r.ArchMin = value
		case 3:
			r.ArchMax = value
		default:
			return false
		}
		return true
	}
	return false
}
