# Missão wyd2.md — Fase 2: cruzamento com o legado (2026-11-10)

> Ponto de partida: `pesquisa-wyd2-missao-fase1-2026-11-10.md` + `wyd2.md` integral.
> Nada aqui vira regra do servidor por existir neste arquivo. Promoção segue o rito:
> fonte → `game_setting`/catálogo com `content_source` → teste.
> Níveis: `fontes.md` §1 (A/B/C/D/E). "Legado" = `Jean1dev/w2pp-OpenWYD@main` (Go,
> alvo 7662/proto 7640) + `Source/` C++ citado por ele.

## 1. Fontes abertas nesta fase (todas lidas)

| # | Fonte | O que extrai | URL/caminho |
|---|---|---|---|
| L1 | `docs/migration/protocol-spec.md` | HEADER 12 B, INITCODE `0x1F11F311`, framing 12–8192, ofuscação a partir do offset 4, checksum não-rejeitante, catálogo 198 Types, `MSG_AccountLogin` 116 B, `MSG_Action` 52 B, `MSG_Attack` com `Dam[13]`, `APP_VERSION 7640`, `MAX_USER 1000` | https://github.com/Jean1dev/w2pp-OpenWYD/blob/main/docs/migration/protocol-spec.md |
| L2 | `docs/migration/data-formats.md` | Regime duplo: save = alinhamento natural MSVC x86, rede = `pack(1)`; `STRUCT_MOB 816` (mapa de padding off 0–816), `STRUCT_ITEM 8`, `STRUCT_SCORE 48`, `STRUCT_QUEST 56`, `STRUCT_MOBEXTRA 552`, `STRUCT_ACCOUNTFILE 7952` (`time_t`=8), variante legada 756 B, `ItemList.csv`/`SkillData.csv` | https://github.com/Jean1dev/w2pp-OpenWYD/blob/main/docs/migration/data-formats.md |
| L3 | `docs/migration/game-rules.md` | EXP/party (`MobKilled.cpp:397-1425`, gate clã≠4, `450/(30+level)`, divisores por tier, corte −40%, `DOUBLEMODE`, `KefraLive`), drop gold (`UNKGOLD`, teto 2000/kill, 2 bi total), drop por slot (`g_pDropRate[64]` por slot, `Basedef.cpp:222-238`), combine (`rand()%115` + fold `>=100→−15`, `combine` 0–100), SancRate (PO/PL/Âmago, `sanc+1`, pity 3/4, limites selado +9/sanc+6), `BASE_GetDamage`/`BASE_GetSkillDamage` + pipeline `_MSG_Attack` (AC/2, maestria, crítico ×1.3–1.6/×2, `Ac*=3` PvP, parry `rand()%1000+1`, dano mín 1) | https://github.com/Jean1dev/w2pp-OpenWYD/blob/main/docs/migration/game-rules.md |
| L4 | `docs/migration/flows.md` | Login C→TM→DB→C, criação/seleção (`SELCHAR→WAITDB→SELCHAR`, `CharacterLogin→CHARWAIT→PLAY`), movimento, ataque→morte→drop→`CreateItem`→`GetItem`, trade atômica, combine, war/castle/billing (war/castle/billing = PARCIAL/UNVERIFIED) | https://github.com/Jean1dev/w2pp-OpenWYD/blob/main/docs/migration/flows.md |
| L5 | `docs/migration/ingame-bugs.md` | B1–B17 cliente real↔Go: ouro negativo `0xCC` (B2 ✓), level+1 (B5 ✓), `ESCENE_FIELD` barra XP (B8 ✓) e morte (B9 ✓), `UpdateEtc`, compra slot ocupado (B11), skill-gate quebrando melee (B12), Elemental (B13), durações (B14/B15), Amunra ×2 (B16), Samaritano (B17) | https://github.com/Jean1dev/w2pp-OpenWYD/blob/main/docs/migration/ingame-bugs.md |
| E1 | `crates/wyd-protocol` + `crates/wyd-server/src/cpsock.rs` | Tabela gerada, `scene.rs` (CreateMob 232 B, RemoveMob 16 B, `ESCENE_FIELD` 30000), dispatch real (AccountLogin/Create/Delete/CharacterLogin/Logout/UseItem/ApplyBonus/Attack×3/Action) | `crates/wyd-protocol/src/wire/generated/opcode_table.rs`, `payload/scene.rs`, `cpsock.rs:432-1624` |
| E2 | `docs/cliente/pacotes.md` | `CreateItem` S→C 30 B packed + `GetItem` C→S 28 B medidos no `.text` 7662; pendências de captura | `docs/cliente/pacotes.md` §"Caixa de item" |

