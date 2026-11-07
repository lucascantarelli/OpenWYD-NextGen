# Issue 21 — Camada de admin no BFF: RPC tipada, resultado→HTTP, auditoria

**Fase:** 6 · **Depende de:** 18, 19 · **Bloqueia:** 22–31 · **Esforço:** M · **Estado:** ✅ concluída (2026-11-03)

## Objetivo

Toda chamada admin do portal passa por **um único caminho** que promisifica a RPC, deriva
`moderator_id` da sessão, mapeia `AdminResult` inteiro para HTTP e registra a ação num log de
auditoria da UI.

## Contexto / evidência

- `AdminResult` viaja **no corpo**; só falha de infraestrutura é erro gRPC
  (`api/web/v1/web.proto:196-208`). Valores: `OK=1`, `FORBIDDEN=2`, `INVALID=3`,
  `NOT_FOUND=4`, `CONTENT_OWNED=5`.
- O bug #257 nasceu de um mapa `AdminResult → HTTP` que não conhecia `CONTENT_OWNED` e devolvia 500
  para um resultado **esperado** de `DeleteNpc` (NPC de conteúdo deve ser ocultado, não apagado) —
  ver `docs/integrations/npc-admin-nextjs.md §4.1`.
- Cada RPC admin recebe `moderator_id` e revalida `account.role` server-side com trilha de auditoria
  (`web.proto:206,478,703,821,1049`). O BFF não pode "confiar" que já verificou.

## Passos

1. `src/server/admin.ts`:
   ```ts
   export const ADMIN_RESULT = { UNSPECIFIED: 0, OK: 1, FORBIDDEN: 2, INVALID: 3, NOT_FOUND: 4, CONTENT_OWNED: 5 } as const;
   const HTTP: Record<number, number> = { 1: 200, 2: 403, 3: 422, 4: 404, 5: 409, 0: 500 };
   const MSG: Record<number, string> = {
     0: "resultado desconhecido do web-api",
     2: "sua conta não tem permissão para esta ação",
     3: "dados inválidos para esta ação",
     4: "o registro não existe mais",
     5: "este item é conteúdo versionado: oculte em vez de apagar",
   };
   export async function adminCall<T extends { result: number }>(
     fn: (req: any, cb: (e: unknown, r: T) => void) => void,
     req: Omit<any, "moderatorId">,
     audit: { action: string; target?: string },
   ) {
     const s = await requireStaff();                    // lança 401/403 tipado
     const res = await promisify<any, T>(fn)({ ...req, moderatorId: s.accountId });
     const status = HTTP[res.result] ?? 500;
     logAudit({ accountId: s.accountId, role: s.role, ...audit, result: res.result, status });
     if (status !== 200) throw new AdminError(status, MSG[res.result] ?? MSG[0]);
     return res;
   }
   ```
2. `src/server/errors.ts`: `AdminError(status, message)` + `toResponse(e)` que converte em
   `Response.json({ error }, { status })` e **nunca** vaza stack, DSN, caminho ou nome de tabela.
   Erros gRPC de infra (`UNAVAILABLE`, `DEADLINE_EXCEEDED`) → **502** `"serviço de administração
   indisponível"`.
3. `src/server/session.ts`: `requireStaff()` — lê sessão, exige `role ∈ {moderator, admin}`, devolve
   `{ accountId, role }`. Usado **só** no server.
4. `src/server/audit.ts`: `logAudit(entry)` → `console.info` estruturado (`JSON.stringify`) com
   `ts, accountId, role, action, target, result, status`. Sem PII além do id. Arquivo
   `portal/logs/audit-YYYY-MM-DD.ndjson` opcional via `AUDIT_LOG_DIR`.
5. `src/server/promisify.ts`: helper único, com timeout de 10s (`grpc.Deadline`) para não pendurar a
   request do browser num web-api travado.
6. Teste unitário `src/server/admin.test.ts` (vitest): os 6 valores de `result` → status/mensagem
   esperados; `UNSPECIFIED` e valor desconhecido → 500 com mensagem genérica.
7. Documentar no README do portal: "para adicionar uma ação admin, use `adminCall`; não chame o
   cliente gRPC direto de uma page".

## Arquivos

- Criados: `portal/src/server/{admin,errors,audit,promisify}.ts`, `portal/src/server/admin.test.ts`,
  `portal/README.md`.
