# FASE-08-10 — Trade atômica por par (bloqueada em layout)

**Fase:** 8 · **Depende de:** FASE-08-6 🚧 (captura do layout) · **Bloqueia:** — · **Esforço:** M · **Estado:** ⛔ bloqueada em layout

## Objetivo

Implementar a troca jogador↔jogador sem dup, com confirmação bilateral e swap
atômico de itens + gold — SOMENTE após a captura do `MSG_Trade` real do 7662.

## Contexto / evidência

Pesquisa `docs/wyd/pesquisa-wyd2-missao-fase3-2026-11-10.md` item 7 (26 campos).
Fluxo do legado (`flows.md` §5): `TradingItem` (oferta) → `Trade` (estado
espelhado) → ambos `MyCheck==1` → swap ATÔMICO (`Item[]`+gold) → `Trade(done)`;
`QuitTrade (0x0384)` cancela ambos (`RemoveMob`/ambos). Opcodes no catálogo
(`Trade/QuitTrade/TradingItem`, `opcode_table.rs`).

**Bloqueio:** `MSG_Trade` tem layout UNVERIFIED (invariante 5 do `AGENTS.md`;
`commerce.rs:3`, `payload/mod.rs:12,293`, `require_ported` → `Unported`). Codar
contra layout chutado é proibido. Até a captura, nada desta issue sai do papel.

## Passos

1. Capturar o `MSG_Trade` real (FASE-08-6): layout campo a campo
   (`Item[15]`, `InvenPos[15]`, `TradeMoney`, `MyCheck`, `OpponentID` —
   `Basedef.h:2435` como hipótese, não verdade).
2. Tipar o payload só com o capturado; handler atômico por par (uma transação:
   dois inventários + gold), `QuitTrade` cancela ambos.
3. Revalidar slots/ouro na confirmação; rollback em desconexão; idempotência
   por oferta; `OpponentID` validado server-side.
4. Testes de contenção: dois swaps no mesmo slot, replay, desconexão no meio.

## Arquivos

- Criados: `crates/wyd-server/tests/trade.rs` (quando o layout existir).
- Alterados: `crates/wyd-protocol/src/payload/commerce.rs` (só com captura),
  `crates/wyd-server/src/{cpsock,characters}.rs`, checklist 5.10.
- Proibido: tipar `Trade` sem captura; tocar a cerca de sessão única.

## Testes / verificação

```bash
cargo test -p wyd-server --test trade
```

Saída esperada: confirmação bilateral debita/credita exatamente uma vez;
cancelamento não move nada; concorrência não duplica.

## Skills Relacionadas do freebuff

- .agents/skills/wyd-rust-net — pacote e sessão.
- .agents/skills/wyd-rust-db — transação por par.
- .agents/skills/wyd-rust-testing — testes de contenção.
- .agents/skills/wyd-client-reverso — captura do layout.

## Documentação de referencia

- docs/wyd/pesquisa-wyd2-missao-fase3-2026-11-10.md (item 7).
- `docs/migration/flows.md` §5.
- `docs/arquitetura/protocolo.md` (quando o layout for promovido).

## Subtasks

- [ ] Codificar issue
- [ ] Testar issue
- [ ] Revisar issue
- [ ] Documentar issue
- [ ] Versionar issue, mergear branch e fechar issue
- [ ] Limpar branch (se aplicável)

## Critérios de aceite

- [ ] Layout capturado e documentado (não herdado do `Basedef.h`).
- [ ] Swap atômico auditado; nenhum dup sob contenção.
- [ ] Desconexão reverte sem perda nem duplicata.

## Riscos / rollback

- Risco: dup econômico (o pior do jogo) → mitigação: transação por par +
  idempotência + testes de race antes de qualquer merge.
- Rollback: remover o handler; `Unported` volta a recusar.
