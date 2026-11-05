# Issue 60 — Mapas (`HeightMap.dat` / `AttributeMap.dat`): banco ou asset?

**Fase:** 7 · **Depende de:** 58 · **Esforço:** M · **Estado:** ⏳ proposta (decisão pendente)

## O que são

Não são configuração: são **rasters espaciais** do mundo, binários e densos.

| Arquivo | Dimensão | Bytes | O que cada byte significa |
|---|---|---|---|
| `HeightMap.dat` | 4096 × 4096 | 16 MiB | altura/terreno por tile. `route.Bake` grava `127` (= `Blocked`) onde o AttributeMap proíbe passagem |
| `AttributeMap.dat` | 1024 × 1024 | 1 MiB | um byte por bloco 4×4 tiles: `0x02` = bloqueado (`route.Bake`), `0x80` = Campo de Treino (issue #383), `0x40`/outros = zona PvP, proibição de invocação etc. Bits além desses são **UNVERIFIED** (`data-formats.md` §2) |

O tmServer carrega os dois no boot, faz o bake uma vez (`walkability grid baked dim=4096`) e mantém a
grade densa em memória — o pathfinding lê célula a célula.

## Dá para migrar? Sim, com trade-off

| Opção | Como | Ganho | Custo |
|---|---|---|---|
| **A. Manter em arquivo** (status quo) | `content.LoadGrid` | zero mudança; 17 MiB lidos do disco em ~ms | fora do banco; o "tudo centralizado" não acontece |
| **B. `map_raster` no banco** | `CREATE TABLE map_raster(kind text PRIMARY KEY, dim int NOT NULL, sha256 text NOT NULL, data bytea NOT NULL)` + seed dos arquivos | fonte única, `sha256` detecta drift entre ambientes, portal pode validar/visualizar | 17 MiB por boot trafegados no gRPC do dbserver; backup/WAL maiores; continua sendo blob opaco (não é editável célula a célula) |
| **C. Derivar tabelas esparsas** | `map_block(x1,y1,x2,y2)` a partir do bake | consultável, leve, o portal desenha zonas | perde altura por tile e os bits não confirmados; bake continua necessário |

## Recomendação

**B, se o objetivo é centralizar; A se o objetivo é performance.** A decisão deve ser medida, não
opinada: implementar B atrás de uma flag de boot (`-maps-from-db`) e medir `walkability grid baked` e
o tempo até `tmserver listening` com o arquivo e com o blob. Adotar B se o delta for < 1 s (o boot já
gasta ~2,5 min aplicando 6.099 NPCs, então 17 MiB é ruído), e então apagar os `.dat` do conteúdo.

O que **não** vale a pena: C sozinho. O servidor precisa da grade densa em memória de qualquer forma;
C só serve como modelo de apresentação, e o `worldatlas` já resolve isso com o PNG derivado.

## Plano de execução (se B)

1. `0044_map_raster.up.sql` (tabela) / `0045_map_raster_seed.up.sql` (`bytea` gerado de
   `cmd/seedgen`, ~17 MiB — avaliar `COPY ... FROM PROGRAM` ou arquivo `.sql.gz` fora do embed).
2. `dbclient` ganha `MapRaster(ctx, kind)` e o tmServer usa quando `-maps-from-db`; fallback para
   `content.LoadGrid` enquanto a flag existir.
3. Gate: `walkability grid baked dim=4096` idêntico e `sha256` da grade em memória igual nos dois
   caminhos (teste de paridade no `route`).
4. Fase 5 da issue 58 remove os `.dat` do runtime; o seed passa a ser a única fonte.

## Critérios de aceite

- [ ] Medição registrada: boot com arquivo vs. com `map_raster` (tempo e memória)
- [ ] `sha256` da grade em memória idêntico nos dois caminhos (4096² e 1024²)
- [ ] Se B for adotado: `.dat` sai do conteúdo e o boot sobe sem eles
- [ ] `gameconfig`/`AttributeMap` bits novos documentados à medida que forem confirmados

## Riscos

- **17 MiB no embed do binário:** empurra o tamanho da imagem e do gRPC; por isso o seed pode ficar
  fora do `//go:embed` (arquivo em `internal/migrations/seeds/`, aplicado pelo `db-baseline.sh`).
- **Bits UNVERIFIED:** migrar o raster não valida semântica; qualquer bit novo continua precisando de
  captura (`docs/migration/data-formats.md`).
