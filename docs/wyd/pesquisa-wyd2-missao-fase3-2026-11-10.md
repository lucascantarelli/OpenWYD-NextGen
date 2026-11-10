# Missão wyd2.md — Fase 3: detalhamento dos 9 itens fatiados (2026-11-10)

> Detalha os 9 itens da Fase 2 §6 nos 26 campos do §11.2 do `wyd2.md`.
> Ordem da Fase 2: priorizados integralmente (itens 1–2); itens 3–9 em formato
> condensado com o mesmo esqueleto e sem preencher campo sem evidência.
> Nada aqui vira regra por existir neste arquivo. Níveis: `fontes.md` §1.
> Nenhuma issue criada, nenhum commit, nenhum arquivo sujo tocado.

Fontes reutilizadas (já abertas nas Fases 1–2, não re-lidas como novas):
L1 protocol-spec, L2 data-formats, L3 game-rules, L4 flows, L5 ingame-bugs
(`Jean1dev/w2pp-OpenWYD@main`), E1 `crates/wyd-protocol` + `cpsock.rs`,
E2 `docs/cliente/pacotes.md` §"Caixa de item" (binário 7662
`SHA-256 04c9d2a1…27ddf8`), `docs/cliente/superficie-conhecida.md`,
`payload/scene.rs`, `gameplay.rs`, `cpsock.rs:1008-1022`.

---

## Item 1 — CreateItem/GetItem no fio (drop→caixa→pickup)

1. Visão geral: materializar no cliente a caixa do item sorteado na morte e
   permitir a coleta com validação servidor-autoritativa.
2. Objetivo: fechar o vão entre `loot_at_death` (sorteio, já existe) e o chão
   visível/coletável.
3. Funcionamento original: morte → `MobKilled()` → drop gold/item (L3) →
   `CreateItem` TM→vizinhos (caixa em `pItem[]`) → `GetItem` C→TM (`ItemID=id+10000`,
   `dist≤3`, `Mode≠0`) → `CNFGetItem` + remoção da caixa; expiração por `DecayItem`
   (L4 §4; E2 confirma direção no binário).
4. Regras de negócio: um `CreateItem` por item caído, com id de caixa do canal;
   pickup valida distância contra posição autoritativa, move para `Carry[]`,
   some da vista de todos; timeout expira sem tabela nova (mapa do canal é a fonte).
5. Fórmulas matemáticas: nenhuma nova. Rolls já existentes (`Gameplay::loot`).
   Faixa de ids de caixa: DESCONHECIDA (E2 §"O que continua UNVERIFIED" item 3).
6. Estruturas de dados: `STRUCT_ITEM` 8 B (L2; E2); mapa do canal
   `id → (item, posição, instante)` — desenho proposto, não implementado.
7. Entidades envolvidas: criatura morta, assassino, vizinhos em visão, caixa.
8. NPCs e mapas: nenhum específico; posição = morte.
9. Requisitos: morte resolvida com loot (existe); `reply_with_body` (existe,
   igual a `CreateMob`).
10. Dependências: `loot_at_death` (pronto); tabela congelada (rito pendente); ids
    de caixa (pendente).
11. Fluxos de execução: morte → sorteio → `event=death/event=loot` (hoje) →
    `CreateItem` (passo 2) → `GetItem` → grava `character_items` → `CNFGetItem` +
    `DecayItem` → expiração (E2 "Menor caminho", passos 1–4).
12. Comportamento do cliente: recebe `0x026E` (handler `0x486c25`), monta `0x0270`
    com `memset 28` + `SendPacket`, `DestType=1` no caminho observado (E2).
13. Comportamento do servidor: anunciar caixa; validar `GetItem`; responder
    `CNFGetItem`/`DecayItem`. Hoje: só sorteia e loga.
14. Persistência: gravar item em `character_items` na coleta; caixa é efêmera.
15. Protocolos: `0x026E CreateItem` **S→C 30 B packed**; `0x0270 GetItem` **C→S
    28 B** (26 + 2 padding zerado); `0x0171 CNFGetItem` S→C; `0x016F DecayItem`
    S→C; `0x0272/0x0175` Drop/CNFDrop. Layouts em E2 (offsets por VA).
16. Estados e transições: caixa viva → coletada (`CNFGetItem`+remoção) ou
    expirada (`DecayItem`); sem máquina além disso.
17. Restrições: `DestType` só visto como 1 (E2 UNVERIFIED 1); `CreateItem` 30 vs
    32 B com padding só fecha com captura (E2 UNVERIFIED 2); ids de caixa sem fonte.
