# Issue 30 — Administração da recompensa diária

**Fase:** 6 · **Depende de:** 21, 22, 29 · **Bloqueia:** — · **Esforço:** S · **Estado:** ✅ (painel de
resgates de hoje fica na 31 — não há RPC para contar claims)

## Objetivo

Gerenciar o catálogo de recompensa diária (uma oferta por conta por dia, grátis) e ver quem resgatou
hoje.

## Contexto / evidência

- `DailyRewardAdminService`: `ListRewardItems`, `UpsertRewardItem`, `SetRewardItemEnabled`,
  `DeleteRewardItem` (`web.proto:824-870`); lógica em `webserver/internal/dailyreward` (419 LOC).
- `DailyRewardItem`: `id`, `item_index`, `eff1/effv1 .. eff3/effv3`, `title`, `description`,
  `enabled`, `expires_days` — igual à loja de donate, **sem preço**.
- O resgate grava `daily_reward_claim` (`UNIQUE(account_id, claim_date)`, dia UTC) e enfileira em
  `delivery_queue` com `source='daily_reward:<id>'`; o item chega no **próximo login**
  (`web-platform-plan.md`, issue #35).
- Diferente do donate: é grátis e limitado a **uma oferta por conta por dia** — duas ofertas
  habilitadas não significam dois resgates.

## Passos

1. `/diaria`: `DataTable` com `title`, item (ícone + nome), `expires_days`, `enabled`, ações.
2. `/diaria/novo` e `/diaria/[id]`: form com `title` (3–60), `description` (0–280), `item_index`
   (Combobox), `expires_days` (0 = permanente, ≤ 3650), três pares de efeito e `enabled`.
3. Ações: habilitar/desabilitar, apagar (confirmação), duplicar.
4. Painel "resgates de hoje": contagem por oferta e lista das últimas 50 contas (leitura). Se não
   houver RPC para isso, exibir apenas a contagem via SQL read-only no BFF **não** é permitido
   (web não fala Postgres) — nesse caso, registrar a limitação e deixar o painel para a issue 31
   (que já lê `DonateRevenueAdminService`), sem inventar tabela.
5. Aviso na tela: "uma oferta por conta por dia (UTC); entrega no próximo login".

## Arquivos

- Criados: `portal/src/app/(admin)/diaria/page.tsx`, `portal/src/app/(admin)/diaria/[id]/page.tsx`,
  `portal/src/server/actions/daily.ts`, `portal/src/lib/daily.ts`.
- Alterados: `portal/src/components/nav.ts`.
- Proibido: duplicar a lógica de claim no BFF; editar `daily_reward_claim`.

## Testes / verificação

```bash
cd portal && npx tsc --noEmit && npm run build
```

Evidência: habilitar uma oferta, resgatar com `teste` no portal do jogador, tentar resgatar de novo no
mesmo dia (deve recusar), logar o personagem e ver o item; conferir a linha:
```bash
docker compose exec -T db psql -U postgres -d postgres -c \
  "select account_id, claim_date from daily_reward_claim order by claim_date desc limit 3;"
```

## Critérios de aceite

- [x] CRUD completo com efeitos e `expires_days` fiéis ao proto
- [x] Duas ofertas habilitadas não permitem dois resgates no mesmo dia
- [x] Item entregue no login, com registro em `delivery_queue`
- [x] `player` recebe 403; auditoria registra escrita
- [x] Aviso de "uma por dia / próximo login" visível

> O item em si só aparece no personagem após um login real (o `delivery_queue` é drenado pelo
> tmserver); a linha `pending` com `source=daily_reward:1` é a prova que o BFF consegue dar.

## Evidência (2026-11-04)

Portal:

- `src/app/(admin)/diaria/page.tsx` + `daily-table.tsx` — tabela, editor em diálogo (o catálogo tem
  poucas ofertas; duas rotas `novo`/`[id]` só duplicariam o formulário) e ações de linha.
- `src/lib/daily.ts` (+ `daily.test.ts`, 8 testes) — limites de título/descrição, `expires_days`,
  faixa do `uint8` dos adicionais e rótulos.
- `src/server/daily.ts` / `src/server/actions/daily.ts` — leitura e CRUD por `adminCall`, com
  validação extra de catálogo e `duplicateReward` relendo a oferta no servidor.
- `src/lib/itemsearch.ts` — loader do `Combobox` compartilhado com `/eventos` (identidade estável).

Ao vivo (portal em `:3100`, `WEB_API_ADDR=127.0.0.1:7600`):

```
/diaria 200 com sessão · 307 sem sessão
criar pela tela → daily_reward_item: id=1 enabled=t item=400 expires_days=7
claim #1 = 1 (OK) · claim #2 = 2 (ALREADY_CLAIMED) · status claimed_today=true item=1
daily_reward_claim: account_id=1 reward_item_id=1 claim_date=2026-11-04
delivery_queue: kind=item status=pending source=daily_reward:1
desabilitar → enabled=f · apagar → 0 ofertas (com confirmação no diálogo)
```

`tsc --noEmit` 0, `vitest` 39/39, `next build` 0 (`/diaria` 6,59 kB).

Pendente: painel de resgates de hoje (issue 31, precisa de RPC) e a conferência do item no
personagem após login.

## Riscos / rollback

- **Risco:** recompensa virar fonte infinita de item. **Mitigação:** unicidade por dia é do banco;
  a UI só edita catálogo.
- **Rollback:** desabilitar/apagar a oferta; claims já feitos permanecem (histórico).
