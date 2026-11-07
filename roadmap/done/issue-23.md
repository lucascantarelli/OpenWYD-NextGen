# Issue 23 — Editor de NPCs e mercadores

**Fase:** 6 · **Depende de:** 20, 21 · **Bloqueia:** 24 · **Esforço:** L · **Estado:** ✅ fechada em 2026-11-04 (UI 03/11 · overlay no mundo + confirmação no jogo 04/11)

## Objetivo

Criar, editar, ocultar e apagar definições de NPC/mercador pela web — posição, template, tipo de
comerciante, visibilidade e estoque da loja — com o tmserver refletindo no próximo boot.

## Contexto / evidência

- `NpcAdminService` já implementa tudo (`webserver/internal/npcadmin/service.go`, 792 LOC) e
  o contrato está descrito em `docs/integrations/npc-admin-nextjs.md` (535 linhas, com tabela
  de RPC e semântica de cada campo).
- `UpsertNpc(slug, template_name, display_name, enabled, map_id, pos_x, pos_y, route_type, merchant)`;
  `SetNpcShop(npc_id, items[])` substitui a loja **inteira** (remover item = reenviar sem ele);
  `SetNpcVisibility(npc_id, enabled)`; `DeleteNpc(npc_id)` devolve `CONTENT_OWNED` para NPC vindo do
  `NPCGener.txt`.
- `ListMerchantTemplates` e `ListItemCatalog` só têm conteúdo quando o web-api sobe com `-content`
  (`docker-compose.yaml:59`).
- Duas pré-condições de runtime, ambas do lado do servidor: seed `dbserver import-npcs -content
  /Release -dsn …` e `W2PP_NPC_EDITING=true` no tmserver — sem isso a edição fica só no banco.
- `map_id` não tem efeito de gameplay hoje (grade única); `ListMapZones` existe só para o picker
  mostrar Armia/Azran/Erion/Nippleheim/Noatum (`webserver/internal/mapzones`).

## Passos

1. `/npcs` (lista): `DataTable` com colunas `slug`, `display_name`, template, `map_id` (nome), posição,
   merchant, visibilidade, origem (`content`/`db`) e ações. Busca por nome/slug; filtro
   "somente ocultos"; badge de origem.
2. Filtro/paginação client-side; contagem no topo via `ListNpcs`.
3. `/npcs/[id]` (detalhe) e `/npcs/novo`:
   - form com `slug` (obrigatório, `[a-z0-9-]{3,40}`), `display_name` (1–40), `template_name`
     (Combobox alimentado por `ListMerchantTemplates`), `map_id` (Select de `ListMapZones`),
     `pos_x`/`pos_y` (−32768..32767, hint das coordenadas das cidades), `route_type` (0–3),
     `merchant` (valores aceitos: ver validação em `webserver/internal/npcadmin`; mostrar legenda),
     `enabled`.
   - validação espelhada com zod, mas a decisão final é do `AdminResult.INVALID`.
4. Aba **Loja** (só quando `merchant != 0`): tabela de 27 slots (`MSG_ShopList` 0..26). Cada linha:
   slot (fixo), item (Combobox do catálogo, mostra índice + nome + ícone), quantidade (1–255), 3 pares
   eff/effv (opcionais). Botões "adicionar item", "limpar slot", "remover". Salvar = enviar **o array
   completo** via `SetNpcShop` (é substituição, não patch).
5. Ações de linha: "ocultar/mostrar" (`SetNpcVisibility`), "duplicar" (novo `slug` com `-copy`),
   "apagar" (Dialog de confirmação exigindo digitar o `slug`).
6. Tratamento de `CONTENT_OWNED` no delete: toast de aviso + botão "ocultar em vez de apagar"
   (não é erro interno).
7. Depois de salvar: toast de sucesso com aviso fixo **"aplica no próximo restart do tmserver"** e
   link para `#restart` na página `/operacao` (issue 38).

## Arquivos

- Criados: `portal/src/app/(admin)/npcs/page.tsx`, `portal/src/app/(admin)/npcs/[id]/page.tsx`,
  `portal/src/app/(admin)/npcs/novo/page.tsx`,
  `portal/src/server/actions/npcs.ts` (list/get/upsert/visibility/shop/delete),
  `portal/src/lib/npc.ts` (tipos + zod), `portal/src/components/npc/{ShopEditor,TemplatePicker}.tsx`.
