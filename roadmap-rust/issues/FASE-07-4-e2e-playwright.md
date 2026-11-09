# Issue 07-4 — E2E Playwright (auth, loja, painel)

**Fase:** 7 · **Depende de:** 07-2, 07-3 · **Bloqueia:** FASE-15 · **Esforço:** M

## Objetivo

Cobrir com Playwright os fluxos críticos (autenticação, loja, painel) com
idempotência de entrega na loja.

## Contexto / evidência

`pre-release-rust.md` FASE 7 §7.2: E2E nos fluxos críticos. Loja exige entrega
rastreável sem duplicidade (regra permanente).

## Passos

1. Escrever specs: login/logout, compra na loja, painel usuário, painel admin.
2. Garantir idempotência de entrega (transação rastreável + teste de duplo
   clique/retry).
3. Rodar contra ambiente local com backend `wyd-admin` (07-1).
4. `portal-test` + `make check`.

## Arquivos

- Criados: specs E2E no portal, fixtures.
- Alterados: `wyd-portal/**` (somente suporte aos testes).
- Proibido tocar: dados de produção, segredos reais nas fixtures.

## Testes / verificação

```bash
cargo test -p wyd-admin
make check
```

Saída esperada: `test result: ok`; E2E verde (log Playwright anexado);
retry de compra não duplica entrega.

## Skills Relacionadas do freebuff

- .agents/skills/wyd-web-admin-ui - fluxos críticos.
- .agents/skills/wyd-rust-db - idempotência transacional.

## Documentação de referencia

- wyd-portal/README.md - fluxos e BFF.
- pre-release-rust.md (FASE 7 §7.2).

## Subtasks

- [ ] Codificar issue
- [ ] Testar issue
- [ ] Revisar issue
- [ ] Documentar issue
- [ ] Versionar issue, mergear branch e fechar issue
- [ ] Limpar branch (se aplicável)

## Critérios de aceite

- [ ] E2E verde nos 3 fluxos; teste de duplicidade de entrega passando.

> Nota (2026-11-07): idempotência de entrega já provada no backend (`retry` devolve recibo original); E2E aguarda transporte + runtime do portal.

## Evidência de execução (2026-11-09, stack viva)

**Status:** ✅ — suíte Playwright versionada, tela de staff e entrega idempotente provadas na stack
 (3/3). A compra na loja e a entrega de item/nível seguem abertas com motivo (fim do arquivo).

- **Fila durável:** `delivery_queue` (migration 0006) ganhou `idempotency_key TEXT UNIQUE` e
  `created_by BIGINT REFERENCES accounts(id)`. `Grants::enqueue` resolve `account_id` pelo
  personagem, insere com `ON CONFLICT (idempotency_key) DO NOTHING` e devolve `(id, replay)`;
  `Grants::claim_gold` credita o personagem e baixa a linha na **mesma instrução**.
  `GrantService` deixou a fila em memória e usa a porta `GrantQueue`; o gateway liga o pool e o
  canal drena a cada 1 s.
- **HTTP real** (token + `x-wyd-actor: 1:admin`): chave `prova-07-4-k1` (gold 123) →
  `grant_id:2, replay:false`; a mesma chave → `grant_id:2, replay:true`; chave nova →
  `grant_id:4`. O banco tem **uma** linha para k1, com `account_id=49`, `character_id=2`,
  `created_by=1` e `status=claimed`.
- **Consumo pelo jogo:** `characters.gold` do personagem 2 passou de `0` para `230`
  (100 da linha pendente antiga + 123 + 7), com `event=grant delivered` no canal para
  `queue=1/2/4`; as três linhas ficaram `claimed`.
- **Idempotência depois da entrega:** repetir k1 manteve `grant_id:2`, `replay:true` e
  `count(*)=1` — retry não duplica entrega nem reaplica gold.
- **Testes:** `cargo test -p wyd-db --test pg` **8/8** com `WYD_TEST_DSN` (novo
  `enqueue_durável_e_idempotente_e_gold_entregue_uma_vez`), `cargo test -p wyd-db -p wyd-admin`
  **107/0**, `cargo test -p wyd-server` **39/0** com `WYD_TEST_DATABASE_URL`.

## Evidência de execução E2E (2026-11-09)

- **Suíte:** `@playwright/test` versionado em `wyd-portal` (`playwright.config.ts`, `e2e/`),
  `npm run test:e2e` → `scripts/e2e.mjs` provisiona o banco **`wyd_e2e`** (role `wyd_e2e`,
  31 migrations reais), gateway/canal efêmeros e um container do portal do compose em 3110
  apontando para esse gateway; teardown no `finally` e exit code do Playwright preservado.
- **Resultado:** `3 passed (23.4s)`, **0 failed** — `entregas.spec` (login real → busca
  "E2E Hero" → ouro 0→250 relido do banco na tela), `idempotencia.spec` (mesma chave duas
  vezes → segundo recibo `replay`, uma linha na fila, gold 300 sem duplicar) e `login.spec`
  (sem sessão → `/login`; staff entra na área administrativa).
- **Tela:** `/entregas` (menu Operação, guarda no `proxy`) lista personagens com `gold`,
  enfileira ouro com chave editável e mostra a fila (`pending`/`entregue`); usa o contrato
  real `character_admin/search_characters`, `character_admin/list_character_grants` e
  `grants/request` (token + `x-wyd-actor`). Handlers novos no gateway, antes 501.
- **Stack manual:** `docker compose build` + `up -d` reconstruídos; portal em **3000**
  (`/entregas` → 307 para `/login` sem sessão, guarda ativa) e `/health` schema 31.

**Aberto (com motivo):**

1. **Compra na loja** depende de decisão de produto: `donate_shop_item` tem preço em créditos,
   mas **não existe saldo/ledger de moeda premium** no banco (`docs/arquitetura/contrato-persistencia.md`
   U-P4). Implementar a compra exige definir a moeda (gold? crédito?) e onde o saldo mora —
   decisão do dono, não do código.
2. **Item/nível na fila:** o dreno entrega `gold`; `item` e `level` ficam pendentes até o
   inventário/nível do Hot Path consumirem a fila (FASE-03/08).

## Rotina de fechamento (ambiente local)

O teste manual do dono roda no portal do compose na **3000**. A partir desta issue, toda
issue só é dada como concluída depois de um rebuild completo verificado:

```bash
docker compose up -d --build --wait   # caminho único; não exige make
make local-rebuild                   # atalho + provas de contrato
```

O comando é o compose nativo: `docker compose build` + `docker compose up -d --wait`, e o
schema entra no boot do gateway/canal (ADR-0024) — não há passo manual nem container de
migration. O alvo encerra com `/login` 200, `/entregas` 307 para `/login`, gateway `/health` com
`schema:31`, portal publicado em `127.0.0.1:3000` e 3100/3110 livres. O `finally` do
`scripts/e2e.mjs` dropa os containers efêmeros e o banco/role `wyd_e2e`, que é migrado pelo
mesmo runner do boot quando o gateway E2E sobe.

## Riscos / rollback

- Risco: E2E flaky mascarando regressão → mitigação: retries limitados +
  evidência de estabilidade (3 execuções verdes).
- Rollback: `git checkout -- wyd-portal/`.
