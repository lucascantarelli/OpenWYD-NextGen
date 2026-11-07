// Família de combinação Odin (GetMatchCombineOdin, GetFunc.cpp:551; despachada por
// Exec_MSG_CombineItemOdin, _MSG_CombineItemOdin.cpp:132-722): 11 receitas
// numeradas mais um fallback padrão "Composição de Sets" disparado pelo campo
// Extra do catálogo, o mesmo mecanismo "_selado" que AnctResult já usa.
package combine

import (
	"openwyd-nextgen/tmserver/internal/refine"
	"openwyd-nextgen/tmserver/internal/world"
)

// Ids de receita Odin, iguais aos valores de retorno de GetMatchCombineOdin. OdinNoMatch
// é o retorno PADRÃO da função (nenhum padrão numerado casou) — NÃO é
// "inválido": o Exec sempre tenta "Composição de Sets" para ele.
const (
	OdinNoMatch          = 0 // "Composição de Sets" — the default fallback
	OdinItemCelestial    = 1 // Exec's own comment for this branch is "Composição de armas"
	OdinPlus12           = 2
	OdinPistaDeRunas     = 3
	OdinDestraveLv40     = 4
	OdinPedraDaFuria     = 5
	OdinSecretaAgua      = 6
	OdinSecretaTerra     = 7
	OdinSecretaSol       = 8
	OdinSecretaVento     = 9
	OdinSementeDeCristal = 10
	OdinCapaCelestial    = 11
)

// blockedItem747 já é declarado em match.go; reusado aqui (Item[747]
// bloqueia toda receita Odin do mesmo jeito que bloqueia Anct).

// maxItemListOdin espelha handler.maxItemList — MAX_ITEMLIST não é documentado;
// mesmo valor provisório UNVERIFIED usado no resto deste fork.
const maxItemListOdin = 30000

// MatchOdin é o porte de GetMatchCombineOdin. items precisa ter exatamente 8
// elementos (protocol.MaxCombine / MAX_COMBINE, Basedef.h:173); slice curto
// nunca casa. Devolve o id da receita 0..11 — 0 significa "nenhum padrão
// numerado casou", que quem chama ainda encaminha para "Composição de Sets",
// NÃO para uma resposta inválida/sem efeito.
func MatchOdin(cat Catalog, items []world.Item) int {
	for _, it := range items {
		if int(it.Index) == blockedItem747 {
			return OdinNoMatch
		}
	}
	if len(items) < 8 {
		return OdinNoMatch
	}
	for _, it := range items[:8] {
		if it.Index < 0 || int(it.Index) >= maxItemListOdin {
			return OdinNoMatch
		}
	}

	idx := func(i int) int { return int(items[i].Index) }
	lvl := func(i int) int { return refine.Level(items[i]) }
	amt := func(i int) int { return refine.Amount(items[i]) }

	switch {
	case idx(0) == cat.Extra[idx(1)] && lvl(0) >= 9 && lvl(1) == 15 &&
		(idx(2) == 542 || idx(2) == 772) &&
		idx(3) == 5334 && idx(4) == 5335 && idx(5) == 5336 && idx(6) == 5337:
		return OdinItemCelestial

	case ((idx(0) == 413 && amt(0) >= 10 && idx(1) == 413 && amt(1) >= 10) || (idx(0) == 4043 && idx(1) == 4043)) &&
		inRange(lvl(2), 10, 15) &&
		isRuneOr3338(idx(3)) && isRuneOr3338(idx(4)) && isRuneOr3338(idx(5)) && isRuneOr3338(idx(6)) &&
		isPlus12Pos(cat.Pos[idx(2)]):
		return OdinPlus12

	case idx(0) == 413 && idx(1) == 413 && idx(2) == 413 && idx(3) == 413 && idx(4) == 413 && idx(5) == 413 && idx(6) == 413:
		return OdinPistaDeRunas

	case idx(0) == 4127 && idx(1) == 4127 && idx(2) == 5135 && idx(3) == 5113 && idx(4) == 5129 && idx(5) == 5112 && idx(6) == 5110:
		return OdinDestraveLv40

	case idx(0) == 5125 && idx(1) == 5115 && idx(2) == 5111 && idx(3) == 5112 && idx(4) == 5120 && idx(5) == 5128 && idx(6) == 5119:
		return OdinPedraDaFuria

	case idx(0) == 5126 && idx(1) == 5127 && idx(2) == 5121 && idx(3) == 5114 && idx(4) == 5125 && idx(5) == 5111 && idx(6) == 5118:
		return OdinSecretaAgua

	case idx(0) == 5131 && idx(1) == 5113 && idx(2) == 5115 && idx(3) == 5116 && idx(4) == 5125 && idx(5) == 5112 && idx(6) == 5114:
		return OdinSecretaTerra

	case idx(0) == 5110 && idx(1) == 5124 && idx(2) == 5117 && idx(3) == 5129 && idx(4) == 5114 && idx(5) == 5125 && idx(6) == 5128:
		return OdinSecretaSol

	case idx(0) == 5122 && idx(1) == 5119 && idx(2) == 5132 && idx(3) == 5120 && idx(4) == 5130 && idx(5) == 5133 && idx(6) == 5123:
		return OdinSecretaVento

	case idx(0) == 421 && idx(1) == 422 && idx(2) == 423 && idx(3) == 424 && idx(4) == 425 && idx(5) == 426 && idx(6) == 427:
		return OdinSementeDeCristal

	case idx(0) == 4127 && idx(1) == 4127 && idx(2) == 5135 && idx(3) == 413 && idx(4) == 413 && idx(5) == 413 && idx(6) == 413:
		return OdinCapaCelestial
	}
	return OdinNoMatch
}

