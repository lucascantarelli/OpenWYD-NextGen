# Issue 31 — Faturamento e relatórios de donate

**Fase:** 6 · **Depende de:** 21, 29 · **Bloqueia:** — · **Esforço:** M · **Estado:** ✅ fechada (2026-11-04)

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
- `RevenueWindow{from_unix, to_unix}`: 0/0 = últimos 30 dias; janela > 366 dias = `INVALID`.
- O proto pede centavos serializados como **string** no JSON público (precisão + formatação). O BFF
  não monta JSON público: os `int64` chegam do proto-loader como `number` e viram texto só na
  formatação, com aritmética inteira — nunca `cents / 100` em float.
- Acesso: relatório de receita é `admin` (o próprio proto comenta que `moderator` é papel grosseiro
  demais para receita — `web.proto:1178`).
- Buckets fecham em `America/Sao_Paulo`; o Brasil não tem mais horário de verão, então o portal
  interpreta as datas digitadas com offset fixo `-03:00`.

## O que foi feito

1. **`/faturamento`** (`app/(admin)/faturamento/page.tsx`): server component que lê
   `searchParams` — `periodo` (`7d`/`30d`/`90d`/`custom`), `desde`/`ate`, `serie`, `conta`/`nome`,
   `status`, `metodo`, `extrato` — e chama o resumo, os pedidos, os compradores e o extrato em
   paralelo. O filtro vive na URL: link compartilhável, sem estado duplicado, e cada mudança é um
   round-trip do servidor (nada de refetch no browser).
2. **Guarda dupla**: middleware (cookie) + `role !== "admin"` na page → `/sem-permissao`. `moderator`
   continua operando `/loja`; receita e dados de pagador são do `admin`.
3. **KPIs em dois blocos** — dinheiro (`Receita paga` com `paid_orders`, `Ticket médio`,
   `Compradores`, `Créditos vendidos`) e operação (`Pedidos criados`, `Aguardando pagamento` com o
   valor e o rótulo "não é receita", `Compras na loja` + donate gastos, `Créditos manuais` +
   concessões). Cada cartão carrega a definição no `hint`, como a issue pediu.
4. **Série sem lib**: barras CSS com largura proporcional ao `gross_cents` do bucket, rótulo
   `01/11` / `sem. 01/11` / `nov/26`, `title` com receita, pedidos e donate. Granularidade padrão
   pela largura (≤45 dias → dia, ≤200 → semana, senão mês) e seletor manual com opção "sem série".
5. **Forma de pagamento**: lista com pedidos, participação % e valor por método, direto de
   `by_method` (repeated, então um `PaymentMethod` novo não quebra a tela).
6. **Tabelas** (`revenue-view.tsx`, client): pedidos (id + referência, criado, conta/e-mail,
   pagador + CPF mascarado, valor, donate, método, status, pago em), top compradores com receita da
   janela, ticket, LTV histórico e saldo atual, e extrato da carteira com `créditos` assinados,
   saldo após e oferta/motivo. `DataTable` faz busca/ordenação/paginação fina; o servidor manda 100
   pedidos e 100 movimentos, avisando quando o total é maior ("estreite o período").
7. **CSV client-side** dos pedidos visíveis: `;` e vírgula decimal (Excel pt-BR), BOM UTF-8,
   `valor_centavos` **e** `valor_reais`, referência externa e datas em Brasília.
8. **Busca de conta reutilizável**: `/api/search/accounts` (staff, mínimo 2 letras) +
   `lib/accountsearch.ts` (`Combobox`), usada no filtro do faturamento **e** no crédito manual da
   loja — o campo "id numérico da conta" da issue 29 deixou de existir.
9. **`lib/money.ts`**: `fmtBRL`/`centsToDecimal` por divisão inteira (sem float), `fmtCount`,
   `averageCents`, `fmtPercent`. **`lib/revenue.ts`**: tipos de wire, `resolveWindow` (valida
   formato, ordem e o teto de 366 dias), `bucketForDays`, rótulos e `ordersCsv`.

## Arquivos

