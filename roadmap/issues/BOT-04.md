# BOT-04 — Navegação com os mapas reais (A*, waypoints e portais)

**Fase:** BOT · **Depende de:** BOT-03 · **Bloqueia:** BOT-06 · **Esforço:** L · **Estado:** ⏳ aberta

## Objetivo

Dar ao bot navegação de verdade: rota entre cidade, área de caça e portal, usando o mesmo mapa que o
servidor usa para andar — não o mapa visual do cliente.

## Contexto / evidência

- O servidor já assa a malha de caminhabilidade a partir de `HeightMap.dat`/`AttributeMap.dat`
  (`route.Bake`; log de boot `walkability grid baked dim=4096`). Altura 127 é bloqueado.
- `tmserver/content/` guarda `Regions.txt` e os mapas (17 MiB, fora do git, `make content-sync`).
- O portal tem atlas (`/atlas`) e rotas de teleporte (`/rotas`), que são a fonte de dados de região e
  portal já validada.
- Suposição a testar: a malha de caminhabilidade do servidor é suficiente para rotear; se não for, o
  bot precisa do mesmo dado de colisão, nunca de leitura de pixel do cliente.

## Passos

1. Exportar a malha de caminhabilidade (ou consumi-la de `internal/route`/`world`) para o bot.
2. Implementar A* sobre a grade com custo uniforme e cache de rotas por par origem/destino.
3. Definir waypoints por região a partir de `Regions.txt` + atlas; incluir portais e cidades.
4. Recalcular rota quando o servidor recusar o passo (`MSG_Action` sem eco) e registrar o motivo.
5. Registrar falha explícita quando o destino for inalcançável, em vez de andar em círculo.

## Arquivos

- `tmserver/internal/bot/nav.go` (novo)
- `tmserver/internal/bot/nav_test.go`
- `tmserver/internal/bot/scenarios/navegacao.json`

## Testes / verificação

```
go test -race ./tmserver/internal/bot/ -run TestNav
```

Saída esperada: rota Armia→Campo de Treino com N passos e custo M; destino bloqueado devolve
`ErrUnreachable`; recálculo após bloqueio devolve rota alternativa.

## Critérios de aceite

- [ ] A* sobre a malha do servidor, com teste de rota conhecida e de destino bloqueado
- [ ] Waypoints de cidade, área de caça e portal versionados em cenário
- [ ] Recálculo em recusa de passo, com motivo no relatório
- [ ] Nenhuma dependência de leitura gráfica do cliente

## Riscos / rollback

Risco: divergência entre a malha do servidor e o que o cliente permite andar. Mitigação: teste e2e de
caminhada comparando posição final. Rollback: bot usa apenas waypoints em linha reta dentro de uma
região.
