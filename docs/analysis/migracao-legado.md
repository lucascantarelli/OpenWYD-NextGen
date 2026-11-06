# Migração do legado C++ (`Source/`) para Go — mapa e limites

Base: 198 arquivos em `Source/` (99 `.cpp`, 44 `.h`, 12 projetos Visual Studio), conferidos contra o
código Go e os documentos de engenharia reversa (`docs/migration/`, 58/58 handlers, `docs/agents/`).

## 1. Inventário por projeto e estado da migração

| Projeto C++ | Arquivos | O que fazia | Estado no Go | Evidência |
|-------------|---------:|-------------|--------------|-----------|
| `TMSrv` | 87 | servidor do jogo: CPSock, mundo, mobs, itens, guild, guerra, castelo | **portado** (exceto detalhes marcados UNVERIFIED) | `tmserver/internal/{world,handler,protocol,route,rng}` + 21 arquivos de contrato em `docs/migration/handlers/` |
| `DBSrv` | 12 | persistência em arquivos `account/<Key>/<NOME>` | **substituído** por Postgres (`dbserver` + `internal/store`), com conversor one-shot | 46 tabelas, 49 `.up.sql`, `dbserver convert` |
| `BISrv` | 7 | billing (`_AUTH_GAME`, 196 B, layout UNVERIFIED) | **reescrito do zero** (`binserver`, allow-all por padrão) | `api/bin/v1`, `binserver/` (5 arquivos) |
| `ClientPatch_v7662` | 5 | patch do cliente: checksum off, força-gráficos, `SkillDelay/4`, título | **não portável** (roda dentro do cliente Windows) | `docs/migration/client-patch-hooks.md`; `client/ClientPatch.dll` |
| `DropTool` (`Source/Code/_legacy/DropTool/`) | 2 | gera `ItemDropList.txt` (relatório inverso) | **substituído** por `/drops` e `/drops/carry` | issue 34, `DropAdminService` |
| `ExpTool` | 2 | recalcula EXP dos templates pela curva | **substituído** por `/mobs` (stats) e pela curva em `game-rules.md` | issue 25 |
| `NPTool` | 7 | edita `NPCGener.txt` (spawn) | **substituído** por `/npcs` (definições) — spawn ainda é arquivo (issue 37) | issues 23, 37 |
| `EDITAPPMOB` | 5 | edita template de mob (816 B) | **substituído** por `/mobs/[template]` (stats + `Equip[]`) | issues 25, 34 |
| `EDITAPPSHOP` | 5 | edita loja de NPC | **substituído** por `/npcs/[id]` | issue 23 |
| `SearchPass` | 2 | busca conta/senha no dump | **irrelevante** (senha só como argon2id; sem texto plano) | `internal/secret` |
| `ZerarSkill` | 2 | zera skills de um personagem | **substituído** por `/personagens/[id]` (leitura) + concessão por fila | issue 43 |
| `AttributeMap_Editor` | 1 | edita `AttributeMap.dat` | **substituído** por `/mapa` (transformação + download) | issue 27 |

**Conclusão:** nenhuma linha de C++ é necessária para o servidor rodar. O que resta são
*referências de comportamento* usadas para tirar dúvidas de paridade.

## 2. Dependências que ainda existem (todas externas ao código)

| Dependência | Por que existe | Dá para eliminar? |
|-------------|----------------|-------------------|
| Cliente `WYD.exe` fechado | é o produto final; define o protocolo | **Não** — só substituindo o cliente (ver [cliente-launcher.md](cliente-launcher.md)) |
| `ClientPatch_v7662.dll` | desliga checksum, ajusta `SkillDelay`, força gráficos | Não sem reescrever o patch; é do cliente |
| Assets do cliente (`.wys`, `.wyt`, `.trn`, `itemicon.bin`) | renderização e ícones | Não; são do cliente (o portal só referencia ícones extraídos) |
| Layouts MSVC x86 (`STRUCT_MOB`=816, `ACCOUNTFILE`=7952, `MOBEXTRA`=552, `QUEST`=56, `ITEM`=8) | save e wire legados | Não — mas já estão isolados em codecs Go explícitos |
| CSC/Basedef/GetFunc do legado | fórmulas e ordem de RNG | Não como código; como **documento** (`game-rules.md`, `parity-tests.md`) |

