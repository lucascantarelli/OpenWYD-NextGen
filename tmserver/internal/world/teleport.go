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

// teleportTable mapeia um tile de origem arredondado (x&0xFFFC, y&0xFFFC) para seu
// destino + custo em ouro (GetTeleportPosition, GetFunc.cpp). O cliente envia um
// _MSG_ReqTeleport vazio quando pisa num tile de teleporte; o servidor resolve
// a rota a partir da posição do jogador.
//
// Os dados em si ficam em internal/teleportdata para o web-api listar e
// simular a mesma tabela (issue 42); este mapa é a forma de execução dela.
var teleportTable = buildTeleportTable()

func buildTeleportTable() map[[2]int16]teleRoute {
	routes := teleportdata.Routes()
	m := make(map[[2]int16]teleRoute, len(routes))
	for _, r := range routes {
		m[[2]int16{r.SrcX, r.SrcY}] = teleRoute{dx: r.DstX, dy: r.DstY, cost: r.Cost}
	}
	return m
}

// OverlayRoute é uma linha de rota editável por moderador (teleport_route, issue 42).
// O overlay é aditivo: uma linha substitui a rota compilada no mesmo tile de
// origem, Enabled=false a remove, e uma origem sem linha mantém a rota
// compilada.
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

// teleportOverlay é config, não estado do mundo: a goroutine de polling publica um novo
// snapshot imutável e o loop do jogo só o lê. Um ponteiro atômico mantém os
// handlers sem lock, que é o que a invariante #1 (dono único do estado do mundo)
// pede — não há nada aqui para possuir, só um snapshot somente leitura.
var teleportOverlay atomic.Pointer[map[[2]int16]overlayRoute]

// SetTeleportOverlay substitui o overlay vivo pelas linhas dadas. Uma fatia
// nil ou vazia o limpa, restaurando a tabela compilada exatamente. Devolve quantas
// linhas foram publicadas, para o log de boot/polling.
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

// TeleportOverlaySize informa quantas linhas de overlay estão vivas (0 = só a tabela
// compilada).
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

// routeAt resolve o tile contra o overlay primeiro, depois contra a tabela compilada.
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

// TeleportAccess é o estado por jogador e do mundo em que algumas rotas são condicionadas.
type TeleportAccess struct {
	// HasFairy: o portal do Vale exige o item 3916 em Equip[13].
	HasFairy bool
	// KefraDefeated: o portal Deserto → Kefra só abre depois que o Kefra está
	// morto (o legado KefraLive != 0, GetFunc.cpp:1006-1010).
	KefraDefeated bool
}

// TeleportDest resolve um teleporte a partir de (x,y): arredonda para o tile, busca
// a rota e devolve o destino (espalhamento +rand%3) e o custo em ouro. ok é
// falso quando não há tile de teleporte naquela posição.
func TeleportDest(x, y int16) (destX, destY int16, cost int32, ok bool) {
	return teleportDest(x, y, TeleportAccess{}, nil)
}

// TeleportDestWithAccess resolve rotas cujo acesso pode depender de itens
// equipados ou estado do mundo (veja TeleportAccess).
func TeleportDestWithAccess(x, y int16, access TeleportAccess, random *rng.MSVC) (destX, destY int16, cost int32, ok bool) {
	return teleportDest(x, y, access, random)
}

func teleportDest(x, y int16, access TeleportAccess, random *rng.MSVC) (destX, destY int16, cost int32, ok bool) {
	tile := [2]int16{x &^ 3, y &^ 3} // arredonda para baixo, múltiplo de 4
	r, requirement, found := routeAt(tile)
	vale, kefra := requirement == teleportdata.RequirementFairy, requirement == teleportdata.RequirementKefra
	if !found || (vale && !access.HasFairy) || (kefra && !access.KefraDefeated) {
		return 0, 0, 0, false
	}
	// O espalhamento continua sendo a ÚLTIMA operação da rota: a ordem do RNG faz parte
	// do contrato de gameplay (a issue 42 proíbe mudá-la), com ou sem overlay.
	spread := func() int { return rand.Intn(3) }
	// Manter a aleatoriedade das rotas existentes inalterada; os ramos do Vale e do Kefra são
	// os recém-portados do caminho legado e usam o fluxo MSVC do mundo.
	if random != nil && (vale || kefra) {
		spread = func() int { return random.Intn(3) }
	}
	return r.dx + int16(spread()), r.dy + int16(spread()), r.cost, true
}
