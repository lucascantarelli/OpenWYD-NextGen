# Issue 10 — Matemática, bug hunting e balanceamento

**Fase:** 10 · **Depende de:** FASE-03, FASE-08 · **Bloqueia:** FASE-15 · **Esforço:** L

## Objetivo

Reconstruir as fórmulas de combate ponta a ponta com ordem de modificadores
documentada, caçar bugs matemáticos classificados e validar tudo no `wyd-sim`
antes de tocar a engine.

## Contexto / evidência

Anomalia prioritária: ~200% de velocidade gerando hits duplicados/DPS anormal
(`pre-release-rust.md` FASE 10 §10.5). Referências: `game-rules.md` (fórmulas),
`ingame-bugs.md` (histórico), código de combate arquivado
(`crates/wyd-sim/src/ + crates/wyd-core/src/systems/`). Esqueleto: `crates/wyd-sim/src/lib.rs`
(`damage_after_absorb`, saturating) — `cargo test -p wyd-sim` verde. Regra:
diferenciar fato (código/observado) de relato de fórum e de **hipótese**.

## Passos

1. Mapear atributos (força/inteligência/destreza/constituição, HP/MP, ataque,
   defesa, velocidade, crítico) com origem, faixa, limites e arredondamento.
2. Reconstruir a ordem do cálculo de dano (base → atributos → equipamentos →
   skill → buffs → crítico → defesa/resistências → resultado).
3. Investigar a anomalia de velocidade/hits com cenários reproduzíveis no
   `wyd-sim` (tabela velocidade×hits, milhões de iterações).
4. Caçar overflow/underflow, divisão por zero, percentuais duplicados e
   divergência cliente×servidor; classificar (severidade/impacto/reprodutibilidade).
5. Propor balanceamento só com evidência de simulação; `make check`.

## Arquivos

- Criados (árvore normativa em `estrutura-e-padroes.md` §14):
  `src/formula/*.rs`, `src/scenario/*.rs`, `src/report/table.rs`,
  `tests/goldens.rs`, doc das equações em `docs/` (manual técnico),
  tabelas de atributos.
- Alterados: `crates/wyd-sim/src/lib.rs`, `crates/wyd-sim/Cargo.toml`
  (deps numéricas via ADR).
- Proibido tocar: `crates/wyd-core/src/` (engine só consome fórmula aprovada),
  balanceamento por opinião (sem simulação, sem merge).

## Testes / verificação

```bash
cargo test -p wyd-sim
make check
```

Saída esperada: `test result: ok`; cenário da anomalia reproduz o comportamento
atual e o proposto diverge de forma explicada (números anexados).

## Skills Relacionadas do freebuff

- .agents/skills/wyd-rust-sim - simulações determinísticas e cenários.
- .agents/skills/wyd-rust-core - aplicação da fórmula na engine (pós-aprovação).
- .agents/skills/software-architecture - fórmulas como funções puras testáveis e patterns idiomáticos.
- .agents/skills/rust-code-quality - sem float onde fixed-point é exigido.

## Documentação de referencia

- crates/wyd-sim/src/formula/ + roadmap-rust/changelog.md (GAME-19) - fórmulas de referência.
- roadmap-rust/issues/GAME-20-velocidade.md - histórico de bugs.
- crates/wyd-sim/src/ + crates/wyd-core/src/systems/ - implementação arquivada.
- pre-release-rust.md (FASE 10) - roteiro da investigação.
- docs/rust/01-dependencias.md - candidatos numéricos.

## Subtasks

- [ ] Codificar issue
- [ ] Testar issue
- [ ] Revisar issue
- [ ] Documentar issue
- [ ] Versionar issue, mergear branch e fechar issue
- [ ] Limpar branch (se aplicável)

## Critérios de aceite

- [ ] Equações completas com ordem, limites e arredondamento documentados.
- [ ] Anomalia de velocidade explicada (causa raiz) com cenário reproduzível.
- [ ] Bugs classificados por severidade/impacto; nenhum balanceamento sem evidência.

## Riscos / rollback

- Risco: "corrigir" comportamento que era mecânica intencional → mitigação:
  hipótese marcada como hipótese até prova em código/captura.
- Rollback: fórmulas versionadas; `git checkout -- crates/wyd-sim/`.
