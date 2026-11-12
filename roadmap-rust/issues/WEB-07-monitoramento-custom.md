# WEB-07 — Monitoramento customizado: logs ao vivo, servidor, DB e alertas

> **ID canônico:** `WEB-07`
> **ID histórico:** `PORTAL-01`
> **Estado 2026-11-12:** ⬜ · ver `roadmap-rust/index.md`.

**Tipo:** Código (portal) · **Esforço:** L · **Depende de:** PROC-24, WEB-01 🚧 · **Bloqueia:** —

## Objetivo

Trocar o embed pesado do Grafana por página customizada com dados via API (cards, gráficos, tabelas do portal) + logs ao vivo com filtros e download, expandindo para CPU/memória/rede/processos, players × mobs separados, histórico, DB e alertas.

## Contexto / evidência

- Hoje: `/monitoramento` (índice) + `/monitoramento/{servidor,grafana,prometheus,logs}`; `monitoring.ts` lê Prometheus/Loki/presença/Grafana em paralelo com degradação por fonte; `loki.ts` só agrega (o visualizador linha a linha foi removido em 2026-11-09 — ver comentário no arquivo).
- `prompt.md` pede: logs ao vivo com filtros + download; servidor com CPU/mem/rede/processos, players e mobs separados (hoje `entities` soma tudo), gráficos + histórico; DB (conexões, queries, resposta); SO/eventos/segurança; alertas/notificações; página custom via API do Grafana em vez de iframe kiosk.

## Passos

1. Logs ao vivo: `GET /api/logs/stream?container=&level=&q=` (SSE, server-only, queries constantes, sem LogQL do browser) + filtros (serviço, nível, texto) + `GET /api/logs/export` (download `.log`/`.ndjson` com limite e `content-disposition`).
2. Servidor: separar `players` × `mobs` (nova métrica `wyd_players_active`/`wyd_mobs_active` no motor ou derivada de `scene_snapshot`; nunca somar no front como "entidades" sem rótulo) + CPU/mem/rede/processo por canal + série histórica (range Prometheus `query_range`, gráfico leve no portal).
3. DB: painel `pg_stat_activity` (conexões por estado), `pg_stat_statements` (top queries por tempo médio/chamadas) e latência de `SELECT 1` — tudo via gateway (portal nunca fala com Postgres).
4. Alertas: regras em `monitoring/prometheus/alerts.yml` + página de alertas com silenciar/acknowledge (via API do Prometheus/Alertmanager, auditado) + notificações para staff.
5. Manter degradação por fonte (invariante atual) e `adminCall` em toda leitura admin; `toResponse` só `{ error }` + `no-store`.

## Arquivos

- Criados: `wyd-portal/src/app/(admin)/monitoramento/ao-vivo/page.tsx`, `wyd-portal/src/app/api/logs/stream/route.ts`, `wyd-portal/src/app/api/logs/export/route.ts`, `wyd-portal/src/server/db-health.ts`, testes `*.test.ts`.
- Alterados: `wyd-portal/src/server/{monitoring,loki,grafana}.ts`, `wyd-portal/src/app/(admin)/monitoramento/{page,servidor/page,logs/page}.tsx`, `crates/wyd-server/src/metrics.rs` (players × mobs), `monitoring/prometheus/alerts.yml`, `roadmap-rust/index.md`, `docs/rust/08-checklist-funcionalidades.md`.
- Proibido: browser falar com Prometheus/Loki/Grafana/DB; DSN no browser; número sem fonte.

## Testes / verificação

```bash
cd wyd-portal && npx tsc --noEmit && npx vitest run
docker compose up -d --build && curl -fsS http://127.0.0.1:3000/monitoramento
```

Saída esperada: typecheck 0, vitest verde; páginas com fonte fora degradam para aviso (nunca 500).

## Skills relacionadas

- frontend-design, ui-ux-practices — cards/gráficos densos sem poluição.
- wyd-rust-obs — métricas, tracing, runbooks.
- software-architecture — BFF puro, falha isolada por fonte.

## Documentação de referência

- `wyd-portal/README.md` (monitoramento), `docs/rust/06-observabilidade-runbooks.md`, `monitoring/` (alertas, dashboards).

## Subtasks

- [ ] SSE de logs + filtros + export
- [ ] Players × mobs + CPU/mem/rede + histórico
- [ ] Saúde do DB via gateway
- [ ] Alertas com acknowledge auditado

## Critérios de aceite

- [ ] Logs ao vivo filtráveis + download funcionam com Loki fora (aviso, sem 500).
- [ ] Nenhum gráfico sem fonte; players e mobs nunca somados sem rótulo.
- [ ] Toda ação admin passa por `adminCall` com auditoria.

## Riscos / rollback

- Risco: SSE caro no BFF → mitigação: limite de linhas/janela, timeout, `AbortSignal`.
- Rollback: reverter rotas `/api/logs/*` (página volta ao agregado).
