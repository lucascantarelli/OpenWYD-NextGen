---
name: wyd-web-admin-ui
description: Constrói e mantém o portal web de administração do WYD2 (Next.js como BFF puro sobre o web-api gRPC do OpenWYD), incluindo login de moderador, editores de NPC/mob/item/drop/rates, relatórios e operação ao vivo. Use ao criar telas, Route Handlers, clientes gRPC, sessão httpOnly, overlays de conteúdo no Postgres ou ao revisar o portal antes de commitar.
---

# Portal web de administração (BFF Next.js ↔ web-api gRPC)

Este portal é a **UI de gestão do servidor**: NPCs/mercadores, stats de mob, catálogo de itens,
drops, EXP, rates, mapas, eventos, loja, faturamento e operação ao vivo. Ele **não** é o cliente do
jogo e **não** é o servidor: é uma casca de administração sobre contratos que já existem (ou que
esta linha de trabalho cria).

Leia junto: `docs/migration/web-platform-plan.md` (decisão de arquitetura),
`docs/integrations/*-nextjs.md` (8 guias por funcionalidade), `AGENTS.md`
(invariantes do servidor) e `roadmap/index.md` §Fase 6 (as issues 18–39).

## 1. As quatro regras que não se quebram

1. **O browser nunca fala gRPC e nunca vê certificado.** Todo acesso ao `web-api` sai de Route
   Handlers / Server Actions. Nenhum `@grpc/*` em componente client.
2. **A web nunca escreve estado vivo de personagem.** Inventário, coin, stats e posição pertencem ao
   loop único do `tmserver`. Concessões vão por `delivery_queue`; o resto é **config fria** em
   Postgres que o tmserver lê no boot.
3. **`Release/` é somente leitura** (bind mount `:ro` no compose). Edição de conteúdo grava
   **overlay no Postgres** (`npc_definition`, `mob_template_stat`, …) e o tmserver aplica no boot.
   Nunca escrever `.txt`/`.csv`/templates binários de `Release/` a partir da web.
4. **Autorização é server-side, duas vezes.** O `role` da sessão serve para mostrar/esconder a rota;
   o `web-api` revalida `account.role` em **toda** RPC admin e audita. Nunca derive `moderator_id` do
   corpo da requisição — sempre do cookie `httpOnly`.

## 2. Topologia e portas

```
Browser ──HTTPS──> portal (Next.js :3000)  ──gRPC──> webserver (web-api :7600) ──> Postgres
                          │                                   └── tmserver lê no boot (overlay)
                          └── cookie httpOnly (account_id + role)
```

- `webserver` = `webserver`, gRPC puro (`grpc.NewServer`), **sem HTTP de UI**. Contrato em
  `api/web/v1/web.proto` (1294 linhas, 14 serviços).
- mTLS é **opcional**: `internal/secure.ServerCreds` devolve `insecure` quando não há certs. Local =
  `127.0.0.1:7600` sem TLS; produção = `scripts/gen-certs.sh` + `W2PP_TLS_{CERT,KEY,CA}` no servidor e
  `client.crt/key` no portal.

## 3. Onde mora o código

```
portal/                                  ← raiz do workspace, IRMÃO de  (não dentro!)
├── src/app/                             rotas (App Router)
│   ├── (public)/login/page.tsx          login
│   ├── (admin)/layout.tsx               shell + guarda de papel
│   ├── (admin)/npcs/…                   editor de NPCs
│   ├── api/auth/{login,logout}/route.ts sessão
│   └── api/health/route.ts              prova viva do gRPC
├── src/server/
│   ├── grpc.ts                          clientes + promisify
│   ├── admin.ts                         wrapper de RPC admin (result → HTTP)
│   ├── session.ts                       cookie httpOnly (jose HS256)
│   └── actions/*.ts                     Server Actions por domínio
├── src/components/                      tabela, form, dialog, combobox, toast
└── src/lib/                             tipos do domínio, formatação, validação (zod)
```

`portal/` fica **fora** de `` de propósito: o build Docker de cada serviço Go faz
`COPY . .` com contexto ``, e um `node_modules` ali dentro explodiria o contexto.

## 4. Contratos disponíveis (estado real hoje)

