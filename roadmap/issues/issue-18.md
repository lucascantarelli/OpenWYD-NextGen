# Issue 18 — Portal Next.js: bootstrap do BFF e cliente gRPC

**Fase:** 6 · **Depende de:** 04 (`webserver` no ar em `:7600`) · **Bloqueia:** 19–39 · **Esforço:** S
· **Estado:** ✅ concluída (2026-11-03)

## Objetivo

Existir `portal/` na raiz do workspace: um app Next.js (App Router, TypeScript) que fala gRPC com o
`webserver` e prova isso numa rota `/api/health`.

## Contexto / evidência

- O `webserver` é gRPC puro: `grpc.NewServer(grpc.Creds(creds))`, `net.Listen("tcp", *addr)` com
  `-addr :7600`, **sem** `ListenAndServe` (`webserver/cmd/webserver/main.go:98,187`). Não
  existe UI no repositório — os 8 guias de `docs/integrations/` descrevem como construí-la.
- mTLS é opcional: `internal/secure.ServerCreds` devolve `insecure` quando certs não estão
  configurados (`internal/secure/tls.go:37-51`). Em dev local conectamos sem TLS.
- O compose publica `7600:7600` explicitamente "so the Next.js BFF (running on the host) can reach
  the web-api over gRPC" (`docker-compose.yaml:65-68`).
- `portal/` **não** pode ficar dentro de ``: o build Docker faz `COPY . .` com esse contexto.

## Passos

1. Criar o app na raiz do workspace:
   ```bash
   cd /d/dolorestec/wyd2
   mkdir portal && cd portal
   npm init -y
   npm i next@15 react@19 react-dom@19 @grpc/grpc-js @grpc/proto-loader jose zod
   npm i -D typescript @types/node @types/react @types/react-dom eslint eslint-config-next
   ```
2. `package.json`: scripts `dev` (`next dev -p 3000`), `build`, `start`, `lint` (`next lint`),
   `typecheck` (`tsc --noEmit`).
3. `tsconfig.json` com `"strict": true`, `"moduleResolution": "bundler"`, alias `@/* → ./src/*`.
4. `next.config.ts`: `{ output: "standalone", experimental: { serverActions: { bodySizeLimit: "2mb" } } }`.
5. `.env.example` (versionado) e `.env.local` (ignorado):
   ```
   WEB_API_ADDR=127.0.0.1:7600
   SESSION_SECRET=<32+ chars>
   # produção: WEB_API_TLS_CA=../certs/ca.crt
   #             WEB_API_TLS_CERT=../certs/client.crt
   #             WEB_API_TLS_KEY=../certs/client.key
   ```
6. `.gitignore`: `node_modules`, `.next`, `.env.local`, `*.tsbuildinfo`.
7. `src/server/grpc.ts` — carregar o proto real do repo (não copiar!):
   ```ts
   import path from "node:path";
   import fs from "node:fs";
   import * as grpc from "@grpc/grpc-js";
   import * as protoLoader from "@grpc/proto-loader";

   const PROTO = path.join(process.cwd(), "..", "OpenWYD", "api", "web", "v1", "web.proto");
   const def = protoLoader.loadSync(PROTO, {
     keepCase: true, longs: Number, enums: Number, defaults: true, oneofs: true,
   });
   const pkg = grpc.loadPackageDefinition(def) as any;
   const addr = process.env.WEB_API_ADDR ?? "127.0.0.1:7600";
   const creds = process.env.WEB_API_TLS_CA
     ? grpc.credentials.createSsl(
         fs.readFileSync(process.env.WEB_API_TLS_CA),
         fs.readFileSync(process.env.WEB_API_TLS_KEY!),
         fs.readFileSync(process.env.WEB_API_TLS_CERT!),
       )
     : grpc.credentials.createInsecure();

   export const itemCatalog = new pkg.web.v1.ItemCatalogService(addr, creds);
   export const accountWeb = new pkg.web.v1.AccountWebService(addr, creds);
   export function promisify<Req, Res>(fn: Function) {
     return (req: Req) => new Promise<Res>((ok, fail) =>
       fn(req, (err: unknown, res: Res) => (err ? fail(err) : ok(res))));
   }
   ```
