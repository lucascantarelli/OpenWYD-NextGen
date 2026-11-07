# Issue 02-2 — Processo de revisão consolidada a cada 10 issues

**Fase:** 2 · **Depende de:** 02-1 · **Bloqueia:** — (contínua) · **Esforço:** S

## Objetivo

Definir o checklist da auditoria de regressão, performance e integração
executada a cada 10 issues concluídas.

## Contexto / evidência

`pre-release-rust.md` FASE 2 item 2: revisão consolidada (regressão, benchmarks,
integração de módulos) a cada 10 issues. Sem checklist fixo, a revisão vira
"olhei e parece ok".

## Passos

1. Definir o checklist: `make check` + `cargo bench` comparativo + testes de
   integração entre módulos + revisão de ADRs pendentes.
2. Definir o gatilho: contador em `roadmap-rust/index.md` (a cada 10 issues
   fechadas).
3. Registrar onde o relatório vive (`docs/` + anexado à issue de revisão).
4. `make check`.

## Arquivos

- Criados: checklist (em `docs/rust/02-execucao.md` ou doc próprio).
- Alterados: `roadmap-rust/index.md` (contador de revisões).
- Proibido tocar: `crates/*/src/`, `.deprecated-go/`.

## Testes / verificação

```bash
make check
```

Saída esperada: gates verdes; checklist publicado com gatilho e local do relatório.

## Skills Relacionadas do freebuff

- .agents/skills/writing-plans - checklist e gatilhos.
- .agents/skills/wyd-rust-toolchain - gates e benches.

## Documentação de referencia

- pre-release-rust.md (FASE 2).
- roadmap-rust/issues/FASE-02-1-ciclo-execucao.md - ciclo base.

## Subtasks

- [ ] Codificar issue
- [ ] Testar issue
- [ ] Revisar issue
- [ ] Documentar issue
- [ ] Versionar issue, mergear branch e fechar issue
- [ ] Limpar branch (se aplicável)

## Critérios de aceite

- [ ] Checklist com regressão, benchmarks e integração; gatilho a cada 10 issues.
- [ ] Local do relatório definido.

## Riscos / rollback

- Risco: revisão pulada sob pressão → mitigação: issue de revisão é bloqueadora
  (contada no ritmo, não opcional).
- Rollback: `git checkout -- docs/rust/02-execucao.md roadmap-rust/index.md`.
