# PORTAL-01 — Matriz de papéis, middleware default-deny e teste de autorização por rota

**Fase:** revisão · **Depende de:** — · **Esforço:** M · **Estado:** ⏳ proposta

## Objetivo

A separação público × admin deixar de depender de cada página lembrar de checar sessão: middleware
default-deny, matriz papel × rota × ação documentada e coberta por teste.

## Contexto/evidência (05/11/2026)

- `portal/src/app` já tem `(admin)`, `(public)`, `api`, `sem-permissao`; `(public)` está raso.
- Toda ação admin passa por `adminCall` (`portal/src/server/admin.ts`: sessão, `moderator_id` da
  sessão, deadline, `AdminResult` → HTTP, auditoria) — o caminho de escrita está correto.
- O que falta é a **leitura**: não há `portal/src/middleware.ts` (nenhum arquivo), então cada página
  guarda a si mesma.
- Verificado nesta sessão que o guarda funciona no caso feliz (login `test/test123` → `role=admin`,
  páginas 200 com dados) e no caso negado (`/drops`, `/itens` → 307 sem sessão), mas isso não está
  automatizado (ver TEST-03) nem documentado como matriz.

## Passos

1. `portal/src/middleware.ts` default-deny: `(admin)/**` exige papel de staff; `(public)/**` só
   libera rotas listadas; `api/**` exige sessão exceto `auth/*` e `health`.
2. Escrever `docs/analysis/portal-papeis.md` com a matriz papel (`player`, `support`, `moderator`,
   `admin`) × rota × ação, citando o RPC que a página chama.
3. Reforçar `requireStaff`/`adminCall` para que nenhuma página leia o gRPC admin direto (checagem
   estática: `grep` no CI por import de cliente gRPC em `app/**`).
4. Teste de autorização por rota (unitário de middleware + fluxo Playwright do TEST-03): anônimo →
   redirect, `player` → `/sem-permissao`, `moderator` → 200 na rota permitida e 403 na de admin.
5. Revisar as rotas `(public)` novas para nunca lerem dado de personagem sem sessão.

## Arquivos

- `portal/src/middleware.ts` (novo)
- `docs/analysis/portal-papeis.md` (novo)
- `portal/src/app/(admin)/**` (remoção de guardas duplicadas)
- `portal/src/server/admin.ts`, `portal/src/server/session.ts`

## Testes/verificação

```bash
cd portal && npm run typecheck && npm test
curl -s -o /dev/null -w "%{http_code}\n" http://127.0.0.1:3000/itens        # sem sessão: 307/302
curl -s -o /dev/null -w "%{http_code}\n" -b cookies.txt http://127.0.0.1:3000/itens   # admin: 200
```

## Critérios de aceite

- [ ] Middleware default-deny cobrindo `(admin)`, `api` e `(public)`.
- [ ] Matriz de papéis documentada e referenciada no `AGENTS.md`.
- [ ] Nenhuma página chamando cliente gRPC admin fora de `adminCall`.
- [ ] Teste automatizado prova 3 casos: anônimo, `player`, `admin`.

## Riscos/rollback

Risco: middleware bloquear rota legítima e derrubar o painel. Mitigação: lista explícita de exceções
e teste de 200 para cada rota admin usada no dia a dia; rollback: desligar o matcher do middleware.
