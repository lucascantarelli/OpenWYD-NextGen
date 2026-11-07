// Package refine porta o núcleo do refino com pó de _MSG_UseItem ("refino com
// poeiras"): a codificação empacotada de sanc e as tabelas de taxa de sucesso. Como
// internal/combine, é puro — sem estado de mundo, sem I/O — porque a codificação
// e as taxas são constantes críticas de paridade da economia (parity-tests.md §4.0).
//
// O nível de refino NÃO mora no cValue do item como número simples. Ele é
// empacotado junto de um segundo contador (Basedef.cpp:2282, BASE_SetItemSanc):
//
//	níveis 0..9   cValue = level + 10*pity     (pity 0..20, então 0..209)
//	níveis 10..15 cValue = base + gem          (base 230/234/238/242/246/250, gem 0..3)
//
// Ler um cValue como está reporta, portanto, um nível muito errado para qualquer
// item que já falhou um refino.
//
// Este pacote reporta o nível VERDADEIRO (0..15). O BASE_GetItemSanc legado
// devolve os sentinelas REF_* não contíguos (Basedef.h:256): REF_10=10,
// REF_11=12, REF_12=15, REF_13=18, REF_14=22, REF_15=27. Toda comparação que o
// caminho de pó faz (== 9, >= 6, >= REF_11) dá a mesma resposta nos dois
// espaços, com uma tradução: `sanc >= REF_11` vira `level >= 11`.
package refine

import "openwyd-nextgen/tmserver/internal/world"

// Ids de efeito que podem carregar nível de refino. BASE_GetItemSanc prefere a
// faixa [116,125] em vez de EF_SANC quando um item tem os dois.
const (
	efSanc      = 43  // EF_SANC (ItemEffect.h:100)
	efSancAltLo = 116 // os carregadores alternativos (ItemEffect.h), usados por alguns itens gerados
	efSancAltHi = 125
)

// Itens de crescimento de montaria (2330..2389) são recusados de imediato por toda
// função da família BASE_GetItemSanc (Basedef.cpp:2138) — eles reusam os slots de
// efeito para dados de crescimento, então ler sanc ali daria lixo.
const (
	growthLo = 2330
	growthHi = 2390
)

// gemBase é o primeiro cValue empacotado de um item +10; cada nível cobre 4 valores,
// um por índice de gema (Basedef.cpp:2294-2308).
const (
	gemBase    = 230
	gemPerLvl  = 4
	minGemLvl  = 10
	maxSancLvl = 15
	maxPity    = 20
)

func isGrowth(it world.Item) bool { return it.Index >= growthLo && it.Index < growthHi }

// readSlot devolve o slot de efeito que BASE_GetItemSanc e BASE_GetItemGem leem
// (Basedef.cpp:2141-2160): TODOS os slots são varridos para a faixa [116,125] primeiro,
// e só então para EF_SANC. -1 quando o item não carrega nenhum dos dois.
func readSlot(it world.Item) int {
	for i, ef := range it.Effects {
		if ef.Effect >= efSancAltLo && ef.Effect <= efSancAltHi {
			return i
		}
	}
	for i, ef := range it.Effects {
		if ef.Effect == efSanc {
			return i
		}
	}
	return -1
}

// writeSlot devolve o slot de efeito em que BASE_SetItemSanc grava e que
// BASE_GetItemSancSuccess lê (Basedef.cpp:2312, :2225): o primeiro slot que
// é EF_SANC *ou* está em [116,125], varrendo 0→1→2. -1 quando não há nenhum, que é
// o retorno FALSE do legado — o nível simplesmente não é guardado.
//
// Peculiaridade de paridade: isto NÃO é readSlot, de propósito. Num item que carrega
// um par EF_SANC e um par [116,125], o legado lê o nível de um slot
// e o pity de outro. O bootstrap do caminho de pó normaliza os itens para
// um único par, então isto só é alcançável em itens gerados fora daqui (save
// legado ou import do banco). Reproduzido em vez de arrumado.
func writeSlot(it world.Item) int {
	for i, ef := range it.Effects {
		if ef.Effect == efSanc || (ef.Effect >= efSancAltLo && ef.Effect <= efSancAltHi) {
			return i
		}
	}
	return -1
}

