# BOT-02 — Cliente headless mínimo, FSM, perfis, cenários e métricas

**Fase:** BOT · **Depende de:** BOT-01 · **Bloqueia:** BOT-03..08 · **Esforço:** M · **Estado:** ✅ concluída em 06/11/2026

## Objetivo

Ter um bot que autentica, seleciona personagem e entra no mundo contra o tmServer real, dirigido por
uma máquina de estados determinística, com cenário versionado e veredito binário — sem cliente
gráfico e sem LLM.

## Contexto / evidência

- O e2e de login (`-tags=e2e`, `TestE2ESmokeLogin`) prova o handshake INITCODE + `MSG_AccountLogin`
  → `MSG_CNFAccountLogin`; a troca de canal prova `MSG_CharacterLogin` → `MSG_CNFCharacterLogin`.
- O executor precisa do `HEADER.ID` do mundo, que vem no header do `CNFCharacterLogin`; usar o id da
  conexão antes disso faz o servidor descartar os pacotes seguintes.
- Sem limite de ritmo, o bot é desconectado por 200 msg/s e o teste "falha" por um motivo que não é
  bug do servidor.

## O que foi entregue

`tmserver/internal/bot`:

| Arquivo | Entrega |
|---|---|
| `client.go` | conexão CPSock, INITCODE, login (erros tipados `ErrAlreadyPlaying`/`ErrStillPlaying`/`ErrLoginRefused`), seleção de personagem, leitura de frame com validação de tamanho/checksum, balde de fichas de 200 msg/s com burst 400 |
| `fsm.go` | FSM com guardas, ações, histórico de transições, erro tipado de transição inexistente |
| `behavior.go` | tabela de transições do ciclo de vida e da caça; escolha de estímulo por estado e perfil |
| `profile.go` | 9 perfis (casual, farmer, combatente, comerciante, explorador, social, quest, stress, bughunter), validação de limiares, RNG por semente, mistura de população com soma exata |
| `scenario.go` | cenário JSON com validação estrita (`DisallowUnknownFields`), métricas e operadores conhecidos, avaliação de critérios |
| `metrics.go` | contadores, p50/p95/p99 amostrados com retenção limitada, exposição Prometheus `openwyd_bot_*` |
| `runner.go` | execução concorrente com semente por bot, prazo, recusa de alvo remoto, relatório com veredito |

`tmserver/cmd/botlab`: `validate`, `run` (com `/metrics` opcional e relatório JSON ou humano),
`metrics`.

Cenários versionados: `tmserver/internal/bot/scenarios/smoke.json` (1 bot) e `populacao.json`
(6 bots, perfis heterogêneos, 2 min).

## Limite declarado (importante)

As ações de combate, coleta, cura e comércio **não enviam pacotes**: elas mutam o modelo local do bot
e contam métricas `sim_*`. O tráfego real hoje é conexão, INITCODE, login, seleção de personagem e
leitura de frames — exatamente o caminho que o cliente gráfico usa. Trocar `sim_*` por pacotes reais
exige provar cada formato contra o cliente 7662 e é o escopo de BOT-03.

## Testes / verificação

```
go build ./...                                                          exit 0
go vet ./tmserver/internal/bot/... ./tmserver/cmd/botlab/...            exit 0
go test -race -count=1 ./tmserver/internal/bot/...                      ok 7,054s
go run ./tmserver/cmd/botlab validate -scenario .../smoke.json          cenário smoke válido: 1 bots, 20s, semente 1, alvo 127.0.0.1:8281
```

Casos cobertos (33 funções de teste): tabela de transições inválida, transição sem destino, guarda escolhendo a
primeira aresta aplicável, `Reset`, ciclo de vida completo, saturação de HP/MP, perfis válidos,
perfil que recua antes de curar rejeitado, RNG reproduzível e diferente entre sementes, mistura de
população somando exatamente `n`, cenário válido, campo desconhecido recusado, treze mutações
inválidas recusadas, recusa de alvo público, aceitação de `127.0.0.1`/`[::1]`/`localhost`, avaliação
de critérios com valor observado, contadores/percentis/exposição Prometheus, servidor fake exigindo
INITCODE e conferindo login/seleção/chat, erros tipados de recusa, conta longa recusada, sequência de
comportamento idêntica para a mesma semente, execução com 2 bots conectados e contabilização de bot
recusado.

## Fora de escopo (registrado)

- Movimento, ataque, loot e comércio por pacote real → BOT-03.
- Navegação por mapa/waypoints → BOT-04.
- Containers, carga progressiva e soak → BOT-05.
- Integração no CI: o alvo `bot-test` existe (offline, com `-race`), mas o `make ci` de hoje só alcança os bots pelo alvo `test` (`go test -race -cover ./...`) — o alvo dedicado e o dashboard ficam em BOT-08.

## Critérios de aceite

- [x] Login + seleção + entrada no mundo por headless, com erros tipados
- [x] FSM com estados, transições, guardas e histórico
- [x] Perfis configuráveis e população heterogênea determinística
- [x] Cenário versionado com critérios de sucesso e falha
- [x] Métricas com percentis e exposição Prometheus
- [x] Recusa de alvo não autorizado por padrão
- [x] Testes sem rede (servidor fake) verdes com `-race`

## Riscos / rollback

Risco: alguém ler as métricas `sim_*` como comportamento real do servidor. Mitigação: prefixo `sim_`,
documentação no ADR e neste registro. Rollback: remover o pacote e os alvos; nada do servidor depende
dele.
