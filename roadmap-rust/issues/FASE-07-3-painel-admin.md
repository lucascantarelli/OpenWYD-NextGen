# Issue 07-3 — Painel administrativo (auditoria, economia, moderação)

**Fase:** 7 · **Depende de:** 07-1 · **Bloqueia:** 07-4 · **Esforço:** M

## Objetivo

Entregar o painel de staff com auditoria de logs, estatísticas de economia e
ferramentas de moderação, tudo rastreável.

## Contexto / evidência

`pre-release-rust.md` FASE 7 §7.2. Auditoria nasce no gateway (07-1); aqui é
visualização + ações moderadas com confirmação.

## Passos

1. Listar ações de moderação e relatórios (economia, logs) com dados fonte.
2. Implementar telas com confirmação para ações destrutivas e trilha de auditoria.
3. Estatísticas de economia a partir de dados auditados (sem número inventado).
4. `portal-test` + `make check`.

## Arquivos

- Criados: telas admin + testes no portal.
- Alterados: `portal/src/**`, `crates/wyd-admin/src/` (endpoint faltante, com teste).
- Proibido tocar: bypass de auditoria, ação sem confirmação destrutiva.

## Testes / verificação

```bash
cargo test -p wyd-admin
make check
```

Saída esperada: `test result: ok`; `portal-test` verde; ação de moderação gera
registro auditável.

## Skills Relacionadas do freebuff

- .agents/skills/wyd-web-admin-ui - auditoria e papéis.
- .agents/skills/frontend-design - direção visual.
- .agents/skills/ui-ux-practices - confirmações e UX de risco.

## Documentação de referencia

- portal/README.md - overlays e auditoria.
- pre-release-rust.md (FASE 7 §7.2).

## Subtasks

- [ ] Codificar issue
- [ ] Testar issue
- [ ] Revisar issue
- [ ] Documentar issue
- [ ] Versionar issue, mergear branch e fechar issue
- [ ] Limpar branch (se aplicável)

## Critérios de aceite

- [ ] Toda ação de moderação auditada e confirmada; economia com fonte auditada.

> Nota (2026-11-07): aguarda transporte do gateway (07-1 backend pronto); auditoria nasce pronta no backend (`auth.denied`, recibos de grant).

## Riscos / rollback

- Risco: ação irreversível sem confirmação → mitigação: confirmação obrigatória
  + soft-action quando possível.
- Rollback: `git checkout -- portal/ crates/wyd-admin/`.
