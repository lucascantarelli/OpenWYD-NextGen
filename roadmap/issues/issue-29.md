# Issue 29 — Administração da loja de donate

**Fase:** 6 · **Depende de:** 21, 22 · **Bloqueia:** 31 · **Esforço:** M · **Estado:** ⏳ aberta

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

- [ ] CRUD de oferta completo, com preço e efeitos fiéis ao proto
- [ ] Habilitar/desabilitar reflete no `ListShopItems` do jogador
- [ ] Crédito manual exige motivo e mostra `new_balance`
- [ ] Entrega comprovada no jogo após login (com registro em `delivery_queue`)
- [ ] `player` recebe 403; auditoria com motivo

## Riscos / rollback

- **Risco:** crédito manual errado virar dinheiro real de alguém. **Mitigação:** confirmação dupla +
  motivo obrigatório + log; correção só por crédito compensatório auditado.
- **Risco:** preço em donate confundido com coin. **Mitigação:** rótulo "donate" em toda a tela.
- **Rollback:** desabilitar a oferta e apagar; crédito não tem rollback automático (por desenho).
