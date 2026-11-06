# Drops

O que cai de um monstro ao morrer: gold, item comum e itens especiais. Tudo sai de uma chamada só, no
fim de `mobkilled.go`, e **depois** do crédito de EXP — trocar essa ordem muda o resultado.

## Duas rolagens diferentes

| Rolagem | Fonte da tabela | Regra |
|---|---|---|
| Gold | faixa do template do mob | `docs/migration/game-rules.md` §2.1 — determinística exceto `rand()` |
| Item comum | `Carry[]` do template do mob | §2.2 — cada slot é um item com chance própria |

O `Carry[]` do mob é a tabela: não existe tabela de drop paralela. É por isso que mudar drop de um mob
é mudar o template (`npc_template` no banco), não o código.

## Onde está

| Peça | Arquivo |
|---|---|
| Rolagem e entrega | `mobkilled.go` |
| Configuração de drop | `internal/dropcfg` |
| Taxa por item | `internal/droprate` |
| Regra | `docs/migration/game-rules.md` §2 |
| Portal | `/drops` (leitura) |

## Como verificar

```bash
go test ./tmserver/internal/handler/ -run 'MobKilled'
go test ./tmserver/internal/droprate/ ./tmserver/internal/dropcfg/
```

No jogo: matar o mesmo mob várias vezes e conferir que o item que cai existe no `Carry[]` do
template — item que cai e não está no template é sinal de drop por outra via (quest, evento).

!!! warning "UNVERIFIED"
    A chance efetiva de um item raro depende do valor no template do mob **daquele** ambiente. O
    repositório não guarda uma tabela de "drop de X": guarda o código da rolagem e o conteúdo no banco.

## Regra de ouro ao mexer aqui

A ordem das chamadas de `rand()` é paridade com o servidor original. Uma chamada a mais antes da
rolagem de item desloca **todos** os drops seguintes, mesmo com a fórmula idêntica.
