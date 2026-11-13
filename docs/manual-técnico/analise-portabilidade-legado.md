# Análise de portabilidade do legado OpenWYD → Rust

> **Data:** 2026-11-08 · **Método:** leitura de `pre-release-rust.md`, `start-rust.md`, `docs/`,
> `AGENTS.md` e skills `wyd-rust-*`; medição do legado em `origin/main` (somente leitura) e do
> workspace Rust atual; conferência dos dois clientes (`client/` e o cliente real do WYD Global).
> **Nada de código foi alterado nesta análise.** Número marcado **(medido)** foi obtido pelos
> comandos do §7 nesta data; o resto é leitura de documento existente ou **hipótese** declarada.

## 1. Por que esta análise existe

O planejamento Rust (`roadmap-rust/`) já cobre processo, arquitetura, banco e protocolo. O que
faltava era uma **medida do tamanho do legado** e um **de-para explícito** entre o que o legado
executava e o que o workspace Rust executa hoje. Sem isso, "portar o OpenWYD" não tem escopo:
ninguém sabe se falta 20% ou 80%, nem qual peça bloqueia o caminho ponta-a-ponta.

## 2. Inventário medido do legado (`origin/main`)

`origin/main` preserva o projeto anterior completo (C++ + Go + docs). `develop` é a árvore Rust.
O legado é **fonte de conhecimento**, nunca de código (`AGENTS.md` §0).

| Pasta | Arquivos (medido) | Papel |
|---|---:|---|
| `Source/` | 142 | C++ original: `TMSrv` (91), `DBSrv` (16), `BISrv` (10), `ClientPatch_v7662` (9), `CPSock`, `Basedef`, `ItemEffect.h` |
| `tmserver/` | 449 | servidor Go (reescrita anterior): `internal/handler` 172, `internal/world` 55, `content/TMsrv` 46, `internal/protocol` 34 |
| `internal/` | 125 | Go compartilhado + `migrations/` (22 `.sql`, **18 MB**) |
| `dbserver/` `binserver/` | 28 + 7 | persistência e billing Go |
| `webserver/` `api/` `portal/` | 94 + 12 + 186 | plataforma web anterior (Go + Next.js) |
| `roadmap/` | 138 | planejamento Go (histórico, auditado em `auditoria-roadmap.md`) |
| `docs/` | 226 | conhecimento: `migration` 79, `archive` 72, `agents` 16, `analysis` 15, `sistemas` 13, `integrations` 9, `adr` 8, `guias` 4, `lore` 3 |

**Volume de código (medido):**

| Fonte | Arquivos | Linhas |
|---|---:|---:|
| C++ (`Source/**/*.cpp|*.h`) | 116 | **70.987** |
| Go (`tmserver/` + `internal/`, total) | 484 | **99.606** |
| Go sem testes | 484 − `_test.go` | **51.582** |
| Rust hoje (`crates/*/src/*.rs`, 7 crates) | 112 | **18.242** |

Os maiores arquivos C++ mostram onde mora a regra de jogo: `TMSrv/Server.cpp` (10.553),
`Basedef.cpp` (7.468), `_MSG_UseItem.cpp` (6.686), `MobKilled.cpp` (3.506),
`ProcessSecMinTimer.cpp` (2.823), `_MSG_Quest.cpp` (2.753), `DBSrv/CFileDB.cpp` (2.712),
`GetFunc.cpp` (2.286), `SendFunc.cpp` (2.145). `Basedef.h` (2.920 linhas) declara **147 structs**
(`STRUCT_ITEM`, `STRUCT_MOB`, `STRUCT_AFFECT`, `STRUCT_QUESTS`, `STRUCT_SELCHAR`…).
O `TMSrv` tem **58 arquivos `_MSG_*.cpp`** — um por handler de cliente —, e o Go anterior
reimplementou **72 handlers** (`internal/handler`, sem os `_test.go`).

> **Leitura:** o Rust cobre ~35% do volume não-teste do Go anterior e ~26% do C++, mas
> **volume não é paridade funcional**. O §4 mede o que existe por crate; o §5 nomeia o que falta.

## 3. De-para: subsistema do legado → destino Rust

> Os três subsistemas com contrato de porte detalhado têm documento próprio, com citação
> `arquivo:linha` do legado e destino por módulo Rust:
> [`contrato-transporte-cpsock.md`](../arquitetura/contrato-transporte-cpsock.md) (framing,
> INITCODE, transformação de keyword, checksum, aceite de sessão),
> [`contrato-dispatch-opcode.md`](../arquitetura/contrato-dispatch-opcode.md) (63 handlers do
> C++ cruzados com 61 rotas Go e 198 opcodes Rust, e o que o cliente alvo envia) e
> [`contrato-persistencia.md`](../arquitetura/contrato-persistencia.md) (56 tabelas legadas →
> 61 tabelas Rust, uso medido e buracos).

