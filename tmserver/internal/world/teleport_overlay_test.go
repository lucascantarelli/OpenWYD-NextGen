package world

import (
	"testing"

	"github.com/jeanluca/w2pp-openwyd/internal/teleportdata"
	"github.com/jeanluca/w2pp-openwyd/tmserver/internal/rng"
)

// TestTeleportOverlayOverridesAndDisables covers issue 42's acceptance criteria
// at the world boundary: a row replaces the compiled route, a disabled row stops
// the tile from teleporting, untouched origins keep the compiled route, and
// clearing the overlay restores the compiled table exactly.
func TestTeleportOverlayOverridesAndDisables(t *testing.T) {
	defer SetTeleportOverlay(nil)

	// Override Armia → Noatum: new destination and a 1000 gold cost.
	if applied := SetTeleportOverlay([]OverlayRoute{{
		SrcX: 2116, SrcY: 2100, DstX: 1200, DstY: 1300, Cost: 1000,
		Requirement: teleportdata.RequirementNone, Enabled: true,
	}}); applied != 1 {
		t.Fatalf("SetTeleportOverlay applied = %d, want 1", applied)
	}
	dx, dy, cost, ok := TeleportDest(2118, 2102) // rounds to 2116,2100
	if !ok || cost != 1000 {
		t.Fatalf("overlaid Armia route: ok=%v cost=%d, want ok cost=1000", ok, cost)
	}
	if dx < 1200 || dx > 1202 || dy < 1300 || dy > 1302 {
		t.Fatalf("overlaid destination = %d,%d, want (1200..1202,1300..1302)", dx, dy)
	}

	// An origin with no overlay row still resolves from the compiled table.
	if _, _, cost, ok := TeleportDest(1044, 1724); !ok || cost != 0 {
		t.Fatalf("compiled Noatum→Armia: ok=%v cost=%d, want ok cost=0", ok, cost)
	}

	// Disabling a route removes the tile entirely.
	SetTeleportOverlay([]OverlayRoute{{SrcX: 2480, SrcY: 1716, Enabled: false}})
	if _, _, _, ok := TeleportDest(2480, 1716); ok {
		t.Fatal("disabled Azran route still teleported")
	}

	// Clearing the overlay restores the compiled route and its 700 gold cost.
	SetTeleportOverlay(nil)
	if _, _, cost, ok := TeleportDest(2480, 1716); !ok || cost != 700 {
		t.Fatalf("after clear: ok=%v cost=%d, want ok cost=700", ok, cost)
	}
}

// TestTeleportOverlayKeepsFairyGateAndRNGOrder proves the overlay cannot weaken a
// gate and that the destination spread keeps consuming the world's MSVC stream
// in the same position (issue 42 forbids changing the RNG order).
func TestTeleportOverlayKeepsFairyGateAndRNGOrder(t *testing.T) {
	defer SetTeleportOverlay(nil)

	SetTeleportOverlay([]OverlayRoute{{
		SrcX: 2548, SrcY: 1740, DstX: 2281, DstY: 3688, Cost: 0,
		Requirement: teleportdata.RequirementFairy, Enabled: true,
	}})

	// Without the fairy: no route and no RNG consumption.
	blocked := rng.NewSeeded(7)
	if _, _, _, ok := TeleportDestWithAccess(2548, 1740, TeleportAccess{}, blocked); ok {
		t.Fatal("overlaid Vale gate resolved without the fairy")
	}
	if got, want := blocked.Intn(1000), rng.NewSeeded(7).Intn(1000); got != want {
		t.Fatalf("blocked portal consumed the MSVC stream: got %d want %d", got, want)
	}

	// With the fairy: destination is base + the next two draws, in order.
	allowed := rng.NewSeeded(7)
	want := rng.NewSeeded(7)
	dx, dy, _, ok := TeleportDestWithAccess(2548, 1740, TeleportAccess{HasFairy: true}, allowed)
	if !ok {
		t.Fatal("overlaid Vale route did not resolve with the fairy")
	}
	if dx != 2281+int16(want.Intn(3)) || dy != 3688+int16(want.Intn(3)) {
		t.Fatalf("spread order changed: got (%d,%d)", dx, dy)
	}
}