## 2. Cruzamento STRUCT/fluxo × Rust (só evidência)

### 2.1 Transporte — CONFIRMADO nos dois lados
Legado L1 = nosso Rust: HEADER 12 B (`Size/KeyWord/CheckSum/Type/ID/ClientTick`),
ofuscação do offset 4 em diante, INITCODE `0x1F11F311` consumido antes do HEADER,
framing `Size∈[12,8192]`, checksum calculado mas **não-rejeitante** (legado tolera;
nosso build 7662 vem com os 3 `jmp EB` desativados — `superficie-conhecida.md` §2).
Versão: legado exige `ClientVersion==7640` (`Basedef.h:102`); nosso fio mede **12000**
no handshake do 7662 (`checklist` 1.4) — **divergência de versão, não de formato**:
manter gate 12000, registrar 7640 como variante documentada.

### 2.2 Saves — duplo regime confirmado, nosso banco está correto
L2 prova: **save = natural, rede = `pack(1)`**. `STRUCT_MOB 816` com padding
(`Coin@28`, `Exp@32`, `BaseScore@44`, `Equip@140`, `Carry@268`, pad final 6 B).
Erro clássico = ler save com `pack(1)` (805 B, −11 B). Nosso caminho
(`character_items`/`vault_items`, `STRUCT_MOB`/`STRUCT_SELCHAR` montados do banco)
**não** reinterpreta dump C — imune a essa classe. Variante 756 B (sem `Quest`,
`Exp` 4 B, scores 28 B) existe nos templates `npc/` do legado; nosso
`npc_template` 816 B + `npc_definition` 6100 linhas é o canônico pós-normalização.

### 2.3 Login/seleção/mundo — fluxo L4 espelha nosso `cpsock.rs`
`INITCODE → 0x020D → DB → 0x010A/SELCHAR → 0x020F/0x0211 → 0x0213 → 0x0114/PLAY`
é o mesmo esqueleto do legado. Diferenças conscientes: senha/PIN **Argon2id**
(legado era texto plano — dívida fechada), sessão única por conta, `STRUCT_SELCHAR`
844 B provisório `rust-v1` (P-7). Entrada no mundo vira entidade + save (FASE-03-8).
Falta: `ESCENE_FIELD` 30000 como cena autoritativa (FASE-03-9) — o legado confirma
a regra pelos bugs B8/B9: sem `ID=ESCENE_FIELD` a barra de XP e a morte **não**
aplicam no cliente, embora `Dam[]` aplique.

### 2.4 Combate — fórmulas legadas extraídas, NÃO promovidas
L3 dá as fórmulas exatas (`BASE_GetDamage`, `BASE_GetSkillDamage`, pipeline
`_MSG_Attack`, parry, reflect, críticos, pisos, dano mín 1). Nosso `wyd-sim`
resolve `AttackOutcome` (hit/crit/dano) e `grant_exp`, mas **não** implementa essa
cadeia — FASE-10-1 continua 🚧. Proibido copiar número sem golden: rolls usam
`rand()` MSVC/LCG; paridade exige o harness de `parity-tests.md`, não transcrição.
Anomalia 200% (FASE-10-2) ganha hipótese testável: `FreeWyd` publica tabela
velocidade/duplo/triplo e declara que lista **o que muda vs original** (fonte C,
nunca constante).