| Serviço | Escrita? | Para quê |
|---|---|---|
| `NpcAdminService` | ✅ | NPC/mercador: posição, visibilidade, loja, preço global, delete |
| `MobTemplateAdminService` | ✅ | stats e `Equip[]` de **qualquer** template (monstros inclusos) |
| `AttributeMapAdminService` | ✅ | colisão/walkability do `AttributeMap.dat` |
| `WorldEventAdminService` | ✅ | evento global (double EXP, item do evento, notice) |
| `DonateAdminService` / `DonateShopService` | ✅ | loja de donate e carteira |
| `DailyRewardAdminService` | ✅ | catálogo de recompensa diária |
| `DonateRevenueAdminService` | ✅ | faturamento (KPIs, pedidos, compradores, extrato, busca de contas) |
| `AccountWebService` | ✅ (criar) | `VerifyCredentials` (login), `CreateAccount`, `IssuePlayCode` |
| `CharacterWebService`, `RankingWebService`, `ItemCatalogService` | ❌ | leitura fria |
| `NpcAdminService.ListDropItems` / `ListMobDrops` | ✅ `/drops` (abas Por item/Por mob, raridade pelo divisor, CSV) | relatórios de drop (item↔mob) |

**Não existe** (a Fase 6 cria): catálogo de itens editável, tabela de drop editável, EXP/curva,
rates e flags do servidor, spawns/`NPCGener`, contas/papéis/ban, live ops (online/kick/teleport/notice).

## 5. O padrão BFF (copie isto, não invente outro)

```ts
// src/server/grpc.ts
import path from "node:path";
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

export const npcAdmin = new pkg.web.v1.NpcAdminService(addr, creds);
// … um cliente por serviço, criado uma vez no módulo
```

```ts
// src/server/admin.ts — TODO RPC admin passa por aqui
export const ADMIN_RESULT = { OK: 1, FORBIDDEN: 2, INVALID: 3, NOT_FOUND: 4, CONTENT_OWNED: 5 } as const;
const HTTP: Record<number, number> = { 1: 200, 2: 403, 3: 422, 4: 404, 5: 409 };

export async function callAdmin<T extends { result: number }>(
  rpc: (req: unknown, cb: (e: unknown, r: T) => void) => void,
  req: unknown,
): Promise<{ status: number; body: T | { error: string } }> {
  const res = await promisify<T>(rpc)(req);
  const status = HTTP[res.result] ?? 500;
  return { status, body: status === 200 ? res : { error: RESULT_MESSAGE[res.result] ?? "resultado desconhecido" } };
}
```

Regra de ouro: **`AdminResult` viaja no corpo**, só falha de infra vira erro gRPC. Um mapa que não
conheça `CONTENT_OWNED` transforma uma resposta esperada (`DeleteNpc` em NPC de conteúdo) em 500 —
foi o bug da issue #257. Mapeie o enum **inteiro**.

## 6. Sessão e autorização

```ts
// src/server/session.ts
const key = new TextEncoder().encode(process.env.SESSION_SECRET!); // ≥32 chars
export async function setSession(accountId: number, role: string) {
  const jwt = await new SignJWT({ sub: String(accountId), role })
    .setProtectedHeader({ alg: "HS256" }).setExpirationTime("8h").sign(key);
  (await cookies()).set("wyd_session", jwt, { httpOnly: true, sameSite: "lax", path: "/",
    secure: process.env.NODE_ENV === "production" });
}
```

- `VerifyCredentials(name, pass)` → `{ ok, account_id, blocked, role }`. `ok=false` → 401 genérico
  (não revelar se a conta existe). `blocked=true` → 403.
- Guarda de rota em `(admin)/layout.tsx`: `role === "moderator" || role === "admin"`, senão
  `redirect("/login")`. Isso é **UX**, não segurança.
- Papel é lido no login; promover alguém no banco **não** muda a sessão aberta. Sessão curta (8h) +
  revalidação no `web-api` cobrem o resto.

## 7. Overlays de conteúdo (escrita de config)

Padrão já provado por `npc_definition` e `mob_template_stat`:

1. Migração em `internal/migrations/NNNN_nome.up.sql` (**próxima livre: 0042**; a última é
   `0041_mob_carry`) + `.down.sql` simétrico. `internal/migrations/migrations.go` embute os
   arquivos — nada de rodar SQL à mão.
2. Store em `dbserver/internal/store` (ou do próprio `webserver/internal/<dominio>`): leitura
   + upsert auditado.
3. Proto em `api/web/v1/web.proto`, com `moderator_id` e `AdminResult`.
4. Aplicação no `tmserver`: um pacote leaf (`tmserver/internal/<dominio>cfg`), um `Source` que busca
   via gRPC do dbServer e um applier **dentro do loop** (`world.World.Run`), atrás de flag
   (`-<dominio>-editing`, default OFF) e aplicado **no boot** (como `-mob-stat-editing`).