## 3. O que falta migrar (ordem sugerida)

1. **Spawn de mobs** (`NPCGener.txt` → banco/overlay) — issue 37 já planejada; fecha o último
   grande arquivo de conteúdo editável fora do banco.
2. **Quests** — o motor existe em parte (handler + `Quest`), mas conteúdo e NPCs de quest são
   data-driven e ainda não têm editor; ver issue 44/45 e [qa-ingame.md](qa-ingame.md).
3. **Ferramentas restantes** — nenhuma. O trabalho é **consolidar** o que o portal já substitui e
   remover a duplicação (issue 56).
4. **Paridade pendente**: coeficientes Dex/Str por classe×arma, árvore de `DoubleCritical`/`ParryRate`
   e `_AUTH_GAME` (marcados UNVERIFIED) — fechar por golden cases, não por leitura de código.

## 4. Limites reais da migração completa

| Limite | Natureza | Consequência |
|--------|----------|--------------|
| Protocolo CPSock e pacotes S→C | cliente fixo | o Go continuará "preso" ao wire; é intencional (`migration-plan.md` §3.5) |
| Save layouts por offset | dados históricos + cliente | codecs explícitos permanecem; nunca usar struct nativa |
| Assets e engine do cliente | propriedade de terceiros | modernização gráfica exige patch/recompilar (C++) ou port WASM |
| Paridade de RNG | determinismo do MSVC | qualquer `rand()` fora de ordem muda drops/refino; testes de paridade são obrigatórios |
| Distribuição legal do cliente | jurídico | o repo não distribui instalador; launcher deve apontar para fonte do usuário |

## 5. Checklist para declarar o legado "arquivado"

- [ ] `git grep -i "Source/"` sem referências de build (hoje só docs).
- [ ] Nenhuma ferramenta operacional depende de `.exe` em `Release/` (`DropTool`, `ExpTool`…).
- [ ] Spawn/quedas/catálogos editáveis **somente** pelo portal.
- [ ] `docs/migration/` cobre 100% dos handlers com evidência (hoje 58/58).
- [ ] `Source/` movido para `archive/` ou fora do checkout padrão, com nota no README.

## 6. Arquivamento das ferramentas substituídas (05/11/2026)

As sete ferramentas que a §1 marca como substituídas saíram de `Source/Code/` e foram para
`Source/Code/_legacy/` (mesmo destino do `DropTool`, arquivado pela issue 61), com um
`README` em `Source/Code/_legacy/` que aponta o substituto de cada uma:

| De | Para | Por quê |
|----|------|---------|
| `ExpTool/` | `_legacy/ExpTool/` | alvo (`npc/`) não existe mais; curva vive em `tmserver/internal/exptool` |
| `NPTool/` | `_legacy/NPTool/` | spawn agora é `npc_definition` (`/npcs`); `NPCGener.txt` só alimenta o importador |
| `EDITAPPMOB/` | `_legacy/EDITAPPMOB/` | `/mobs/[template]` |
| `EDITAPPSHOP/` | `_legacy/EDITAPPSHOP/` | `/npcs/[id]` |
| `SearchPass/` | `_legacy/SearchPass/` | senha só existe como argon2id; não há texto plano para buscar |
| `ZerarSkill/` | `_legacy/ZerarSkill/` | `/personagens/[id]` + `delivery_queue` |
| `AttributeMap_Editor/` | `_legacy/AttributeMap_Editor/` | `/mapa` + `/api/mapa/download` |

Continuam em `Source/Code/`, como referência de wire/layout/fórmula citada por `docs/migration/`:
`TMSrv/`, `DBSrv/`, `BISrv/`, `Basedef.*`, `CPSock.*`, `ItemEffect.h`. `ClientPatch_v7662/` fica
porque é dependência do **cliente** (REVIEW-07). Remoção definitiva de `_legacy/` quando a paridade
UNVERIFIED fechar; até lá o custo é só o tamanho do checkout (~330 KB).
