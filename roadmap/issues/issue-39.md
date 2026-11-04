# Issue 39 — QA final do portal: E2E, segurança e fechamento da Fase 6

**Fase:** 6 · **Depende de:** 18–38 · **Bloqueia:** — · **Esforço:** M · **Estado:** ⏳ aberta

## Objetivo

Provar que o portal está completo e seguro: suíte E2E cobrindo login, guarda de papel e um fluxo de
cada editor; revisão de exposição de segredos/gRPC; acessibilidade e performance mínimas; documentação
fechada.

## Contexto / evidência

- Até aqui há testes unitários por issue, mas nenhum caminho de ponta a ponta browser→BFF→gRPC→DB.
- Riscos reais que só aparecem juntos: cookie de sessão vazando papel antigo, bundle do cliente com
  string de conexão, gRPC acessível fora do host, `Release/` alterado por engano, 3.2k itens travando
  a primeira renderização.
- O projeto já tem o hábito de exigir evidência (log, teste verde) para fechar issue
  (`roadmap/index.md` §"Como ler uma issue").

## Passos

1. **Playwright** (`portal/e2e/`) com `webServer` apontando para a stack local; `storageState` por
   papel. Cenários mínimos:
   - `login.spec.ts`: credencial errada 401; `player` cai em `/sem-permissao`; `admin` entra;
     logout limpa.
   - `npcs.spec.ts`: criar → ocultar → loja com 2 itens → apagar (ou 409 em conteúdo).
   - `mobs.spec.ts`: abrir template sem override (read-through), salvar `exp`, reverter.
   - `itens.spec.ts`: buscar item, abrir detalhe, conferir ícone/fallback.
   - `config.spec.ts`: mudar uma rate, restaurar; mudar papel de `teste` e voltar.
   - `operacao.spec.ts`: teletransportar o char de teste e ver a posição mudar no snapshot.
2. **Segurança** (checklist com evidência):
   - `grep -r "moderatorId" portal/src/app` não encontra valor vindo de body;
   - `grep -rn "grpc\|7600" portal/.next/static` sem resultados (nada de gRPC no bundle);
   - `grep -rn "SESSION_SECRET\|postgres://" portal/.next/static` sem resultados;
   - cookie com `HttpOnly`/`SameSite` conferido via `curl -i`;
   - request cross-origin a uma action → recusada (Server Actions do Next já validam Origin — provar);
   - `docker compose exec webserver` sem `W2PP_TLS_*` **apenas** em dev; documentar produção com mTLS.
3. **Acessibilidade**: `npx @axe-core/cli http://localhost:3000/login` e `/npcs` sem violações
   críticas; navegação só teclado em 3 telas.
4. **Performance**: primeira carga de `/itens` < 2s local; navegação entre páginas < 500ms; conferir
   que `ListItems` é chamado 1× por versão de catálogo (log do BFF).
5. **Documentação**: `portal/README.md` (setup, env, papéis, como rodar E2E), atualizar
   `roadmap/index.md` (Fase 6 concluída + changelog), `docs/migration/SESSION-PRIMER.md`
   (linha do portal) e `AGENTS.md` (§1 inclui `portal/`; §7 inclui a skill `wyd-web-admin-ui`).
6. **Fechamento**: checklist final mapeando 18–38, com o que ficou explicitamente fora de escopo
   (edição de `nSkillData.csv`, market/loja de jogador, gateway de pagamento, reset de senha).

## Arquivos

- Criados: `portal/playwright.config.ts`, `portal/e2e/*.spec.ts`, `portal/e2e/fixtures.ts`,
  `portal/README.md` (se ainda não existir), `portal/.github/workflows/portal.yml` (se o repo usar
  Actions) ou alvo no `Makefile`.
- Alterados: `roadmap/index.md`, `AGENTS.md`, `docs/migration/SESSION-PRIMER.md`.
- Proibido: rodar E2E contra a stack que alguém está testando à mão sem avisar; apagar dados de
  personagem real; usar a conta `test` como cobaia destrutiva (criar `e2e-admin`/`e2e-player`).

## Testes / verificação

```bash
cd portal
npx tsc --noEmit && npm run lint && npm run build
npx playwright test --reporter=list
npx @axe-core/cli http://localhost:3000/login
grep -rn "7600\|grpc" .next/static | head    # esperado: vazio
```

```bash
# contas de teste dedicadas
docker compose run --rm dbserver seed-account -name e2e-admin -pass e2e12345 \
  -dsn "postgres://postgres:dev@db:5432/postgres"
docker compose exec -T db psql -U postgres -d postgres -c "update account set role='admin' where name='e2e-admin';"
```

Saída esperada: todos os specs verdes; zero violação crítica de a11y; greps vazios; `npm run build`
sem warning de segredo.

## Critérios de aceite

- [ ] Suíte E2E cobre login/guarda + 1 fluxo de cada editor principal, com contas dedicadas
- [ ] Nenhuma credencial, DSN ou endpoint gRPC no bundle do cliente (evidência dos greps)
- [ ] Cookie `HttpOnly`; mutação cross-origin recusada
- [ ] a11y sem violação crítica em login e uma tela de tabela
- [ ] `/itens` carrega < 2s local e chama o catálogo 1× por versão
- [ ] Documentação atualizada (README do portal, roadmap, SESSION-PRIMER, AGENTS.md)
- [ ] Lista explícita do que ficou fora de escopo

## Riscos / rollback

- **Risco:** E2E instável por depender do estado do mundo. **Mitigação:** cada spec cria o que usa
  (`slug` único por execução) e limpa no `afterEach`; nunca assumes dados existentes.
- **Risco:** rodar E2E e atrapalhar teste humano. **Mitigação:** janela combinada; stack separada
  quando possível; nunca durante validação do usuário.
- **Rollback:** E2E é aditivo; remover os specs não altera o produto.