Legenda: ✅ existe · 🚧 parcial · ❌ ausente.

| Domínio | Legado (evidência) | Destino Rust | Estado |
|---|---|---|---|
| Transporte CPSock (TCP 8281, framing, sessão) | `CPSock.cpp/h`, Go `internal/protocol/{framing,header,keytable,checksum}.go` | `crates/wyd-net/` (framing/sessão) | ❌ só `limits/` + `metrics.rs` (458 linhas) |
| Contrato de wire (HEADER, ofuscação, opcodes, payloads) | `Basedef.h`, `protocol-spec.md`, 198 Types | `crates/wyd-protocol/` (199 opcodes no `opcode_table.rs`; 7 domínios de payload) | ✅ (goldens sintéticos) |
| Login / seleção / sessão de personagem | `_MSG_AccountLogin.cpp`, `_MSG_CharacterLogin.cpp`, Go `login.go`, `character.go` | `payload/auth.rs` ✅ código; máquina de sessão ❌ | 🚧 |
| Mundo: tick, grade, spawn, respawn, teleporte, cidade | `ProcessSecMinTimer.cpp`, Go `world/{tick,grid,spawn,respawn,teleport,city}.go` | `core/{tick,spatial}` ✅; `systems/movement.rs` ✅; spawn/respawn/teleporte/cidade ❌ | 🚧 |
| Combate, morte, EXP, drop | `MobKilled.cpp`, `GetFunc.cpp`, Go `combat.go`, `mobkilled.go`, `death_exp.go`, `pvpkilled.go` | `systems/combat.rs` ✅ (dano via `DamageModel`); morte/EXP/drop ❌; fórmulas em `wyd-sim` 🚧 | 🚧 |
| Skills, affects, Arch | `_MSG_Action.cpp`, Go `skill.go`, `affect*.go`, `arch_progression.go` | `systems/skill.rs` ✅ parcial; affects/progressão Arch ❌ | 🚧 |
| Inventário, cargo, equipamento | `CItem.cpp`, `_MSG_GetItem/Deposit/Withdraw/UpdateItem.cpp`, Go `item.go` | `systems/inventory.rs` ✅ + gema ✅ | ✅ (eixo inventário) |
| Refino / combine | `_MSG_CombineItem*.cpp` (10 variantes), Go `refine*.go`, `combine*.go` | nenhum `systems/refine`; taxas só no banco | ❌ |
| Comércio: loja, trade, autotrade | `_MSG_REQShopList/ReqBuy/Sell/Trade/TradingItem/SendAutoTrade.cpp`, Go `shop.go`, `trade.go` | `payload/commerce.rs` ✅; `systems/commerce.rs` ❌ | 🚧 |
| Party, guilda, chat, duelo, PK | `_MSG_InviteGuild/GuildAlly/AcceptParty/MessageChat.cpp`, Go `party.go`, `guild.go`, `chat.go`, `duel.go`, `pkmode.go` | nenhum crate | ❌ |
| Guerra, castelo, reinos, torres | `CCastleZakum.cpp`, `CWarTower.cpp`, `_MSG_War.cpp`, Go `castle.go`, `towerwar.go`, `kingdom.go` | nenhum crate | ❌ |
| Quests (story, runas, newbie, Kefra) | `_MSG_Quest.cpp`, Go `runequest*.go`, `newbiequest.go`, `kefra*.go` | nenhum crate | ❌ |
| Eventos, clima, água, nightmare, campo de treino | Go `worldevent.go`, `weather.go`, `water.go`, `nightmare*.go`, `training_field.go` | nenhum crate | ❌ |
| IA de NPC | `CMob.cpp`, `CNPCGene.cpp`, Go `mobai.go` | `core/npc/*` + `systems/npc.rs` ✅ (5 Hz, orçamento por tick) | ✅ |
| Autoridade de movimento/dano/skill/inventário | `CUser.cpp`, `Server.cpp` | `systems/*` com teste de pacote forjado | ✅ |
| Persistência | `DBSrv/CFileDB.cpp` (arquivos `account/<Key>/<NOME>`), Go `internal/dbclient` | `crates/wyd-db/` model/repo/pool/transfer; `sqlx` adiado (PROC-16) | 🚧 |
| Conteúdo estático (itens, skills, NPCs, mundo) | `content/TMsrv` (46), `ItemList.bin`/`SkillData.bin` | PostgreSQL, 24 migrations, 59 tabelas (ADR-0009); carga no boot ❌ | 🚧 |
| Billing | `BISrv/`, Go `billing.go` | nenhum crate; `_AUTH_GAME` fora de `payload/` por contrato | ❌ |
| Plataforma web / portal | `webserver/` (94), `portal/` (186), `api/` (12) | `wyd-portal/` Next.js + `wyd-admin` (HTTP/JSON, ADR-0008) | 🚧 (painel admin WEB-03 ⬜) |
| Bots de carga | Go `internal/bot` (14) | `crates/wyd-bots` inexistente (QA-11) | ❌ |
| Launcher / patch | `WYDLauncher.exe`, Go `client-tools` | `crates/wyd-launcher` (manifest+patch+distribuição+bootstrap) | ✅ |

