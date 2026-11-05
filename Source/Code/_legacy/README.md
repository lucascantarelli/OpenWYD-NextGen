# `_legacy/` — ferramentas C++ substituídas

Nada aqui entra em build, imagem Docker ou runtime. Estas pastas são **referência de comportamento**,
mantidas só enquanto alguém precisar comparar uma decisão do Go/portal com o que a ferramenta Win32
fazia, e são candidatas naturais a sair do repositório (critério em
[`docs/analysis/migracao-legado.md`](../../../docs/analysis/migracao-legado.md) §5).

Cada ferramenta aqui opera num alvo que **não existe mais** no runtime atual: arquivos de `npc/`,
`ItemList.csv`, dumps de conta em arquivo ou `NPCGener.txt` como fonte viva. Ou seja, elas não são
apenas substituídas — são inexecutáveis contra o estado atual (Postgres + portal).

## O que está arquivado e por quem foi substituído

| Pasta | O que fazia | Alvo antigo | Substituto hoje | Evidência |
|-------|-------------|-------------|-----------------|-----------|
| `DropTool/` | gerava `ItemDropList.txt` / `MobDropList.txt` (relatório inverso item↔mob) | `npc/` + `ItemList.csv` | `/drops` e `/drops/carry` (`DropAdminService`) | issue 26 ✅, issue 61 (arquivado em 05/11) |
| `ExpTool/` | recalculava a EXP dos templates pela curva | `npc/` | `tmserver/internal/exptool` (curva reimplementada) + `/mobs` para stats | issue 25, `game-rules.md` |
| `NPTool/` | editava `NPCGener.txt` (posição/tipo dos spawns) | `NPCGener.txt` | `/npcs` sobre `npc_definition`; o arquivo virou **insumo do importador** do dbserver | issues 23 ✅, 57 ✅ |
| `EDITAPPMOB/` | editava o template de mob (816 B) | `npc/` | `/mobs/[template]` (stats + `Equip[]`) | issue 25 |
| `EDITAPPSHOP/` | editava a loja de um mercador | `npc/` | `/npcs/[id]` (`Carry[]` da loja) | issue 23 ✅ |
| `SearchPass/` | buscava conta/senha em texto plano no dump | `account/<Key>/<NOME>` | senha só existe como **argon2id** (`internal/secret`); não há o que buscar | `docs/analysis/migracao-legado.md` |
| `ZerarSkill/` | zerava as skills de um personagem | dump de conta | `/personagens/[id]` (leitura) + concessão por `delivery_queue` | issues 43 🟡, 57 |
| `AttributeMap_Editor/` | editava `AttributeMap.dat` célula a célula | `AttributeMap.dat` | `/mapa` (histograma + transformação) + `/api/mapa/download` | issue 27 ✅ |

## O que **não** está aqui (e por quê)

- `TMSrv/`, `DBSrv/`, `BISrv/`, `Basedef.*`, `CPSock.*`, `ItemEffect.h`: são a **referência de wire,
  layout e fórmula** citada por `docs/migration/*` (`arquivo.cpp:linha`). Continuam em `Source/Code/`
  até a paridade UNVERIFIED fechar (coeficientes Dex/Str, `DoubleCritical`/`ParryRate`, `_AUTH_GAME`).
- `ClientPatch_v7662/`: é dependência **do cliente** (gera a DLL injetada no `WYD.exe`), não do
  servidor; sai só quando o cliente for substituído (REVIEW-07).

## Citação de caminho

Docs antigos com data no nome (`docs/agents/*-2026-06-19*`) citam os caminhos originais
(`Source/Code/ExpTool/...`); eles são **retratos** daquela data e não foram reescritos. A
documentação viva, o código Go e o proto apontam para `Source/Code/_legacy/<Pasta>/`.
