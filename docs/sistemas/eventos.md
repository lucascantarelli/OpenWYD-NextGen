# Eventos globais

Evento global é o que muda o mundo para todos os jogadores ao mesmo tempo: clima, noite de pesadelo,
invasão, pílula do orc, Molarg/Gargula. A regra vale para todos eles: **um dono do estado** — o evento
roda dentro do loop do mundo, com timer próprio, e não em goroutine paralela.

## Catálogo de eventos

| Evento | Handler | Estado no mundo |
|---|---|---|
| Evento base / agenda | `worldevent.go` | `internal/world/worldevent.go`, `internal/world/event.go` |
| Clima | `weather.go` | `internal/worldevents/weather.go` |
| Noite de pesadelo | `nightmare.go`, `nightmare_exp.go` | `internal/world/nightmare.go` |
| Molarg / Gargula | `molargargula.go` | `internal/world/event.go` |
| Pílula do orc | `pilulaorc.go` | `internal/world/event.go` |
| Castelo e torre | `castle.go`, `towerwar.go` | `internal/worldevents/{castle,tower}.go` |

EXP de evento (pesadelo) tem handler próprio (`nightmare_exp.go`) porque o multiplicador só vale dentro
da janela — e precisa sair no **painel**, não como fala. O ciclo do evento é tratado em `nightmare.go`
(a `nightmare_event_test.go` cobre o ciclo, mas não existe handler separado com esse nome).

## Como um evento começa e termina

1. `internal/worldevents` calcula a agenda (relógio do servidor, não do cliente).
2. Ao abrir a janela, o estado entra no mundo e o cliente recebe o aviso (`notice.go`).
3. Durante a janela, os handlers consultam o estado do evento antes de aplicar bônus.
4. Ao fechar, o estado é limpo e **nada** é persistido além do que a regra do evento mandar persistir.

## Onde está

| Peça | Arquivo |
|---|---|
| Agenda e ciclo | `internal/worldevents/` |
| Estado no mundo | `internal/world/worldevent.go`, `internal/world/event.go` |
| Regra e timers | `docs/migration/game-rules.md` §6 |
| Portal | `/eventos` (leitura e configuração auditada) |

## Como verificar

```bash
go test ./tmserver/internal/worldevents/ ./tmserver/internal/handler/ -run 'Weather|Nightmare|WorldEvent|Molarg|Pilula'
go test ./tmserver/internal/world/ -run 'Event|Nightmare'
```

No jogo: forçar a janela de um evento curto e conferir que o anúncio chega uma vez, que o bônus vale
durante a janela e que ele **para** ao fechar.

!!! warning "UNVERIFIED"
    A cadência real (a cada quantas horas abre cada evento) é configuração de conteúdo + agenda do
    `internal/worldevents`; não há documento único com a tabela de horários. Para afirmar "abre às 20h",
    meça no servidor em execução — o portal mostra a janela que o servidor aplicou, não a que o texto
    promete.

## Cuidado ao mexer

Evento que aplica multiplicador de EXP precisa preservar a ordem de `rand()` do cálculo de EXP; um
`rand()` extra dentro da janela muda o ganho de **todos** os jogadores, não só do que está no evento.
