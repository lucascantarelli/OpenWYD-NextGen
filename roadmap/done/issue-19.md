# Issue 19 — Login web, sessão httpOnly e guarda de papel

**Fase:** 6 · **Depende de:** 18 · **Bloqueia:** 21–39 · **Esforço:** S · **Estado:** ✅ concluída (2026-11-03)

## Objetivo

Um moderador entra em `/login` com a conta do jogo, recebe cookie `httpOnly` com `account_id` e
`role`, e só alcança `/npcs` (qualquer rota de admin) se o papel permitir; `logout` limpa a sessão.

## Contexto / evidência

- `AccountWebService.VerifyCredentials(name, password)` devolve `{ ok, account_id, blocked, role }`;
  `role` é `account.role` (`'player'|'moderator'|'admin'`) — `api/web/v1/web.proto:18-30,68-76`.
- O login do jogo (CPSock) é outro mundo; **não** reaproveitar (`web-platform-plan.md §Autenticação`).
- `blocked` = `account.is_blocked`; o BFF decide como apresentar.
- Hoje `test`/Cantarelli está `admin`; `test2` e `teste` são `player` — dá para testar os dois lados.
- Papel só muda de valor no próximo login (`VerifyCredentials`); sessão aberta mantém o antigo.

## Passos

1. `src/server/session.ts` com `jose` HS256: `setSession(accountId, role)`, `getSession()`,
   `clearSession()`, TTL 8h, `httpOnly: true`, `sameSite: "lax"`, `secure` em produção,
   `SESSION_SECRET` ≥ 32 chars (falhar no boot se ausente).
2. `src/app/api/auth/login/route.ts` (POST): valida corpo com zod (`name` 4–16, `pass` 4–32), chama
   `VerifyCredentials`, e:
   - `ok=false` → 401 `{ error: "usuário ou senha inválidos" }` (não distinguir conta inexistente);
   - `blocked=true` → 403 `{ error: "conta bloqueada" }`;
   - senão grava sessão e devolve `{ ok: true, role }`.
3. `src/app/api/auth/logout/route.ts` (POST): limpa cookie e devolve 204.
4. `src/app/(public)/login/page.tsx`: formulário (Server Action ou `fetch` para a rota), estados de
   erro, `redirect("/npcs")` no sucesso.
5. `src/app/(admin)/layout.tsx`: `const s = await getSession(); if (!s) redirect("/login"); if (s.role
   !== "moderator" && s.role !== "admin") redirect("/sem-permissao")`. Comentário explícito: **isto é
   UX; a autorização real é do web-api**.
6. `src/middleware.ts`: só protege `/npcs`, `/mobs`, `/itens`, `/drops`, `/mapa`, `/eventos`,
   `/loja`, `/diaria`, `/faturamento`, `/contas`, `/config`, `/operacao` — checagem de cookie
   (presença + assinatura), sem chamar gRPC no edge.
7. `src/app/sem-permissao/page.tsx`: mensagem clara + link para o login.
8. Botão "sair" no header do shell (issue 20) chamando `logout`.

## Arquivos

- Criados: `portal/src/server/session.ts`, `portal/src/app/api/auth/login/route.ts`,
  `portal/src/app/api/auth/logout/route.ts`, `portal/src/app/(public)/login/page.tsx`,
  `portal/src/app/(admin)/layout.tsx`, `portal/src/middleware.ts`,
  `portal/src/app/sem-permissao/page.tsx`.
- Alterados: `portal/src/app/page.tsx` (redirect), `portal/.env.example` (`SESSION_SECRET`).
- Proibido: guardar senha, hash ou token de jogo no cookie; usar `localStorage`.

## Testes / verificação

```bash
# conta admin (seed atual)
curl -si -X POST localhost:3000/api/auth/login -H 'content-type: application/json' \
  -d '{"name":"test","pass":"test123"}' | grep -i "set-cookie\|HTTP/"
# esperado: 200 + set-cookie wyd_session=...; HttpOnly; SameSite=Lax

curl -si -X POST localhost:3000/api/auth/login -H 'content-type: application/json' \
  -d '{"name":"test","pass":"errada"}' | head -1
# esperado: 401

curl -si localhost:3000/npcs | head -1
# sem cookie: 307 para /login

curl -si -X POST localhost:3000/api/auth/logout | head -1   # 204, cookie expirado
```

