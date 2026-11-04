// Package droprate holds the base drop divisors per Carry slot — the table the
// game formulas and the web editor must agree on.
//
// It lives in the module-level internal/ (not tmserver/internal) because both
// sides need it: tmserver/internal/loot copies this table into its mutable
// DropRate (which the moderator overlay rewrites at boot) and
// webserver/internal/dropadmin shows these values as the "base" the editor is
// overriding. Real values from Basedef.cpp:222 (game-rules.md §2.2).
//
// The value is a DIVISOR: the bigger it is, the rarer the drop (1 = always).
package droprate

// MaxRate is the largest accepted divisor; mirrors the CHECK on
// drop_rate_override.value. A typo above it is rejected instead of clamped.
const MaxRate = 1_000_000

// Rate is g_pDropRate[64]: base odds per Carry slot of the mob, NOT per item.
var Rate = build()

func build() [64]int {
	var out [64]int
	fill := func(from, to, v int) {
		for i := from; i <= to; i++ {
			out[i] = v
		}
	}
	fill(0, 7, 900)     // common equip
	fill(8, 11, 4)      // very common (gold/potion)
	fill(12, 15, 900)   //
	fill(16, 23, 20000) // ultra rare
	fill(24, 47, 2000)  //
	fill(48, 55, 3000)  //
	out[56] = 1         // always drops
	for i, v := range []int{35, 500, 2500, 5000, 5000, 10000, 20000} {
		out[57+i] = v
	}
	return out
}
