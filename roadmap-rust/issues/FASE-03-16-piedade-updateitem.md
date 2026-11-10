# FASE-03-16 — Piedade de refino persistida + `UpdateItem` de resposta

**Fase:** 3 · **Depende de:** FASE-08-2 ✅ (fio) · **Bloqueia:** FASE-10 (fechar taxas) · **Esforço:** M

## Objetivo

Persistir a piedade (pity) de refino por item e responder o refino com o pacote
de atualização visível ao cliente, sem mudar nenhuma taxa vigente.

## Contexto / evidência

Pesquisa `docs/wyd/pesquisa-wyd2-missao-fase3-2026-11-10.md` item 6 (26 campos).
O caminho principal já está no fio (`Opcode::UseItem` → `SelectionStore::use_item`
→ `Gameplay::use_item` → `Gameplay::refine`, SancRate viva por tentativa,
`MissingRate` sem mutação, prova de recarga em `tests/config_gameplay.rs`).
Faltam: piedade por item (hoje zerada por chamada: `pity: 0`,
`crates/wyd-server/src/gameplay.rs:460`) e o pacote de resposta (`UpdateItem`;
o cliente só redesenha o item no relogin).

Fonte oficial de época: `docs/wyd/ongame-guia.md` (Guia OnGame via Wayback,
Ori até +6/100% até +2, Lac até +9/100% até +4, jóias +4/+8/+15/+32%,
+10/+11 e gems) — confrontar com o cliente 7662 antes de virar regra.

Regras do legado (citadas, não copiadas sem golden):
`SancRate` indexa `sanc+1`; pity soma `{5,5,5,5,4,4,3,3,2,1,0}` ao próximo nível;
acúmulo em 3/4 dos fails (`rand()%4<=2`), Lac sempre, Ori só até o teto
(`_MSG_UseItem.cpp:943`, via `game-rules.md` §3.3); travas Ori +6 / Lac +9 /
selado +9 (destrói); `EF_NOSANC` (126) bloqueia; `Lactolerium_100` (4141) força a
linha do Âmago.

## Passos

1. Persistir a piedade por item (nova coluna/tabela, migration versionada):
   ler na abertura do `use_item`, aplicar no cálculo da taxa, gravar no sucesso
   (zera) e na falha qualificada (acumula 3/4).
2. Emitir o pacote de atualização do item refinado ao final do `use_item`
   (reaproveitar `reply_with_body`, como `CreateMob`).
3. Subpasso B16: `itemAbilityRefined` completo (`(sanc+10)/10`, promoção
   `sanc==9→10` em acessório, lista de isenção, clamp `EF_RUNSPEED`,
   thresholds `+25 AC`/`+40 dano` por cima) com goldens.
4. Testes: taxa por passo antes/depois da recarga; pity acumula/zera;
   `MissingRate` sem mutação; travas +6/+9/selado; `EF_NOSANC`.

## Arquivos

- Criados: migration da piedade, `crates/wyd-server/tests/refino_piedade.rs`.
- Alterados: `crates/wyd-server/src/gameplay.rs`, `crates/wyd-sim/src/formula/refine.rs`,
  `crates/wyd-protocol/src/payload/*` (só o já provado), `docs/rust/08-checklist-funcionalidades.md` (5.4).
- Proibido: mudar qualquer taxa sem golden + captura; inventar layout.

## Testes / verificação

```bash
cargo test -p wyd-server --test refino_piedade
cargo test -p wyd-sim
```

Saída esperada: tudo `ok`; recarga SancRate muda o resultado sem restart.

## Skills Relacionadas do freebuff

- .agents/skills/wyd-rust-core — autoridade do inventário.
- .agents/skills/wyd-rust-sim — goldens de taxa/piedade.
- .agents/skills/wyd-rust-db — migration da piedade.
- .agents/skills/wyd-rust-testing — harness de fio.

## Documentação de referencia

- docs/wyd/pesquisa-wyd2-missao-fase3-2026-11-10.md (item 6).
- docs/wyd/sistemas/refino.md + `refino-auditoria.md`.
- `docs/migration/game-rules.md` §3 (legado `Jean1dev/w2pp-OpenWYD@main`).

## Subtasks

- [x] Codificar issue
- [x] Testar issue
- [x] Revisar issue
- [x] Documentar issue
- [ ] Versionar issue, mergear branch e fechar issue
- [ ] Limpar branch (se aplicável)

## Correção de rota (prova no fio)

O pacote de resposta **não** é `UpdateItem` (0x0374 = porta/door no legado)
— é **`SendItem` (0x0182)**, 24 B: invType u16+12, slot u16+14, item[8]+16.
Prova: RE do cliente 7662 (handler `0x182`) + `EncodeSendItemBody` do porte do
mesmo alvo. Na recusa do refino nada é reenviado (cliente já mostra o estado
correto; reenviar o source seria comportamento inventado).

## Evidência (2026-11-10, sem commit)

- `cargo test --workspace --offline`: exit 0, 69 suítes `ok`, 0 falhas.
- `cargo clippy --workspace --offline --all-targets -- -D warnings`: exit 0.
- `cargo fmt --all -- --check`: limpo. `migrations-check`: 43/84 ok.
- Banco real (Postgres 17 da stack, rede `wyd`):
  `tests/refino_piedade.rs` **5/5** (falha acumula piedade 1 e o par sanc no
  fio carrega `4 + 10×1`; sucesso sobe +5 e zera; `EF_NOSANC` veta sem mutação;
  poda de piedade órfã; layout do `SendItem` byte a byte) +
  `tests/pg.rs` **8/8** (sem regressão) + `tests/catalogo.rs` **3/3**.
- Piedade em tabela própria `refine_pity(character_id, slot)` (migration
  `0043`); `replace_items` grava itens + piedade na mesma transação e poda
  slots fora do inventário; B16 restrito ao encoding do par EF_SANC no fio
  (estatística +25AC/+40dmg fica como resíduo FASE-10).

## Critérios de aceite

- [x] Piedade persiste entre tentativas e sessões.
- [x] Cliente vê o resultado do refino sem relogar.
- [x] Nenhuma taxa mudou sem golden + evidência.
- [x] P-10 (5% vs 4% no +12) registrada, não decidida no código.

## Riscos / rollback

- Risco: migrar taxa errada e quebrar economia → mitigação: goldens travam cada passo.
- Rollback: coluna nova ignorada; comportamento volta ao `pity: 0` atual.