### 2.5 EXP/drop — o maior delta mapeado
Legado: EXP por membro em `±HALFGRID`, `GetExpApply` + `450/(30+level)` + divisores
por tier (Mortal/Arch/Celestial distintos) + corte −40% + cap + bônus
(item/fada/RvR/newbie/`DOUBLEMODE`/`KefraLive` ±15%) + clamp `g_pNextLevel`.
Nosso `gain_exp` usa curva `g_pNextLevel` + multiplicador `exp` do snapshot —
**núcleo compatível, árvore de bônus ausente**. Drop legado: gold determinístico +
item por **slot** (`g_pDropRate[64]` é divisor por slot, slot 56 = sempre), bônus
via `10000/(bônus+1)`. Nosso `Gameplay::loot_at_death` faz 1 roll/item da tabela
do snapshot — **modelo divergente** (por item, não por slot): registrar como
decisão consciente até captura; sem inventar tabela por slot.

### 2.6 Refino/combine — separar os três rolls
L3 fecha: combine Anct = `rand()%115` + fold; poeira em equipamento =
`rand()%100` puro; selado/celestial/pedra = `rand()%115` + fold com destruição ou
`sanc→0`. `SancRate` indexa `sanc+1`; pity `+{5,5,5,5,4,4,3,3,2,1,0}`, 3/4 acumula;
Ori trava +6, Lac +9; selado destrói. Nosso `Gameplay::refine/use_item`
(SancRate viva, `UpdateItem` pendente, piedade não persistida) cobre o caminho
principal — alinhar pity/limites indexados antes de FASE-10.

## 3. Divergência `CreateItem 0x026E` — reavaliação (P-0 da Fase 1)

| Camada | Declara | Evidência |
|---|---|---|
| `Basedef.h`/tabela congelada/legado L1 Apêndice B | `0x026E = CLIENT2GAME` | tabela 198 Types do legado |
| Binário 7662 (`docs/cliente/pacotes.md`) | **S→C, 30 B packed** | `cmp edx,0x26e → call 0x486c25` no roteador de recepção; `GetItem 0x0270` montado para `SendPacket` |
| Fluxo L4 | `CreateItem` TM→vizinhos após `MobKilled` | `CreateItem no chão → pItem[]`, pickup `ItemID=id+10000`, `dist≤3` |

**Veredito Fase 2:** o binário + o fluxo concordam (servidor anuncia a caixa).
A tabela congelada está errada nesse Flow — correção exige rito próprio
(`generated/` congelada) e **não** foi feita aqui. Regra operacional: implementar
`CreateItem S→C` + `GetItem C→S` + `CNFGetItem/CNFDropItem S→C` conforme §"Caixa"
de `pacotes.md`; marcar a linha da tabela como `DIVERGENTE-CONFIRMADA` até o rito.
Nenhum pacote novo foi emitido nesta fase.

## 4. Pendências P-1…P-14 — reavaliação com o legado

