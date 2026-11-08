# Issue 03-6 — IA de NPC no Hot Path (FSM + utilidade, determinística)

**Fase:** 3 · **Depende de:** 03-2, 03-3, 03-4 · **Bloqueia:** 15-3 · **Esforço:** M
**Estado:** ✅ entregue em 2026-11-08 (ADR-0016)

## Objetivo

Dar comportamento a NPC/monstro dentro do tick: aggro, perseguição, ataque, fuga e retorno
ao spawn, com decisão determinística, orçamento medido e nenhum I/O.

## Contexto / evidência

`pre-release-rust.md` FASE 3 §3.1 manda o ECS cobrir "Players, NPCs, Monsters, Items no
chão". A decisão e as fronteiras estão em `docs/arquitetura/adr/0010-ia-npc-e-bots.md`; o
modelo de decisão (P1–P3, cadência, D-IA01..06) em `docs/arquitetura/ia-bots.md`; a execução
em `docs/rust/07-ia-e-bots.md`. A entrega **não** ficou presa a `mob_template_stat` (vazia):
o perfil de comportamento chega por parâmetro (`NpcProfile`), validado no boot.

## O que foi entregue

| Peça | Conteúdo |
|---|---|
| `crates/wyd-core/src/npc/state.rs` | `NpcMode` (7 estados), `NpcState`/`NpcStates` em `Vec` contíguo por `Entity.index`, `ThreatTable` (teto 8, TTL em ticks, desempate pelo menor `entity`), RNG `xorshift64` semeado por `(world_seed, entity_index)` |
| `crates/wyd-core/src/npc/behavior.rs` | FSM pura com precedência explícita: `Flee` → `Return` (leash) → `Attack` → `Chase` → `Aggro` → `Patrol` → `Idle` |
| `crates/wyd-core/src/npc/brain.rs` | P1 percepção (alcance inclusivo), P2 alvo pela tabela de ameaça, P3 **intenção** (`Idle`/`MoveTo`/`Attack`/`Flee`/`Return`); `NpcProfile` validado (`InvalidNpcProfile`) |
| `crates/wyd-core/src/npc/route.rs` | Rota ancorada: patrulha limitada ao raio da âncora (`route_type` segue UNVERIFIED) |
| `crates/wyd-core/src/npc/budget.rs` | Cadência `tick % 4 == entity % 4` (5 Hz) + `DecisionBudget` com teto por tick |
| `crates/wyd-core/src/systems/npc.rs` | Integração com o tick: vivo → criatura → cadência → orçamento → decisão; **não** move nem fere, devolve intenção para `step`/`attack` validarem |
| `crates/wyd-core/benches/tick.rs` | Bench executável (`harness = false`, sem dependência nova): 500 NPCs + 50 jogadores, mede a cadeia percepção → alvo → intenção → orçamento e **falha** se passar de 10% do tick |
| `crates/wyd-core/tests/npc.rs` | 12 testes: golden FNV-1a determinístico, cadência, aggro no limite, ataque no alcance, alvo fora ignorado, desempate pelo menor `entity`, fuga com HP baixo, leash/retorno, recusa atômica (entidade morta e jogador), intenção forjada recusada por `step` e “decidir não publica evento” |

Recusas novas, todas tipadas e sem `panic!`: `NotACreature`, `ThreatTableFull`,
`InvalidNpcProfile`. Leitura nova no ECS: `ComponentStore::kind_of` (dado, não regra).

Cópia de ameaça em relação a `crates/wyd-sim/src/formula/threat.rs`: **deliberada**, porque o
DAG proíbe `wyd-core → wyd-sim` (ADR-0011). A unificação continua dívida da FASE-10-1 e o
golden de `tests/npc.rs` impede divergência silenciosa.

## Testes / verificação (medidos nesta entrega)

```bash
cargo test --workspace                                  # verde, 30 alvos
cargo clippy --workspace --all-targets -- -D warnings   # sem warning
cargo fmt --all --check                                 # limpo
cargo bench -p wyd-core --bench tick                    # 88,6–204,5 µs/tick = 0,18%–0,41% do orçamento
node tools/content/check-migrations.mjs                 # 24 migrations
node tools/content/gen-catalogo-banco.mjs --check       # catálogo em sincronia
node tools/content/verify-schema.mjs                    # 59 tabelas / 457 colunas
node tools/content/migrate-cycle.mjs                    # up/down/up
```

## O que segue UNVERIFIED

- Custo em hardware de produção: o número medido é do host de desenvolvimento (Windows,
  release); não há medição em máquina de servidor ainda.
- `route_type`/`generator_data` sem trace do cliente → patrulha só na âncora.
- Ameaça por dano 1:1 com teto 8 (D-IA02) sem confirmação por captura.
- Perfil derivado do banco (`npc_behavior`) depende de `mob_template_stat` e de FASE-04-3.
- Tabelas `npc_behavior`/`npc_dialogue`/`llm_job` não criadas (schema sem produtor seria
  tabela órfã).

## Skills Relacionadas do freebuff

- .agents/skills/wyd-rust-core - ECS, tick, dados contíguos.
- .agents/skills/rust-code-quality - sem `unsafe`, sem alocação quente.
- .agents/skills/wyd-rust-obs - métricas de decisão e profundidade de fila.

## Documentação de referencia

- docs/arquitetura/adr/0016-ia-npc-deterministica.md
- docs/arquitetura/adr/0010-ia-npc-e-bots.md
- docs/arquitetura/ia-bots.md
- docs/rust/07-ia-e-bots.md
- docs/arquitetura/estrutura-e-padroes.md §10
- pre-release-rust.md FASE 3 §3.1

## Subtasks

- [x] Codificar issue
- [x] Testar issue
- [x] Revisar issue
- [x] Documentar issue
- [ ] Versionar issue, mergear branch e fechar issue (árvore segue só com alterações locais, por instrução)
- [ ] Limpar branch (se aplicável)

## Critérios de aceite

- [x] FSM cobre os 7 estados com teste por transição.
- [x] Determinismo provado por golden (mesmo estado e seed → mesmas intenções, hash FNV-1a).
- [x] Custo da IA ≤ 10% do orçamento do tick com 500 NPCs — **medido** em 5 execuções:
  88,6–204,5 µs/tick = 0,18%–0,41% do tick de 50 ms (pior p99 0,74%) com 500 NPCs + 50
  jogadores; 0,71–1,64 µs/decisão.
- [x] Nenhum I/O, nenhuma alocação no quente, `clippy -D warnings` limpo.

## Riscos / rollback

- Risco: IA cara no tick → mitigação: cadência escalonada + teto em `budget.rs`, bench no aceite.
- Risco: `route_type` UNVERIFIED virar comportamento inventado → mitigação: conservador (âncora).
- Rollback: reverter `npc/`, `systems/npc.rs`, a linha de `systems/mod.rs` e a de `lib.rs`;
  conteúdo não muda.