18. Casos especiais: slot destino ocupado (cf. B11 do L5: reenviar `SendItem`
    para ressincronizar em vez de `return` mudo).
19. Bugs conhecidos: B11 (compra com slot ocupado falhava em silêncio — padrão a
    replicar na coleta).
20. Vulnerabilidades: pickup à distância (validar `dist≤3`); duplo pickup
    concorrente (idempotência por id de caixa); spawn forjado C→S (tabela declara
    `CreateItem` como `Client2Game` — não aceitar nessa direção).
21. Testes necessários: golden 30 B/28 B por VA; coleta grava e responde;
    concorrência de pickup; expiração; rejeição de `CreateItem` C→S.
22. Implementação atual no Rust: sorteio até `event=loot` (E1);
    `CreateItem/GetItem` sem fio; tabela com `Flow::Client2Game` oposto ao binário
    (`opcode_table.rs:881-883`) — rito próprio pendente.
23. Gaps identificados: emitir caixa; tratar `GetItem`; `CNFGetItem`/`DecayItem`;
    faixa de ids; `UpdateItem` de refresh (passo 3 do E2 sugere reaproveitar).
24. Issues relacionadas: (a criar na Fase 5) "caixa no chão + pickup"; depende do
    rito da tabela congelada.
25. Fontes e evidências: L1 Apêndice B (Types), L3 §2 (drop), L4 §4, E2 (VAs
    `0x495dc5/0x486c25/0x4fee1b-7e/0x54da23`), E1 (`loot_at_death`, tabela).
26. Pendências de investigação: `DestType≠1`; 30 vs 32 B; faixa de ids de caixa.
    Grau: PROVÁVEL (direção/layout) / DESCONHECIDO (ids).

## Item 2 — Cena autoritativa ESCENE_FIELD (30000)

1. Visão geral: todo evento de cena (spawn, ataque, XP, morte) chega ao cliente
   com `HEADER.ID = 30000`, nunca com o id da conexão do dono.
2. Objetivo: destravar barra de EXP/HP e estado de morte no cliente 7662.
3. Funcionamento original: `GetAttack → sm->ID=ESCENE_FIELD` (`GetFunc.cpp`,
   via L5 B8/B9); `SendWeather` broadcast com `ID=ESCENE_FIELD` (L1 §3.2).
4. Regras de negócio: `CreateMob` com `ID=ESCENE_FIELD`, `MobID` em +16;
   `RemoveMob` com `ID` = entidade e corpo `RemoveType` 0/1/2 (E1 `scene.rs`).
5. Fórmulas: nenhuma.
6. Estruturas: `CreateMob` 232 B, `RemoveMob` 16 B (E1, golden em teste).
7. Entidades: todas visíveis na cena.
8. NPCs/mapas: `sync_npcs` já reconcilia sem duplicar (Fase 1).
9. Requisitos: motor no processo (existe); `reply_with_body` com `ID_SCENE`.
10. Dependências: entrada no mundo (existe); captura 08-6 (confirmação final).
11. Fluxos: ataque → `Engine::attack_at` → broadcast cena → `RemoveMob(DEATH)` +
    `UpdateEtc/UpdateScore` no abate (hoje parcial).
12. Cliente: aplica `Dam[]` independente do header, mas só aplica
    `CurrentExp/CurrentHp/CurrentMp` e morte quando `ID=ESCENE_FIELD` (L5 B8/B9).
13. Servidor: emitir ataque do mob com `ID=ESCENE_FIELD` (legado já faz;
    `handler/mobai.go` no Go); guarda `HP<=0` endurecida (L5 B9).
14. Persistência: nenhuma nova.
15. Protocolos: `0x0364` 232 B, `0x0165` 16 B, `0x7530` (constante existe,
    nenhum pacote emitido — invariante 4).
16. Estados: `JUST_ENTERED=2`, `OUT_OF_VIEW=0`, `DEATH=1`, `LOGOUT=2` (E1).
17. Restrições: não usar id da conexão no header de cena (bug B8/B9).
18. Casos especiais: morto continua agindo se a cena não chegar (B9) —
    endurecer guarda + `SendHpMode/AddCrackError` equivalentes.
19. Bugs: B8 (XP não anda), B9 (morto ataca/usa item) — causa raiz comum.
20. Vulnerabilidades: cliente dita dano (cf. B12: claims inesperados resolvem
    como melee com log, nunca zeram; dano é server-authoritative).
