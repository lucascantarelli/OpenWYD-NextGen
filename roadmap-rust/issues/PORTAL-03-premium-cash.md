# Issue PORTAL-03 — Entregas → Premium Mail + Cash + gestão financeira

**Tipo:** Código (portal+gateway+jogo) · **Esforço:** L · **Depende de:** ROAD-01, FASE-07-4 ✅ · **Bloqueia:** —

## Objetivo

Substituir `/entregas` genérica por Premium Mail + Cash: jogador recebe itens/recompensas com segurança e rastreabilidade, com gestão financeira (saldo, histórico, compras) e sem sistema paralelo de "entregas".

## Contexto / evidência

- Hoje: `/entregas` enfileira **ouro** via `grants/request` idempotente (`delivery_queue`, dreno no `Carry`) + fila por personagem; E2E prova idempotência (gold 0→250, replay sem duplicar).
- `prompt.md` pede: remover entregas soltas; premium mail + cash; histórico/status/notificações; gestão financeira (transações, saldo, compras); auditoria; visuais (gráficos/tabelas); referência em privados/docs/offgame oficial; itens com origem rastreável (`wyd2.md` §4.25).

## Passos

1. Pesquisa de referência (dentro do repo primeiro): `docs/wyd/economia.md`, `docs/wyd/fontes.md`, `droplist`/Kersef citados — registrar o que é mecânica (vale) vs número de servidor (vira `game_setting`).
2. Modelo: `premium_mail` (id, conta/personagem, item snapshot + origem, estado `pending/claimed/expired`, expiração) + `cash_balance`/`cash_ledger` (crédito, débito, saldo, motivo, idempotência) — migrations reversíveis, sem `ALTER` no `up`.
3. Gateway: `premium_mail/*` (enviar, listar, resgatar com claim idempotente no dreno do jogo) + `cash/*` (creditar via `donate_admin`, extrato, saldo) — `501` onde não houver tabela; `result` 1..5 + auditoria.
4. Portal: `/correio` (jogador: caixa, resgatar, expiração) + `/admin/correio` e `/admin/financeiro` (staff: enviar, histórico, status, notificações, gráficos de concessões/vendas) — tudo por `adminCall`; portal nunca credita direto.
5. Identificação de item: preservar origem (`content_source`, grant id) em troca/armazenamento/movimentação para anti-dupe.

## Arquivos

- Criados: migrations `premium_mail`, `cash_*`; `crates/wyd-admin/src/transport/premium.rs`; `wyd-portal/src/app/(public)/correio/*`, `wyd-portal/src/app/(admin)/financeiro/*`; testes.
- Alterados: `wyd-portal/src/server/{deliveries→premium,catalog,economy}.ts`, `crates/wyd-db/src/repo/*`, `crates/wyd-server/src/*dreno*` (claim no jogo), `docs/wyd/economia.md`, `docs/rust/08-checklist-funcionalidades.md`.
- Proibido: portal escrever estado vivo; número externo virar constante (vai para `game_setting`).

## Testes / verificação

```bash
cargo test -p wyd-db -p wyd-admin
cd wyd-portal && npx tsc --noEmit && npx vitest run
```

Saída esperada: claim idempotente (replay = uma linha, sem duplicar item/gold); expiração respeitada; saldo = soma do ledger.

## Skills relacionadas

- wyd-economy (legada, só contexto) → sucessora Rust `wyd-rust-db`.
- wyd-rust-testing — goldens de claim, concorrência.
- frontend-design — extrato, gráficos sem número inventado.

## Documentação de referência

- `wyd2.md` §4.25 (cash/premium/identificação), `docs/wyd/economia.md`, `docs/arquitetura/modelo-banco.md`
- `crates/wyd-db/migrations/*delivery*`, `wyd-portal/src/server/deliveries.ts`

## Subtasks

- [x] Correio staff: conceder item (`character_admin/grant_item`) + fila com status/histórico (`/correio`) + replay de item provado (`item_grant_is_idempotent_across_calls`, mesma chave → mesmo `grant_id`)
- [x] Financeiro staff: vitrine CRUD + crédito idempotente (`donate_admin/*`, `/financeiro`)
- [ ] Inbox do jogador — BLOQUEADO: fila só legível por staff (`character_admin/*` exige staff); sem rota nova no gateway não há inbox honesto
- [ ] Remover `/entregas` — ADIADO de propósito: E2E (`entregas.spec`, `idempotencia.spec`, `login.spec`) prova o fluxo de ouro nela; ouro fica em `/entregas`, item no `/correio`
- [ ] Extrato/receita por comprador — BLOQUEADO: `donate_revenue_*` declarado sem handler (501); tela avisa o que não é servido

## Critérios de aceite

- [ ] Replay com mesma chave não duplica item/gold (teste dedicado).
- [ ] Item de premium preserva origem após troca/armazenamento.
- [ ] `/entregas` antiga removida ou redirecionada sem quebrar E2E (atualizar spec).

## Riscos / rollback

- Risco: dupe em claim concorrente → mitigação: `UNIQUE(idempotency_key)` + claim transacional no jogo.
- Rollback: manter `grants/request` legada até migração completa (feature flag).

## Verificação on-line (2026-11-11)

- **Entrega de ouro ponta a ponta**: clique → server action → `grants/request`
  → `delivery_queue`. Recibo exibido: `entrega #1 enfileirada`; a tela
  `/economia` passou a mostrar **1 pedido / 100 de ouro** na mesma sessão.
- **`/entregas` foi refeita nesta rodada**: o formulário de concessão ocupava
  uma célula inteira (linha de ~215 px, tabela estourando a largura e
  "CHAVE DE IDEMPOTÊNCIA" exposta). Agora é uma linha compacta por personagem
  com valor + "Conceder" + "ver fila"; a chave é gerada e mantida oculta.
- **Inbox do jogador**: BLOQUEADO. `character_admin/*` exige staff; não há rota
  de leitura de caixa para o dono da conta. Precisa de `premium_mail/*` no
  gateway + tabela `premium_mail` (migration) antes de qualquer tela.
- **Extrato/receita por comprador**: BLOQUEADO. `donate_revenue_*` está
  declarado no transporte sem handler (responde 501).
- **Cash**: só existe crédito de saldo de donate (`donate_admin/credit_*`);
  débito/compra no jogo e ledger de cash não têm rota.
