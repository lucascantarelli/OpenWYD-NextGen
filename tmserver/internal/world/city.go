package world

import "math/rand"

// city é uma zona de spawn (subconjunto de STRUCT_GUILDZONE). A tabela é fixa no
// original (Basedef.cpp:54): retângulo CityLimit + área padrão CitySpawn.
type city struct {
	spawnX, spawnY   int16
	limitX1, limitY1 int16
	limitX2, limitY2 int16
	// warX1..warY2 é o retângulo WarArea da zona de guilda (BASE_GetArena), uma pequena
	// caixa própria por zona — não a arena PvP de Armia, que fica dentro dos
	// limites da cidade de Armia.
	warX1, warY1 int16
	warX2, warY2 int16
}

// cities é a tabela fixa das 5 cidades (Armia, Azran, Erion, Nippleheim, Noatum).
var cities = [5]city{
	{2086, 2093, 2052, 2052, 2171, 2163, 197, 213, 238, 230},     // 0 Armia
	{2494, 1707, 2432, 1672, 2675, 1767, 197, 149, 238, 166},     // 1 Azran
	{2453, 2000, 2448, 1966, 2476, 2024, 141, 213, 182, 230},     // 2 Erion
	{3652, 3122, 3605, 3090, 3690, 3260, 141, 149, 182, 166},     // 3 Nippleheim
	{1050, 1706, 1036, 1700, 1072, 1760, 4000, 4000, 4010, 4010}, // 4 Noatum
}

// Arena devolve a zona de guilda (0..4) cuja WarArea contém (x,y), ou -1
// (BASE_GetArena, Basedef.cpp:4903).
func Arena(x, y int16) int {
	for i := range cities {
		c := cities[i]
		if x >= c.warX1 && x <= c.warX2 && y >= c.warY1 && y <= c.warY2 {
			return i
		}
	}
	return -1
}

// PvPZone informa se (x,y) está dentro de uma cidade ou área de guerra de guilda: ali as
// regras legadas de PvP pulam Guilty/CantKill (_MSG_Attack.cpp "PK - War - Miss") e
// uma morte em PvP não custa EXP (MobKilled.cpp: arena != 5 || village != 5).
func PvPZone(x, y int16) bool {
	return Arena(x, y) >= 0 || Village(x, y) >= 0
}

// Village devolve o índice da cidade (0..4) cujo retângulo contém (x,y), ou -1
// (BASE_GetVillage).
func Village(x, y int16) int {
	for i := range cities {
		c := cities[i]
		if x >= c.limitX1 && x <= c.limitX2 && y >= c.limitY1 && y <= c.limitY2 {
			return i
		}
	}
	return -1
}

// cityTax é g_pGuildZone[village].CityTax — o imposto percentual que uma vila cobra numa
// venda em loja pessoal (issue #115). O init estático é 5 para toda vila
// (Basedef.cpp:54-61); em execução um cerco pode sobrescrevê-lo (0..20, padrão 10,
// CReadFiles.cpp:809), o que a reescrita ainda não modela. Indexado por Village().
var cityTax = [5]int16{5, 5, 5, 5, 5}

// CityTax devolve o percentual de imposto de venda para um índice de vila (0..4), ou 0 se fora
// da faixa (BASE_GetVillage devolveu -1 → sem imposto). Leitura só no loop.
func CityTax(village int) int16 {
	if village < 0 || village >= len(cityTax) {
		return 0
	}
	return cityTax[village]
}

// CitySpawn devolve uma posição de spawn padrão para a cidade dada (CitySpawn +
// rand%15). city é limitada a 0..3 (a "última cidade" salva só ocupa 2 bits;
// Noatum=4 cai para Armia, espelhando o overflow Merchant<<6 do original).
func CitySpawn(city int) (int16, int16) {
	if city < 0 || city > 3 {
		city = 0
	}
	c := cities[city]
	return c.spawnX + int16(rand.Intn(15)), c.spawnY + int16(rand.Intn(15))
}

// nonCombatNPC classifica NPCs de serviço/cidade independentemente do byte Merchant
// visível ao cliente. Alguns templates reais de cidade trazem Merchant==0 (por exemplo estátuas
// de combine e atores de adorno da cidade), então confiar só no Merchant os torna
// alvos de combate. Criaturas hostis de clã dentro/perto dos retângulos de cidade continuam podendo combater.
func nonCombatNPC(merchant, clan uint8, x, y int16) bool {
	if merchant != 0 {
		return true
	}
	return Village(x, y) >= 0 && !ClanHostile(0, clan)
}
