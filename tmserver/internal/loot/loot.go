// Package loot guarda as fórmulas de drop de morte de mob como funções puras
// (game-rules.md §2, MobKilled.cpp). Como o pacote combat, as chamadas de rand()
// passam por um Rand injetado na ordem original, para o LCG da MSVC reproduzir os
// drops exatamente (parity-tests.md §4.0).
package loot

import "openwyd-nextgen/internal/droprate"

// Rand é o RNG mínimo; *rng.MSVC o satisfaz. Intn(n) == rand()%n em C.
type Rand interface {
	Intn(n int) int
}

// DropRate é g_pDropRate[64] — a chance base de drop POR slot de Carry do mob
// (NÃO por item): quanto maior o valor, mais raro o drop (é um divisor).
// Valores reais de Basedef.cpp:222 (game-rules.md §2.2).
//
// É inicializada como CÓPIA de internal/droprate.Rate: a sobreposição do moderador
// (dropcfg) reescreve esta tabela no boot, e o editor web continua lendo os
// valores base originais do pacote compartilhado.
var DropRate = droprate.Rate

// DropBonus é g_pDropBonus[64] — todos 100 (neutro) por default (Basedef.cpp).
var DropBonus [64]int

func init() {
	for i := range DropBonus {
		DropBonus[i] = 100
	}
}

// EffectiveDropRate calcula a chance de drop por slot depois do bônus de drop do
// matador e do ajuste pelo nível do alvo (game-rules.md §2.2). Resultado maior é
// mais raro. killerBonus é o DropBonus do matador (0 = nenhum).
func EffectiveDropRate(slot, killerBonus, mobLevel int) int {
	droprate := DropRate[slot]
	dropbonus := DropBonus[slot] + killerBonus
	if dropbonus != 100 {
		dropbonus = 10000 / (dropbonus + 1)
		droprate = dropbonus * droprate / 100
	}
	if slot < 60 {
		switch slot / 8 {
		case 0, 1, 2:
			switch {
			case mobLevel < 10:
				droprate = 4 * droprate / 100
			case mobLevel < 20:
				droprate = 5 * droprate / 100
			case mobLevel < 30:
				droprate = 6 * droprate / 100
			case mobLevel < 40:
				droprate = 7 * droprate / 100
			case mobLevel < 60:
				droprate = 8 * droprate / 100
			default:
				droprate = 99 * droprate / 100
			}
		}
	} else {
		switch {
		case mobLevel < 170:
			droprate = 90 * droprate / 100
		case mobLevel < 200:
			droprate = 60 * droprate / 100
		case mobLevel < 230:
			droprate = 50 * droprate / 100
		case mobLevel < 255:
			droprate = 43 * droprate / 100
		case mobLevel < 320:
			droprate = 38 * droprate / 100
		default:
			droprate = 50 * droprate / 100
		}
	}

	// Essas quatro posições de Carry são sobreposições duras aplicadas depois de todo
	// ajuste de nível em MobKilled.cpp. O slot 11 é, portanto, garantido.
	switch slot {
	case 8, 9, 10:
		droprate = 4
	case 11:
		droprate = 1
	}
	if droprate > 32000 {
		droprate = 32000
	}
	return droprate
}

// Drops rola um único drop contra uma taxa efetiva: dropa sse rand()%rate == 0
// (o padrão de chance-como-divisor, como o gate de ouro).
//
// UNVERIFIED: a comparação final exata está truncada no fonte
// (MobKilled.cpp após :2800); este é o padrão convencional e deve ser
// confirmado por captura.
func Drops(r Rand, rate int) bool {
	if rate <= 0 {
		return true // rate 0 ⇒ always (slot 56 = 1 still rolls 0)
	}
	return r.Intn(rate) == 0
}

// GoldDrop devolve o ouro dropado por um mob que morre (game-rules.md §2.1,
// MobKilled.cpp:2693). Consome dois valores de rand(): um gate de drop e, se
// dropar, a quantidade. Limitado a 2000 por kill. 0 significa sem ouro.
func GoldDrop(r Rand, mobLevel, mobCoin int) int {
	unkGold := 18
	switch {
	case mobLevel < 10:
		unkGold = 2
	case mobLevel < 20:
		unkGold = 4
	case mobLevel < 30:
		unkGold = 6
	case mobLevel < 50:
		unkGold = 9
	}
	if mobCoin == 0 || r.Intn(unkGold+1) != 0 {
		return 0
	}
	q := (mobCoin + 1) / 4
	coin := 4 * (r.Intn(q+1) + q + mobCoin)
	if coin > 2000 {
		coin = 2000
	}
	return coin
}