// decodeLevel mapeia um cValue empacotado para o nível de refino verdadeiro 0..15
// (BASE_GetItemSanc, Basedef.cpp:2162-2180).
func decodeLevel(v uint8) int {
	if v >= gemBase && v <= gemBase+(maxSancLvl-minGemLvl+1)*gemPerLvl-1 {
		return minGemLvl + int(v-gemBase)/gemPerLvl
	}
	// Todo o resto (incluindo o inalcançável 210..229 e a cauda 254/255)
	// dobra por %10, que é o que retira o pity.
	return int(v) % 10
}

// encode empacota um nível e seu contador aux num cValue (BASE_SetItemSanc,
// Basedef.cpp:2282-2310). aux é o contador de pity abaixo de +10 e o índice de gema
// de +10 para cima.
func encode(level, aux int) uint8 {
	// Peculiaridade legada: nível fora da faixa zera o item para +0 — NÃO
	// limita a +15 (Basedef.cpp:2285).
	if level < 0 || level > maxSancLvl {
		level = 0
	}
	if aux < 0 {
		aux = 0
	}
	if aux > maxPity {
		aux = maxPity
	}
	if level >= minGemLvl {
		// Estoura além de +15 com aux grande, exatamente como a atribuição legada
		// a um unsigned char faz.
		return uint8(gemBase + (level-minGemLvl)*gemPerLvl + aux)
	}
	return uint8(level + 10*aux)
}

// Level devolve o nível de refino verdadeiro do item, 0..15 (BASE_GetItemSanc). 0 significa
// sem refino.
func Level(it world.Item) int {
	if isGrowth(it) {
		return 0
	}
	i := readSlot(it)
	if i < 0 {
		return 0
	}
	return decodeLevel(it.Effects[i].Value)
}

// Pity devolve o contador acumulado de "sucesso" (BASE_GetItemSancSuccess).
// Refinos falhos o aumentam e ele volta para a taxa da próxima tentativa. De
// +10 para cima o mesmo campo guarda o índice de gema, então os dois não podem ser
// misturados — o caminho de pó só lê Pity abaixo de +10.
func Pity(it world.Item) int {
	if isGrowth(it) {
		return 0
	}
	i := writeSlot(it)
	if i < 0 {
		return 0
	}
	return int(it.Effects[i].Value) / 10
}

// Gem devolve o índice de gema de um item +10..+15, ou -1 abaixo de +10
// (BASE_GetItemGem).
func Gem(it world.Item) int {
	if isGrowth(it) {
		// O legado devolve FALSE aqui, ou seja, gema 0 — não o -1 que significa "sem
		// gema" (Basedef.cpp:2186). Reproduzido.
		return 0
	}
	i := readSlot(it)
	if i < 0 {
		return -1
	}
	v := it.Effects[i].Value
	if v < gemBase {
		return -1
	}
	return int(v-gemBase) % gemPerLvl
}

// Set guarda um nível e o contador aux no slot de sanc do item, reportando se
// havia slot disponível (BASE_SetItemSanc). Nunca aloca um: quem chama
// planta o par EF_SANC primeiro (Bootstrap).
func Set(it *world.Item, level, aux int) bool {
	i := writeSlot(*it)
	if i < 0 {
		return false
	}
	it.Effects[i].Value = encode(level, aux)
	return true
}

// Bootstrap planta um par EF_SANC zerado no primeiro slot livre para recebê-lo, para
// um item nunca refinado ter onde guardar seu nível
// (_MSG_UseItem.cpp:814-840). Um slot conta como livre quando está vazio ou já
// carrega um par de sanc. Retorna falso quando os três slots têm efeitos reais —
// o legado recusa refinar tal item.
func Bootstrap(it *world.Item) bool {
	for i, ef := range it.Effects {
		if ef.Effect == 0 || ef.Effect == efSanc || (ef.Effect >= efSancAltLo && ef.Effect <= efSancAltHi) {
			it.Effects[i] = world.Effect{Effect: efSanc, Value: 0}
			return true
		}
	}
	return false
}
