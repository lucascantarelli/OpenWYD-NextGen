// Package rng reimplementa o LCG do rand() da CRT da Microsoft usado pelo servidor
// original, para drops, refinos e críticos serem reproduzidos byte a byte
// (parity-tests.md §4.0).
//
// Não existe srand() no TMSrv/DBSrv (o único é em CombineItemOdin), então a CRT
// usa a semente default = 1 em todo boot e produz o mesmo fluxo. Reproduzir este
// LCG mais a ordem original das chamadas de rand() gera resultados idênticos aos
// de uma captura controlada.
package rng

// MSVC é o gerador rand() da CRT da Microsoft. NÃO é seguro para uso concorrente;
// no servidor ele pertence à goroutine do laço do jogo (como o estado global
// rand() original). Para testes determinísticos, semeie explicitamente.
type MSVC struct {
	state uint32
}

// New devolve um gerador com a semente default da CRT (1), igual a um boot novo
// do servidor original.
func New() *MSVC { return &MSVC{state: 1} }

// NewSeeded devolve um gerador com semente explícita (o equivalente a
// srand(seed)); usado para casos golden e para o caminho de reseed do CombineItemOdin.
func NewSeeded(seed uint32) *MSVC { return &MSVC{state: seed} }

// Rand devolve o próximo valor em [0, 32768), exatamente como o rand() da MSVC:
//
//	state = state*214013 + 2531011
//	return (state >> 16) & 0x7FFF
func (r *MSVC) Rand() int {
	r.state = r.state*214013 + 2531011
	return int((r.state >> 16) & 0x7FFF)
}

// Intn devolve Rand() % n, igual ao idioma `rand()%n` do original. Dá panic
// se n <= 0 (erro de programação). Nota: como na versão em C, há viés para n que
// não divide 32768 — esse viés faz parte do comportamento a preservar.
func (r *MSVC) Intn(n int) int {
	if n <= 0 {
		panic("rng: Intn requires n > 0")
	}
	return r.Rand() % n
}