- Alterados: `portal/src/components/nav.ts` (tirar "em breve" de `/npcs`).
- Proibido: escrever em `Release/TMsrv/run/NPCGener.txt`; apagar NPC `origin=content` (usar ocultar).

## Testes / verificação

```bash
# pré-requisitos no servidor (uma vez)
# (comandos rodam na raiz do repositório)
docker compose run --rm dbserver import-npcs -content /Release -dsn "postgres://postgres:dev@db:5432/postgres"
# ligar o overlay e reiniciar só o tmserver
W2PP_NPC_EDITING=true docker compose up -d tmserver
docker compose logs --tail 20 tmserver | grep -i npc
```

```bash
cd portal && npx tsc --noEmit && npm run build
# smoke do contrato (sem UI)
node -e '…' # ou usar a tela; o importante é a evidência:
```

Evidência esperada, na tela: criar `slug=ferreiro-teste` em Armia `(100,120)` merchant=1, adicionar 2
itens na loja, salvar → aparece na lista; ocultar → `enabled=false`; entrar no jogo e abrir o NPC:
estoque novo presente (após restart do tmserver, log `npc overlay applied`).

## Critérios de aceite

- [x] Criar, editar, ocultar e apagar funcionam com `AdminResult` tratado (inclui 409 em conteúdo)
- [x] Loja salva o array completo e remove item ausente (sem RPC extra)
- [x] `template_name` e item só por picker (sem digitar índice à mão)
- [x] Aviso de restart visível após toda escrita
- [x] Conta `player` recebe 403 em toda ação; auditoria registra a tentativa
- [x] Evidência no jogo: NPC novo com estoque, após restart — 04/11: o `QA Ferreiro` apareceu em
  Armia com os três itens e **os três foram comprados** no cliente

## Riscos / rollback

- **Risco:** ligar `-npc-editing` sem o seed derruba os mercadores do mundo. **Mitigação:** a tela
  mostra banner vermelho quando `ListNpcs` volta vazio; ordem do seed documentada no README.
- **Risco:** `SetNpcShop` parcial apagar itens. **Mitigação:** editor sempre envia o array completo e
  exibe diff antes de salvar.
- **Risco:** duplicar `slug` (unique). **Mitigação:** validação local + `INVALID` do servidor.
- **Rollback:** `SetNpcVisibility(false)` no NPC criado errado; `-npc-editing` off volta ao
  `NPCGener.txt` puro.

## Evidência de execução (2026-11-03)

```text
$ npx tsc --noEmit    # tsc_exit=0
$ npm run build       # build_exit=0

# probe /api/npc-probe (rota temporária, removida depois) chamando as server actions
templates = 228
UpsertNpc(ferreiro-teste)      → {ok:true, npcId:629}
GetNpc(629)                    → origin=custom, shop=0, enabled=true
SetNpcShop([0:1×5, 1:2×1])     → shop = ["0:1x5","1:2x1"]
SetNpcShop([1:2×3])            → shop = ["1:2x3"]      ← slot 0 removido pela substituição
SetNpcVisibility(629,false)    → {ok:true}; GetNpc → enabled=false
DeleteNpc(<NPC de conteúdo>)   → {ok:false,status:409,contentOwned:true}; NPC continua no banco
DeleteNpc(ferreiro-teste)      → {ok:true}; GetNpc → "o registro não existe mais"

# player (conta teste) chamando uma ação admin
{ok:false,status:403,error:"sua conta não tem permissão para esta ação"}
[audit] {…,"accountId":3,"role":"player","action":"npc.upsert","target":"npc:negado-teste","result":2,"status":403}

# páginas
/npcs 200 · /npcs/novo 200 · /npcs/1 200 · /npcs/999999 404 · 568 definições / 228 mercadores
```

- Criados: `app/(admin)/npcs/{page.tsx,novo/page.tsx,[id]/page.tsx}`,
  `components/npc/{npc-table,npc-form,shop-editor}.tsx`, `server/npcs.ts`,
  `server/actions/npcs.ts`, `lib/npc.ts`. Alterados: `server/grpc.ts` (cliente `NpcAdminService`),
  `server/admin.ts` (negação auditada), `globals.css` (`.ui-banner`, `.ui-shop-grid`).
