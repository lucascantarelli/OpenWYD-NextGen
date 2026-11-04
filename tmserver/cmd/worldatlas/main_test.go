package main

import (
	"testing"

	"github.com/jeanluca/w2pp-openwyd/tmserver/internal/content"
)

// smallWorld builds a 2×2 attribute grid over an 8×8 height grid (each attribute
// cell covers 4×4 blocks). The lower-right attribute cell is walkable and has
// relief; the rest is blocked or flat.
func smallWorld() (*content.Grid, *content.Grid) {
	attrs := &content.Grid{Dim: 2, Data: []byte{127, 127, 127, 0}}
	heights := &content.Grid{Dim: 8, Data: make([]byte, 64)}
	for y := 4; y < 8; y++ {
		for x := 4; x < 8; x++ {
			heights.Data[y*8+x] = 50
		}
	}
	return heights, attrs
}

// TestPlayableBounds: only walkable cells WITH relief count, and the box comes
// back in world coordinates (one attribute cell = 4 blocks).
func TestPlayableBounds(t *testing.T) {
	heights, attrs := smallWorld()

	all := playableBounds(heights, attrs, nil)
	if all == nil || *all != (bounds{X1: 4, Y1: 4, X2: 8, Y2: 8}) {
		t.Fatalf("playableBounds = %+v, want 4,4 → 8,8", all)
	}

	// Clip ending exactly where the cell starts: exclusive on the far edge.
	if got := playableBounds(heights, attrs, &bounds{X1: 0, Y1: 0, X2: 4, Y2: 4}); got != nil {
		t.Fatalf("playableBounds(clip before the cell) = %+v, want nil", got)
	}
}

// TestPlayableBoundsIgnoresVoid is the regression that made `bounds` wrap the
// whole world: the AttributeMap reports "free" outside the islands, so without
// the height check the box was 0..4096 and cropped nothing.
func TestPlayableBoundsIgnoresVoid(t *testing.T) {
	_, attrs := smallWorld()
	flat := &content.Grid{Dim: 8, Data: make([]byte, 64)}

	if got := playableBounds(flat, attrs, nil); got != nil {
		t.Fatalf("playableBounds(no relief) = %+v, want nil", got)
	}
}