| # | Pergunta | Reavaliação Fase 2 | Default mantido |
|---|---|---|---|
| P-1 | skill/nível (4/nível 1–200?) | Legado: `LearnedSkill` bitmask + `SkillBonus`; tabela Kersef é **config de servidor** (B). Sem `SkillData.csv` do 7662, não derivar | `skill_point` no catálogo |
| P-2 | +10: 50kk vs 5kk | Divergência PT/EN do mesmo artigo = versões distintas; fluxo L4 não fixa custo | 50kk em `decisoes-servidor.md` |
| P-3 | 8 vs 2 canais | Legado e Global confirmam multicanal; `serverlist.bin` suporta 10/grupo; guerra entre canais é estado por canal | 2 canais (linha em `server_channel`) |
| P-4 | bônus cidadania | Sem número no legado; imposto castelo 40% / portal 700 (B, Global atual) | 0 até número aprovado |
| P-5 | teto 400/camada | Legado: divisores até 400 e `g_pNextLevel[MAX+1]`; 800/1500 = custom | 400/camada |
| P-6 | Arch (4 pedras+Sephirot) no 1º corte? | `celestial-system-plan.md` + L4: Arch/Celestial têm máquinas próprias, depois do mundo visível | depois de 03-9 |
| P-7 | `STRUCT_SELCHAR` provisório? | L2 dá o layout canônico, mas offsets de fio 7662 só fecham com captura 08-6 | sim, `rust-v1` + UNVERIFIED + golden |
| P-8 | donate no servidor? | `web-platform-plan.md`: web-api + `delivery_queue`, nunca estado vivo (invariante 7) | só portal/fila |
| P-9 | Soul 70/40+30 e 120/80+40? +20% Reinforce? | D-4: "20% em cada" era a passiva Reinforce, não o percentual; números worldwyd = fonte única | não aplicar; +20% como multiplicador separado configurável |
| P-10 | +12 em 5% e abençoada 2–15%? | Legado `CompRate/Odin` dá a família; 4% vs 5% = versão | default 5%/2–15% doc; seed em 4%/1–10 até decisão |
| P-11 | gate Arch 300/355/400? | D-3 resolvido: 300 criação, 355 capa/destrave, 400 Celestial | 300 mínimo + `game_setting` |
| P-12 | RvR sáb vs seg–sex? | Fonte oficial se contradiz (versões) | agenda por canal em `game_setting` |
| P-13 | montaria no 1º corte? qualidade 70? | Sistema ovo→cria→montaria inteiro ausente nos dois lados | depois; qualidade como taxa |
| P-14 | Trombeta fan-out + consumo? | `_MSG_MagicTrumpet 0x0D1D` (DB2GAME+GAME2DB+GAME2CLIENT) confirma o canal; consumo não confirmado | evento de chat; consumo a decidir |

Nenhuma P foi promovida a regra. D-3 segue resolvido; D-4 segue falso positivo corrigido.

## 5. Cobertura atualizada dos 33 domínios (Fase 1 → Fase 2)

Só muda o que o legado evidenciou. ✅/⚠️/❌ como na Fase 1.