// inRange diz se lo < v <= hi (a janela de nível da receita "+12+").
func inRange(v, lo, hi int) bool { return v > lo && v <= hi }

// isRuneOr3338 é o gate de ingrediente por slot da receita "+12+": uma das
// quatro pedras elementais Secreta (5334-5337), ou a pedra de runa neutra 3338.
func isRuneOr3338(idx int) bool {
	return (idx >= 5334 && idx <= 5337) || idx == 3338
}

// isPlus12Pos limita a receita "+12+" aos slots de equip a que ela se aplica
// (g_pItemList[Item[2].sIndex].nPos == um destes).
func isPlus12Pos(pos int) bool {
	switch pos {
	case 2, 4, 8, 16, 32, 64, 192, 128:
		return true
	}
	return false
}

// g_pItemSancRate12 / g_pItemSancRate12Minus (Basedef.cpp:115-120): os bônus
// de taxa da receita "+12+" (por pedra de runa presente, e pelo nível de sanc
// da própria pedra neutra) e as penalidades (pelo nível atual do alvo).
var (
	itemSancRate12      = [11]int{1, 2, 3, 4, 5, 6, 7, 8, 9, 12, 15}
	itemSancRate12Minus = [4]int{2, 3, 4, 5}
)

// Plus12Rate acumula a taxa de sucesso da receita "+12+"
// (_MSG_CombineItemOdin.cpp:257-322): uma taxa base, +itemSancRate12[0] por pedra
// de runa (5334-5337) entre os 8 slots, +itemSancRate12[1..9] indexado pelo
// nível da pedra 3338 (0..9), a contagem de ocorrências do item 4043 ("protect") e
// por fim uma penalidade de itemSancRate12Minus indexada pelo nível ATUAL do
// alvo (12/13/14/15 — 11 não paga penalidade). items precisa ser a mesma entrada
// de 8 slots que MatchOdin viu; target é items[2], lido antes do aumento de nível.
func Plus12Rate(baseRate int, items []world.Item) (rate, protect int) {
	rate = baseRate
	for _, it := range items {
		idx := int(it.Index)
		if idx >= 5334 && idx <= 5337 {
			rate += itemSancRate12[0]
		}
		if idx == 4043 {
			protect++
		}
		if idx == 3338 {
			if l := refine.Level(it); l >= 0 && l <= 9 {
				rate += itemSancRate12[l+1]
			}
		}
	}
	switch refine.Level(items[2]) {
	case 12:
		rate -= itemSancRate12Minus[0]
	case 13:
		rate -= itemSancRate12Minus[1]
	case 14:
		rate -= itemSancRate12Minus[2]
	case 15:
		rate -= itemSancRate12Minus[3]
	}
	return rate, protect
}

// Plus12NewLevel é a escada de nível da receita "+12+"
// (_MSG_CombineItemOdin.cpp:333-352): +11→12, +12→13, +13→14, +14→15. Qualquer
// outro nível (já excluído pelo gate level>10 && level<=15 do chamador,
// mas protegido aqui também) não reporta aumento.
func Plus12NewLevel(level int) (int, bool) {
	switch level {
	case 11:
		return 12, true
	case 12:
		return 13, true
	case 13:
		return 14, true
	case 14:
		return 15, true
	}
	return level, false
}