- Alterados: `portal/src/server/grpc.ts` (deadline default), `portal/src/server/session.ts`.
- Proibido: `try/catch` que engole `AdminResult != OK`; qualquer `any` no retorno de `adminCall`
  usado pelo componente.

## Testes / verificação

```bash
cd portal && npx vitest run src/server/admin.test.ts && npx tsc --noEmit
```

Cenário manual (após a issue 23): logado como `teste` (`player`), tentar salvar um NPC →
`403 "sua conta não tem permissão"`; logado como `test`, apagar NPC de conteúdo → `409` com a
mensagem de ocultar; derrubar o `webserver` (`docker compose stop webserver`) → `502`.

## Critérios de aceite

- [x] Nenhuma page/action chama cliente gRPC admin sem passar por `adminCall`
- [x] `moderatorId` sempre da sessão; grep do repo não acha `moderatorId` vindo de body
- [x] Os 5 resultados de negócio mapeados + `UNSPECIFIED`/desconhecido → 500 genérico
- [x] `CONTENT_OWNED` → 409 com mensagem acionável (não 500)
- [x] Toda ação registra auditoria com conta, papel, alvo e resultado
- [x] Timeout de 10s e 502 quando o web-api está fora

## Riscos / rollback

- **Risco:** o wrapper virar gargalo de tipos. **Mitigação:** genéricos por RPC + tipos derivados de
  `web.proto` em `src/lib/proto-types.ts`.
- **Risco:** auditoria com dado sensível. **Mitigação:** só ids, nomes de campo e resultado; nunca
  senha, hash ou conteúdo de item inteiro.
- **Rollback:** `adminCall` é uma função pura; reverter exige ajustar as actions que o usam (poucas,
  porque todas as telas seguintes dependem dele).

## Evidência de execução (2026-11-03)

```text
$ npx vitest run src/server/admin.test.ts    # 8 passed, vitest_exit=0
$ npx tsc --noEmit                           # tsc_exit=0

# sonda temporária (rota /api/probe, removida depois) chamando adminCall + toResponse
sem sessão          http=401 {"error":"sessão expirada: entre novamente"}
admin ok            http=200 {"ok":true,"result":1,"moderatorId":1}
admin content_owned http=409 {"error":"este item é conteúdo versionado: oculte em vez de apagar"}
admin web-api fora  http=502 {"error":"serviço de administração indisponível"}
player              http=403 {"error":"sua conta não tem permissão para esta ação"}

# auditoria no log do processo (console.info estruturado)
[audit] {"ts":"…","accountId":1,"role":"admin","action":"probe.check","target":"npc:42","result":1,"status":200}
[audit] {"ts":"…","accountId":1,"role":"admin","action":"probe.check","target":"npc:42","result":5,"status":409}

# o promisify novo (deadline via CallOptions) continua servindo o caminho real
$ curl -s 127.0.0.1:3000/api/health → {"ok":true,"items":3220,…}  http=200
```

- Criados: `src/server/{admin,admin-result,errors,audit,promisify}.ts`, `src/server/admin.test.ts`,
  `README.md`. Alterados: `src/server/{grpc,session}.ts`, `package.json` (vitest + `npm test`).
- **Desvio 1:** o mapa `AdminResult → HTTP` vive em `src/server/admin-result.ts`, módulo **puro** que
  `admin.ts` consome. Motivo: o teste unitário precisa rodar sem `next/headers` nem
  `SESSION_SECRET`; manter o mapa dentro do wrapper acoplaria o teste ao runtime do Next.
- **Desvio 2:** `promisify` saiu de `grpc.ts` para `src/server/promisify.ts` e agora aplica deadline
  por `CallOptions` (o timer antigo só evitava promise pendurada). `grpc.ts` reexporta
  `promisify`/`Unary`, então `api/health`, `api/auth/login` e `server/catalog.ts` não mudaram.
- **Pendente:** o 502 foi exercitado com `UNAVAILABLE` simulado, não derrubando o `webserver`; o
  deadline de 10s está aplicado, mas não foi observado contra um peer que aceita a conexão e trava.
- **Pendente:** `moderator_id` só será enviado de verdade quando a primeira tela admin existir
  (issue 22/23); hoje o grep confirma que nenhum valor de `moderator_id` vem de body.