| Domínio | Fase 1 | Fase 2 (delta do legado) |
|---|---|---|
| 4.1 atributos/pontos | ⚠️ | ⚠️ (bitmask `LearnedSkill` + bônus mapeados; distribuição `ApplyBonus` segue pendente) |
| 4.2 combate | ⚠️ | ⚠️ (fórmulas extraídas L3, não portadas; hipótese 200% identificada) |
| 4.3 EXP | ⚠️ | ⚠️ (árvore completa extraída; nosso `gain_exp` = núcleo, bônus ausentes) |
| 4.4 Mortal/Arch/Celestial/Sub | ⚠️ | ⚠️ (`celestial-system-plan` + gates D-3 alinhados; números ainda D) |
| 4.5 cidadania/reinos | ⚠️ | ⚠️ (confirmado estado por canal; sem número de bônus) |
| 4.6 mapas/teleportes | ⚠️ | ⚠️ |
| 4.7 monstros/bosses/IA | ✅/⚠️ | ⚠️+ (spawn 6100 + IA; respawn/drop→caixa seguem ❌) |
| 4.8–4.9 itens/drop | ⚠️ | ⚠️ (modelo por slot × por item divergente e registrado) |
| 4.10 refino/combine | ✅(caminho) | ✅ (três rolls separados + pity/indexação mapeados; falta `UpdateItem`/piedade) |
| 4.11 montarias | ❌ | ❌ |
| 4.12 capas/pinturas | ❌ | ❌ |
| 4.13 skills/Sephira/Soul | ⚠️ | ⚠️ (`SkillData.csv` é o caminho; percentuais Soul não aplicar) |
| 4.14 quests | ⚠️ | ⚠️ |
| 4.15 NPCs/lojas | ⚠️ | ⚠️ (loja/banco parciais B7; quest/combine NPCs ❌) |
| 4.16 consumíveis | ⚠️ | ⚠️ |
| 4.17 guilds/guerras | ⚠️ | ⚠️ (sequência exata UNVERIFIED) |
| 4.18 eventos | ⚠️ | ⚠️ |
| 4.19 Kefra/especiais | ⚠️ | ⚠️ |
| 4.20 cidades/impostos | ⚠️ | ⚠️ |
| 4.21 economia | ⚠️ | ⚠️ (teto 2000/kill, 2 bi total extraídos; mapa de fluxo pendente) |
| 4.22 troca/grupo/loja | ❌ | ❌ (fluxo L4 exige atomicidade por par — anotado, não implementado) |
| 4.23 PVP/PK/chaos | ❌ | ❌ (`Ac*=3`, reflect extraídos, não portados) |
| 4.24 chat/Trombeta | ❌ | ❌ (`0x0D1D` confirma existência; fio ❌) |
| 4.25 cash/premium | ⚠️ | ⚠️ (billing 196 B cru, layout UNVERIFIED) |
| 4.26 canais/troca | ⚠️ | ⚠️ (esperas `WAITDB` viram async — desenho, não código) |
| 4.27 login/logout | ⚠️ | ⚠️ (B8/B9 explicam `ESCENE_FIELD`; logout explícito segue próximo) |
| 4.28 tela personagem | ⚠️ | ⚠️ |
| 4.29 barra skills | ✅ persistência | ✅ |
| 4.30 efeitos/aparência | ❌ | ❌ |
| 4.31 iniciais | ✅ | ✅ |
| 4.32 buffs/timers | ⚠️ | ⚠️ (affects B13–B17: portar com gate `IsPlayer` e política de duração) |
| 4.33 resets/ciclos | ❌ | ❌ |
| §5 funcionalidades básicas | ⚠️ | ⚠️ |
| §6 bugs/hacks | ❌ | ⚠️ (B1–B17 dão o esqueleto do catálogo; nosso catálogo próprio segue ❌) |
| §7 XP jogadores | ❌ | ❌ |
| §8 matemática | ⚠️ | ⚠️ (contrato de ordem + fórmulas legadas citadas, não copiadas) |
| §9 bots/LLM/stress | ❌ | ❌ (`parity-tests.md` dá o harness de replay como base futura) |

## 6. Fatiamento pronto para a Fase 3 (issues propostas, NÃO criadas)

Ordem por desbloqueio; cada uma testável e sem inventar número:

1. `CreateItem/GetItem` no fio (S→C 30 B / C→S 28 B + `CNFGetItem`, `DecayItem`, pickup `dist≤3`, `ItemID=id+10000`) + rito da tabela congelada — desbloqueia drop→caixa→pickup.
2. `ESCENE_FIELD` 30000 como cena autoritativa (B8/B9) — barra XP e morte no cliente.
3. Logout explícito `0x0215` (FASE-08-9) — fecha §4.27.
4. Cadeia de dano L3 em `wyd-sim` com goldens (`BASE_GetDamage/SkillDamage`, parry, reflect, pisos, mín 1) — pré-requisito de FASE-10-1.
5. Árvore de EXP L3 por trás de flags (`DOUBLEMODE`, `KefraLive`, RvR, newbie ±15%) em `game_setting` — sem mudar o default atual.
6. Pity/limites de refino indexados (`sanc+1`, 3/4, travas +6/+9/selado) + `UpdateItem` + piedade persistida.
7. Trade atômica por par (L4 §5) + `QuitTrade` — com teste de concorrência, sem `MSG_Trade` inventado.
8. Catálogo de ameaças B1–B17 + dup (trade/drop/canal/sessão) como checklist de segurança.
9. Harness de replay/paridade (LCG MSVC, fixture + proxy nos 3 links) como base de `wyd-bots`.

*Branch `develop`, árvore suja preservada, nenhum commit, nenhuma issue criada nesta fase.*
