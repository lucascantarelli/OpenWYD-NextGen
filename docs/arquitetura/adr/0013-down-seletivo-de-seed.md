# ADR-0013 — `down` de seed é seletivo e restaura o estado anterior da fonte

- **Estado:** aceita (2026-11-08)
- **Data:** 2026-11-08
- **Decisores:** dono do projeto + agente (missão "tornar a `0021` genuinamente reversível")
- **Governa:** `crates/wyd-db/migrations/0021_seed_skill_lineage.sql`,
  `tools/content/gen-skill-lineage.mjs`
- **Depende de:** ADR-0009 (conteúdo estático), `docs/arquitetura/modelo-banco.md` §4

## Contexto

`0019_seed_skill_catalog.sql` publica as 151 skills do cliente e já classifica 12 delas
(`rule:name_confirmed`). `0021_seed_skill_lineage.sql` é gerada depois e classifica 63 skills —
6 das quais **são as mesmas linhas da 0019**, com valor curado diferente (ex.: `Clod_Attack` sai
de `magic_sword` e entra em `faith`; `Another_Change` sai de `passive` e entra em `buff`).

O `down` original da 0021 zerava a classificação de todas as 63 linhas
(`class_code = NULL`, `classification_source = 'unmapped'`). Consequência: reverter **só** a 0021
apagava a classificação que a 0019 tinha publicado — perda de dado herdado, não reversão. O
`migrate-cycle` não pegava porque ele desce **todas** as migrações: a 0019 também desce e a
tabela fica vazia por outro motivo. A limitação estava declarada no cabeçalho gerado e em
`docs/wyd/classes.md` em vez de corrigida.

## Decisão

Adotamos **`down` seletivo que restaura o estado anterior**, porque a migração precisa ser
reversível sozinha — um `down` que destrói dado de outra migração não é reversão, é efeito
colateral documentado.

Regras que decorrem dela:

1. O `down` restaura as colunas que o `up` escreve (`class_code`, `lineage_code`, `skill_role`,
   `classification_source`, `confidence`) ao valor que a **fonte** (`0019`) publicou — nunca a
   `primary_system`, que esta migração não toca.
2. As linhas que a 0021 classificou pela primeira vez voltam a `'unmapped'` com `NULL`.
3. Os valores restaurados são **lidos do seed pelo gerador** (`tools/content/gen-skill-lineage.mjs`
   parseia as tuplas da 0019), não copiados à mão: mudar o seed e não regerar a 0021 quebra o
   `check-migrations`/ciclo, em vez de deixar o `down` desatualizado em silêncio.
4. Nenhuma linha é apagada no `down` de seed: `DELETE` só existe onde a migração criou a linha
   (ex.: `0024`).

## Consequências

- **Positivas:** reverter a 0021 sozinha devolve exatamente o estado anterior; o conflito
  seed × curadoria (`Clod_Attack`, `Another_Change`) volta ao valor do seed no `down`, o que é a
  definição correta de reverter; o ciclo up→down→up reproduz o mesmo estado.
- **Negativas:** o arquivo gerado carrega a lista de restauração (6 tuplas), então ele cresce com
  a curadoria. É aceitável: é dado derivado da fonte e provado pelo próprio gate.
- **Invariantes afetados:** nenhum código Rust; o contrato de migração fica mais forte
  (`down` sempre devolve o estado anterior, nunca o de outra migração).

## Alternativas consideradas

| Alternativa | Por que foi descartada |
|---|---|
| Publicar uma migração corretiva (`0025`) que "recompõe" o seed | a 0021 ainda não foi publicada (árvore local, sem release); corrigir a geradora é o caminho mais barato e não deixa migração fantasma |
| Tabela de rastreio (`seed_lineage_origin`) | cria tabela órfã só para reversão; o estado anterior já existe na fonte |
| `DELETE` das 63 linhas no `down` | apagaria linhas da 0019 junto (a 0021 não as criou) |
| Manter a limitação documentada | o requisito é reversibilidade real; documentar perda de dado não é aceitável quando a correção custa uma tabela de `VALUES` |

## Rastreabilidade

- Issue consumidora: `roadmap-rust/changelog.md` (PROC-09, concluída; migrations/seed).
- Docs atualizadas: `docs/wyd/classes.md` (fim da limitação), `docs/wyd/decisoes-servidor.md`
  (números e `down` seletivo), `docs/rust/03-progresso.md` (entrada datada),
  `crates/wyd-db/migrations/README.md` (regra do `down` de seed).
- Evidência: cabeçalho gerado da `0021` + `migrate-cycle` (24 migrations, up/down/up) +
  prova de restauração no Postgres (6 herdadas voltam ao seed, 57 novas ficam `unmapped`,
  nenhuma linha fora do conjunto muda).

## Revisão

- Revisar quando: a 0024 (seed da Kersef) precisar de `down` que devolva estado de outra
  migração, ou quando o `migrate-cycle` passar a comparar conteúdo (não só contagens).
- Sinais de que envelheceu: `down` de seed voltando a usar `DELETE`/`UPDATE` amplo; curadoria
  mudando o seed sem regerar a 0021.
