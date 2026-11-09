# Portal (WYD2)

Next.js 16 (App Router) como **BFF puro** sobre o `wyd-admin` via HTTP/1.1 + JSON
em loopback (`:7600`, ADR-0008). O browser nunca fala com o gateway nem vê o
token de staff.

```bash
cd wyd-portal
npm install
npm run dev          # http://127.0.0.1:3000
npm run typecheck    # tsc --noEmit
npm test             # vitest
npm run build        # next build (standalone)
```

## Variáveis de ambiente

| Variável | Padrão | Uso |
|---|---|---|
| `WEB_API_ADDR` | `http://127.0.0.1:7600` | URL do `wyd-admin` (loopback) |
| `WEB_API_TOKEN` | — | token de staff (`authorization: Bearer`), processo→processo |
| `SESSION_SECRET` | — | assinatura HS256 do cookie `wyd_session` (>= 32 caracteres) |
| `AUDIT_LOG_DIR` | — | opcional: grava `audit-YYYY-MM-DD.ndjson` além do log do processo |
| `PROMETHEUS_URL` | `http://prometheus:9090` | leitura de métricas e alertas (server-side) |
| `PROMETHEUS_PUBLIC_URL` | `http://127.0.0.1:9090` | link “abrir no Prometheus” |
| `LOKI_URL` | `http://loki:3100` | agregado de logs por serviço (server-side) |
| `GRAFANA_URL` | `http://grafana:3000` | saúde e catálogo de dashboards |
| `GRAFANA_PUBLIC_URL` | `http://127.0.0.1:3001` | link “abrir no Grafana” |

## Telas

O portal só tem tela para o que o contrato do gateway serve hoje
(`crates/wyd-admin/src/transport/mod.rs`). Menu não anuncia rota sem contrato.

| Rota | Acesso | Fonte |
|---|---|---|
| `/painel` | staff | `server_admin/list_channel_presence`, Prometheus, Loki, Grafana e `item_catalog/list_items` |
| `/contas` | staff | `accounts/find` (busca exata) e `accounts/create` |
| `/itens` | staff | `item_catalog/list_items` (somente leitura) |
| `/monitoramento` | staff | presença no banco, Prometheus, Loki e dashboards do Grafana |
| `/conta` | jogador | `panel/self`, `panel/characters`, `panel/rankings` e `account_web/change_numeric_pin` |
| `/guias`, `/enciclopedia` | público | JSON versionado gerado de `docs/` e do catálogo |
| `/login` | público | `account_web/verify_credentials` |

Fora do escopo por falta de contrato no gateway: listagem paginada de contas,
edição de personagem, conteúdo (NPC, mob, drop, preço, evento), economia e
faturamento. Essas telas existiam sobre o servidor antigo e foram removidas —
não há chamada para rota que o `wyd-admin` não exponha.

## Identidade e desenho

`docs/arquitetura/identidade.tokens` é a **fonte única** de cor, espaço, raio e
tipografia. `src/app/globals.css` consome os tokens; `src/identidade.test.ts`
reprova cor fora da identidade, token ausente e divergência de valor. A mesma
identidade gera os assets do launcher.

O desenho é obsidiana + ouro velho: superfícies em camadas, luz âmbar no topo da
página, tipografia densa e números tabulares. Componentes pequenos e reusados
(`Stat`, `Badge`, `DataTable`, `Dialog`, `Field`, `Toast`) e ícones `lucide-react`.

Referências consultadas para a reformulação (2026-11-09):

