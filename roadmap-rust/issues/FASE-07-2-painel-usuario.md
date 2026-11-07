# Issue 07-2 — Painel do usuário (conta, recuperação, rankings, personagens)

**Fase:** 7 · **Depende de:** 07-1 · **Bloqueia:** 07-4 · **Esforço:** M

## Objetivo

Entregar o painel do jogador contra o `wyd-admin`: gestão de conta, recuperação,
rankings em tempo real e visualização de personagens.

## Contexto / evidência

`pre-release-rust.md` FASE 7 §7.2. Dados vêm do banco (FASE-04) via gateway
(07-1); portal nunca escreve estado vivo.

## Passos

1. Listar telas e dados necessários por tela (fonte: gateway 07-1).
2. Implementar leitura via BFF (`adminCall`, sessão, `no-store` onde pessoal).
3. Rankings em tempo real com política de cache documentada (sem dado pessoal).
4. `portal-test` (tsc + vitest) + `make check`.

## Arquivos

- Criados: telas/componentes no portal, testes.
- Alterados: `portal/src/**` (somente o necessário), `crates/wyd-admin/src/`
  (endpoint faltante, se surgir — com teste).
- Proibido tocar: estado de personagem vivo, segredos no browser.

## Testes / verificação

```bash
cargo test -p wyd-admin
make check
```

Saída esperada: `test result: ok`; `portal-test` verde (log anexado).

## Skills Relacionadas do freebuff

- .agents/skills/wyd-web-admin-ui - BFF e telas.
- .agents/skills/frontend-design - direção visual.
- .agents/skills/ui-ux-practices - revisão de UX.

## Documentação de referencia

- portal/README.md - telas e overlays.
- pre-release-rust.md (FASE 7 §7.2).
- roadmap-rust/issues/FASE-07-1-gateway.md - contrato consumido.

## Subtasks

- [ ] Codificar issue
- [ ] Testar issue
- [ ] Revisar issue
- [ ] Documentar issue
- [ ] Versionar issue, mergear branch e fechar issue
- [ ] Limpar branch (se aplicável)

## Critérios de aceite

- [ ] 4 fluxos (conta, recuperação, rankings, personagens) funcionando via gateway.
- [ ] Respostas pessoais com `no-store`; sem segredo no cliente.

## Riscos / rollback

- Risco: vazar dado pessoal em cache/SSR → mitigação: revisão de
  cache-headers por tela.
- Rollback: `git checkout -- portal/ crates/wyd-admin/`.
