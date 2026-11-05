package world

import (
	"math/rand"
	"sync/atomic"

	"openwyd-nextgen/internal/teleportdata"
	"openwyd-nextgen/tmserver/internal/rng"
)

type teleRoute struct {
	dx, dy int16
	cost   int32
}

// teleportTable maps a rounded origin tile (x&0xFFFC, y&0xFFFC) to its
// destination + gold cost (GetTeleportPosition, GetFunc.cpp). The client sends an
// empty _MSG_ReqTeleport when it steps on a teleport tile; the server resolves
// the route from the player's position.
//
// The data itself lives in internal/teleportdata so the web-api can list and
// simulate the same table (issue 42); this map is the runtime form of it.
var teleportTable = buildTeleportTable()

func buildTeleportTable() map[[2]int16]teleRoute {
	routes := teleportdata.Routes()
	m := make(map[[2]int16]teleRoute, len(routes))
	for _, r := range routes {
		m[[2]int16{r.SrcX, r.SrcY}] = teleRoute{dx: r.DstX, dy: r.DstY, cost: r.Cost}
	}
	return m
}

// OverlayRoute is one moderator-editable route row (teleport_route, issue 42).
// The overlay is additive: a row overrides the compiled route at the same origin
// tile, Enabled=false removes it, and an origin with no row keeps the compiled
// route.
type OverlayRoute struct {
	SrcX, SrcY  int16
	DstX, DstY  int16
	Cost        int32
	Requirement string
	Enabled     bool
}

type overlayRoute struct {
	dx, dy      int16
	cost        int32
	requirement string
	enabled     bool
}

// teleportOverlay is config, not world state: the poll goroutine publishes a new
// immutable snapshot and the game loop only reads it. An atomic pointer keeps
// handlers lock-free, which is what invariant #1 (single owner of world state)
// asks for — there is nothing here to own, only a read-only snapshot.
var teleportOverlay atomic.Pointer[map[[2]int16]overlayRoute]

// SetTeleportOverlay replaces the live overlay with the given rows. An empty or
// nil slice clears it, restoring the compiled table exactly. It returns how many
// rows were published, for the boot/poll log.
func SetTeleportOverlay(routes []OverlayRoute) int {
	if len(routes) == 0 {
		teleportOverlay.Store(nil)
		return 0
	}
	m := make(map[[2]int16]overlayRoute, len(routes))
	for _, r := range routes {
		m[[2]int16{r.SrcX, r.SrcY}] = overlayRoute{
			dx: r.DstX, dy: r.DstY, cost: r.Cost,
			requirement: normalizeRequirement(r.Requirement),
			enabled:     r.Enabled,
		}
	}
	teleportOverlay.Store(&m)
	return len(m)
}

// TeleportOverlaySize reports how many overlay rows are live (0 = compiled table
// only).
func TeleportOverlaySize() int {
	if m := teleportOverlay.Load(); m != nil {
		return len(*m)
	}
	return 0
}

func normalizeRequirement(req string) string {
	switch req {
	case teleportdata.RequirementFairy:
		return teleportdata.RequirementFairy
	case teleportdata.RequirementKefra:
		return teleportdata.RequirementKefra
	default:
		return teleportdata.RequirementNone
	}
}

// routeAt resolves the tile against the overlay first, then the compiled table.
func routeAt(tile [2]int16) (teleRoute, string, bool) {
	if m := teleportOverlay.Load(); m != nil {
		if r, ok := (*m)[tile]; ok {
			if !r.enabled {
				return teleRoute{}, "", false
			}
			return teleRoute{dx: r.dx, dy: r.dy, cost: r.cost}, r.requirement, true
		}
	}
	r, ok := teleportTable[tile]
	if !ok {
		return teleRoute{}, "", false
	}
	return r, teleportdata.RequirementAt(tile[0], tile[1]), true
}

// TeleportAccess is the per-player and world state some routes are gated on.
type TeleportAccess struct {
	// HasFairy: the Vale portal requires item 3916 in Equip[13].
	HasFairy bool
	// KefraDefeated: the Deserto → Kefra portal only opens once the Kefra is
	// dead (the legacy KefraLive != 0, GetFunc.cpp:1006-1010).
	KefraDefeated bool
}

// TeleportDest resolves a teleport from (x,y): it rounds to the tile, looks up
// the route, and returns the destination (+rand%3 spread) and gold cost. ok is
// false when there is no teleport tile at that position.
func TeleportDest(x, y int16) (destX, destY int16, cost int32, ok bool) {
	return teleportDest(x, y, TeleportAccess{}, nil)
}

// TeleportDestWithAccess resolves routes whose access can depend on equipped
// items or world state (see TeleportAccess).
func TeleportDestWithAccess(x, y int16, access TeleportAccess, random *rng.MSVC) (destX, destY int16, cost int32, ok bool) {
	return teleportDest(x, y, access, random)
}

func teleportDest(x, y int16, access TeleportAccess, random *rng.MSVC) (destX, destY int16, cost int32, ok bool) {
	tile := [2]int16{x &^ 3, y &^ 3} // round down to a multiple of 4
	r, requirement, found := routeAt(tile)
	vale, kefra := requirement == teleportdata.RequirementFairy, requirement == teleportdata.RequirementKefra
	if !found || (vale && !access.HasFairy) || (kefra && !access.KefraDefeated) {
		return 0, 0, 0, false
	}
	// The spread stays the LAST operation of the route: the RNG order is part of
	// the gameplay contract (issue 42 forbids changing it), overlay or not.
	spread := func() int { return rand.Intn(3) }
	// Keep existing route randomness unchanged; the Vale and Kefra branches are
	// the ones newly ported from the legacy path and use the world's MSVC stream.
	if random != nil && (vale || kefra) {
		spread = func() int { return random.Intn(3) }
	}
	return r.dx + int16(spread()), r.dy + int16(spread()), r.cost, true
}