## 4. Cobertura atual do workspace Rust (medido)

| Crate | Arquivos `src` | Linhas | Arquivos de teste em `tests/` |
|---|---:|---:|---:|
| `wyd-protocol` | 21 | 3.145 | 3 |
| `wyd-core` | 29 | 6.754 | 8 |
| `wyd-net` | 4 | 458 | 2 |
| `wyd-db` | 15 | 618 | 2 |
| `wyd-admin` | 16 | 2.463 | 3 |
| `wyd-sim` | 20 | 3.276 | 3 |
| `wyd-launcher` | 7 | 1.528 | 0 (testes unitários in-crate) |
| **Total** | **112** | **18.242** | **21** |

O desequilíbrio é a informação: `wyd-protocol` + `wyd-core` + `wyd-sim` concentram 72% das linhas,
enquanto `wyd-net` (transporte) e `wyd-db` (persistência real) somam 6%.

## 5. Lacunas que bloqueiam o objetivo declarado

Objetivo em `AGENTS.md` §1: *servidor WYD2 em Rust operacional, com login pelo cliente 7662,
personagens, batalha, comércio e quests funcionando*.

| # | Lacuna | Consequência hoje |
|---|---|---|
| L1 | **Transporte CPSock servidor não existe** (`wyd-net` = limites + métricas) | o cliente 7662 **não consegue conectar**; não há login ponta-a-ponta; não há captura ao vivo (CLIENT-06 bloqueada) |
| L2 | **Persistência real** (`sqlx`) e carga do conteúdo no boot fora (PROC-16 🚧) | o core não sobe sem Postgres e ninguém carrega `item_catalog`/`npc_template` para dentro da engine |
| L3 | **Camada de dispatch por opcode** (o `_MSG_*` do legado) não existe | 199 opcodes mapeados, 7 payloads tipados, nenhum caminho "pacote → intenção → `systems`" |
| L4 | **Comércio, party, guilda, guerra, quests, eventos, refino/combine** ausentes | dos 5 itens do objetivo, só "batalha" tem base |
| L5 | **Billing, ranking, painel admin do portal, E2E** | operação real do servidor fica sem retaguarda |
| L6 | **Conhecimento do legado só existe em `origin/main`** (§6) | risco real de perda: `develop` não tem `docs/migration/`, `docs/sistemas/`, `docs/lore/`, `docs/guias/`, `docs/analysis/`, `docs/agents/` |

## 6. Conhecimento que só existe em `origin/main` (resgatar antes de perder)

`develop` tem `docs/{arquitetura,cliente,manual-técnico,manual-usuario,rust,wyd}`. **Não** tem
`docs/migration` (79), `docs/sistemas` (13), `docs/lore` (3), `docs/guias` (4), `docs/analysis` (15),
`docs/integrations` (9), `docs/agents` (16), `docs/adr` (8). Prioridade de resgate:

| Prioridade | Artefato | Por que |
|---|---|---|
| P0 | `docs/migration/game-rules.md` (639 linhas) | fórmulas verificadas contra `Source/`: `BASE_GetDamage`, `SkillDamage`, crítico duplo, golpe em área, acerto/parry/reflect, curva de EXP, drop, refino — é a matéria-prima da FASE-10 |
| P0 | `docs/migration/parity-tests.md` | esquema de fixture, harness de replay e **paridade de RNG pelo LCG do MSVC** — define o que significa "byte-idêntico" |
| P0 | `docs/migration/data-formats.md` | offsets medidos: `STRUCT_ACCOUNTFILE` 7952 B, `STRUCT_MOB` 816, `MOBEXTRA` 552, `QUEST` 56, `ITEM` 8, `SCORE` 48; `pack(1)` vs alinhamento natural |
| P1 | `docs/migration/handlers/` (58/58 contratos) + `domain-model.md` + `flows.md` | contrato de cada `_MSG_*`: entrada, validação, efeito, resposta |
| P1 | `docs/migration/protocol-spec.md` (198 Types) | conferência cruzada com `protocolo.md`/`opcode_table.rs` |
| P2 | `docs/sistemas/*` (13), `docs/lore/*`, `docs/guias/*` | conhecimento de domínio que `docs/wyd/` já cobre em parte; conferir divergência em vez de duplicar |
| P2 | `docs/agents/component-analysis-*` (Basedef/CPSock/TMSrv-Core/CItem/CMob/CUser/CastleWar/BISrv/DBSrv) | análise de risco por componente antes da reescrita |
| P3 | `docs/analysis/*`, `docs/integrations/*`, `docs/archive/*`, `docs/adr/*` | contexto histórico; valor só quando uma issue tocar o tema |