- **Correção durante a issue:** `requireStaff` lançava antes de qualquer `logAudit`, então a
  tentativa de uma conta `player` não deixava rastro. `adminCall` agora usa `guardStaff`, que grava
  a negação (`result=FORBIDDEN`, `status` 401/403, `accountId=0` quando anônimo) e só então lança.
  Foi o que fez a sonda de player devolver 500 virar 403 auditado.
- **Desvio 1:** o detalhe é rota (`/npcs/[id]`) com abas, não drawer — mesmo motivo da issue 22.
- **Desvio 2:** o combobox de item é textual (índice + nome). O `Combobox` não tem slot de ícone e o
  pack não existe nesta máquina; ícone no picker só faz sentido junto da issue de ícones.
- **Desvio 3:** "duplicar" copia também a loja (sem isso a cópia de um mercador nasce vazia) e usa o
  sufixo `-copy`; colisão de slug aparece como `INVALID` do servidor.
- **Resolvido em 04/11 (ver seção de 2026-11-04):** o compose passou a repassar os overlays, então
  `W2PP_NPC_EDITING=true docker compose up -d tmserver` funciona. Falta apenas a conferência na tela.
- **Pendente:** botões de "ver preço/quem dropa/usar em loja" do detalhe continuam texto, porque as
  telas 24/26 ainda não existem.

## Evidência de execução (2026-11-04) — overlay aplicado no mundo

Seed já existente: 568 definições / 108 mercadores. Definição criada espelhando a transação da
server action (definição + loja + bump de versão):

```text
insert npc_definition(slug=qa-ferreiro, template=Acessorios, Armia 2090,2106, merchant=1) → id 2273
insert npc_shop_item slots 0..2                                                           → itens 591, 592, 593
update npc_config_meta set version = version + 1                                          → 11

$ W2PP_NPC_EDITING=true docker compose up -d tmserver
msg="tmserver config" npc_editing=true …
msg="npc config overlay enabled (moderator editing)"
msg="NPCs spawned" generators=6099 mobs=11126 templates=620 merchant_blocks_skipped=548
msg="npc config applied at boot" version=11 npcs=568

# hot-reload, sem restart: a contagem anda ±1 com o `enabled` do qa-ferreiro
update enabled=false + bump → msg="npc config reloaded" version=13 npcs=567
update enabled=true  + bump → msg="npc config reloaded" version=14 npcs=568
```

- A entidade gerida é a nossa: esconder a definição remove exatamente uma entidade do mundo
  (568 → 567) e reativá-la devolve (567 → 568); o hot-reload aplicou três versões sem restart.
- `npcs=568` para 569 linhas: uma definição do seed não materializa — investigar separadamente.
- **Correção de infraestrutura durante o teste:** o passo documentado não funcionava porque o
  serviço `tmserver` não tinha bloco `environment:` no compose (o `docker compose up` não recriava
  nada). Adicionado passthrough dos seis overlays com default `false`.
- **Desvio:** a linha foi gravada por SQL espelhando a transação da server action; o caminho
  portal → web-api → auditoria já tem evidência de 2026-11-03.

### Confirmação no jogo (2026-11-04, 20:2x)

- O `QA Ferreiro` **apareceu** em Armia, abriu a loja com os **três itens** (591/592/593) e os três
  foram **comprados** no cliente — critério 6 cumprido, overlay de NPC comprovado ponta a ponta.
- **Bug encontrado no mesmo teste (B18, `docs/migration/ingame-bugs.md`):** quase todos os mercadores
  de Armia apareciam em dobro. Eram **20 linhas legadas** do seed pré-0018 (`generator_index` NULL,
  `generator_data` vazio, `display_name` = template) na mesma posição/template de um bloco NPCGener
  já importado — o overlay materializava o bloco **e** a sombra. Somado a isso, `applyNPCConfig`
  refazia o conjunto inteiro a cada bump de versão.
- **Correção:** as 20 sombras foram desativadas (`enabled=false`), o `store.SeedNPCDefinitions`
  passou a aposentá-las (`retireShadowedDefinitions`) e o reconcile virou **diff** — no reload
  seguinte o log mostra `npc config reloaded version=16 npcs=548` (só as sombras saíram) contra
  `npcs=568` antes, com as 549 definições ativas preservadas.
