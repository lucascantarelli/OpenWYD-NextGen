# Issue 30 — Administração da recompensa diária

**Fase:** 6 · **Depende de:** 21, 22, 29 · **Bloqueia:** — · **Esforço:** S · **Estado:** ⏳ aberta

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

- [ ] CRUD completo com efeitos e `expires_days` fiéis ao proto
- [ ] Duas ofertas habilitadas não permitem dois resgates no mesmo dia
- [ ] Item entregue no login, com registro em `delivery_queue`
- [ ] `player` recebe 403; auditoria registra escrita
- [ ] Aviso de "uma por dia / próximo login" visível

## Riscos / rollback

- **Risco:** recompensa virar fonte infinita de item. **Mitigação:** unicidade por dia é do banco;
  a UI só edita catálogo.
- **Rollback:** desabilitar/apagar a oferta; claims já feitos permanecem (histórico).
