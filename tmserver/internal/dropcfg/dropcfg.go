// Package dropcfg dobra a sobreposição de taxa de drop do moderador (issue 34)
// nos defaults de código do pacote loot.
//
// DropRate é um divisor por slot de Carry (1 = sempre, 20000 = ultra raro) e é
// **código**, não conteúdo: loot.DropRate é preenchido no init() a partir de
// Basedef.cpp:222. A sobreposição, portanto, substitui valores nessa tabela em vez
// de ler um arquivo, e precisa ser aplicada exatamente uma vez, no boot, antes do
// laço do mundo começar — a mesma regra de -rates-editing e -mob-stat-editing.
//
// Paridade de RNG: este pacote só escreve números. Nenhuma chamada de rand() é
// adicionada, removida ou reordenada, então um boot com a sobreposição consome a
// mesma sequência de um sem ela (a diferença é quais slots podem dropar).
package dropcfg

import (
	"openwyd-nextgen/internal/droprate"
	"openwyd-nextgen/tmserver/internal/loot"
)

// MaxRate é o maior divisor aceito, casando com o CHECK em
// drop_rate_override.value e com a faixa do editor web. Qualquer valor acima
// disso é tratado como inválido em vez de ser limitado em silêncio: um erro de
// digitação que torna um drop impossível precisa ser visível.
const MaxRate = droprate.MaxRate

// Override é um slot de Carry editado, como o dbServer o entrega.
type Override struct {
	Slot  int
	Value int64
}

// Apply escreve toda sobreposição válida em loot.DropRate e informa quantas foram
// aplicadas/ignoradas. Slots e valores fora da faixa são ignorados (e contados),
// nunca limitados: o portal já valida a mesma faixa, então uma linha ignorada
// significa que a tabela foi editada fora de banda.
func Apply(overrides []Override) (applied, ignored int) {
	for _, o := range overrides {
		if o.Slot < 0 || o.Slot >= len(loot.DropRate) || o.Value < 1 || o.Value > MaxRate {
			ignored++
			continue
		}
		loot.DropRate[o.Slot] = int(o.Value)
		applied++
	}
	return applied, ignored
}

// Snapshot copia a tabela atual, para logging e para testes que precisam
// restaurar os defaults de código.
func Snapshot() [64]int { return loot.DropRate }
