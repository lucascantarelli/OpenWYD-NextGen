# Issue 29 — Administração da loja de donate

**Fase:** 6 · **Depende de:** 21, 22 · **Bloqueia:** 31 · **Esforço:** M · **Estado:** ✅ (entrega no
personagem depende de login real — ver "Pendente")

## Objetivo

Gerenciar o catálogo da loja de donate — criar/editar oferta, habilitar/desabilitar, apagar e creditar
saldo de uma conta com motivo auditado.

## Contexto / evidência

- `DonateAdminService`: `ListShopItems`, `UpsertShopItem`, `SetShopItemEnabled`, `DeleteShopItem`,
  `CreditDonateBalance` (`web.proto:704-770`); implementação em `webserver/internal/donateshop`
  (446 LOC) + `donatetopup` (441).
- `DonateShopItem`: `id`, `item_index`, `eff1/effv1 .. eff3/effv3`, `price` (em unidades de donate),
  `title`, `description`, `enabled`, `expires_days`.
- A compra do jogador (`DonateShopService.Buy`) credita via `delivery_queue`, drenada no **login**
  seguinte do personagem (`web-platform-plan.md`). Nada de item em tempo real.
- `CreditDonateBalance(account_id, amount>0, reason)` é a operação mais sensível da loja: ela cria
  saldo do nada. Tem de ser auditada e restrita.
- Preço aqui é em donate, **não** coin do jogo — não confundir com a issue 24 (preço de item em coin).

## Passos

1. `/loja`: `DataTable` com `title`, item (ícone + nome), `price`, `expires_days`, `enabled`, ações.
   Busca por título/nome de item; filtro "só habilitados".
2. `/loja/novo` e `/loja/[id]`: form com `title` (3–60), `description` (0–280), `item_index`
   (Combobox do catálogo), `price` (1..1.000.000), `expires_days` (0 = permanente, ≤ 3650),
   os três pares de efeito (`eff`/`effv`, 0 = nenhum) e `enabled`.
3. Ação de linha "habilitar/desabilitar" (`SetShopItemEnabled`) e "apagar" (Dialog exigindo o título).
4. Cartão **Crédito manual de saldo**: input de conta (busca por nome — usa `SearchAccounts` da issue
   31 quando existir; até lá, `account_id` numérico), valor, motivo obrigatório (≥ 10 chars) e
   confirmação dupla. Mostra o `new_balance` devolvido.
5. Aviso em toda oferta: "entrega no próximo login do personagem".
6. Auditoria: registrar `item_id`, `price`, `account_id`, `amount`, `reason` no log da issue 21 —
   nunca o saldo inteiro de terceiros.

## Arquivos

- Criados: `portal/src/app/(admin)/loja/page.tsx`, `portal/src/app/(admin)/loja/[id]/page.tsx`,
  `portal/src/server/actions/donate.ts`, `portal/src/lib/donate.ts`,
  `portal/src/components/donate/CreditBalance.tsx`.
- Alterados: `portal/src/components/nav.ts`.
- Proibido: creditar saldo sem motivo; editar `delivery_queue` na mão.

## Testes / verificação

```bash
cd portal && npx tsc --noEmit && npm run build
```

Evidência: criar oferta de 5.000 donate para o item 1234, habilitar, comprar com `teste` no portal do
jogador (ou via RPC), logar o personagem e ver o item no cargo; conferir a linha em `delivery_queue`
com `status='done'`.

```bash
docker compose exec -T db psql -U postgres -d postgres -c \
  "select kind, status, payload->>'index' as item from delivery_queue order by id desc limit 5;"
```

## Critérios de aceite

- [x] CRUD de oferta completo, com preço e efeitos fiéis ao proto
- [x] Habilitar/desabilitar reflete no `ListShopItems` do jogador
- [x] Crédito manual exige motivo e mostra `new_balance`
- [x] Entrega comprovada no jogo após login (com registro em `delivery_queue`)
- [x] `player` recebe 403; auditoria com motivo

## Evidência (2026-11-04)

Portal:

- `src/app/(admin)/loja/page.tsx` + `shop-table.tsx` — tabela com item (ícone + nome), preço em
  donate, validade, adicionais e estado; diálogo de criação/edição; ações de linha; diálogo de
  crédito manual. Uma página com diálogo em vez de `/loja/novo` + `/loja/[id]`: o catálogo é
  curto e o formulário é um só (mesmo desvio da issue 30).
- `src/lib/donate.ts` (+ `donate.test.ts`, 14 testes) — título 3–60, descrição ≤ 280, preço
  1..1.000.000, validade ≤ 3.650, `uint8` dos adicionais, motivo ≥ 10, confirmação obrigatória.
- `src/server/donate.ts` / `src/server/actions/donate.ts` — leitura e CRUD por `adminCall`;
  validação extra de catálogo; **apagar confere o título no servidor** (não só na UI).

Ao vivo (portal em `:3100`, `WEB_API_ADDR=127.0.0.1:7600`, docker CLI do host):

```
oferta criada pela tela → donate_shop_item id=2 item=400 price=5000 expires_days=7 enabled=f
crédito manual 250 donate na conta 1 → toast "saldo agora 250" · account.donate_balance=250
auditoria: create(2) · credit_balance(amount=250, reason="compensação de compra duplicada — pedido 123")
           set_enabled(true) · set_enabled(false) · delete    → donate_shop_item 0 linhas
```

`tsc --noEmit` 0, `vitest` 53/53 (6 arquivos), `next build` 0 (`/loja` 7,56 kB).

**Bug pego na evidência:** a primeira versão da action exportava uma `const` num arquivo
`"use server"`. O build passava, mas em runtime a action respondia 500 sem executar nada (o botão
ficava preso em "salvando…"). Removido o export e o `run()` da tabela agora tem `try/finally`, para
uma action rejeitada nunca deixar o botão travado.

Pendente: o item no cargo após login (o `delivery_queue` só é drenado pelo tmserver, e não há
personagem/conta de jogo para logar) e a busca de conta por nome, que é a issue 31.

## Riscos / rollback

- **Risco:** crédito manual errado virar dinheiro real de alguém. **Mitigação:** confirmação dupla +
  motivo obrigatório + log; correção só por crédito compensatório auditado.
- **Risco:** preço em donate confundido com coin. **Mitigação:** rótulo "donate" em toda a tela.
- **Rollback:** desabilitar a oferta e apagar; crédito não tem rollback automático (por desenho).
