---
name: wyd-economy
description: Balanceia economia e progressão do WYD (preços, drops, refino, combines, EXP) com dados versionados e simulação. Use para mudar preço/drop/EXP, criar receita ou investigar item que não refina, não cai ou não equipa.
---

# Economia e progressão (Rust)

Norma: `docs/wyd/economia.md`, `docs/wyd/itens.md`, `docs/wyd/sistemas/refino.md`,
`docs/arquitetura/matematica-sim.md` e `docs/arquitetura/modelo-banco.md`.
Taxas, receitas, ordem de fórmula e tabelas de catálogo vivem nesses docs.

## Decisões específicas

- Valor vem de dado versionado (`item_catalog`, `skill_catalog`, `npc_template`, seeds
  `DEV-ONLY`), não de constante enterrada no código. Sem catálogo no banco o mundo sobe
  sem NPCs.
- Valide a mudança em `wyd-sim`/golden antes de tocar a engine; fórmula só muda com
  table-driven test e motivo.
- Item novo precisa existir também no cliente (`ItemList.bin`/`itemname.bin`); preço
  ausente é linha ausente, não bug de código.
- UNVERIFIED (curva celestial, coefs classe×arma, DoT sem tabela) não calibra economia.
- Registre antes/depois e a fonte do número na issue; não crie doc legado paralelo.

## Prova

Simulação/goldens de `loot`/`combat`/`refine` com `cargo test -p wyd-sim`; depois
`cargo test --workspace` e `make check`.
