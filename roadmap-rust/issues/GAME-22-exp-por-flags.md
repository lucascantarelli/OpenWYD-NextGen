# GAME-22 — Árvore de EXP por flags (default inalterado)

> **ID canônico:** `GAME-22`
> **ID histórico:** `FASE-10-4`
> **Estado 2026-11-12:** ⬜ · ver `roadmap-rust/index.md`.

**Fase:** 10 · **Depende de:** GAME-08 ✅ (entrada/salva) · **Bloqueia:** FASE-15 · **Esforço:** M

## Objetivo

Aplicar a árvore completa de EXP do legado por trás de flags em `game_setting`,
sem mudar um ponto do comportamento atual com as flags desligadas.

## Contexto / evidência

Pesquisa `docs/wyd/pesquisa-wyd2-missao-fase3-2026-11-10.md` item 5 (26 campos).
Hoje: `grant_exp` + `exp_reward` com multiplicador (curva LEGACY, teto retido,
`progression.rs:106-130`, teste `grant_exp_levels_up`). Árvore do legado
(`MobKilled.cpp:397-1425`, via `game-rules.md` §1): gate `Clan!=4`; base
`GetExpApply`; bônus de party (`NumMob`, `PARTYBONUS`, loop em `±HALFGRID`);
`450/(30+level)`; gate `0<exp≤10⁷` (não clampa, pula); divisores por tier
(Mortal/Arch/Celestial); corte −40% (`6*exp/10`); cap `eMob`; bônus item/fada,
RvR +5%, newbie +25%, `DOUBLEMODE ×2`, `KefraLive==0 ÷2`, ±15%; clamp
`g_pNextLevel`; `Hold` de EXP; log diário.

Valores 7662 por flag: UNVERIFIED (sem captura) — cada flag nasce desligada com
o default atual.

## Passos

1. Uma flag em `game_setting` por ajuste (party, tier, corte, cada bônus,
   `DOUBLEMODE`, `KefraLive`, ±15%, `Hold`), todas default = comportamento atual.
2. Implementar na ordem do legado (L3 §1.5), uma por vez, cada uma com teste
   isolado + combinadas.
3. `Hold` (trava que consome o ganho até quitar) e log diário como subpassos com
   testes próprios.
4. `make check`.

## Arquivos

- Criados: migration(s) de `game_setting`, `crates/wyd-sim/tests/exp_flags.rs`
  (ou extensão de `progression.rs`).
- Alterados: `crates/wyd-sim/src/formula/progression.rs`,
  `crates/wyd-server/src/gameplay.rs`, seeds de `game_setting`.
- Proibido: mudar o default com flags off; derivar número de servidor privado.

## Testes / verificação

```bash
cargo test -p wyd-sim
cargo test -p wyd-server --test config_gameplay
```

Saída esperada: tudo `ok`; com flags off, resultado idêntico ao atual
(regressão travada em teste).

## Skills Relacionadas do freebuff

- .agents/skills/wyd-rust-sim — árvore e goldens.
- .agents/skills/wyd-rust-core — consumo pós-aprovação.
- .agents/skills/wyd-rust-db — flags versionadas.

## Documentação de referencia

- docs/wyd/pesquisa-wyd2-missao-fase3-2026-11-10.md (item 5).
- docs/wyd/progressao.md §6 (teto retido).
- `docs/migration/game-rules.md` §1.

## Subtasks

- [ ] Codificar issue
- [ ] Testar issue
- [ ] Revisar issue
- [ ] Documentar issue
- [ ] Versionar issue, mergear branch e fechar issue
- [ ] Limpar branch (se aplicável)

## Critérios de aceite

- [ ] Flags off = comportamento atual byte a byte (teste de regressão).
- [ ] Cada flag isolada e combinada com teste.
- [ ] `current_exp` do cliente continua ignorado.

## Riscos / rollback

- Risco: flag ligada por engano muda a economia de EXP → mitigação: default off
  + revisão de seed.
- Rollback: desligar flags; `grant_exp` atual permanece.
