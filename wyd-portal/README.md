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
| `/contas` | staff | `accounts/find`, `accounts/create` e moderação (`get_moderation`/`set_role`/`set_blocked`) |
| `/itens` | staff | `item_catalog/list_items` (somente leitura) |
| `/monitoramento` | staff | índice enxuto: saúde dos canais, motor (ticks/s, p99, entidades) e atalhos |
| `/monitoramento/servidor` | staff | motor do jogo por canal: ticks/s, p99, entidades, orçamento de tick |
| `/monitoramento/grafana` | staff | dashboards do Grafana embutidos (kiosk) |
| `/monitoramento/prometheus` | staff | alvos, regras de alerta e séries coletadas |
| `/monitoramento/logs` | staff | volume e erros por serviço no Loki |
| `/auditoria` | admin | `audit_admin/recent` (`audit_log`, append-only) |
| `/economia` | staff | `economy_admin/summary` (`city_tax_ledger` + `delivery_queue`) |
| `/conta` | jogador | `panel/self`, `panel/characters`, `panel/rankings` e `account_web/change_numeric_pin` |
| `/guias`, `/enciclopedia` | público | JSON versionado gerado de `docs/` e do catálogo |
| `/login` | público | `account_web/verify_credentials` |

Fora do escopo por falta de contrato no gateway: listagem paginada de contas,
edição de personagem, escrita de conteúdo (upsert/set/delete de NPC, mob, drop,
preço, evento) e faturamento (`donate_revenue_admin`). Essas telas existiam
sobre o servidor antigo e foram removidas — não há chamada para rota que o
`wyd-admin` não exponha. **Leitura de conteúdo já tem contrato** (FASE-07-5):
`npc_admin/list_npcs`, `mob_admin/list_mob_templates` e
`item_effect_admin/list_effects` (paginadas, teto 200) mais as já servidas
`item_catalog/list_items`, `drop_admin/list_drop_rates` e
`teleport_admin/list_routes` — prontas para as telas de conteúdo do portal.
Economia existe no recorte que tem fonte auditada (imposto e concessões);
receita de doacao continua sem contrato. **Operação já tem contrato** (FASE-07-6,
sem telas novas): `daily_reward_admin` (lista/upsert/delete), `donate_admin`
(loja + `credit_donate_balance` idempotente), `world_event_admin` (foto com
horários + liga/desliga), `account_admin/list_accounts`/`get_account` (sem
segredo) e `character_admin/grant_item` (fila idempotente com dreno no `Carry`) —
prontas para as telas de operação do portal. Seguem `501` por falta de tabela:
`daily_reward_admin/set_reward_item_enabled` e `donate_revenue_admin/*`.

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

O módulo nasceu como uma página única e hoje é um **índice** (`/monitoramento`) com
subpáginas para o que cresce em volume e merece leitura própria:

- `/monitoramento` — foto enxuta: presença viva por canal, motor (ticks/s, p99, entidades),
  alertas disparando e links para as subpáginas.
- `/monitoramento/servidor` — o jogo em si: ticks/s, p99 e entidades por canal contra o
  orçamento de tick (`TICK_BUDGET_US`), para responder “o canal está atrasando?”.
- `/monitoramento/grafana` — dashboards em kiosk (inclui `wyd-saude.json` com orçamento de
  tick, entidades e ticks por canal).
- `/monitoramento/prometheus` — alvos, regras ativas e séries que o portal consulta.
- `/monitoramento/logs` — volume e erros por serviço no Loki.

Cada bloco falha sozinho (Prometheus, Loki, gateway e Grafana são consultados em
paralelo e degradam para aviso). O portal nunca escreve estado vivo do jogo.