5. Tela no portal + item no menu + teste.

Nunca: escrever em `Release/`, criar caminho de escrita que toque personagem vivo, ou aplicar overlay
sem flag desligável.

## 8. Ciclo de verificação (o mínimo antes de dizer "pronto")

```bash
# portal
cd portal && npx tsc --noEmit && npm run lint && npm run build

# BFF vivo contra a stack
curl -s localhost:3000/api/health          # {"ok":true,"items":3220,...}

# Go, sem Go no host
MSYS_NO_PATHCONV=1 docker run --rm -v "D:/dolorestec/wyd2/OpenWYD:/src" -w /src \
  golang:1.26-alpine sh -c "go build ./... && go test ./webserver/internal/... ./tmserver/internal/..."
```

Após mexer em overlay: `docker compose up -d tmserver` e conferir no log `content loaded`,
`NPCs spawned`, ausência de `unbalanced Exp`.

## 9. Armadilhas conhecidas

| Armadilha | Consequência | Como evitar |
|---|---|---|
| `longs: Number` ausente | `int64` chega como objeto `Long` e quebra JSON | sempre `longs: Number` |
| `enums: String` + mapa numérico | `AdminResult` vira `"OK"` e nada casa | `enums: Number` |
| `keepCase` divergente | campos viram `moderatorId` vs `moderator_id` | fixe `keepCase: true` |
| `npc_definition` vazio | o mundo sobe sem NPCs (o banco é a única fonte, issue 57) | rodar `dbserver import-npcs` (o dbserver também semeia no boot) |
| `-mob-stat-editing` | aplica **uma vez** no boot | reiniciar tmserver após editar |
| Catálogo de itens (3.2k) por request | portal lento | cachear por `catalog_version` |
| `unstable_cache` lendo artefato gerado (`dist/world-atlas.json`) | regenerar não aparece na tela e o `null` fica gravado em disco | cache manual por `mtime` (`server/atlas.ts`): arquivo mudou, cache cai |
| `AdminResult` comparado com `0` (`res.result === 0`) | escrita bem-sucedida aparece como "o web-api recusou a alteração": `ADMIN_RESULT_OK` é **1** (`ADMIN_RESULT_UNSPECIFIED` é 0) | usar `isOk()` de `server/admin-result.ts`, nunca um número solto |
| Upsert de stat de mob recebendo o `Equip[]` do browser | o upsert substitui o override inteiro: salvar stats com uma cópia velha do equip apaga a edição de equip feita logo antes | o BFF relê o equip do servidor (`GetMobTemplateStat`) e grava junto; o equip tem ação própria (`SetMobTemplateEquip`) |
| `webserver/internal/*` importando um pacote `tmserver/internal/*` | `use of internal package ... not allowed`: `tmserver/internal` só é importável de dentro de `tmserver` | tabela/constante compartilhada vai para `internal/<nome>` (foi assim com `internal/droprate`); o pacote do tmserver fica com a cópia mutável |
| Listar template de mob e querer saber se tem override | não existe no arquivo (é estado do Postgres); um `Get` por item da lista seriam 1.991 RPCs | `MobTemplateFile.overridden` é resolvido no web-api com **uma** consulta por listagem (`ListMobTemplateStats`) e a lista devolvida é cópia do snapshot de boot |
| Indexar célula de taxa por posição numérica | `CompRate.txt` tem **várias chaves por família** (`Compositor Item_+7`, `Item_+8`, `Item_+9`) e todas caíam na mesma `pos` | chave textual `(file, family, name)`: índice do refino, chave do CompRate ou `"<tier>.<coluna>"` |
| Editar adds de item por posição (`slot 1..3`) | `item_catalog` traz até 12 pares `EF_*` por item (a 1188 tem 6) e o servidor indexa por código | overlay com PK `(item_index, effect)` — a posição não existe como conceito no conteúdo |
| Estado de módulo (`setBaseTable`) num `lib/` compartilhado | o server component e o bundle do client são instâncias separadas do módulo: o valor escrito no SSR não existe no browser e a conta sai zerada | passe a tabela por prop (`effectiveDropRate(slot, bonus, level, base)`) ou serializável; nunca guarde dado de request em variável de módulo isomórfico |
| Retângulo digitado em célula quando o contrato é mundo | o servidor divide por 4 de novo e a área atingida fica 16× menor (ou no lugar errado) | digite sempre em 0..4095, mostre `describeRect` (mundo → célula) ao lado do input e valide com o mesmo limite do backend |
| Editor que só desenha linhas preenchidas e envia o array completo | posição vazia pode ser lida como "manter" e o drop antigo volta sem ninguém pedir | no carry o vazio é decisão explícita: envie só as ocupadas e deixe o servidor substituir a tabela do template inteira; nunca complete com os valores lidos |
| Borda de input quase invisível | `--border`/`--border-strong` são divisores (1,45:1 / 1,87:1) e não passam em WCAG 1.4.11 quando delimitam um campo | use `--border-control: #707b8d` (4,42:1 no bg, 3,77:1 no surface2) para input/select/chip/botão; meça com script de luminância, não a olho |
| Precisa validar RPC/tela sem abrir navegador | o painel de preview nem sempre anexa e o curl não fala gRPC | script CJS de 20 linhas em `portal/` com `@grpc/proto-loader` + `@grpc/grpc-js` no `127.0.0.1:7600` (`moderator_id:1`), e para a página `curl -c/-b /tmp/cj.txt` após `POST /api/auth/login` com `{"name":"test","pass":"test123"}`; apague o script no fim |
| Tela de erro sem saída para o operador | leitura falhou e só a mensagem aparece | use `components/RetryButton.tsx` (`router.refresh()` em `useTransition`), que também atualiza caches de boot do BFF |
| Botão de tab/cabeçalho de tabela sem anel de foco | `all: unset` na classe cancela o `outline` da regra global de `:focus-visible` | redeclare `X:focus-visible { outline: 2px solid var(--accent) }`; o mesmo vale para qualquer classe que use `all: unset` |
| Build de Go falhando com `Espaço insuficiente no disco` no link | o `$WORK` do Go fica no `%TEMP%` (drive `C:`), que nesta máquina encheu | `TMPDIR=D:/tmp-go GOTMPDIR=D:/tmp-go go build ./...`; se o Docker também cair com `Docker Desktop is unable to start`, é o mesmo disco — libere `C:` |
| Provar o read-through reiniciando o `tmserver` ao vivo | o restart dispara save-on-shutdown e atrapalha teste humano | suba um container one-off da mesma imagem na rede do compose, em portas próprias: `docker run -d --name probe --network openwyd_default -v "<repo>/Release:/Release:ro" openwyd-tmserver -addr :8282 -dbserver dbserver:7514 -content /Release -status-addr :8088 -item-effect-editing`, leia o log e `docker rm -f probe` |
| Serviço gRPC novo respondendo `Unimplemented: unknown service` | o container `dbserver` roda a imagem antiga, não o código que você acabou de escrever | `docker compose build dbserver && docker compose up -d dbserver` antes de testar; idem para `webserver`/`tmserver` |
| Escrever estado que o `tmserver` guarda em memória (Kefra, dono do castelo, guildas) e prometer efeito imediato | o mundo carrega isso no boot; o save do processo em execução pode até sobrescrever a linha | gravar no banco e dizer na tela que vale **no próximo restart**; para o Kefra suba a `revision` no mesmo `UPDATE` (o guard de `SaveKefraState` recusa revisão menor), e só implemente escrita ao vivo quando houver RPC no tmserver (issue 38) |
| Bucket/porta 7600 exposta | qualquer host fala gRPC | em produção só na rede interna + mTLS |
| `next build` com `node_modules` dentro de `` | contexto Docker gigante | portal na raiz do workspace |

