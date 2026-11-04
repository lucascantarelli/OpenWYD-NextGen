# Issue 23 — Editor de NPCs e mercadores

**Fase:** 6 · **Depende de:** 20, 21 · **Bloqueia:** 24 · **Esforço:** L · **Estado:** 🟡 UI ✅ (2026-11-03) — falta a evidência in-game

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
- [ ] Evidência no jogo: NPC novo com estoque, após restart

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
- **Pendente (critério 6):** evidência in-game não foi feita. Exige `W2PP_NPC_EDITING=true` no
  `tmserver` (a flag **não** está no `docker-compose.yaml`) e restart do serviço de jogo; a escrita
  ficou só no Postgres. O compose e o restart não foram alterados sem o OK do usuário.
- **Pendente:** botões de "ver preço/quem dropa/usar em loja" do detalhe continuam texto, porque as
  telas 24/26 ainda não existem.