**Regra:** resgatar para `docs/` como conhecimento datado, não como código. O que já foi superado
(ex.: taxas de servidor privado) entra marcado; o que é evidência de binário entra com build+hash.

## 7. Riscos de portabilidade (o que quebra a paridade se ignorado)

1. **Dialeto de struct por build.** 7662: `MAX_EQUIP` 16 (`Source/Code/Basedef.h:135`) e `MAX_CARRY` 64 (`Source/Code/Basedef.h:136` — 64 slots de mochila, coerente com o `STRUCT_MOB` de 816 B), `STRUCT_SELCHAR` 840 B;
   fork OSS v769: 18 / 120 / 904 B. Transporte é igual, **struct não é**. O servidor fala 7662.
2. **Checksum não rejeitante.** Os 3 sites já estão `EB` no executável de referência; rejeitar por
   padrão quebraria o cliente (`docs/cliente/superficie-conhecida.md` §2).
3. **`ESCENE_FIELD` = 30000** continua sendo o pacote autoritativo de cena (EXP/HP na barra).
4. **Ordem de RNG é contrato.** `game-rules.md` exige ordem de chamadas do LCG do MSVC para drop,
   crítico e refino. O `wyd-sim` usa `XorShift64` semeado (ADR-0011): determinístico, **não** LCG do
   MSVC. Ou se declara "não perseguimos paridade de RNG do original" (e se fixa o RNG novo como
   verdade do servidor), ou os goldens de captura nunca vão casar. Decisão precisa de ADR — hoje
   está implícita.
5. **Conteúdo só no PostgreSQL** (ADR-0009): sem `sqlx` o servidor não sobe; `.txt/.bin` legado é
   insumo de `tools/content/`, jamais leitura em runtime.
6. **Dois clientes na máquina.** `client/` é o alvo; o cliente do WYD Global é **referência
   de terceiro** e não pode virar contrato — ver `docs/cliente/referencia-wyd-global.md`.
7. **Volume do legado ≠ dívida a pagar.** Reimplementar 72 handlers Go linha a linha não é o
   objetivo: o que vale é o contrato de comportamento (entrada, validação, efeito) já documentado.

## 8. Próximos passos recomendados (ordem por desbloqueio)

1. **Resgate P0/P1 da §6** — só documentação, sem risco, e trava o conhecimento antes de qualquer
   limpeza de histórico. (Sugerido como issue nova da FASE-01/FASE-11.)
2. **L1 — transporte CPSock em `wyd-net`** (framing 12 B, ofuscação por `pKeyWord`, sessão,
   8281) com ADR de fronteira. É o que transforma protocolo em servidor e libera a captura ao vivo
   da CLIENT-06 (goldens reais em vez de sintéticos).
3. **L2 — PROC-16 (`sqlx` + pool + carga de conteúdo no boot)** — sem isso o core não é
   configurável nem persistente.
4. **L3 — camada de dispatch** (opcode → intenção tipada → `systems/*`), decidindo onde mora
   (ADR-0002 manda não inventar arquivo fora da árvore).
5. **L4 por ordem de valor**: comércio → quests → party/guilda → refino/combine → guerra/castelo.
6. **GAME-20** (anomalia de velocidade) só depois da captura real: sem trace, é chute.

## 9. Como revalidar esta análise

```bash
# legado (somente leitura, de origin/main)
for d in Source tmserver internal dbserver binserver webserver api portal roadmap docs; do
  printf "%-10s %s\n" "$d" "$(git ls-tree -r --name-only origin/main -- $d | wc -l)"; done
rm -rf /tmp/legacy && mkdir -p /tmp/legacy
git archive origin/main Source tmserver internal | tar -x -C /tmp/legacy
find /tmp/legacy/Source \( -name '*.cpp' -o -name '*.h' \) | xargs wc -l | tail -1
find /tmp/legacy/tmserver /tmp/legacy/internal -name '*.go' | xargs wc -l | tail -1

# workspace Rust
for c in crates/wyd-*; do
  echo "$c $(find $c/src -name '*.rs' | wc -l) $(find $c/src -name '*.rs' | xargs cat | wc -l)"
done

# cliente alvo (build 7662)
sha256sum client/WYD.exe   # 04c9d2a1…27ddf8
```

Se qualquer contagem divergir de forma material, esta análise está vencida: atualizar na mesma
entrega que mudou o número (`docs/rust/02-execucao.md` §5).
