# Issue 04-3 — Persistência assíncrona fora do tick

**Fase:** 4 · **Depende de:** 04-2 · **Bloqueia:** FASE-07 · **Esforço:** L

## Objetivo

Implementar o pool async (`sqlx`) com transações curtas e fila de persistência,
sem nenhum I/O bloqueante no tick loop.

## Contexto / evidência

`pre-release-rust.md` FASE 4 §4.2: persistência async em segundo plano, sem I/O
bloqueante no tick. Regra Hot/Cold (`AGENTS.md`): banco é Cold Path.

## Passos

1. Implementar pool em `crates/wyd-db/src/` (timeouts, limite de conexões,
   transações curtas por operação).
2. Implementar a fila de persistência (drain periódico pelo core, sem lock
   global — canal/fila dedicada).
3. Definir política de save (periódico, on-demand, on-shutdown) e documentar.
4. Teste: latência de tick inalterada sob carga de saves (números anexados).

## Arquivos

- Criados: pool, fila e testes em `crates/wyd-db/src/`.
- Alterados: `crates/wyd-db/Cargo.toml` (sqlx via ADR).
- Proibido tocar: `crates/wyd-core/src/tick.rs` (nenhum await de banco no
  quente), segredos.

## Testes / verificação

```bash
cargo test -p wyd-db
make check
```

Saída esperada: `test result: ok`; teste de carga de saves sem degradar o tick.

## Skills Relacionadas do freebuff

- .agents/skills/wyd-rust-db - pool e transações.
- .agents/skills/wyd-rust-core - interface de drain pelo core.
- .agents/skills/rust-code-quality - timeouts, erros tipados.

## Documentação de referencia

- pre-release-rust.md (FASE 4 §4.2, §0.2 Hot/Cold).
- roadmap-rust/issues/FASE-04-2-migrations.md - schema base.

## Subtasks

- [ ] Codificar issue
- [ ] Testar issue
- [ ] Revisar issue
- [ ] Documentar issue
- [ ] Versionar issue, mergear branch e fechar issue
- [ ] Limpar branch (se aplicável)

## Critérios de aceite

- [ ] Zero await de banco no tick; fila drenada com política documentada.
- [ ] Save-on-shutdown sem perda (teste dedicado).

## Riscos / rollback

- Risco: perda de dados em crash entre saves → mitigação: janela de perda
  documentada + teste de shutdown.
- Rollback: `git checkout -- crates/wyd-db/`.
