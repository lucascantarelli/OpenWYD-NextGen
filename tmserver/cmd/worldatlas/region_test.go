package main

import "testing"

// TestRegionColorDistinct: 52 named regions need 52 distinguishable tints. The
// golden-angle walk must not repeat a colour over the whole catalog, which a
// fixed palette could not promise.
func TestRegionColorDistinct(t *testing.T) {
	seen := make(map[string]int, 64)
	for i := 0; i < 64; i++ {
		c := regionColor(i)
		if len(c) != 7 || c[0] != '#' {
			t.Fatalf("regionColor(%d) = %q, want #rrggbb", i, c)
		}
		if other, dup := seen[c]; dup {
			t.Fatalf("regionColor(%d) = %q já usado por %d", i, c, other)
		}
		seen[c] = i
	}
}

// TestRegionColorStable pins the first tints: the mask PNG and the table swatch
// are drawn by different code paths, so a change here would silently
// desynchronise them.
func TestRegionColorStable(t *testing.T) {
	if got := regionColor(0); got != "#d36868" {
		t.Errorf("regionColor(0) = %q, want #d36868", got)
	}
	if got := regionColor(1); got != "#68d387" {
		t.Errorf("regionColor(1) = %q, want #68d387", got)
	}
}
