package content

// ItemEffectOverride is one moderator-edited base effect (issue 44) as the world
// consumes it: Value replaces Effect on ItemIndex. The key is the ItemEffect.h
// code — the same key BaseEffects uses — not a position in the 3 STRUCT_ITEM
// slots, because a catalog row can carry up to a dozen pairs (item 1188 has six).
type ItemEffectOverride struct {
	ItemIndex int
	Effect    uint8
	Value     int16
}

// EffectOverrideStats counts what the boot-time overlay actually did, so the
// server log can show it instead of a bare row count.
type EffectOverrideStats struct {
	Replaced int // the item already declared this effect: value swapped
	Added    int // the item did not declare it: pair appended
	Ignored  int // code the score model cannot represent: left out on purpose
}

// scoreEffectCodes is the set of codes BaseEffects is able to emit (the efName
// values). An override for anything else is refused: handlers fold equipment into
// the score by walking exactly these codes, and a code injected here would be
// silently dead (worse, it would make the catalog claim a bonus the game never
// applies). EF_RANGE, EF_VOLATILE and friends have their own readers.
var scoreEffectCodes = func() map[uint8]struct{} {
	out := make(map[uint8]struct{}, len(efName))
	for _, code := range efName {
		out[code] = struct{}{}
	}
	return out
}()

// ApplyItemEffectOverrides folds the web overlay into the catalog effects.
//
// Precedence stays instance > overlay > catalog: this only rewrites the catalog
// layer, so an item's own refinements (stEffect/stValue) still win at score time.
// The input map is never mutated — an empty overlay returns the catalog as-is, so
// the flag being off and being on with no rows are indistinguishable.
func ApplyItemEffectOverrides(base map[int][]BaseEffect, rows []ItemEffectOverride) (map[int][]BaseEffect, EffectOverrideStats) {
	var stats EffectOverrideStats
	if len(rows) == 0 {
		return base, stats
	}

	out := make(map[int][]BaseEffect, len(base))
	for idx, effs := range base {
		out[idx] = effs
	}

	for _, row := range rows {
		if _, ok := scoreEffectCodes[row.Effect]; !ok {
			stats.Ignored++
			continue
		}
		// Copy-on-write per item: the catalog slices are shared with the caller.
		effs, seen := out[row.ItemIndex], false
		next := make([]BaseEffect, len(effs))
		copy(next, effs)
		for i := range next {
			if next[i].Eff == row.Effect {
				next[i].Val = row.Value
				seen = true
				break
			}
		}
		if seen {
			stats.Replaced++
		} else {
			next = append(next, BaseEffect{Eff: row.Effect, Val: row.Value})
			stats.Added++
		}
		out[row.ItemIndex] = next
	}
	return out, stats
}
