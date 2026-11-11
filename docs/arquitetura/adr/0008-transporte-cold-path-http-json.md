# ADR-0008 — Transporte do Cold Path: HTTP/1.1 + JSON em loopback (sem gRPC/proto)

- **Estado:** aceita
- **Data:** 2026-11-07
- **Decisores:** dono do projeto + agente (decidido nesta sessão, a pedido do dono)
- **Governa:** `roadmap-rust/issues/FASE-07.md`, `FASE-07-1-gateway.md`,
  `FASE-07-2`/`07-3`/`07-4` (telas e E2E consomem este contrato)

## Contexto

A fronteira portal ↔ `wyd-admin` é o **único** ponto do projeto onde cabe um
transporte entre processos: o Hot Path é monolítico e proíbe gRPC/HTTP no tick
(ADR-0001). O planejamento antigo dizia "gRPC/REST" sem escolher
(`pre-release-rust.md` §7, `docs/rust/00-visao-estrutura.md` §3).

Fatos do repositório que obrigam a decidir agora:

- **Não existe contrato nem transporte.** Nenhum `*.proto` versionado; o antigo
  `api/web/v1/web.proto` (origem Go) foi removido com a árvore legada.
  `wyd-portal/src/server/grpc.ts:21` aponta para
  `crates/wyd-admin/proto/admin.proto`, que **não existe** — o módulo ficou lazy,
  então o portal compila e testa (118/118), mas qualquer ação admin falha em runtime.
- **Um único consumidor, todas as chamadas unárias.** `grpc.ts` exporta 16
  serviços (`ItemCatalogService`, `AccountAdminService`, `CharacterAdminService`,
  `NpcAdminService`, `WorldStateAdminService`, …) e nenhum usa streaming.
- **O envelope já é nosso, não do protocolo.** `admin.ts` lê `result: 1..5`
  (`ADMIN_RESULT` em `admin-result.ts`) e mapeia para HTTP; `promisify.ts` aplica
  deadline de 10 s e propaga `x-trace-id`. gRPC status é traduzido de volta para
  `AdminResult` — três vocabulários para uma chamada.
- **Zero dependências externas** (`docs/rust/01-dependencias.md`): o workspace só
  tem `path` deps. `tonic`/`prost` arrastam `tokio` + `hyper` + `h2`, que a
  ADR-0006 ainda difere, mais `protoc`/`buf` e codegen TS.
- **O "benefício" do gRPC não existe aqui.** O portal mantém ~40 interfaces TS à
  mão; não há segundo cliente nem linguagem terceira para justificar schema
  compartilhado. O ganho real (não reescrever os 48 módulos) é obtido mantendo a
  **fachada TS**, não o protocolo binário.

## Decisão

Adotamos **HTTP/1.1 + JSON sobre loopback (`127.0.0.1:7600`)** como transporte
único da fronteira portal ↔ `wyd-admin`, com o envelope `AdminResult` no corpo e
`POST /admin/v1/{servico}/{metodo}` como forma de chamada — porque o único
consumidor é unário e local, e gRPC/proto só acrescentariam toolchain, HTTP/2 e
um vocabulário de erro a traduzir.

### 1. Contrato método → params → resposta

- **Rota:** `POST /admin/v1/{servico}/{metodo}` — `servico` e `metodo` em
  `snake_case` (ex.: `/admin/v1/account_admin/set_role`,
  `/admin/v1/item_catalog/list_items`). Um método = uma rota; sem query string.
- **Serviços:** exatamente os 16 já exportados em `wyd-portal/src/server/grpc.ts`
  (`item_catalog`, `account_web`, `account_admin`, `character_admin`, `npc_admin`,
  `content_rates`, `drop_admin`, `daily_reward_admin`, `donate_admin`,
  `donate_revenue_admin`, `world_event_admin`, `world_state_admin`, `mob_admin`,
  `attribute_map_admin`, `teleport_admin`, `item_effect_admin`). Métodos = os
  exports do mesmo arquivo; nenhum método novo nasce sem entrar no contrato.
- **Request:** corpo = objeto de parâmetros (JSON, `UTF-8`); **sem**
  `moderator_id`/`moderatorId` — autor vem da sessão. Payload binário
  (`AttributeMap.transform`) viaja em base64 no campo `data`.
