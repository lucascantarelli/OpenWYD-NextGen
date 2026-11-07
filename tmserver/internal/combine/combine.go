// Package combine guarda o sorteio de sucesso de refino/combinação como função pura
// (game-rules.md §3.1, _MSG_CombineItem.cpp:80-84). É a constante de economia
// crítica de paridade compartilhada pelas ~9 variantes de combinação.
package combine

// Rand é o RNG mínimo; *rng.MSVC o satisfaz (Intn(n) == rand() do C % n).
type Rand interface {
	Intn(n int) int
}

// Constantes do sorteio (_MSG_CombineItem.cpp:80-82).
const (
	RollModulo       = 115 // rand() % 115
	FlattenThreshold = 100 // valores >= 100 ...
	FlattenAmount    = 15  // ... são reduzidos em 15 (então 100..114 → 85..99)
)

// Roll faz o sorteio de sorte da combinação contra a taxa da receita (0..100). Devolve
// o valor sorteado (achatado) e se teve sucesso.
//
// O achatamento torna os resultados 85..99 DUAS vezes mais prováveis que 0..84 (vêm
// tanto do 85..99 original quanto do 100..114 reduzido). Esta distribuição exata é
// parte da economia e NÃO pode ser trocada por rand()%100.
func Roll(r Rand, rate int) (value int, success bool) {
	v := r.Intn(RollModulo)
	if v >= FlattenThreshold {
		v -= FlattenAmount
	}
	return v, v <= rate
}

// Constantes do sorteio Odin (_MSG_CombineItemOdin.cpp:124-129): uma dobra distinta
// da de Roll — rand()%199, e valores acima de 100 caem 99. NÃO é a
// mesma distribuição da dobra 115/-15 de Roll; as duas não podem ser confundidas.
const (
	OdinRollModulo    = 199
	OdinFlattenAbove  = 100
	OdinFlattenAmount = 99
)

// RollOdin faz o sorteio de sorte da família Odin (_MSG_CombineItemOdin.cpp:125-129).
// Diferente de Roll, quem chama compara o resultado com a taxa específica da receita
// (cada ramo Odin calcula seu próprio limiar), então isto devolve só
// o valor dobrado.
func RollOdin(r Rand) int {
	v := r.Intn(OdinRollModulo)
	if v > OdinFlattenAbove {
		v -= OdinFlattenAmount
	}
	return v
}
