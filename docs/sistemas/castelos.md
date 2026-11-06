# Castelos, Kefra e guerra de torre

O que o jogador chama de "guerra" tem três mecânicas distintas no código, e confundi-las gera promessa
que o servidor não cumpre:

| Mecânica | Handler | O que muda no jogo |
|---|---|---|
| Domínio de castelo | `castle.go` | dono do castelo, taxa de quest do período |
| Kefra | `kefra.go`, `kefra_event.go` | evento cíclico com recompensa (ciclo em `game-rules.md` §1.6) |
| Guerra de torre | `towerwar.go` | disputa da torre, com placar do evento |

## Castelo

`castle.go` aplica o resultado da disputa e persiste o dono. `internal/worldevents/castle.go` controla a
janela do evento (quando abre, quando fecha) e `Release/Common/Settings/CastleQuest.txt` define o que a
disputa paga. O portal mostra o estado em `/castelo`.

## Kefra

`kefra.go` trata a interação do jogador, `kefra_event.go` o ciclo do evento e `internal/world/kefra.go`
o estado no mundo. Persistência tem teste próprio (`kefra_persistence_test.go`): progresso de Kefra que
não sobrevive a restart é bug, não "sorte".

## Reino e coroa

`kingdom.go` cobre o reino e a coroa (cape/pedra), com testes de diálogo e de pagamento
(`king_cape_dialogue_test.go`, `king_cape_payment_test.go`). `liderapprentice.go` trata o aprendiz do
líder. São mecânicas separadas da guerra: não misture na mesma tela nem no mesmo texto.

## Onde está

| Peça | Arquivo |
|---|---|
| Domínio de castelo | `castle.go`, `internal/worldevents/castle.go` |
| Kefra | `kefra.go`, `kefra_event.go`, `internal/world/kefra.go` |
| Guerra de torre | `towerwar.go`, `internal/worldevents/tower.go` |
| Reino e coroa | `kingdom.go`, `liderapprentice.go` |
| Conteúdo | `Release/Common/Settings/CastleQuest.txt` |
| Fluxos | `docs/migration/flows.md` §7 (torre) e §8 (castelo) |
| Portal | `/castelo` |

## Como verificar

```bash
go test ./tmserver/internal/handler/ -run 'Castle|Kefra|Tower|Kingdom|Leader'
go test ./tmserver/internal/worldevents/
```

No jogo: conferir a janela do evento no horário configurado e que o dono do castelo continua o mesmo
depois de um restart do tmServer.

!!! warning "UNVERIFIED"
    Duração e horário exatos da disputa dependem do arquivo de conteúdo e do relógio do servidor
    (`America/Sao_Paulo` nas telas do portal). O repositório não valida esse arquivo contra o cliente;
    divergência aparece como "o evento não abriu", sem erro de código.

## Regra de autoridade

O portal **lê** o estado e envia intenção administrativa auditada; ele não decreta vencedor escrevendo
no banco. Quem decide o resultado da disputa é o servidor de jogo.
