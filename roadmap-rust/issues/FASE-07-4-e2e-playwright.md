# Issue 07-4 — E2E Playwright (auth, loja, painel)

**Fase:** 7 · **Depende de:** 07-2, 07-3 · **Bloqueia:** FASE-15 · **Esforço:** M

## Objetivo

Cobrir com Playwright os fluxos críticos (autenticação, loja, painel) com
idempotência de entrega na loja.

## Contexto / evidência

`pre-release-rust.md` FASE 7 §7.2: E2E nos fluxos críticos. Loja exige entrega
rastreável sem duplicidade (regra permanente).

## Passos

1. Escrever specs: login/logout, compra na loja, painel usuário, painel admin.
2. Garantir idempotência de entrega (transação rastreável + teste de duplo
   clique/retry).
3. Rodar contra ambiente local com backend `wyd-admin` (07-1).
4. `portal-test` + `make check`.

## Arquivos

- Criados: specs E2E no portal, fixtures.
- Alterados: `portal/**` (somente suporte aos testes).
- Proibido tocar: dados de produção, segredos reais nas fixtures.

## Testes / verificação

```bash
cargo test -p wyd-admin
make check
```

Saída esperada: `test result: ok`; E2E verde (log Playwright anexado);
retry de compra não duplica entrega.

## Skills Relacionadas do freebuff

- .agents/skills/wyd-web-admin-ui - fluxos críticos.
- .agents/skills/wyd-rust-db - idempotência transacional.

## Documentação de referencia

- portal/README.md - fluxos e BFF.
- pre-release-rust.md (FASE 7 §7.2).

## Subtasks

- [ ] Codificar issue
- [ ] Testar issue
- [ ] Revisar issue
- [ ] Documentar issue
- [ ] Versionar issue, mergear branch e fechar issue
- [ ] Limpar branch (se aplicável)

## Critérios de aceite

- [ ] E2E verde nos 3 fluxos; teste de duplicidade de entrega passando.

## Riscos / rollback

- Risco: E2E flaky mascarando regressão → mitigação: retries limitados +
  evidência de estabilidade (3 execuções verdes).
- Rollback: `git checkout -- portal/`.
