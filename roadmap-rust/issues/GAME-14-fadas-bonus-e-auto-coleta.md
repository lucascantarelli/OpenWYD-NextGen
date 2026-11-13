# GAME-14 — Fadas: bônus de drop/EXP, duração e auto-coleta

> **ID canônico:** `GAME-14`
> **ID histórico:** `FASE-03-14`
> **Estado 2026-11-12:** ⬜ · ver `roadmap-rust/index.md`.

**Fase:** 03 (core/gameplay) · **Depende de:** GAME-11 (drop no inventário + caixa); GAME-12 ✅
(teto de pilha por item) · **Dossiê:** `docs/wyd/sistemas/fadas.md`

## 1. Contexto

Fada é item do **slot Familiar (`Equip[13]`)** que modifica drop/EXP enquanto equipada, com prazo
por item. Prateada e Dourada **automatizam a coleta**: agrupam aglutináveis e vendem os não
agrupáveis. Hoje o servidor só aplica o bônus da gema de diamante (+8%); fada não existe.

Provas já levantadas (não repetir a pesquisa):

- `Equip[13]` = slot Familiar — legado: `Equip[13].Index == 769` (Nyerds) e
  `SendItem(cId, EQUIP, 13, Item)` no timer de fada;
- faixa de índices de fada no legado: **3900–3908**, com contador em `EFV1/EFV2/EFV3` decrescente e
  **destruição** ao expirar (`AmountMinus`);
- bônus por fada (**B**, guia oficial): azul 32% drop (80 NP), verde 16% EXP (120 NP), vermelha
  32% drop + 16% EXP (200 NP), prateada = vermelha + Flash Prateado, dourada 50% drop + 18% EXP
  (+ Flash Dourado); duração 3 dias, só correndo equipada;
- teto de agrupamento **120 por slot** confirmado por servidor oficial (WYD Global, 2026).

## 2. Escopo

1. **Constantes de bônus como configuração** (`game_setting`/`config_rate`), nunca hard-code:
   `fairy_drop_bps`, `fairy_exp_bps`, por índice de item de fada.
2. **Aplicação no abate**: roll por item → bônus de fada/gema sobre a taxa com teto na escala →
   depósito → auto-coleta.
3. **Duração**: contador no item (3 bytes do legado ou data de expiração), decremento só com o
   jogador em jogo e a fada equipada; expirar **destrói**.
4. **Prateada/Dourada**: lista de itens agrupáveis por fada (tabela de servidor, não regra do
   item); não agrupável → **auto-venda** com log `event=autosell` (item, qty, gold), nunca
   silenciosa.
5. **Ordem com a caixa**: item que sobra e é agrupável pela fada não vira caixa; item fora da
   lista segue a regra de 30 s / 3 min.

## 3. Fora de escopo

- Pet/combate: fada não tem HP nem ataca.
- Lista completa de itens agrupáveis da Dourada: publicada só parcialmente (**C**).

## 4. Critério de aceite

- [ ] Teste: mesma seed e mesma tabela de drop, com e sem fada, resulta em taxa efetiva diferente
      e **teto na escala** (1.000.000).
- [ ] Teste: fada expirada é destruída e deixa de contar no bônus.
- [ ] Teste: item agrupável da lista não vira caixa com a fada; item fora da lista vira.
- [ ] Teste: auto-venda gera linha `event=autosell` com gold creditado.
- [ ] `cargo fmt --check`, `clippy -D warnings`, `cargo test --workspace` em zero.

## 5. UNVERIFIED

| Ponto | Motivo |
|---|---|
| índice exato de cada fada (azul/verde/vermelha/dourada) | legado só prova a faixa 3900–3908 |
| bônus da Dourada em 2026 | números são de guia 2009–2011 |
| prazo corre com o jogador morto? | sem fonte |
| lista completa de agrupáveis por fada | só a da Prateada está publicada |

## Skills relacionadas

- `wyd-rust-core` — buffs e auto-coleta no tick.
- `wyd-rust-sim` — duração e bônus de drop/EXP.
- `rust-code-quality` / `rust-test-quality` — determinismo e testes.