## 10. Checklist de PR do portal

- [ ] Rota nova exige sessão `moderator`/`admin` no layout **e** chama RPC admin que revalida
- [ ] `moderator_id` vem do cookie; nenhum id sensível no body do browser
- [ ] `AdminResult` mapeado por inteiro, com mensagem por valor (inclui `CONTENT_OWNED`)
- [ ] Erros de infra → 502 com mensagem genérica; nada de stack/paths no cliente
- [ ] Nenhuma escrita em `Release/`; config nova tem migração + flag OFF + applier no boot
- [ ] `tsc --noEmit`, `next build` e o teste Go afetado verdes; evidência colada
- [ ] Textos de UI em português; números formatados (coin, EXP) como o jogo

## 11. Padrão de UI/UX (issue 47)

Vem de duas skills de front: `frontend-design` (direção visual) e `ui-ux-practices` (esta é de
**outro** produto — Tauri 2, `pnpm`, `gp100-core`; só os princípios valem aqui, nunca os caminhos,
os gates `pnpm` ou o IPC).

1. **Nome antes de ID.** Toda lista/detalhe mostra o nome do catálogo; o índice vira link
   (`/itens/<index>`) e `title`, nunca rótulo principal. Resolver no BFF — o catálogo tem 3,2 mil
   linhas e o componente client não importa `webserver/internal/*`.
