// Package droprate mantém os divisores base de drop por slot de Carry — a tabela
// em que as fórmulas do jogo e o editor web precisam concordar.
//
// Mora no internal/ do módulo (não em tmserver/internal) porque os dois lados
// precisam dela: tmserver/internal/loot copia esta tabela para o seu DropRate
// mutável (reescrito pelo overlay do moderador no boot) e
// webserver/internal/dropadmin mostra estes valores como a "base" que o editor
// sobrescreve.
//
// O valor é um DIVISOR: quanto maior, mais raro o drop (1 = sempre).
package droprate

// MaxRate é o maior divisor aceito; espelha o CHECK de
// drop_rate_override.value. Um valor acima dele é recusado, não truncado.
const MaxRate = 1_000_000

// Rate é g_pDropRate[64]: chance base por slot de Carry do mob, NÃO por item.
var Rate = build()

func build() [64]int {
	var out [64]int
	fill := func(from, to, v int) {
		for i := from; i <= to; i++ {
			out[i] = v
		}
	}
	fill(0, 7, 900)     // common equip
	fill(8, 11, 4)      // very common (gold/potion)
	fill(12, 15, 900)   //
	fill(16, 23, 20000) // ultra rare
	fill(24, 47, 2000)  //
	fill(48, 55, 3000)  //
	out[56] = 1         // always drops
	for i, v := range []int{35, 500, 2500, 5000, 5000, 10000, 20000} {
		out[57+i] = v
	}
	return out
}
