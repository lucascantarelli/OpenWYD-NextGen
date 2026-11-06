# Sistemas de jogo

Uma página por sistema. Cada uma diz **onde o servidor trata**, **qual regra documenta**, **como se
verifica** e o que ainda não está confirmado (marcado UNVERIFIED). O mapa abaixo é a porta de entrada;
o detalhamento mora na página, não aqui.

| Sistema | Página | Handler (tmServer) | Portal |
|---|---|---|---|
| Personagem e evolução | [Personagem](personagem.md) | `character.go`, `login.go`, `score_derive.go` | `personagens` |
| Combate e dano | [Combate](combate.md) | `combat.go`, `mobkilled.go`, `death_exp.go` | — |
| Skills e afects | [Skills](skills.md) | `skill.go`, `affect_score.go`, `affect_tick.go` | `skills` (em breve) |
| Drops | [Drops](drops.md) | `mobkilled.go`, `internal/dropcfg` | `drops` |
| Refino e combine | [Refino](refino.md) | `refine.go`, `combine.go` | `refino` (em breve) |
| Quests e diária | [Quests](quests.md) | `runequest.go`, `newbiequest.go` | `diaria` |
| Comércio e loja | [Comércio](comercio.md) | `shop.go`, `trade.go`, `item.go` | `itens`, `precos`, `npcs` |
| Guildas | [Guildas](guildas.md) | `guild.go`, `guild_state.go` | `guildas` |
| Castelos, Kefra e torre | [Castelos](castelos.md) | `castle.go`, `kefra.go`, `towerwar.go` | `castelo` |
| Eventos globais | [Eventos](eventos.md) | `worldevent.go`, `weather.go`, `nightmare.go` | `eventos` |
| Canais e troca de servidor | [Canais](canais.md) | `serverchange.go`, `newbie_channel.go` | — |
| Mapa e teleporte | [Mapa](mapa.md) | `movement.go`, `world/teleport.go` | `atlas`, `mapa`, `teleportes` |

Regras transversais valem para todos: **estado do mundo em uma goroutine** (nada de lock), **ordem de
`rand()` preservada** em código de gameplay e **EXP sempre no painel**, nunca como fala no chat.

## Como ler uma página de sistema

Cada página segue o mesmo contrato:

1. **Fluxo** — o caminho de um pedido do cliente até o efeito no mundo.
2. **Onde está** — tabela peça → arquivo; nomes de arquivo são verificados por teste.
3. **Como verificar** — comando de teste e o que observar no jogo.
4. **UNVERIFIED** — o que depende de conteúdo ou de medição que ainda não existe.
5. **Limite atual** — o que o servidor não faz, dito explicitamente.

O teste `portal/src/server/guias.test.ts` reprova página de sistema que cite arquivo `.go` inexistente
na pasta de handlers, **inclusive este índice**. Foi assim que nomes velhos saíram daqui: o que se
chamava attack.go hoje é combat.go, npcshop*.go é shop.go e worldevents.go é worldevent.go (escritos
sem crase de propósito: são história, não citação).
