// Package teleportdata is the compiled teleport route table, shared by tmServer
// (gameplay) and the web-api (the moderator teleport editor, issue 42).
//
// It lives outside tmserver/internal on purpose: the web-api has to list and
// simulate exactly the routes the world resolves, and Go forbids importing
// tmserver/internal from there. This is static game data (GetTeleportPosition,
// GetFunc.cpp), not world state — the live world still owns its own overlay.
package teleportdata

// Requirement values a route can carry. They mirror teleport_route.requirement
// (internal/migrations/0034_teleport_route.up.sql) and the gates teleportDest
// applies: the Vale portal needs the Valley Fairy (item 3916 in Equip[13]) and
// Deserto → Kefra only opens once the Kefra is dead (KefraLive != 0,
// GetFunc.cpp:1006-1010).
const (
	RequirementNone  = "none"
	RequirementFairy = "fairy"
	RequirementKefra = "kefra"
)

// Route is one compiled teleport route. SrcX/SrcY are the rounded origin tile
// (x &^ 3), which is exactly the key teleportDest looks up.
type Route struct {
	SrcX        int16
	SrcY        int16
	DstX        int16
	DstY        int16
	Cost        int32
	Requirement string
}

// routes is the legacy table, comment by comment as ported from
// GetTeleportPosition. Noatum is the hub: the three cities pay 700 to reach it;
// travel out of Noatum is free.
var routes = []Route{
	{2116, 2100, 1044, 1724, 700, RequirementNone},  // Armia → Noatum
	{2480, 1716, 1044, 1716, 700, RequirementNone},  // Azran → Noatum
	{2456, 2016, 1044, 1708, 700, RequirementNone},  // Erion → Noatum
	{1044, 1724, 2116, 2100, 0, RequirementNone},    // Noatum → Armia
	{1044, 1716, 2480, 1716, 0, RequirementNone},    // Noatum → Azran
	{1044, 1708, 2456, 2016, 0, RequirementNone},    // Noatum → Erion
	{2548, 1740, 2281, 3688, 0, RequirementFairy},   // Azran → Vale (Fada do Vale, item 3916)
	{2364, 3924, 3250, 1703, 0, RequirementKefra},   // Deserto → Kefra (Kefra defeated)
	{1052, 1708, 3650, 3110, 0, RequirementNone},    // Noatum → Nippleheim
	{3648, 3108, 1054, 1710, 0, RequirementNone},    // Nippleheim → Noatum
	// Fields / dungeons (subset of GetTeleportPosition).
	{2140, 2068, 2588, 2096, 0, RequirementNone},  // Armia → Armia Field
	{2468, 1716, 2248, 1556, 0, RequirementNone},  // Azran → Azran Field
	{2364, 2284, 144, 3788, 0, RequirementNone},   // Armia Field → Dungeon 1
	{144, 3788, 2364, 2284, 0, RequirementNone},   // Dungeon 1 → Armia Field
	{2668, 2156, 148, 3774, 0, RequirementNone},   // Armia Field → Dungeon 1 (alt)
	{144, 3772, 2668, 2156, 0, RequirementNone},   // Dungeon 1 → Armia Field (alt)
	{1824, 1772, 1172, 4080, 0, RequirementNone},  // Azran Field → Underworld
	{1172, 4080, 1824, 1772, 0, RequirementNone},  // Underworld → Azran Field
	// GetFunc.cpp: both adjacent entrance blocks lead to floor 2; the return
	// deliberately lands in the eastern block, not the western entrance.
	{144, 3780, 1004, 4028, 0, RequirementNone}, // Dungeon 1 → Dungeon 2
	{148, 3780, 1004, 4028, 0, RequirementNone}, // Dungeon 1 → Dungeon 2
	{1004, 4028, 148, 3780, 0, RequirementNone}, // Dungeon 2 → Dungeon 1
	{408, 4072, 1004, 4064, 0, RequirementNone}, // Dungeon 1 → Dungeon 2 (alt)
	{1004, 4064, 408, 4072, 0, RequirementNone}, // Dungeon 2 → Dungeon 1 (alt)
}

// Routes returns a copy of the compiled table. Callers may sort or mutate it.
func Routes() []Route {
	out := make([]Route, len(routes))
	copy(out, routes)
	return out
}

// Lookup returns the compiled route whose origin is the rounded tile (srcX,
// srcY). Callers must pass the already-rounded tile (x &^ 3).
func Lookup(srcX, srcY int16) (Route, bool) {
	for _, r := range routes {
		if r.SrcX == srcX && r.SrcY == srcY {
			return r, true
		}
	}
	return Route{}, false
}

// RequirementAt reports the compiled requirement of an origin tile (None when
// the tile is not a teleport).
func RequirementAt(srcX, srcY int16) string {
	if r, ok := Lookup(srcX, srcY); ok {
		return r.Requirement
	}
	return RequirementNone
}