- **Response 200:** envelope `{ "result": 1..5, "message": "...", "trace_id": "..." }`
  mais os campos do domínio no mesmo objeto (ex.: `"npcs": [...]`,
  `"delivery_ids": [...]`). `result` segue `ADMIN_RESULT`
  (`admin-result.ts`): `1` OK, `2` FORBIDDEN, `3` INVALID, `4` NOT_FOUND, `5` CONTENT_OWNED.
- **Falha de transporte ≠ recusa de negócio:** resultado de negócio (inclusive
  `FORBIDDEN` por papel) volta **HTTP 200** com `result` no corpo; só o transporte
  usa status: `400` JSON inválido, `401` sem token/ inválido (não há autor para
  auditar), `404` método desconhecido, `413` corpo acima do limite, `405` verbo
  diferente de POST, `500` erro interno (sem stack no corpo).
- **Headers:** request `content-type: application/json`, `authorization: Bearer
  <token-de-sessão-staff>`, `x-trace-id` (obrigatório); response
  `content-type: application/json`, `cache-control: no-store`, `x-trace-id`
  ecoado. Sem cookies: o BFF já tem a sessão httpOnly; o token viaja só no
  processo servidor→servidor.
- **Limites:** corpo ≤ 32 MiB (mapa de atributos), resposta ≤ 32 MiB, leitura com
  timeout de 5 s no servidor, deadline de 10 s mantida no cliente
  (`promisify.ts`), `Connection: close` (sem keep-alive na v1).

### 2. Auth e auditoria (regras que o transporte não pode afrouxar)

1. **Bind só em loopback** (`127.0.0.1`, porta `7600`); recusar bind em `0.0.0.0`.
2. **`StaffSession` é construída do token**, nunca do corpo
   (`crates/wyd-admin/src/auth/session.rs`); `player` → `result: 2` (403),
   token ausente/inválido → `401` de transporte.
3. **Toda chamada autenticada audita** com ação `<servico>.<metodo>`
   (`crates/wyd-admin/src/audit/log.rs`), incluindo leitura; negação por papel
   registra `auth.denied`; `401` anônimo não gera linha (não há autor).
4. **`moderator_id` das intenções sai da sessão** (`StaffSession::moderator_id`),
   como já ocorre em `adminCall`.
5. **Escritas continuam indo para `delivery_queue`** (nunca estado vivo); o
   transporte não altera o caminho de grants.

### 3. Módulo novo na árvore do `wyd-admin`

Acrescentado ao §13 de `docs/arquitetura/estrutura-e-padroes.md` (ADR-0002):

```
crates/wyd-admin/
├── src/
│   ├── transport/
│   │   ├── mod.rs        # roteador servico/metodo → api/*, envelope + trace_id
│   │   ├── http.rs       # HTTP/1.1 mínimo em std: loopback, Content-Length,
│   │   │                 #   limite de corpo, Connection: close, status do ADR
│   │   └── json.rs       # JSON do subconjunto do contrato (objeto/array/
│   │                     #   string/número/bool/null; escape e números testados)
└── tests/
    └── contract.rs       # rota→método, 401 sem token, 403 player, envelope, limites
```

`transport/` é Cold Path puro: não é importado por `wyd-core`, `wyd-net` nem pelo
tick. Nenhum `.proto`, nenhum `protoc`, nenhuma dependência externa na v1.

### 4. Portal (mínimo churn, regra preservada)

- `wyd-portal/src/server/grpc.ts` vira `admin-client.ts` **mantendo as mesmas
  exportações** (`accountAdmin.listAccounts(...)`, `itemCatalog.listItems(...)`),
  com `fetch` + `AbortSignal.timeout`; os 48 módulos que importam
  `@/server/grpc` passam a importar o novo cliente (ou o antigo reexporta).
- `promisify.ts` deixa de depender de `@grpc/grpc-js`: aplica deadline, header
  `x-trace-id` e traduz `401/403` de transporte em `AdminError`. `adminCall`
  (`admin.ts`) não muda de contrato: sessão, `moderator_id` da sessão, auditoria,
  `AdminResult` → HTTP do browser.
