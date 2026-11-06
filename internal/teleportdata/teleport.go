// Package teleportdata é a tabela compilada de rotas de teleporte, compartilhada
// pelo tmServer (gameplay) e pela web-api (o editor de teleporte do moderador).
//
// Mora fora de tmserver/internal de propósito: a web-api precisa listar e
// simular exatamente as rotas que o mundo resolve, e Go proíbe importar
// tmserver/internal de lá. Isto é dado estático de jogo, não estado do mundo — o
// mundo vivo continua dono do seu próprio overlay.
package teleportdata

// Valores de exigência que uma rota pode carregar. Espelham a coluna
// teleport_route.requirement e os portões que o destino aplica: o portal do Vale
// exige a Fada do Vale (item 3916 em Equip[13]) e Deserto → Kefra só abre com o
// Kefra morto (KefraLive != 0).
const (
	RequirementNone  = "none"
	RequirementFairy = "fairy"
	RequirementKefra = "kefra"
)

// Route é uma rota de teleporte compilada. SrcX/SrcY são o tile de origem
// arredondado (x &^ 3), exatamente a chave que o destino consulta.
type Route struct {
	SrcX        int16
	SrcY        int16
	DstX        int16
	DstY        int16
	Cost        int32
	Requirement string
}

// routes é a tabela legada, portada comentário a comentário. Noatum é o hub: as
// três cidades pagam 700 para chegar lá; sair de Noatum é de graça.
var routes = []Route{
	{2116, 2100, 1044, 1724, 700, RequirementNone}, // Armia → Noatum
	{2480, 1716, 1044, 1716, 700, RequirementNone}, // Azran → Noatum
	{2456, 2016, 1044, 1708, 700, RequirementNone}, // Erion → Noatum
	{1044, 1724, 2116, 2100, 0, RequirementNone},   // Noatum → Armia
	{1044, 1716, 2480, 1716, 0, RequirementNone},   // Noatum → Azran
	{1044, 1708, 2456, 2016, 0, RequirementNone},   // Noatum → Erion
	{2548, 1740, 2281, 3688, 0, RequirementFairy},  // Azran → Vale (Fada do Vale, item 3916)
	{2364, 3924, 3250, 1703, 0, RequirementKefra},  // Deserto → Kefra (Kefra defeated)
	{1052, 1708, 3650, 3110, 0, RequirementNone},   // Noatum → Nippleheim
	{3648, 3108, 1054, 1710, 0, RequirementNone},   // Nippleheim → Noatum
	// Campos e masmorras (subconjunto da tabela legada).
	{2140, 2068, 2588, 2096, 0, RequirementNone}, // Armia → Armia Field
	{2468, 1716, 2248, 1556, 0, RequirementNone}, // Azran → Azran Field
	{2364, 2284, 144, 3788, 0, RequirementNone},  // Armia Field → Dungeon 1
	{144, 3788, 2364, 2284, 0, RequirementNone},  // Dungeon 1 → Armia Field
	{2668, 2156, 148, 3774, 0, RequirementNone},  // Armia Field → Dungeon 1 (alt)
	{144, 3772, 2668, 2156, 0, RequirementNone},  // Dungeon 1 → Armia Field (alt)
	{1824, 1772, 1172, 4080, 0, RequirementNone}, // Azran Field → Underworld
	{1172, 4080, 1824, 1772, 0, RequirementNone}, // Underworld → Azran Field
	// Os dois blocos de entrada adjacentes levam ao andar 2; a volta cai de
	// propósito no bloco leste, não na entrada oeste.
	{144, 3780, 1004, 4028, 0, RequirementNone}, // Dungeon 1 → Dungeon 2
	{148, 3780, 1004, 4028, 0, RequirementNone}, // Dungeon 1 → Dungeon 2
	{1004, 4028, 148, 3780, 0, RequirementNone}, // Dungeon 2 → Dungeon 1
	{408, 4072, 1004, 4064, 0, RequirementNone}, // Dungeon 1 → Dungeon 2 (alt)
	{1004, 4064, 408, 4072, 0, RequirementNone}, // Dungeon 2 → Dungeon 1 (alt)
}

// Routes devolve uma cópia da tabela compilada. O chamador pode ordenar ou
// alterar a cópia.
func Routes() []Route {
	out := make([]Route, len(routes))
	copy(out, routes)
	return out
}

// Lookup devolve a rota compilada cuja origem é o tile arredondado (srcX, srcY).
// O chamador precisa passar o tile já arredondado (x &^ 3).
func Lookup(srcX, srcY int16) (Route, bool) {
	for _, r := range routes {
		if r.SrcX == srcX && r.SrcY == srcY {
			return r, true
		}
	}
	return Route{}, false
}

// RequirementAt devolve a exigência compilada de um tile de origem (None quando
// o tile não é teleporte).
func RequirementAt(srcX, srcY int16) string {
	if r, ok := Lookup(srcX, srcY); ok {
		return r.Requirement
	}
	return RequirementNone
}
