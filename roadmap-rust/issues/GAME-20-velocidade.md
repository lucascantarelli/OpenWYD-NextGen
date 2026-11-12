# GAME-20 — Anomalia de velocidade (~200%) e hits duplicados

> **ID canônico:** `GAME-20`
> **ID histórico:** `FASE-10-2`
> **Estado 2026-11-12:** ⬜ · ver `roadmap-rust/index.md`.

**Fase:** 10 · **Depende de:** GAME-19 · **Bloqueia:** GAME-21 · **Esforço:** M

## Objetivo

Explicar a causa raiz da anomalia de velocidade de ataque com cenário
reproduzível e comportamento desejado modelado deterministicamente.

## Contexto / evidência

`pre-release-rust.md` FASE 10 §10.2: ~200% de velocidade gerando hits
duplicados/DPS anormal. Base: equações GAME-19 + `ingame-bugs.md` (histórico).

## Passos

1. Reproduzir no `wyd-sim`: tabela velocidade×hits que mostre a anomalia.
2. Isolar a causa (ordem de aplicação? arredondamento? janela de tick?) com
   experimentos controlados.
3. Modelar o comportamento desejado (função determinística, anti-exploit de
   pacotes de velocidade).
4. Documentar causa raiz + correção proposta (implementação é da GAME-21/engine).

## Arquivos

- Criados: cenários de reprodução em `crates/wyd-sim/`, doc da causa raiz.
- Alterados: `crates/wyd-sim/src/lib.rs` (cenários, não a fórmula final sem ADR).
- Proibido tocar: `crates/wyd-core/src/`, nerf sem evidência.

## Testes / verificação

```bash
cargo test -p wyd-sim velocidade 2>&1 | tail -n 5
make check
```

Saída esperada: cenário reproduz a anomalia (números anexados: antes × desejado).

## Skills Relacionadas do freebuff

- .agents/skills/wyd-rust-sim - reprodução determinística.
- .agents/skills/wyd-rust-core - janela de tick como suspeita.

## Documentação de referencia

- pre-release-rust.md (FASE 10 §10.2).
- roadmap-rust/issues/GAME-20-velocidade.md.
- roadmap-rust/issues/GAME-19-equacoes.md - equações base.

## Subtasks

- [ ] Codificar issue
- [ ] Testar issue
- [ ] Revisar issue
- [ ] Documentar issue
- [ ] Versionar issue, mergear branch e fechar issue
- [ ] Limpar branch (se aplicável)

## Critérios de aceite

- [ ] Causa raiz explicada com cenário reproduzível; correção modelada.
- [ ] Manipulação de pacote de velocidade pelo cliente é inútil no modelo.

## Riscos / rollback

- Risco: corrigir sintoma e não causa → mitigação: experimento que elimina cada
  suspeita antes de concluir.
- Rollback: `git checkout -- crates/wyd-sim/`.
