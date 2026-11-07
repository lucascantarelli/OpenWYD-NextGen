// Package worldevents guarda a lógica pura de decisão dos eventos de mundo que o
// servidor legado conduz por seus timers de segundo/minuto (ProcessSecMinTimer.cpp):
// sorteios de clima, a janela de Guerra de Guild/Torre, o atraso de limpeza de
// área do reino (RvR) e a contagem de Castelo/Zakum.
//
// Tudo aqui é de propósito livre de tipos de world/protocol/session: são
// máquinas de estado guiadas por tabela que recebem o relógio e o RNG como
// parâmetros e devolvem uma decisão. O Dispatcher é dono das instâncias e executa
// os efeitos (pacotes, dano, spawns) — ver tmserver/internal/handler.
//
// Disciplina de relógio: nenhum tipo deste pacote guarda relógio. Quem chama passa
// `now time.Time` (o Dispatcher passa d.now()), e é isso que torna todo
// gate de calendário testável sem esperar uma terça real às 20:00.
package worldevents

// Rand é o RNG mínimo de que estes timers precisam. *rng.MSVC o satisfaz.
//
// Os timers de evento de mundo sorteiam de um fluxo DEDICADO, não do fluxo de
// paridade do mundo: o legado sorteia do rand() global único, mas nossos testes
// golden de drop/refino/crítico fixam a ordem exata de chamadas daquele fluxo, e
// acrescentar um sorteio por minuto o invalidaria (issue #116).
type Rand interface {
	Intn(n int) int
}
