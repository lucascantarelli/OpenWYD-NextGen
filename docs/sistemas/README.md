# Sistemas de jogo

Uma linha por sistema: onde o servidor trata, qual regra documenta e onde o portal mostra. O
detalhamento de cada sistema mora no doc de regra citado — esta página é o mapa, não a explicação.

| Sistema | Handler (tmServer) | Regra / evidência | Portal |
|---|---|---|---|
| Login e sessão | `login.go`, `characterlogin.go` | `migration/protocol-spec.md` §4 | `contas` |
| Personagem e save | `charcreate.go`, `savefmt` | `migration/data-formats.md` | `personagens` |
| Combate e dano | `attack.go`, `mobkilled.go` | `migration/game-rules.md` §1–§3 | — |
| Skills e afects | `skill*.go`, `affect*.go` | `migration/game-rules.md` §4 | — |
| Drops | `mobkilled.go`, `internal/droprate` | `migration/game-rules.md` §2.2 | `drops` |
| Refino e combine | `combine*.go`, `internal/ratetables` | `migration/game-rules.md` §5 | `refino` |
| Quests e diária | `quest*.go`, `daily*.go` | `migration/game-rules.md` §6 | `diaria` |
| Comércio e loja de NPC | `npcshop*.go`, `item*.go` | `migration/game-rules.md` §7 | `npcs`, `itens`, `precos` |
| Guildas e castelo | `guild*.go`, `castle*.go` | `migration/game-rules.md` §8 | `guildas`, `castelo` |
| Eventos globais | `worldevents.go` | `migration/flows.md` §eventos | `eventos` |
| Troca de canal | `serverchange.go`, `internal/crossover` | `migration/canais.md` §8 | — |
| Teleporte e mapa | `teleport*.go`, `route` | `migration/config-ops.md` | `teleportes`, `mapa`, `atlas` |

## O que ainda falta aqui

Hoje esta página é um **índice**: ela diz onde olhar, mas não explica o sistema para quem não é do
time. A DOC-01 pede uma página por sistema com fluxo, fórmulas e exemplos; o mapa acima é o
esqueleto, e a primeira a ganhar página própria deve ser **combate** (é o sistema com mais defeito
histórico registrado em `migration/ingame-bugs.md`).