21. Testes: header de ataque-mob = cena; `UpdateEtc` no level-up e refreshes de
    gold (L5 B8/B10); regressão melee `SkillIndex=0` (L5 B12).
22. Implementação atual: `CreateMob/RemoveMob` portados + provados no fio
    (6100 frames); ataque no fio sem cena completa.
23. Gaps: ligar `ESCENE_FIELD` no broadcast de ataque/morte/XP.
24. Issues: (a criar) "cena autoritativa"; desbloqueia mundo visível.
25. Fontes: L1 §3.2, L5 B8/B9/B12, E1 `scene.rs:19-22`, `cpsock.rs` attack.
26. Pendências: captura 08-6. Grau: VALIDADO (duas fontes independentes: legado
    + binário via sintomas).

## Itens 3–9 (condensado, mesmo esqueleto)

### Item 3 — Logout explícito `0x0215`
Fluxo L4: `CharacterLogout → volta à seleção`; hoje `cpsock.rs:1012-1022`
responde `CNFCharacterLogout` e libera a sessão (idle 120 s + queda de socket).
Campos §11.2: objetivo = liberar sem depender de timeout (requisito §4.27);
persistência = save já existente; teste = relogin imediato; gap = pacote de saída
sem teste dedicado (checklist 1.13 🚧). Fontes: L4 §2, E1 dispatch. Grau: PROVÁVEL.

### Item 4 — Cadeia de dano em `wyd-sim`
Fórmulas L3 §4 (`BASE_GetDamage/SkillDamage`, pipeline `_MSG_Attack`, parry
`rand()%1000+1`, reflect, críticos, pisos, mín 1) citadas, não copiadas.
Nosso `damage_chain` (E1 `damage.rs`) é placeholder (identidade/clamps).
26 campos: estruturas `DamageInput`; ordem base→…→resultado como contrato;
PROIBIDO portar sem golden + LCG MSVC (`parity-tests.md`). Gap: FASE-10-1 🚧.

### Item 5 — EXP por flags
Árvore L3 §1 (gate clã≠4, `450/(30+level)`, divisores por tier, −40%, bônus
item/fada/RvR/newbie/`DOUBLEMODE`/`KefraLive` ±15%, clamp `g_pNextLevel`).
Nosso `grant_exp` = núcleo + multiplicador do snapshot. 26 campos: dependência
`game_setting` por flag; default atual inalterado; teste por flag isolada.
DESCONHECIDO: valores de `PARTYBONUS`/`GameConfig` do 7662 sem captura.

### Item 6 — Pity/limites de refino + `UpdateItem`
L3 §3: três rolls separados, `SancRate[sanc+1]`, pity `{5,…,0}` 3/4, travas
Ori +6/Lac +9/selado +9-destrói, cel/HC e pedras com `sanc→0`. Nosso
`gameplay.rs` (E1) aplica `LEGACY` + `powder_rate_bps` com `roll=rand()%100`;
falta piedade persistida + `UpdateItem`. Teste: taxa por passo antes/depois da
recarga (padrão já usado em `config_gameplay.rs`).

### Item 7 — Trade atômica por par
L4 §5: `TradingItem` → `Trade` estado → ambos `MyCheck==1` → swap atômico
(item+gold); `QuitTrade` cancela ambos. `MSG_Trade` UNVERIFIED (invariante 5;
`commerce.rs:3`, `payload/mod.rs:12`). 26 campos: concorrência por par +
idempotência; teste de dup sob contenção; sem inventar layout até captura.

### Item 8 — Catálogo de ameaças B1–B17 + dup
L5 dá o esqueleto (ouro `0xCC`, level+1, cena, slot ocupado, skill-gate, affects,
durações, Amunra ×2, Samaritano) + pontos críticos de dup em trade/drop/canal/
sessão (L4 §5). Nosso checklist §6 segue ❌. 26 campos: cada ameaça com
causa/impacto/condição/prevenção/validação; revisar se o Rust já protege;
`B6` (posição exata não persiste) é comportamento esperado, não bug.

### Item 9 — Harness de replay/paridade
L1 `parity-tests.md` (golden por subsistema, fixture + replay, proxy nos 3 links,
paridade de RNG via LCG MSVC, matriz de cobertura). Base futura de `wyd-bots`
(FASE-15-3, `crates/wyd-bots` inexistente). 26 campos: determinismo antes de
carga; sanitize de fixtures; sem `#[ignore]`; bot descobre, teste reproduz sem rede.

*Branch `develop`; árvore suja intocada; nenhum commit; nenhuma issue criada.*
