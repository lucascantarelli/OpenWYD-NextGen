# Issue 10-1 — Equações exatas de combate

**Fase:** 10 · **Depende de:** 03-5 · **Bloqueia:** 10-3 · **Esforço:** L

## Objetivo

Documentar e codificar as equações de combate (base → atributos → equipamentos →
skill → buffs → crítico → defesa) com ordem, limites e arredondamento.

## Contexto / evidência

`pre-release-rust.md` FASE 10 §10.1. Fontes: `game-rules.md`,
`.deprecated-go/tmserver/internal/combat/`. Diferenciar fato de relato de fórum
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

- .deprecated-go/docs/migration/game-rules.md.
- .deprecated-go/tmserver/internal/combat/.
- pre-release-rust.md (FASE 10 §10.1).

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
