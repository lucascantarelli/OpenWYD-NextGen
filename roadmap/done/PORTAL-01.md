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

## Resultado (06/11/2026) — guarda de rota provada por teste

**A afirmação "não há `portal/src/middleware.ts`" estava errada**: o arquivo existe desde a entrega
do portal e valida o JWT HS256 do cookie no edge. O defeito real era outro, e pior:

| Achado | Evidência |
|---|---|
| `matcher` cobria **14 de 19** rotas admin: `/precos`, `/atlas`, `/guildas`, `/castelo` ficavam abertas | derivado do disco por `middleware.test.ts` |
| havia um array `GUARDED` morto (`void GUARDED`) listando as 19 — documentava proteção que o `matcher` não aplicava | `git show 3157207` |
| `/operacao` estava no `matcher` sem página existir | idem |

Corrigido: `matcher` agora tem as 19 rotas que existem, `/operacao` saiu (a página ainda não existe;
quando nascer, o teste de cobertura cobra a entrada).

### Testes (rodam no `npm test`)

`portal/src/middleware.test.ts` — 8 casos: sem cookie → 307 para `/login` com query limpa; rota
aninhada barrada; cookie válido passa (`x-middleware-next`); cookie assinado com outra chave → 307;
cookie expirado → 307; **cobertura** derivada de `app/(admin)` no disco (toda página no matcher);
**cobertura** derivada do menu (`SHELL_ROUTES` sem `soon`); nenhum padrão do matcher apontando para
rota inexistente; rotas públicas fora do matcher; regressão nomeada de `/precos`, `/atlas`,
`/guildas`, `/castelo`.

`portal/src/app/api/search/accounts/route.test.ts` — 3 casos na fronteira de autorização real (o
`route.ts` que existe, não um endpoint inventado para o teste): sem sessão → **401**, papel `player` →
**403**, staff → 200 com `cache-control: private, no-store`.

```
npx vitest run   →  12 arquivos, 93 testes, todos passando
npx tsc --noEmit →  exit 0
```

O teste reprova de fato: ao rodá-lo com o `matcher` de 06/11 ele apontou as 4 rotas abertas e o
`/operacao` órfão antes de qualquer correção.

### O que continua aberto nesta issue

- Matriz papel × rota × ação em `docs/analysis/portal-papeis.md` (não escrita).
- Checagem estática no CI de cliente gRPC admin fora de `adminCall`.
- Default-deny para `api/**` (hoje cada rota checa a si mesma; só `/api/search/accounts` tem teste).
- Fluxo Playwright por papel — pertence à TEST-03 (Fila B).

## Riscos/rollback

Risco: middleware bloquear rota legítima e derrubar o painel. Mitigação: lista explícita de exceções
e teste de 200 para cada rota admin usada no dia a dia; rollback: desligar o matcher do middleware.
