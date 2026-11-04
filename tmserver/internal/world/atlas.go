package world

import (
	"sort"

	"github.com/jeanluca/w2pp-openwyd/internal/teleportdata"
)

// AtlasCity is a read-only view of the fixed city table (city.go). It exists so
// the world-atlas generator (webserver/cmd/worldatlas) can publish the same
// numbers the server uses at runtime instead of keeping a second copy of them.
type AtlasCity struct {
	Index   int    `json:"index"`
	Name    string `json:"name"`
	SpawnX  int    `json:"spawn_x"`
	SpawnY  int    `json:"spawn_y"`
	LimitX1 int    `json:"limit_x1"`
	LimitY1 int    `json:"limit_y1"`
	LimitX2 int    `json:"limit_x2"`
	LimitY2 int    `json:"limit_y2"`
	WarX1   int    `json:"war_x1"`
	WarY1   int    `json:"war_y1"`
	WarX2   int    `json:"war_x2"`
	WarY2   int    `json:"war_y2"`
}

// cityNames is the display name per village index (Basedef.cpp:54 order).
var cityNames = [5]string{"Armia", "Azran", "Erion", "Nippleheim", "Noatum"}

// AtlasCities returns the five city zones in table order.
func AtlasCities() []AtlasCity {
	out := make([]AtlasCity, 0, len(cities))
	for i, c := range cities {
		out = append(out, AtlasCity{
			Index: i, Name: cityNames[i],
			SpawnX: int(c.spawnX), SpawnY: int(c.spawnY),
			LimitX1: int(c.limitX1), LimitY1: int(c.limitY1),
			LimitX2: int(c.limitX2), LimitY2: int(c.limitY2),
			WarX1: int(c.warX1), WarY1: int(c.warY1),
			WarX2: int(c.warX2), WarY2: int(c.warY2),
		})
	}
	return out
}

// AtlasTeleport is one teleportTable entry, sorted by origin.
type AtlasTeleport struct {
	SrcX        int    `json:"src_x"`
	SrcY        int    `json:"src_y"`
	DstX        int    `json:"dst_x"`
	DstY        int    `json:"dst_y"`
	Cost        int    `json:"cost"`
	Requirement string `json:"requirement"` // none | fairy | kefra
}

// AtlasTeleports returns every route of teleportTable in a stable order. The
// requirement is the same gate teleportDest applies (Vale needs the fairy,
// Deserto→Kefra needs the Kefra defeated).
func AtlasTeleports() []AtlasTeleport {
	routes := teleportdata.Routes()
	sort.Slice(routes, func(i, j int) bool {
		if routes[i].SrcX != routes[j].SrcX {
			return routes[i].SrcX < routes[j].SrcX
		}
		return routes[i].SrcY < routes[j].SrcY
	})

	out := make([]AtlasTeleport, 0, len(routes))
	for _, r := range routes {
		out = append(out, AtlasTeleport{
			SrcX: int(r.SrcX), SrcY: int(r.SrcY),
			DstX: int(r.DstX), DstY: int(r.DstY),
			Cost: int(r.Cost), Requirement: r.Requirement,
		})
	}
	return out
}