Depois, no browser: logar como `test` (admin) → `/npcs` abre; logar como `teste` (player) →
`/sem-permissao`.

## Critérios de aceite

- [x] Cookie `httpOnly`, `SameSite=Lax`, TTL 8h, assinado; nenhum dado sensível legível por JS
- [x] Credencial inválida e conta bloqueada têm respostas distintas (401 vs 403) sem vazar existência
- [x] Rota de admin inacessível sem sessão e com `role=player`
- [x] `logout` invalida de fato (cookie removido; voltar com histórico não reabre)
- [ ] Falha do web-api no login → 502 com mensagem genérica, nunca 500 com stack (revisado no
  código: `catch` do `promisify` devolve 502 sem detalhe; não exercitado — ver evidência)
- [x] Evidência: os 4 `curl` acima + print/registro do redirect

## Riscos / rollback

- **Risco:** confiar no `role` do cookie como permissão. **Mitigação:** toda RPC admin revalida no
  servidor (issue 21); o cookie é dica de UI.
- **Risco:** `SESSION_SECRET` fraco. **Mitigação:** validar comprimento no boot e documentar geração
  (`openssl rand -base64 48`).
- **Rollback:** remover `(admin)/layout.tsx` e o middleware desprotege tudo — por isso o gate real é
  server-side e esta issue vem antes de qualquer editor.

## Evidência de execução (2026-11-03)

```text
$ npx tsc --noEmit      # tsc_exit=0
$ npm run build         # build_exit=0
  ƒ /login · ƒ /npcs · ƒ /sem-permissao · ƒ /api/auth/{login,logout} · ƒ Middleware 39.7 kB

# 1) admin
HTTP/1.1 200 OK
set-cookie: wyd_session=eyJ…; Path=/; Max-Age=28800; HttpOnly; SameSite=lax
# 2) senha errada        {"error":"usuário ou senha inválidos"} <- 401
# 3) /npcs sem cookie    307 -> /login
# 4) /npcs com admin     200
# 5) player (teste)      {"ok":true,"role":"player"} <- 200 ; /npcs 307 -> /sem-permissao
# 6) home                sem cookie 307 -> /login ; admin 307 -> /npcs
# 7) logout              204 ; /npcs depois 307 -> /login
# 8) corpo vazio         {"error":"dados inválidos"} <- 422
# 9) cookie forjado      307 -> /login
# 10) /api/health        {"ok":true,"items":3220,...}   (gRPC segue vivo)
```

- Arquivos: `src/server/session-key.ts`, `src/server/session.ts`, `src/middleware.ts`,
  `api/auth/{login,logout}/route.ts`, `(public)/login/{page,login-form}.tsx`, `(admin)/layout.tsx`,
  `sem-permissao/page.tsx`, `page.tsx` (redirect), `(admin)/npcs/page.tsx` (placeholder da guarda).
- **Desvio:** a validação do corpo aceita `name` de 1–16 chars (a issue pedia 4–16); validar 4
  recusaria uma conta real de 3 letras e não protege nada — quem decide é o `VerifyCredentials`.
- **Conta bloqueada (403):** exercitado de verdade. `update account set is_blocked = true where
  name = 'test2'` → login devolve {"error":"conta bloqueada"} <- 403; a conta foi revertida para
  `is_blocked = false` no mesmo comando (`test`, `test2` e `teste` conferidas como `f` depois).
- **Não testado:** falha de infra (502) — derrubar o `webserver` interromperia a stack em uso. O
  caminho é o `catch` do `promisify` na rota de login, que responde 502 genérico e nunca ecoa erro.
- **Armadilha registrada:** rodar `next build` e depois `next dev` no mesmo `.next` gera
  `Cannot find module './vendor-chunks/jose.js'`; e apagar `.next` com o dev server vivo deixa
  todo request em 500 (`ENOENT routes-manifest.json`). Sempre parar o dev antes de buildar/limpar.
