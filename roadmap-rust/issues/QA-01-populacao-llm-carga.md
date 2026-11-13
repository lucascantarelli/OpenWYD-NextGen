# QA-01 — Bots população real + LLM + gestão de logs de teste

> **ID canônico:** `QA-01`
> **ID histórico:** `BOTS-01`
> **Estado 2026-11-12:** ⬜ · ver `roadmap-rust/index.md`.

**Tipo:** Código (bots+sim) · **Esforço:** L · **Depende de:** PROC-24, BUG-01 · **Bloqueia:** QA-09/QA-10

## Objetivo

Evoluir `wyd-bots` de harness de protocolo para população simulada jogável (caçar, quest, grupo, loja, PvP, guerra) com gestão de logs para balanceamento, mais arquitetura LLM híbrida (determinístico no jogo, LLM só em decisão social/alto nível).

## Contexto / evidência

- Hoje: `wyd-bots` (4 raças, handshake, `_MSG_Quest`, `read_scene`/`attack_at`/`kill`/`open_gate`, 4/4 testes) + `GAME-23` 🟡 (Campo) + ADR-0010/0016 (IA determinística, sem LLM no tick).
- `prompt.md` pede: bots com todas as funções de jogador real (Mortal→Subcelestial, atributos, skills, refino, drops, guerras, PvP, grupos, trocas, lojas, teleporte, morte/ressurreição); população com perfis variados; LLM para objetivos/planejamento/memória/social; stress/load/concorrência/persistência/economia; gestão de logs para análise e balanceamento.
- Invariantes: Hot Path sem I/O, servidor autoridade, `unsafe` só em interop, `clippy -D warnings`.

## Passos

1. Núcleo jogável: FSM por bot (login → PIN → seleção → mundo → loop caçar/loot/quest/grupo/loja/morrer-ressuscitar/trocar canal) reutilizando `wyd-sim` (fórmulas) e `payload/*` provados; sem pacote inventado.
2. População: perfis (classe, nível, estilo, horário, social) via config + seeds determinísticas (`XorShift64(world_seed, entity)`); grupos, comércio e guildas entre bots.
3. LLM híbrida: motor decide ação (validado pelo servidor); LLM sugere objetivo/rota/fala em lote fora do tick, com orçamento e fallback determinístico; sem LLM em dano/acerto/drop/autoridade.
4. Carga: cenários reproduzíveis (N bots, taxa, duração) + asserts de invariante (sem dupe, sem gold negativo, sem entidade órfã) + relatório (p50/p99 tick, sessões, drops/hora, mortes).
5. Logs: `bots_logs` (run id, bot, ação, resultado, latência) + pipeline para análise (curvas EXP/drop, hotspots de morte, economia) alimentando `wyd-sim` e `WEB-07`.

## Arquivos

- Criados: `crates/wyd-bots/src/{population,scenarios,llm,logs}.rs`, `crates/wyd-bots/tests/*`, migration `bots_logs` (se persistir em Postgres; senão NDJSON auditável).
- Alterados: `crates/wyd-bots/{src/lib.rs,src/tela.rs}`, `crates/wyd-sim/*` (cenários), `docs/arquitetura/ia-bots.md`, `docs/rust/07-ia-e-bots.md`, `docs/rust/08-checklist-funcionalidades.md`.
- Proibido: LLM no Hot Path; bot com privilégio (usa o mesmo fio do jogador); segredo de LLM no repo.

## Testes / verificação

```bash
cargo test -p wyd-bots
cargo test -p wyd-sim
```

Saída esperada: campanha do Campo ponta a ponta com fixtures; cenário de carga com relatório sem invariante quebrado.

## Skills relacionadas

- wyd-rust-testing — harness, goldens, carga sem cliente gráfico.
- wyd-rust-sim — fórmulas puras, cenários, goldens.
- wyd-rust-core — FSM fora do tick, determinismo.
- software-architecture — fronteira determinístico × LLM.

## Documentação de referência

- `docs/arquitetura/adr/0010-ia-npc-e-bots.md`, `docs/arquitetura/ia-bots.md`, `docs/rust/07-ia-e-bots.md`
- `wyd2.md` §9 (bots/LLM/stress), `roadmap-rust/issues/GAME-23-bots-do-campo-de-treinamento.md`

## Subtasks

- [ ] Loop jogável completo por bot
- [ ] Perfis de população com seeds
- [ ] Camada LLM híbrida com fallback
- [ ] Cenários de carga + logs analisáveis

## Critérios de aceite

- [ ] N bots executam quest/carga sem violar autoridade (servidor recusa forjado).
- [ ] Relatório de carga com p50/p99 + economia/drops por hora.
- [ ] Logs permitem reproduzir falha (run id + seed).

## Riscos / rollback

- Risco: LLM alucina ação inválida → mitigação: servidor valida tudo; LLM nunca assina pacote.
- Rollback: desabilitar camada LLM (bots seguem determinísticos).
