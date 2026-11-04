// Package dropcfg folds the moderator drop-rate overlay (issue 34) into the
// code defaults of package loot.
//
// DropRate is a divisor per Carry slot (1 = always, 20000 = ultra rare) and it is
// **code**, not content: loot.DropRate is filled in its init() from
// Basedef.cpp:222. The overlay therefore replaces values in that table instead of
// parsing a file, and it must be applied exactly once, at boot, before the world
// loop starts — the same rule as -rates-editing and -mob-stat-editing.
//
// RNG parity: this package only writes numbers. No rand() call is added, removed
// or reordered, so a boot with the overlay consumes the same sequence as one
// without it (the difference is which slots can drop).
package dropcfg

import (
	"github.com/jeanluca/w2pp-openwyd/internal/droprate"
	"github.com/jeanluca/w2pp-openwyd/tmserver/internal/loot"
)

// MaxRate is the largest accepted divisor, matching the CHECK on
// drop_rate_override.value and the web editor's range. Anything above it is
// treated as invalid instead of silently clamping: a typo that makes a drop
// impossible should be visible.
const MaxRate = droprate.MaxRate

// Override is one edited Carry slot as the dbServer serves it.
type Override struct {
	Slot  int
	Value int64
}

// Apply writes every valid override into loot.DropRate and reports how many were
// applied/ignored. Out-of-range slots and values are ignored (and counted), never
// clamped: the portal already validates the same range, so an ignored row means
// the table was edited out of band.
func Apply(overrides []Override) (applied, ignored int) {
	for _, o := range overrides {
		if o.Slot < 0 || o.Slot >= len(loot.DropRate) || o.Value < 1 || o.Value > MaxRate {
			ignored++
			continue
		}
		loot.DropRate[o.Slot] = int(o.Value)
		applied++
	}
	return applied, ignored
}

// Snapshot copies the current table, for logging and for tests that need to
// restore the code defaults.
func Snapshot() [64]int { return loot.DropRate }