8. `src/app/api/health/route.ts` — chama `ItemCatalogService.ListItems`:
   ```ts
   export const dynamic = "force-dynamic";
   export async function GET() {
     try {
       const r = await promisify<any, any>(itemCatalog.listItems.bind(itemCatalog))({});
       return Response.json({ ok: true, items: r.items?.length ?? 0, catalogVersion: r.catalog_version });
     } catch (e) {
       return Response.json({ ok: false, error: "web-api indisponível" }, { status: 502 });
     }
   }
   ```
9. `src/app/page.tsx`: redireciona para `/login` (a página real vem na issue 19).

## Arquivos

- Criados: `portal/package.json`, `portal/tsconfig.json`, `portal/next.config.ts`,
  `portal/.env.example`, `portal/.gitignore`, `portal/src/server/grpc.ts`,
  `portal/src/app/api/health/route.ts`, `portal/src/app/page.tsx`, `portal/src/app/layout.tsx`.
- Alterados: `AGENTS.md` (§1 tabela de partes do workspace e §7 tabela de skills), `.gitignore` raiz
  se necessário (não adicionar `portal/` — o portal é versionado).
- Proibido tocar: `` (Go), `Release/`, `client/`.

## Testes / verificação

```bash
cd portal
npx tsc --noEmit
npm run build
npm run dev &                     # ou: npx next dev -p 3000
curl -s localhost:3000/api/health
```

Saída esperada: `{"ok":true,"items":3220,"catalogVersion":"<hash>"}` (o valor de `items` é o mesmo
`items=3220` do boot do tmserver). Sem a stack: `{"ok":false,...}` com **502**, não 500 nem stack.

## Critérios de aceite

- [ ] `portal/` na raiz do repositório, com `node_modules` ignorado
- [ ] Proto carregado por caminho relativo ao repositório (uma única fonte da verdade)
- [ ] `/api/health` devolve contagem real do catálogo e versão
- [ ] Falha do web-api vira 502 com mensagem genérica
- [ ] `tsc --noEmit` e `next build` verdes, saída colada na issue
- [ ] `.env.example` documenta as três variáveis de TLS de produção

## Riscos / rollback

- **Risco:** caminho do proto quebrar no build standalone. **Mitigação:** `path.join(process.cwd(),
  "..", …)` + teste no `npm run build`; em deploy, montar `api` ao lado do app.
- **Risco:** alguém commitar `.env.local` com segredo. **Mitigação:** `.gitignore` no passo 6 e
  revisão no PR.
- **Rollback:** apagar `portal/` — nada do servidor depende dele.

## Evidência de execução (2026-11-03)

```text
$ node -v && npm -v
v26.4.0
11.17.0

$ npm install --no-audit --no-fund
added 59 packages in 3m          # EXIT=0

$ npx tsc --noEmit               # tsc_exit=0
$ npm run build                  # build_exit=0
   ▲ Next.js 15.5.28
 ✓ Compiled successfully in 14.5s
 ┌ ƒ /api/health   127 B   103 kB

$ curl -s localhost:3000/api/health
{"ok":true,"items":3220,"catalogVersion":"acce2f7cf63fb856","iconPackVersion":"3260f9a7065f88ef"}
```

- `items=3220` é o mesmo valor de `items=3220` no boot do tmserver — a leitura vem do
  `-content /Release` do web-api, não de um mock.
- **Desvio consciente:** `src/app/page.tsx` é uma página de status (não um redirect para `/login`),
  porque `/login` só existe na issue 19. A issue 19 substitui esta página.
- **Pendência registrada:** `npm install` avisou que o postinstall de `protobufjs` não rodou
  (allow-scripts). O `proto-loader` funcionou nos testes; se aparecer erro de runtime de protobuf,
  rodar `npm approve-scripts protobufjs`.
