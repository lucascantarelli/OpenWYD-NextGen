package refine

import "openwyd-nextgen/tmserver/internal/world"

// Linhas de bigorna da tabela SancRate. "OriLacto" é o nome legado do
// índice, derivado do EF_VOLATILE do pó como Vol-4 (_MSG_UseItem.cpp:849).
const (
	Ori   = 0 // Poeira de Oriharucon (412), EF_VOLATILE 4 — caps the item at +6
	Lac   = 1 // Poeira de Lactolerium (413), EF_VOLATILE 5 — caps the item at +9
	Amago = 2 // forced by Lactolerium_100 (4141); always a flat 100%
)

// RateTable fornece as linhas de SancRate indexadas por bigorna (Ori/Lac/Amago) e por
// nível+1. *content.SancRate a satisfaz; o formato de interface mínima espelha
// combine.Rand.
type RateTable interface {
	Rate(anvil, idx int) int
}

// sancGrade é g_pSancGrade (Basedef.cpp:77): o passo de refino concedido por
// grade de EF_ITEMLEVEL. Toda entrada é 1, e SancRate.txt não traz linhas
// PO_A..PL_E para sobrescrever, então um refino bem-sucedido é sempre +1.
var sancGrade = [2][5]int{
	{1, 1, 1, 1, 1},
	{1, 1, 1, 1, 1},
}

// pityWeight é g_pSuccessRate (Basedef.cpp:122): quantos pontos de taxa cada
// ponto de pity acumulado vale. Indexado como a tabela de taxa, em nível+1.
var pityWeight = [11]int{5, 5, 5, 5, 4, 4, 3, 3, 2, 1, 0}

// Step devolve o incremento de refino de um item da grade EF_ITEMLEVEL informada
// (_MSG_UseItem.cpp:862-874). Grades fora de [1,5] caem num +1 simples —
// e note que o legado pula os tetos de nível nesse fallback, o que quem
// chama reproduz.
func Step(anvil, grade int) int {
	if anvil < Ori || anvil > Lac || grade < 1 || grade > len(sancGrade[0]) {
		return 1
	}
	return sancGrade[anvil][grade-1]
}

// SuccessRate porta BASE_GetSuccessRate (Basedef.cpp:2243): a chance 0..100
// de refinar `it` com a bigorna informada com sucesso. Lê o próprio nível e
// pity do item, exatamente como a assinatura legada faz.
//
// Note o índice nível+1: o slot 0 de uma linha é inútil, e um refino de +N lê
// o slot N+1.
func SuccessRate(t RateTable, it world.Item, anvil int) int {
	if isGrowth(it) {
		return 0
	}
	if anvil < Ori || anvil > Amago {
		return 0
	}
	level := Level(it)
	// +10 é o único nível que as tabelas não cobrem; é fixo a montante.
	if level == minGemLvl {
		if anvil == Amago {
			return 100
		}
		return 15
	}
	if anvil == Amago {
		return 100 // a garantia do Lactolerium_100, antes de qualquer consulta à tabela
	}
	// O caminho de pó bloqueia +9 e +11.. antes do sorteio, então só 0..8 chegam a
	// uma consulta. O legado dobraria os sentinelas REF_* com `sanc %= 10`
	// e leria uma linha sem sentido aqui; devolver 0 (sempre falha) evita que uma
	// entrada inalcançável indexe fora dos limites.
	if level < 0 || level+1 >= len(pityWeight) {
		return 0
	}
	return t.Rate(anvil, level+1) + Pity(it)*pityWeight[level+1]
}