2. **Tokens de espaço** base 4, escala de Fibonacci: 4 · 8 · 12 · 20 · 32 · 52 · 84. Nada de `10`,
   `14`, `18` soltos em `padding`/`margin`/`gap` novos.
3. **A11y**: AA (4,5:1 texto, 3:1 UI), `:focus-visible` 2px com 3:1, alvo ≥ 32×32, estado nunca só
   por cor.
4. **Estados explícitos**: `loading` / `empty` / `error` / `ready`; erro sempre com ação de
   recuperação (o `ui-empty` já é o formato).
5. **Copy**: português, sentence case, botão diz o resultado do fluxo, erro sem desculpa e sem
   vagueza.
6. **Um acento por tela.** `--accent` = ação primária + foco. Metadado curto pode usar `·`; cadeia de
   `a · b · c` como decoração de cabeçalho é o "tell" de página gerada que a `frontend-design`
   proíbe.
7. **A UI não fala protocolo**: nada de `eff1/effv1`, enum ou hex na tela; o BFF entrega tipo nomeado.

Nomes vêm de catálogo no **web-api**, nunca do browser: `itemcatalog` (`item_catalog`),
`skillcatalog` (`skill_catalog`, 151 linhas) e `itemeffects` (`Source/Code/ItemEffect.h`) resolvem
índice/código → nome, e a RPC devolve só o que o personagem tem. Código de efeito fora do header cai
em `efeito N` — não inventar rótulo.

## 12. Mapa das issues