| Referência | O que foi observado |
|---|---|
| [WYD International](https://wyd-international.com/home) | painel do jogador, ranking, busca de personagem e lista de drops como navegação pública |
| [DarkCMS Painel](https://darkmucms.online/painel.html) | separação entre área do jogador e área de staff, com papéis; painel em Next.js |
| [GameCP Admin Dashboard](https://devsbr-org.github.io/admin-dashboard) | KPIs de contas/personagens/online e log de ações administrativas na primeira tela |
| [Palworld Server Dashboard](https://palworld-server-dashboard.vercel.app/docs) | painel self-hosted com saúde do servidor, jogadores e visão restrita para moderador |
| [Game Admin Panel (Dribbble)](https://dribbble.com/shots/26221693--Game-Admin-Panel-Dashboard-UI-Design) | tema escuro, hierarquia de KPIs, moderação e economia sem poluição visual |

O que foi adotado: navegação curta por domínio, KPIs com fonte explícita,
status por serviço, tabelas densas e área do jogador separada da administração.
O que foi descartado: cards decorativos sem dado real, gráficos inventados e
qualquer número que o backend não forneça.

## Regra: toda ação admin passa por `adminCall`

`src/server/admin.ts` é o único caminho para chamada administrativa. Ele exige
sessão de staff, injeta `moderator_id` **da sessão** (nunca do corpo), aplica
deadline de 10 s, mapeia `AdminResult` → HTTP e registra auditoria.

```ts
// src/app/(admin)/contas/actions.ts
const account = await createAccount({ login, role });
```

**Não** chame `admin-client` direto de uma page ou action: sem `adminCall` não há
`moderator_id` da sessão, nem mapeamento de `CONTENT_OWNED`, nem auditoria.

## `AdminResult` → HTTP

`src/server/admin-result.ts` (puro, testado em `admin.test.ts`):

| `AdminResult` | HTTP | Mensagem |
|---|---|---|
| `OK` (1) | 200 | — |
| `FORBIDDEN` (2) | 403 | sua conta não tem permissão para esta ação |
| `INVALID` (3) | 422 | dados inválidos para esta ação |
| `NOT_FOUND` (4) | 404 | o registro não existe mais |
| `CONTENT_OWNED` (5) | 409 | este item é conteúdo versionado: oculte em vez de apagar |
| `UNSPECIFIED` (0) / desconhecido | 500 | resultado desconhecido do serviço de administração |

Falha de transporte (conexão recusada, timeout de 10 s ou 5xx) → **502**
“serviço de administração indisponível”. Stack, DSN e caminho nunca vão para o
browser: `toResponse` só devolve `{ error }` com `cache-control: no-store`.

## Senha numérica (“segunda senha”)

A senha numérica é **por conta**, pedida no cliente depois do login, na tela de
personagem. O primeiro cadastro acontece ali: o jogador digita os dígitos e o
servidor grava o hash. Nas entradas seguintes, o mesmo pacote apenas valida.

- **Cliente:** cria na primeira entrada e troca pelo fluxo de duas etapas do
  `0x0FDE` (`change_numeric` 1 com a atual, 2 com a nova) — `docs/wyd/senha-numerica.md`.
- **Portal:** `/conta` oferece a troca da senha **existente**. O BFF chama
  `account_web/change_numeric_pin`; o gateway abre uma conexão CPSock real e
  repete o mesmo fluxo do cliente. Nada de hash no portal.
- **Regras:** 4 a 6 dígitos; 5 erros bloqueiam a conta por 10 minutos; PIN, senha e
  hash nunca entram em log, resposta de erro ou browser storage.

Se a conta ainda não tem senha numérica, a tela do portal não finge “criar”: a
criação é do primeiro acesso no cliente, e a mensagem diz isso.

## Estrutura

```
src/app/(admin)/            shell administrativo (sidebar, topbar, guarda de UX)
src/app/(public)/           login, área do jogador, guias e enciclopédia
src/app/api/               rotas do BFF (auth, painel, health)
src/components/             componentes de UI sem regra de negócio
src/lib/                    formatação, papéis e textos (i18n)
src/server/admin-client.ts  cliente HTTP/JSON do wyd-admin + tipos do contrato
src/server/admin.ts         adminCall (sessão + moderator_id + resultado + auditoria)
src/server/accounts.ts      busca e criação de conta
src/server/catalog.ts       catálogo de itens (cache curto de processo)
src/server/panel.ts         painel do jogador (account_id da sessão)
src/server/monitoring.ts    foto de Prometheus/Loki/presença/Grafana
src/server/session.ts       cookie httpOnly HS256, getSession/requireStaff
src/proxy.ts                guarda de rota (matcher literal, coberto por teste)
scripts/build-guias.mjs     gerador de /guias a partir de docs/ (npm run guias)
```

## Docker

O compose da raiz sobe o portal junto com o gateway:

```bash
docker compose up -d --build portal   # http://localhost:3000
```

A imagem (`docker/Dockerfile.portal`) usa o output `standalone` do Next, com
contexto de build na raiz do repositório. No container,
`WEB_API_ADDR=http://wyd-admin:7600`; `SESSION_SECRET` tem default apenas de
desenvolvimento (>= 32 caracteres em produção).

## Guias e enciclopédia

`/guias` publica a documentação de `docs/` gerada por `npm run guias`
(`src/content/guias.json`, versionado). `/enciclopedia` lê o catálogo estático
gerado por `tools/content/` — são rotas públicas e não passam pelo gateway.

## Monitoramento

Uma página, quatro fontes independentes: Prometheus (métricas e alertas), Loki
(volume e erros por serviço), gateway (presença viva por canal, lida de
`character_session`) e Grafana (dashboards em kiosk). Cada bloco falha sozinho.
O portal nunca escreve estado vivo do jogo.
