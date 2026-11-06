# Guildas

Guilda é estrutura social com estado no servidor: membros, cargos, depósito e chat. O portal lê e
administra; o jogo executa.

## Estrutura

- `guild.go` trata criação, convite, entrada, saída e expulsão.
- `guild_state.go` mantém o estado da guilda no mundo (quem está online, cargos).
- `internal/world/guild.go` guarda a representação do mundo; `internal/world/clan.go` cobre a
  estrutura de clã.
- Chat de guilda tem teste próprio (`guild_chat_test.go`): mensagem de guilda não pode vazar para o
  canal geral.

## Regras de autoridade

Quem pode expulsar, promover ou dissolver é decidido **no servidor**, pelo cargo do personagem, não
pelo botão que o cliente mostra. Guilda com líder offline não é dissolvida automaticamente — dissolver
é ação explícita (ou administrativa, auditada).

## Onde está

| Peça | Arquivo |
|---|---|
| Operações de guilda | `guild.go` |
| Estado no mundo | `guild_state.go`, `internal/world/guild.go` |
| Clã | `internal/world/clan.go` |
| Guerra de guilda / torre | `towerwar.go` — ver [Castelos e guerras](castelos.md) |
| Portal | `/guildas` (leitura) |

## Como verificar

```bash
go test ./tmserver/internal/handler/ -run 'Guild'
go test ./tmserver/internal/world/ -run 'Guild|Clan'
```

No jogo: criar guilda, convidar um segundo personagem, conferir cargo na lista de membros e que o chat
de guilda chega só aos membros.

!!! warning "UNVERIFIED"
    Limites (tamanho máximo de nome, número de membros, custo de criação) vêm do conteúdo/config do
    cliente e da configuração do servidor; não há tabela única no repositório que os consolide. Antes
    de prometer um limite, meça no servidor em execução.

## O que não existe

Declaração formal de guerra entre guildas com prazo, placar e recompensa automática — ver
[Guerras](../lore/guerras.md) para o que existe hoje (disputa de castelo).
