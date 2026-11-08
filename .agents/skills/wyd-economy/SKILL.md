---
name: wyd-economy
description: Balanceia economia e progressão do WYD (preços, drops, refino, combines, EXP) com dados versionados e simulação. Use para mudar preço/drop/EXP, criar receita ou investigar item que não refina, não cai ou não equipa.
---

# Economia e progressão (Rust)

> Sucessão 09-2: mantido o nome; fonte de dados migrada para Postgres + `wyd-sim`. Fonte: `crates/wyd-db/`, `crates/wyd-sim/`, `docs/arquitetura/modelo-banco.md`.

## Regras

1. **Valor vem de dado, não de código.** `item_catalog`/`skill_catalog`/`npc_template` no banco (seeds `DEV-ONLY`); taxas e drops versionados; sem catálogo no banco o mundo sobe sem NPCs.
2. **Valide com simulação antes de tocar a engine.** Fórmulas puras em `crates/wyd-sim/src/formula/` (ordem base→atrib→equip→skill→buff→crit→defesa); cenários reproduzíveis + goldens (`cargo test -p wyd-sim`).
3. **Ordem e paridade são contrato.** RNG/tick determinísticos; chamada nova no meio de caminho coberto por golden muda tudo — mude o teste com motivo, nunca remova asserção.
4. **Cliente precisa conhecer o item.** Item novo no servidor sem entrada em `ItemList.bin`/`itemname.bin` fica invisível; preço ausente = linha ausente, não bug de código.
5. **UNVERIFIED não calibra.** Curva celestial, coefs por classe×arma e DoT sem tabela canônica não viram verdade de balanceamento.

## Rotina

1. Meça (EXP/min, gold/min, itens/min) com harness/bot dedicado.
2. Mude dado/flag; fórmula só com teste table-driven.
3. `cargo test --workspace` + goldens de `loot`/`combat`/`refine`.
4. Registre antes/depois e motivo na issue (não em doc legado).
