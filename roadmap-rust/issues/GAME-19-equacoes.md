# GAME-19 — Equações exatas de combate

> **ID canônico:** `GAME-19`
> **ID histórico:** `FASE-10-1`
> **Estado 2026-11-12:** 🚧 · ver `roadmap-rust/index.md`.

**Fase:** 10 · **Depende de:** GAME-05 · **Bloqueia:** GAME-21 · **Esforço:** L

## Objetivo

Documentar e codificar as equações de combate (base → atributos → equipamentos →
skill → buffs → crítico → defesa) com ordem, limites e arredondamento.

## Contexto / evidência

`pre-release-rust.md` FASE 10 §10.1. Fontes: `game-rules.md`,
`crates/wyd-sim/src/ + crates/wyd-core/src/systems/`. Diferenciar fato de relato de fórum
e de **hipótese**.

## Passos

1. Mapear atributos (Força, Inteligência, Destreza, Constituição, HP/MP, ataque,
   defesa, velocidade, crítico): origem, faixa, limites, arredondamento.
2. Reconstruir a ordem do cálculo de dano passo a passo com exemplo numérico.
3. Codificar como funções puras testáveis (base do `wyd-sim`).
4. Teste por etapa da ordem (cada modificador isolado); `make check`.

## Arquivos

- Criados: equações em `crates/wyd-sim/src/` + doc em `docs/` (manual técnico).
- Alterados: `crates/wyd-sim/src/lib.rs`, `Cargo.toml` (numéricos via ADR).
- Proibido tocar: `crates/wyd-core/src/` (engine só consome após aprovação),
  balanceamento sem simulação.

## Testes / verificação

```bash
cargo test -p wyd-sim
make check
```

Saída esperada: `test result: ok` com exemplo numérico documentado reproduzido.

## Skills Relacionadas do freebuff

- .agents/skills/wyd-rust-sim - cenários e funções puras.
- .agents/skills/solid-design-patterns - fórmulas testáveis.
- .agents/skills/rust-code-quality - sem float onde fixed-point é exigido.

## Documentação de referencia

- crates/wyd-sim/src/formula/ + roadmap-rust/issues/GAME-19-equacoes.md.
- crates/wyd-sim/src/ + crates/wyd-core/src/systems/.
- pre-release-rust.md (FASE 10 §10.1).

## Progresso 2026-11-08 — composição codificada, ordem provisória

- **Criados:** `crates/wyd-sim/src/formula/{attributes,progression,hit,resist,effect,drop,regen,threat}.rs`,
  `scenario/combat.rs`, `report/table.rs`, `units.rs`, `error.rs`, `roll/{mod,xorshift64}.rs`;
  `tests/determinism.rs`. `formula/damage.rs` virou cadeia (`DamageInput` → dano final)
  mantendo `damage_after_absorb`.
- **Documentado:** `docs/arquitetura/matematica-sim.md` (por sistema: entradas, saídas, unidade,
  ordem no tick, arredondamento) + ADR-0011 (árvore).
- **Não fechado:** a **ordem exata** (defesa subtrativa × multiplicativa, posição do crítico e
  da absorção, grandeza do crítico de item) é hipótese D-02..D-04/D-06 — exige trace do
  cliente (CLIENT-06) e stats reais (`mob_template_stat`, hoje vazia).
- **Prova:** `cargo test -p wyd-sim` (goldens existentes com as mesmas asserções + 6 testes de
  determinismo + unidade por módulo).

## Subtasks

- [ ] Codificar issue
- [ ] Testar issue
- [ ] Revisar issue
- [ ] Documentar issue
- [ ] Versionar issue, mergear branch e fechar issue
- [ ] Limpar branch (se aplicável)

## Critérios de aceite

- [ ] Ordem completa documentada com exemplo numérico; cada etapa testada.
- [ ] Hipóteses marcadas como hipóteses.

## Riscos / rollback

- Risco: "corrigir" mecânica intencional → mitigação: prova em código/captura
  antes de qualquer mudança.
- Rollback: `git checkout -- crates/wyd-sim/ docs/`.
