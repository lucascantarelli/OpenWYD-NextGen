# Issue 01-1 — Auditoria do estado real das issues (Go + Rust)

**Fase:** 1 · **Depende de:** 00-3 · **Bloqueia:** 01-2 · **Esforço:** S

## Objetivo

Inventariar `roadmap/` (Go) e `roadmap-rust/` por estado real com evidência,
sem mover nada para `done/` sem prova.

## Contexto / evidência

`pre-release-rust.md` §§1.1–1.6: auditar abertas/fechadas/em andamento,
`done/` e `test/`, verificando evidência real em vez de status manual.
Arquivo Go vive em `roadmap-rust/`; Rust em `roadmap-rust/`.

## Passos

1. Listar e contar: `ls roadmap-rust/issues roadmap-rust/done roadmap-rust/test roadmap-rust/issues`.
2. Para cada issue Go marcada concluída, buscar evidência (commit/teste/código);
   sem evidência, marcar como **hipótese** de incompleta.
3. Publicar tabela (issue → estado declarado → evidência → estado auditado).
4. `make check` (sanidade; sem código nesta issue).

## Arquivos

- Criados: tabela de auditoria (nesta issue ou `roadmap-rust/`).
- Alterados: `roadmap-rust/index.md` (contadores auditados).
- Proibido tocar: `done/` histórico (sem mover
  nada), código.

## Testes / verificação

```bash
make check
```

Saída esperada: gates verdes; tabela publicada com 100% das issues classificadas.

## Skills Relacionadas do freebuff

- .agents/skills/writing-plans - estruturar o inventário.
- .agents/skills/documentation-writer - tabela de auditoria.
- .agents/skills/wyd-rust-toolchain - `make check`.

## Documentação de referencia

- pre-release-rust.md (§§1.1–1.6, FASE 1).
- pre-release.md (§§1.1–1.6) - regras de auditoria do legado.
- roadmap-rust/index.md - contadores a atualizar.

## Subtasks

- [ ] Codificar issue
- [ ] Testar issue
- [ ] Revisar issue
- [ ] Documentar issue
- [ ] Versionar issue, mergear branch e fechar issue
- [ ] Limpar branch (se aplicável)

## Critérios de aceite

- [ ] Todas as issues classificadas com evidência ou **hipótese** explícita.
- [ ] Nada movido para `done/` sem prova; evidência anexada.

## Riscos / rollback

- Risco: classificar errado por falta de contexto → mitigação: marcação
  **hipótese** obrigatória quando incerto.
- Rollback: `git checkout -- roadmap-rust/`.
