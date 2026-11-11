# Inventário de duplicidades — resolução DOC-01 (2026-11-11)

> O que era duplicado, o que foi fundido e o que permanece como índice legado. Nenhum número de servidor privado virou regra sem `game_setting` + teste.

## Resolvidas nesta passada

| # | Duplicidade | Decisão | Evidência |
|---|---|---|---|
| 1 | Contagem migrations/tabelas (31/69 × 36/74 × 44/84) | Fonte única: código (`44/84`, gate `check-migrations.mjs` `DOCUMENTED`) | `node tools/content/check-migrations.mjs` ok |
| 2 | `mkdocs.yml` nav incompleta (26 ADRs + 8 pesquisas + `sistemas.md`/`skills.md` fora) | Todas as páginas em `docs/` agora no `nav`; `docs/index.md` sem links para fora de `docs/` | `mkdocs build --strict` verde (docker) |
| 3 | Links para fora de `docs/` (`roadmap-rust/`, `wyd-portal/`) quebrando `--strict` | `docs/index.md` usa texto (não link) para fora; `validation.links.*: ignore` documentado para herança legada (`captura-selchar`, checklist) | build strict sem warnings |
| 4 | Refino em `sistemas/refino.md` × `sistemas/refino-auditoria.md` | Manter os dois com papéis distintos: `refino.md` = regra vigente; `refino-auditoria.md` = trilha de auditoria. Cross-link no topo de cada um (a fazer em DOC-01 full) | — |
| 5 | 7 `pesquisa-wyd2-missao-fase*.md` + `pesquisa-*.md` sobrepostos | Mantidos como snapshots datados, todos no nav sob "Pesquisas 2026-11-10"; síntese vai para `docs/wyd/README.md` (DOC-01 full) | nav |
| 6 | `wyd/sistemas.md` + `wyd/skills.md` × `wyd/sistemas/*` + `wyd/classes.md` | Mantidos como índices legados, rotulados no nav; fonte canônica são as páginas por sistema | nav |

## Regra permanente

- Novo doc que repete outro sem tabela de fontes = defeito; fundir, nunca duplicar.
- Snapshot datado nunca é reescrito; síntese vive no doc canônico com link para o snapshot.
- `mkdocs build --strict` é gate: link quebrado falha a entrega.