- `@grpc/grpc-js` e `@grpc/proto-loader` saem do `package.json` quando a fiação
  terminar (FASE-07-1), não antes.
- **Implementado em 2026-11-08:** `grpc.ts` removido, `admin-client.ts` com as 16
  exportações sobre `fetch`, `promisify.ts` sem `@grpc/*`, dependências e
  referências de `.proto` fora do `package.json`/lock e do `Dockerfile.portal`.

## Consequências

- **Positivas:** build reprodutível offline (zero deps novas); contrato
  auditável com `curl`/log de texto; um vocabulário de erro só (`AdminResult`);
  nenhum `protoc`/`buf` no CI; transporte trocável depois por `axum`/`hyper` sem
  tocar no BFF; portal retoma funcionalidade admin sem Go.
- **Negativas:** JSON de subconjunto próprio (~200 linhas + testes) até
  `serde_json` poder entrar; sem codegen, a paridade Rust↔TS depende de
  `tests/contrato_transporte.rs` (rota/params por serviço) e de revisão; HTTP/1.1 sem
  keep-alive é mais lento por chamada (irrelevante em loopback, tráfego de admin).
- **Invariantes afetados:** gRPC/HTTP continuam **proibidos no tick** (ADR-0001);
  `.proto` fica proibido na árvore (o contrato é este ADR + `tests/contrato_transporte.rs`);
  `moderator_id` do corpo permanece proibido; a web segue sem escrever estado vivo;
  nenhum bind fora de loopback.

## Alternativas consideradas

| Alternativa | Por que foi descartada |
|---|---|
| gRPC (`tonic` + `prost` + `protoc`/`buf`) | toolchain dupla, HTTP/2 + `tokio` (ADR-0006) e status codes extras para 16 serviços unários de um só cliente |
| Manter proto só como schema, transporte REST | duas fontes de verdade (proto + TS) sem codegen; custo de manter as duas em sincronia |
| REST "clássico" com rota por recurso (`/accounts/{id}`, verbos HTTP) | reescreveria os 48 módulos do portal; `adminCall` passaria a mapear REST→domínio em vez de método→método |
| tRPC / GraphQL | sem cliente Rust nativo; resolve o problema do lado TS e cria outro do lado Rust |
| Acesso direto do portal ao Postgres | viola "a web nunca escreve estado vivo" e acopla o portal ao schema (`AGENTS.md` §3.7) |
| N-API / addon nativo JS↔Rust | acopla o build do Node ao do Rust e quebra o `standalone` do Next/Docker |

## Rastreabilidade

- Issues consumidoras: `roadmap-rust/issues/FASE-07.md`,
  `FASE-07-1-gateway.md` (transporte + contrato), `FASE-07-2`/`07-3` (telas),
  `FASE-07-4` (E2E).
- Docs atualizadas nesta ADR: `docs/arquitetura/estrutura-e-padroes.md` §13
  (árvore do `wyd-admin`), `docs/arquitetura/visao-pastas.md`,
  `docs/rust/00-visao-estrutura.md`, `docs/rust/05-stack-docker.md`,
  `.agents/skills/wyd-web-admin-ui/SKILL.md`, `roadmap-rust/index.md`
  (status/pendência da FASE-07), `docs/rust/03-progresso.md`.
- Evidência: nenhum `*.proto` versionado (`find . -name '*.proto'` só em
  `node_modules/`); nenhuma dep `tonic`/`prost` (`crates/*/Cargo.toml` só `path`);
  `crates/wyd-admin` já entrega auth/auditoria/grants (11 lib + 4 integração) e
  `tests/contrato_autorizacao.rs` prova 401/403.

## Revisão

- Revisar quando: surgir cliente que não seja o BFF (ex.: app mobile), houver
  streaming real (eventos ao vivo no portal) ou o volume de chamadas exigir
  multiplexação — aí gRPC volta à mesa **com medição**.
- Sinais de que a decisão envelheceu: `json.rs` crescendo com casos de borda;
  `tests/contrato_transporte.rs` divergindo do TS com frequência; latência do painel
  dominada pelo transporte (medir antes de trocar).
