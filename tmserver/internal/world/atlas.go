package world

import (
	"sort"

	"openwyd-nextgen/internal/teleportdata"
)

// AtlasCity é uma visão somente leitura da tabela fixa de cidades (city.go). Ele existe para
// o gerador de atlas do mundo (webserver/cmd/worldatlas) publicar os mesmos
// números que o servidor usa em execução em vez de manter uma segunda cópia deles.
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

// cityNames é o nome de exibição por índice de vila (ordem de Basedef.cpp:54).
var cityNames = [5]string{"Armia", "Azran", "Erion", "Nippleheim", "Noatum"}

// AtlasCities devolve as cinco zonas de cidade na ordem da tabela.
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

// AtlasTeleport é uma entrada de teleportTable, ordenada por origem.
type AtlasTeleport struct {
	SrcX        int    `json:"src_x"`
	SrcY        int    `json:"src_y"`
	DstX        int    `json:"dst_x"`
	DstY        int    `json:"dst_y"`
	Cost        int    `json:"cost"`
	Requirement string `json:"requirement"` // none | fairy | kefra
}

// AtlasTeleports devolve toda rota de teleportTable numa ordem estável. O
// requisito é o mesmo gate que teleportDest aplica (o Vale precisa da fada,
// Deserto→Kefra precisa do Kefra derrotado).
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
