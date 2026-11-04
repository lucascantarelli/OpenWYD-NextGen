# Issue 38 — Operação ao vivo: online, kick, teleport e aviso global

**Fase:** 6 · **Depende de:** 32, 36 · **Bloqueia:** 39 · **Esforço:** L · **Estado:** ⏳ aberta

## Objetivo

Ver quem está online e agir sobre o mundo em execução — derrubar conexão, teleportar personagem e
anunciar mensagem global — sem violar o dono único do estado.

## Contexto / evidência

- O `tmserver` **não** tem servidor gRPC: só CPSock (`:8281`) e status HTTP (`:80`). Ele é **cliente**
  de dbServer/binserver. Portanto, comando da web → jogo não pode ser RPC direta.
- O padrão que já existe para isso é `delivery_queue` (web grava, tmserver drena no login) e
  `WorldEventConfig` ("o portal escreve esta config fria; o tmserver faz polling via dbServer e
  aplica dentro do loop" — `web.proto:876-880`).
- Todo estado de jogo vive numa goroutine (`world.World.Run`); mutação fora do loop é proibida
  (`AGENTS.md` §3.1). Um kick é fechar a conexão **pelo loop**; teleport é mover o char **pelo loop**.
- `/gm notice`, `/gm kick`, `/gm goto` já existem in-game (`docs/game.md`) — a web é o equivalente
  remoto, não um novo mecanismo de jogo.

## Passos

1. **Migração** `0040_live_ops`:
   ```sql
   CREATE TABLE IF NOT EXISTS live_command (
     id bigserial PRIMARY KEY,
     kind text NOT NULL CHECK (kind IN ('kick','teleport','notice')),
     target_account text, target_character text,   -- kick/teleport
     payload jsonb NOT NULL DEFAULT '{}'::jsonb,    -- {x,y} | {text}
     status text NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','done','failed','expired')),
     created_by bigint NOT NULL REFERENCES account(id),
     created_at timestamptz NOT NULL DEFAULT now(),
     applied_at timestamptz, error text
   );
   CREATE TABLE IF NOT EXISTS live_world_snapshot (
     id bigserial PRIMARY KEY,
     taken_at timestamptz NOT NULL DEFAULT now(),
     online jsonb NOT NULL   -- [{character, account, level, x, y, hp, mp, map}]
   );
   ```
2. **tmserver**: poller de 2s (fora do loop) → entrega ao loop por canal; o loop aplica e devolve
   resultado. Regras: comando com mais de 60s vira `expired`; `kick` fecha a sessão e dispara
   save-on-quit; `teleport` valida coordenada `0..4095` e célula não bloqueada (`route.Bake`);
   `notice` usa o mesmo caminho do `/gm notice`. Snapshot do mundo a cada 10s (somente leitura).
3. **Proto**: `LiveOpsService` com `ListOnline`, `Kick(target)`, `Teleport(target, x, y)`,
   `BroadcastNotice(text)`, `ListCommands(status)`; `moderator_id` + `AdminResult`; escrita só para
   `moderator`/`admin`, `BroadcastNotice` só `admin` (evita spam de moderador).
4. **Portal** `/operacao`:
   - tabela de online (personagem, conta, nível, posição, HP/MP, tempo desde o snapshot) com
     auto-refresh de 10s (botão para pausar);
   - ações por linha: teletransportar (dialog com X/Y e validação), expulsar (confirmação com nome
     digitado);
   - faixa de aviso global com contador de caracteres (1–120) e confirmação;
   - histórico de comandos com status (`pending/done/failed/expired`) e erro, atualizado a cada 5s;
   - cartão "saúde do stack": `web-api` ok? `tmserver` respondeu snapshot nos últimos 30s? (deriva de
     `live_world_snapshot.taken_at`), sem shell.
5. Rate limit no BFF: 1 comando por conta a cada 3s; `notice` 1 por 30s. O servidor também limita
   (defesa em profundidade).
6. Nunca permitir `shutdown`, restart ou execução de comando arbitrário pela web nesta issue.

## Arquivos

- Criados: `internal/migrations/0040_live_ops.{up,down}.sql`,
  `tmserver/internal/liveops/{liveops.go,liveops_test.go}` (poller + applier no loop),
  `webserver/internal/liveadmin/{service.go,service_test.go}`,
  `webserver/internal/grpcsrv/liveadmin.go`,
  `portal/src/app/(admin)/operacao/page.tsx`, `portal/src/server/actions/live.ts`,
  `portal/src/components/live/{OnlineTable,CommandHistory}.tsx`.
- Alterados: `api/web/v1/web.proto`, `webserver/cmd/webserver/main.go`,
  `tmserver/cmd/tmserver/main.go`, `tmserver/internal/world/world.go` (canal de
  comandos no loop), `portal/src/components/nav.ts`.
- Proibido: mutar estado do mundo fora do loop; `rand()` novo no caminho (paridade);
  restart/shutdown pela web.

## Testes / verificação

```bash
# (comandos rodam na raiz do repositório)
MSYS_NO_PATHCONV=1 docker run --rm -v "D:/dolorestec/wyd2/OpenWYD:/src" -w /src golang:1.26-alpine \
  sh -c "go build ./... && go test -race ./tmserver/internal/liveops/ ./tmserver/internal/world/"
```

```bash
docker compose up -d tmserver webserver
docker compose logs --tail 30 tmserver | grep -i "live command"
```

Evidência com o cliente aberto: `/operacao` mostra o personagem online; teletransportar para
`(2100, 2100)` move o char; `notice` aparece no chat; `kick` derruba e o personagem salva (conferir
`character.x/y` no banco depois); comando para char offline → `failed` com erro claro.

## Critérios de aceite

- [ ] Online reflete o mundo em ≤ 10s, com dados corretos de nível/posição
- [ ] Kick fecha a conexão e **salva** o personagem (sem duplicação de item)
- [ ] Teleport valida coordenada e colisão; recusa célula bloqueada
- [ ] Notice só para `admin`, com rate limit efetivo
- [ ] Histórico mostra `pending→done/failed/expired` e sobrevive a restart do portal
- [ ] Nenhuma mutação fora do loop (revisão do diff + `-race` verde)
- [ ] Evidência dos 4 comandos no jogo

## Riscos / rollback

- **Risco:** kick no meio de um trade duplicar item. **Mitigação:** o kick passa pelo caminho normal
  de desconexão (mesmo save-on-quit do cliente), coberto pelo teste de trade.
- **Risco:** teleport para célula inválida prender o char. **Mitigação:** validar contra o
  `AttributeMap` antes de aplicar; comando falha, não move.
- **Risco:** poller competir com o loop. **Mitigação:** polling fora do loop + canal não-bloqueante
  com fila limitada (drop de comandos antigos).
- **Rollback:** flag `-live-ops` OFF desliga poller e aplicação; tabelas ficam inertes.
