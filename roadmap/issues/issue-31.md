# Issue 31 — Faturamento e relatórios de donate

**Fase:** 6 · **Depende de:** 21, 29 · **Bloqueia:** — · **Esforço:** M · **Estado:** ⏳ aberta

## Objetivo

Painel de receita: KPIs do período, pedidos de recarga, maiores compradores, gasto em donate e busca
de contas — tudo leitura, restrito a `admin`.

## Contexto / evidência

- `DonateRevenueAdminService`: `GetRevenueSummary`, `ListTopupOrders`, `ListTopBuyers`,
  `ListDonateSpend`, `SearchAccounts` (`web.proto:1066-1090`); lógica em
  `webserver/internal/donaterevenue` (1005 LOC). Guia completo em
  `docs/integrations/faturamento-nextjs.md` (406 linhas).
- `RevenueTotals`: `paid_orders`, `gross_cents`, `credits_sold`, `distinct_buyers`, `created_orders`,
  `pending_orders`, `pending_cents`, `shop_purchases`, `credits_spent`, `manual_credits`,
  `credits_granted`. **`pending_cents` não é receita** — nunca somar em `gross_cents`.
- `RevenueWindow{from_unix, to_unix}`: 0/0 = últimos 30 dias.
- O proto pede centavos serializados como **string** no JSON público (precisão + formatação).
- Acesso: relatório de receita é `admin` (o próprio proto comenta que `moderator` é papel grosseiro
  demais para receita — `web.proto:1178`).

## Passos

1. `/faturamento`: filtro de período (7/30/90 dias + intervalo custom, convertendo para Unix seconds).
2. Cartões KPI: receita paga (BRL), pedidos pagos, compradores distintos, créditos vendidos, gasto em
   loja, créditos concedidos manualmente. Tooltip explicando cada um e o aviso de `pending`.
3. Gráfico simples (SVG/barras CSS, sem lib) de receita por dia e por método (`pix_*`, `card_*`).
4. Tabelas: pedidos (`ListTopupOrders` com status, valor, método, conta), top compradores
   (`ListTopBuyers`), gasto em donate (`ListDonateSpend`, com `action='purchase'` vs
   `'credit_balance'`).
5. Busca de contas (`SearchAccounts`) no topo, usada também pela issue 29 (crédito manual) — expor um
   `Combobox` reutilizável.
6. Formatadores: `fmtBRL(cents)` com `Intl.NumberFormat("pt-BR", { style: "currency", currency:
   "BRL" })` a partir da string de centavos; nunca float.
7. Export CSV dos pedidos do período (client-side).
8. Guarda dupla: layout `(admin)` + checagem `role === "admin"` na page; `moderator` vê
   `/sem-permissao`. O backend continua sendo a autoridade.

## Arquivos

- Criados: `portal/src/app/(admin)/faturamento/page.tsx`,
  `portal/src/server/actions/revenue.ts`, `portal/src/lib/money.ts`,
  `portal/src/components/revenue/{KpiCard,RevenueBars,OrdersTable}.tsx`.
- Alterados: `portal/src/components/nav.ts`.
- Proibido: somar `pending_cents` à receita; usar `number` para centavos; consultar Postgres direto.

## Testes / verificação

```bash
cd portal && npx tsc --noEmit && npm run build
```

Evidência: com um pedido confirmado (`ConfirmTopupOrder`) e uma compra na loja, o resumo mostra
`paid_orders=1`, `gross_cents` igual ao valor do pedido e `shop_purchases=1`; alterar o período muda
os números de forma coerente; pedido pendente aparece em `pending_orders` **sem** entrar na receita.

## Critérios de aceite

- [ ] KPIs batem com o banco para um período fechado (conferir 3 valores via SQL)
- [ ] `pending_cents` nunca somado a `gross_cents`
- [ ] Centavos tratados como string/inteiro, formatados em BRL
- [ ] `moderator` não acessa; `admin` acessa
- [ ] CSV do período abre corretamente
- [ ] Nenhuma chamada ao Postgres a partir do portal

## Riscos / rollback

- **Risco:** número errado virar decisão de negócio. **Mitigação:** definição de cada KPI em tooltip +
  conferência SQL na issue.
- **Risco:** período custom em timezone errado. **Mitigação:** tratar tudo em UTC no BFF e exibir em
  `America/Sao_Paulo` só na formatação.
- **Rollback:** read-only.