| # | Entrega |
|---|---|
| 18 | Bootstrap do portal + client gRPC + health |
| 19 | Login, sessão httpOnly, guarda de papel |
| 20 | Shell: layout, menu, tabela/form/dialog/toast |
| 21 | Camada de admin no BFF (result→HTTP, auditoria de UI) |
| 22 | Catálogo de itens com ícones (leitura) |
| 23 | Editor de NPCs/mercadores |
| 24 | Preços globais de itens |
| 25 | Editor de stats/equip de mob |
| 26 | Relatórios de drop item↔mob |
| 27 | Editor do mapa de colisão — ✅ `/mapa` (histograma, significados, operações em massa, retângulo em mundo e download `AttributeMap_New.dat` com sha no header) |
| 28 | Eventos globais — ✅ `/eventos` (badge ativo/esgotado/incompleto, item pelo catálogo, divisor lido como "1 em cada N", faixa com progresso, gate de EXP dobrada e "desligar tudo"; a action relê o servidor depois do `Set`, e o tmserver aplica em até 15 s sem restart) |
| 29 | Loja de donate — ✅ `/loja` (item por nome/ícone, **preço em donate** com o rótulo em toda linha, validade, adicionais, estado; diálogo de criação/edição com `eff`/`effv` e `expires_days`; habilitar/desabilitar; **apagar exige digitar o título** e o servidor confere; crédito manual exige motivo ≥ 10 e confirmação dupla e devolve `new_balance`). A action recusa item fora do catálogo e — armadilha real — **nada de `export const` num arquivo `"use server"`**: o build passa e a action responde 500 em runtime |
| 30 | Recompensa diária — ✅ `/diaria` (tabela com item por nome/ícone, validade, adicionais e estado; diálogo de criação/edição com os três pares `eff`/`effv` e `expires_days`; habilitar, duplicar — a cópia nasce desabilitada — e apagar com confirmação). Resgate é do `DailyRewardService`, nunca do BFF: claim duplicado devolve `ALREADY_CLAIMED` e a entrega é `delivery_queue` (`source=daily_reward:<id>`) drenada no login |
| 31 | Faturamento — ✅ `/faturamento` (só `admin`): janela 7/30/90 dias ou intervalo digitado em `America/Sao_Paulo`, KPIs de dinheiro (`gross_cents`, ticket, compradores) separados dos de donate, aviso de que `pending_cents` não é receita, série em barras CSS por dia/semana/mês, pedidos com CSV client-side, top compradores com LTV e extrato da carteira. O filtro mora na URL; a busca de conta (`SearchAccounts`) virou `Combobox` e também substituiu o "id numérico" do crédito manual da issue 29 |
| 32 | Contas, papéis e ban — ✅ `/contas` (lista com cursor keyset + filtro de login/papel/bloqueio, detalhe com **auditoria** e formulário de papel/bloqueio com motivo ≥ 10, criar conta já com papel). Regras no **store**, dentro da transação: ninguém altera a si mesmo, a última conta admin ativa não é rebaixada nem bloqueada, e `moderator` só bloqueia `player`. `adminCall` agora prefere o `message` de negócio que o web-api devolve ("esta é a última conta admin ativa — promova outra antes") ao texto genérico do `AdminResult`. Falta a evidência in-game do bloqueio |
| 33 | Catálogo de itens editável (overlay) |
| 34 | Tabela de drop editável — `/drops/editar` (divisor por slot, simulador, faixas nomeadas, slots 8–11 "fixo no código") e `/drops/carry` (Carry[] por template, faixas de raridade, adicionar na próxima posição livre, restaurar) |
| 35 | EXP de mob e curva |
| 36 | Rates e flags do servidor |
| 37 | Spawns e localização de mob |
| 38 | Live ops (online/kick/teleport/notice) |
| 39 | QA final: E2E, segurança, hardening |
| 40 | Logs dos containers ao vivo — ✅ `/logs` com **dois modos** atrás de `server/docker.ts`: no compose, o sidecar **`logpipe`** (Go stdlib, socket read-only, `/status` + `/logs/<serviço>`, sem escrita/exec, nonroot + `read_only` + `no-new-privileges` + `group_add: 0`) e o portal só autentica e repassa; no host, `docker compose logs -f`. Whitelist de `tmserver`/`dbserver`/`binserver`/`webserver`/`db`/`portal` + `all`, `redact` de ANSI/DSN/segredo, `tail` 1..2000, `since` só `10m`, SSE com 401/403/404 antes de abrir conexão, chips de status cacheados 5 s, pausa que não fecha o stream, filtro, limpar e `.txt`. **O socket nunca é montado no portal** |
| 41 | Atlas do mundo: zonas, coordenadas, portais e spawns (leitura + mapa) — ✅ `/atlas`, artefato gerado por `tmserver/cmd/worldatlas` |
| 42 | Rotas de teleporte editáveis (overlay + tmserver read-through) — ✅ `/teleportes`, `TeleportAdminService`, overlay com poll de 15 s |
| 36 | Rates de conteúdo — 🟡 `/config` (abas Refino/Combine/Quests + seção Flags), `ContentRatesService`, overlay `content_rate_override` + auditoria; parser único em `internal/ratetables`; read-through no tmserver (`-rates-editing`, logs `rate_overrides_applied` + linha efetiva); flags lidas de `server_instance_flags`, gravadas pelo tmserver no boot via `InstanceFlagsService` |
| 45 | Guildas, castelo e Kefra — 🟡 `/guildas` (snapshot + diálogo com `tornar mestre`, `remover` e relação, tudo com motivo auditado em `guild_admin_audit`) e `/castelo` (dono da cidade = `guild_zone.charge_guild`, dono da torre = `guild_tower_state.owner_guild`, taxa por zona e card do Kefra), `WorldStateAdminService`; falta a janela de horários (issue 36) |
| 43 | Inventário e status de personagem (edição offline + delivery_queue) — 🟡 leitura + concessão de item por fila; gold e edição de campo pendentes |
| 44 | Adds/efeitos de item, refino e combine — 🟡 leitura ✅ (`ItemCatalogEntry.effects`, `effect_types`, validação de adds na concessão); edição pendente |
| 45 | Guildas, castelo/Zakum e Kefra |
| 46 | Catálogo de skills (overlay de `skill_catalog`) |
| 47 | UI sem IDs crus + padrão de UI/UX (tokens, estados, a11y) — 🟡 itens, skills, adds, atlas, loja e teleportes já mostram nome (nome de lugar via `lib/zone.ts`, coordenada no `title`); escala `--space-1..7` e foco visível nos alvos que usam `all: unset` |
| 48 | Mapa de navegação: relevo (geral + detalhe 4096²), zoom/pan, leitura de ponto, calor de spawn, “ir para coordenada”, teleportes com nome de lugar e raster de regiões (contorno colado ao terreno) — 🟡 feito; faltam mapa em largura total e pinch-zoom |