- Criados: `portal/src/lib/money.ts`, `portal/src/lib/revenue.ts`, `portal/src/lib/accountsearch.ts`,
  `portal/src/server/revenue.ts`, `portal/src/app/(admin)/faturamento/{page,revenue-view}.tsx`,
  `portal/src/app/api/search/accounts/route.ts`, `portal/src/lib/{money,revenue}.test.ts`.
- Alterados: `portal/src/server/grpc.ts` (`donateRevenueAdmin` + tipos), `portal/src/components/nav.ts`
  (`/faturamento` deixa de ser "em breve"), `portal/src/app/(admin)/loja/shop-table.tsx` (conta por
  login), `portal/src/lib/donate.ts` (mensagem de validação).
- Proibido e respeitado: somar `pending_cents` à receita; `float` para centavos; Postgres a partir
  do portal. Nenhum arquivo de `Release/` foi tocado e nada escreve estado vivo.

## Testes / verificação

```bash
cd portal && npx tsc --noEmit && npx vitest run && npm run build
```

- `tsc --noEmit` exit 0; `vitest` **68/68** em 8 arquivos (15 novos: 7 `money`, 8 `revenue`);
  `next build` exit 0 — `/faturamento` 3,18 kB e `/api/search/accounts` presente.
- Evidência ao vivo no portal do host (`:3100`, `next start`), com dados de fixture inseridos por
  SQL (não pelo portal) e sessão `admin` assinada:

  | KPI | Tela | SQL |
  |---|---|---|
  | Receita paga | `R$ 50,00` · 1 pedido | `sum(amount_cents) = 5000` |
  | Aguardando pagamento | 1 · `R$ 37,50` (não é receita) | `count = 1`, `sum = 3750` |
  | Créditos vendidos | 250 | `sum(credits) = 250` |
  | Pedidos criados | 2 | `count(*) = 2` |
  | Compras na loja | 1 · 500 donate | `count = 1`, `after->>'price' = 500` |
  | Créditos manuais | 200 · 1 concessão | `after->>'amount' = 200` |

  `pending_cents` apareceu em cartão separado e `gross_cents` continuou 5000 — não houve soma.
- `moderator` → `307` para `/sem-permissao`; sem sessão → `307` para `/login`; `/api/search/accounts`
  sem sessão → `401`. Com sessão, a API devolveu `{"value":"1","label":"test","hint":"#1 · 0 donate ·
  admin"}`, e a page mostrou "exportar CSV" e o `Combobox` "filtrar por conta".
- Fixtures removidas ao final (`donate_topup_order` = 0, `donate_shop_audit` = 0,
  `donate_payer_profile` = 0), então o banco voltou ao estado anterior.

## Critérios de aceite

- [x] KPIs batem com o banco para um período fechado (conferir 3 valores via SQL) — seis conferidos
- [x] `pending_cents` nunca somado a `gross_cents`
- [x] Centavos tratados como string/inteiro, formatados em BRL (sem float em nenhum ponto)
- [x] `moderator` não acessa; `admin` acessa
- [x] CSV do período abre corretamente (`;`, vírgula decimal, BOM, valores em reais e centavos)
- [x] Nenhuma chamada ao Postgres a partir do portal

## Pendente / ressalvas

- O proto entrega `payer_cpf_masked` (mascarado de propósito) e `provider` vazio: a tela não constrói
  nada sobre provedor, como o contrato manda.
- Conciliação por `external_reference` é manual (copiar do CSV); não existe RPC de estorno/chargeback.
- O top de compradores é limitado a 50 e o extrato/pedidos a 100 por carga: para janelas muito
  largas o painel pede para estreitar o período, em vez de paginar no browser.
- Relatório de resgates da recompensa diária (issue 30) continua sem RPC; não foi inventado aqui.

## Riscos / rollback

- **Risco:** número errado virar decisão de negócio. **Mitigação:** definição de cada KPI no cartão +
  conferência SQL documentada acima.
- **Risco:** período custom em timezone errado. **Mitigação:** datas digitadas viram `[from, to)`
  com offset `-03:00` e a tela mostra a janela que o servidor aplicou, não a digitada.
- **Rollback:** read-only — apagar a rota e a linha do menu.
