# Missão wyd2.md — Fase 3: detalhamento dos 9 itens fatiados (2026-11-10)

> Detalha os 9 itens da Fase 2 §6 nos 26 campos do §11.2 do `wyd2.md`.
> Itens 1–2 preservados da entrega anterior; itens 3–9 agora no mesmo nível de
> detalhe. Nada aqui vira regra por existir neste arquivo. Níveis: `fontes.md` §1.
> Nenhuma issue criada, nenhum commit, nenhum arquivo sujo tocado.

Fontes (já abertas nas Fases 1–2): L1 protocol-spec, L2 data-formats, L3
game-rules, L4 flows, L5 ingame-bugs (`Jean1dev/w2pp-OpenWYD@main`), E1 crates
Rust, E2 `docs/cliente/pacotes.md` §"Caixa de item" (7662 `04c9d2a1…27ddf8`).

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
    faixa de ids; `UpdateItem` de refresh.
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
26. Pendências: captura 08-6. Grau: VALIDADO (legado + binário via sintomas).

## Item 3 — Logout explícito `0x0215`

1. Visão geral: saída controlada do mundo de volta à seleção, liberando a sessão
   na hora sem depender do timeout de ociosidade.
2. Objetivo: cumprir o requisito §4.27 do `wyd2.md` (login/logoff completos, sem
   depender exclusivamente de timeout).
3. Funcionamento original: `C → _MSG_CharacterLogout (0x0215)` → TM volta à
   seleção (`Mode PLAY→SELCHAR`); `DBSaveMob (0x0807)` / `SavingQuit (0x0806)`
   salvam; `DBNoSave (0x0805)` desconecta sem salvar (L1 §3.3; L4 §2).
4. Regras de negócio: resposta `CNFCharacterLogout (0x0116)` só de cabeçalho;
   save enfileirado na saída; sessão liberada para relogin imediato.
5. Fórmulas matemáticas: nenhuma.
6. Estruturas de dados: `SessionLease`/`SessionGate` (E1 `cpsock.rs:52,198,383`);
   `Drop` do lease libera a vaga.
7. Entidades envolvidas: conexão, conta, personagem em jogo.
8. NPCs e mapas: nenhum.
9. Requisitos: handshake/CNF e seleção existentes.
10. Dependências: `leave_world` + `Drop` do `SessionLease` (existem, E1).
11. Fluxos de execução: `0x0215` → log `result=ok reason=explicit` →
    `reply(CNFCharacterLogout)` → `false` (encerra) → `leave_world` + save
    (`cpsock.rs:1008-1023`, comentário "sem esperar o timeout").
12. Comportamento do cliente: envia `0x0215` ao voltar à seleção ou fechar a
    janela (comentário E1 `cpsock.rs:1008-1010`); UNVERIFIED o byte-exato do
    frame de saída — fechar pela captura 08-6.
13. Comportamento do servidor: responder `0x0116` e encerrar aqui; queda de
    socket e ociosidade (120 s) continuam como rede de segurança.
14. Persistência: save do estado vivo na saída (posição/HP/MP/EXP via caminho
    FASE-03-8); sem `DBNoSave` salvo decisão explícita.
15. Protocolos: `0x0215` C→S (só header, `MSG_STANDARD`), `0x0116` S→C só header
    (E1; `pacotes.md` registra `CNFCharacterLogout` como PORTADO).
16. Estados e transições: `InGame → seleção`; sessão `ativa → liberada`; relogin
    imediato permitido (sessão única por conta, `0030_sessao_unica`).
17. Restrições: não confundir saída voluntária com queda (ambas liberam, só a
    primeira responde `0x0116`); não reter vaga após `Drop`.
18. Casos especiais: cliente fechado à força (sem `0x0215`) — cai no timeout/
    queda; CHARACTERLOGFAIL não existe no catálogo (só `CharacterLoginFail`).
19. Bugs conhecidos: "cliente fechado continua online" (comentário E1) —
    resolvido pelo `leave_world` + `Drop` na saída explícita.
20. Vulnerabilidades: sessão presa (duplo login negado); replay de logout sem
    sessão — responder sem mutação.
21. Testes necessários: logout responde `0x0116`; sessão libera (relogin imediato
    aceito); save executado; queda sem `0x0215` libera por timeout (existente).
22. Implementação atual no Rust: handler implementado (`cpsock.rs:1012-1023`);
    checklist 1.13 segue 🚧 por falta de teste dedicado ("só o timeout tem código"
    está desatualizado no texto, mas o teste continua pendente).
23. Gaps identificados: teste dedicado de logout explícito (FASE-08-9); captura
    do frame real de saída.
24. Issues relacionadas: (a criar) FASE-08-9 "logout explícito"; sem dependência
    de captura para o código, com captura para o golden do frame.
25. Fontes e evidências: L1 §3.1/§3.3, L4 §2, E1 `cpsock.rs:1008-1023`,
    `session.rs` (`SessionGate`/`SessionLease`), migration `0030_sessao_unica`.
26. Pendências de investigação: byte-exato do `0x0215` no 7662 (captura 08-6).
    Grau: PROVÁVEL (fluxo) / UNVERIFIED (frame de saída).

## Item 4 — Cadeia de dano em `wyd-sim`

1. Visão geral: reconstruir a ordem de modificadores do dano ponta a ponta com
   goldens, antes de tocar a engine.
2. Objetivo: fechar FASE-10-1 com fórmulas provadas, não opinião.
3. Funcionamento original: `dam = atacante.CurrentScore.Damage`; crítico parcial
   ×1.3–1.4 (player) / ×1.5–1.6 (mob) (`DoubleCritical&2`); `Ac` do alvo
   (player `Ac*=3`); `BASE_GetDamage(dam, Ac, master)`; área extra FM/skill
   0x200000 25%; crítico total `×2` (`DoubleCritical&1`); acerto/parry;
   reflect; clamps; `+ForceMobDamage` vs mob (`_MSG_Attack.cpp:440-480,1415-1440,
   1496-1508,1470`, via L3 §4).
4. Regras de negócio: mitigação `tdam = dam − ac/2`; maestria limita variância
   (`combat=min(combat/2,7)`, teto 15 em skill); esquiva `rand()%1000+1 <
   parryretn → dam=-3/-4`; dano mínimo 1 (L3 §4.1/4.2/4.4).
5. Fórmulas matemáticas: melee `rnd = rand()%(12−combat)+combat+99`,
   `tdam = rnd*tdam/100` + escada de pisos (`<-50→0`, `-50..0→(t+50)/7`,
   `0..50→5t/4+7`); skill `rnd = rand()%(21−combat)+combat+90` + escada própria
   (`0..45→5t/4+5`) (L3 §4.1/4.2, `Basedef.cpp:1265/1486`) — CITADAS, não
   copiadas para o Rust sem golden + LCG.
6. Estruturas de dados: `DamageInput` + `STAGE_ORDER` de 7 estágios
   (`damage.rs:17-24`); `attack_power`, `effective_defense`, `damage_chain`.
7. Entidades envolvidas: atacante, alvo (player/mob), 13 slots `Dam[]`
   (`STRUCT_DAM {TargetID, Damage}`, L1 §3.5 `MSG_Attack`).
8. NPCs e mapas: nenhum específico; `ForceMobDamage` soma por cima vs mob.
9. Requisitos: `parryretn` de `GetParryRate` (UNVERIFIED no detalhe por
   classe/arma — L3 §4.6 manda tabelar `Basedef.cpp:3245+`).
10. Dependências: `getDamage`/`SkillDamage` do legado como oráculo; LCG MSVC para
    paridade de `rand()` (`parity-tests.md`).
11. Fluxos de execução: intenção → `DamageInput` → `attack_power` → skill →
    buff → crítico → `pierce/defense` → resist → `absorb` →
    `damage_after_absorb` (saturação em zero, E1 `damage.rs:91-104`).
12. Comportamento do cliente: envia `SkillIndex`/`Dam[]` (pode mandar dano local
    — IGNORADO; B12); aplica o resultado do broadcast.
13. Comportamento do servidor: autoridade total; claims inesperados resolvem como
    melee com log, nunca zeram (L5 B12); morto com `HP<=0` endurecido (L5 B9).
14. Persistência: nenhuma (dano é efêmero; morte persiste via HP/EXP).
15. Protocolos: `0x0367 MSG_Attack` 164 B (`Dam[13]` 104 B, L1 §3.5);
    `0x0366/0x0368` movimento/ação vizinhos.
16. Estados e transições: hit/crit/miss(`-3`)/block(`-4`); `AttackOutcome`
    (`resolve_attack`, E1 `wyd-sim/src/lib.rs:26`).
17. Restrições: ordem de modificadores é contrato (E1 `STAGE_ORDER` travada por
    teste); sem `f64`; sem `rand()` implícito (seed explícita).
18. Casos especiais: skill 79/22 reduzem parry a 30%; `Rsv&0x40` soma 500 no
    `attackdex`; `Rsv&0x200` + `rd<100` → `-4` (L3 §4.4).
19. Bugs conhecidos: B12 (skill-gate zerava melee), B16 (Amunra sem ×`(sanc+10)/10`
    + promoção acessório `sanc==9→10`), B8/B9 (cena).
20. Vulnerabilidades: dano ditado pelo cliente; `Dam[]` fora de `-2/-1/0` tratado
    como melee (B12); overflow saturado (`saturating_*`).
21. Testes necessários: goldens `BASE_GetDamage/SkillDamage` (incl. pisos e
    `rnd` por maestria); parry/reflect; regressão `TestMeleeAlwaysDamagesMob`;
    cadeia `163` do `mod.rs:52-57` como trava atual.
22. Implementação atual no Rust: cadeia hipotética documentada como provisória
    (`damage.rs:1-13`, decisões D-02/D-03/D-04 em `matematica-sim.md` §4);
    único passo provado = `damage_after_absorb` (saturação).
23. Gaps identificados: portar L3 §4 com goldens; tabelar coeficientes
    Dex/Str por classe×arma (`Basedef.cpp:3245+`); anomalia 200% (FASE-10-2).
24. Issues relacionadas: (a criar) FASE-10-1 "cadeia de dano com goldens";
    bloqueia FASE-15.
25. Fontes e evidências: L3 §4/§7, L5 B8/B9/B12/B16, E1 `damage.rs`, `mod.rs`,
    `lib.rs:26`, `skills.md` §5 + `matematica-sim.md` §4.
26. Pendências: trace que fecha D-02/D-03/D-04/D-06; LCG exato. Grau: HIPÓTESE
    (cadeia atual) / LEGADO-CONFIRMADO (fórmulas citadas, não portadas).

## Item 5 — EXP por flags

1. Visão geral: aplicar a árvore completa de EXP do legado por trás de flags
   configuráveis, sem mudar o default atual.
2. Objetivo: paridade de progressão sem quebrar o jogo atual.
3. Funcionamento original: `MobKilled.cpp:397-1425` — gate `Clan!=4`; base
   `GetExpApply`; bônus party (`NumMob`, `PARTYBONUS`, loop `MAX_PARTY+1` em
   `±HALFGRID`); `450/(30+level)`; gate `0<exp≤10⁷`; divisores por tier
   (Mortal/Arch/Celestial tabelados, L3 §1.4); corte −40%; cap `eMob`; bônus
   item/fada, RvR +5%, newbie +25%, `DOUBLEMODE ×2`, `KefraLive==0 ÷2`, ±15%
   newbie; clamp `g_pNextLevel` (L3 §1).
4. Regras de negócio: cada ajuste é uma flag em `game_setting`; default = árvore
   atual (`gain_exp` + multiplicador `exp` do snapshot).
5. Fórmulas matemáticas: `isExp = NumMob*MobExp/100`; `exp = 450*isExp/(30+level)`;
   `exp = 6*exp/10`; `exp += exp*(ExpBonus+fada)/100`; RvR/newbie/DOUBLEMODE/
   Kefra/±15% na ordem L3 §1.5 — CITADAS, não aplicadas globalmente.
6. Estruturas de dados: `LevelState {level, exp_into_level}`, `ExpTable`
   (`progression.rs:95-130`); excedente no teto RETIDO (regra 3 de
   `progressao.md` §6).
7. Entidades envolvidas: killer, membros da party vivos em alcance, mob.
8. NPCs e mapas: `HALFGRIDX/HALFGRIDY` como raio (valor exato UNVERIFIED).
9. Requisitos: curva `g_pNextLevel` (existe); `GameConfig::exp` (existe).
10. Dependências: `Gameplay::exp_reward` (existe); `config_revision`/poll 1 s
    (existe, Fase 1).
11. Fluxos: abate → `gain_exp` (hoje) → (futuro) árvore por flags →
    `UpdateEtc + UpdateScore` (hoje no `attack`, E1).
12. Comportamento do cliente: barra anda via `UpdateEtc` quando o pacote é cena
    (L5 B8); `DisableExpMsg (0x034D)` como opt-out (L1 Apêndice B).
13. Comportamento do servidor: `grant_exp` por limiar cumulativo, sem pular
    nível sem custo (`progression.rs:106-130`, teste `grant_exp_levels_up…`).
14. Persistência: grava `level/exp` transacional (Fase 1, `send_progress`).
15. Protocolos: `0x0336 UpdateScore`, `0x0337 UpdateEtc`, `0x0338 CNFMobKill`
    (L1 Apêndice B).
16. Estados: `Hold` de EXP (trava que consome o ganho até quitar, L3 §1.5);
    log diário com reset (L3 §1.5) — ambos DESENHO, não implementados.
17. Restrições: `0<exp≤10⁷` NÃO clampa, pula o prêmio (L3 §1.3); cap `exp≤eMob`.
18. Casos especiais: `Clan==4` nunca dá EXP (gold/drop fora do gate);
    `myLevel += MAX_LEVEL+1` fora de Mortal/Arch (L3 §1.3).
19. Bugs conhecidos: B8 (barra parada sem cena) — não é fórmula, é fio.
20. Vulnerabilidades: `current_exp` do cliente ignorado (E1 `attack`); farm por
    `KefraLive`/`DOUBLEMODE` auditável via `config_revision`.
21. Testes necessários: um por flag isolada + combinadas; trava `Hold`; clamp;
    regressão do default (multiplicador atual inalterado com flags off).
22. Implementação atual: `grant_exp` + `exp_reward` com multiplicador (E1);
    árvore de bônus ausente por decisão.
23. Gaps: `PARTYBONUS`, `HALFGRID`, `ExpBonus`/fada, RvR/newbie, `DOUBLEMODE`,
    `KefraLive`, ±15%, `Hold`, log diário.
24. Issues: (a criar) "árvore de EXP por flags"; default inalterado.
25. Fontes: L3 §1/§7, E1 `progression.rs:106-130`, `progressao.md` §6,
    `game_setting` (`multiplier:exp` já vivo).
26. Pendências: valores 7662 de cada flag sem captura. Grau: NÚCLEO VALIDADO /
    BÔNUS HIPÓTESE.

## Item 6 — Pity/limites de refino + `UpdateItem`

1. Visão geral: alinhar o refino com poeiras ao legado indexado e persistir a
   piedade, com resposta visível ao cliente.
2. Objetivo: fechar o caminho principal de refino sem inventar taxa.
3. Funcionamento original: refino mora em `_MSG_UseItem (0x0373)`, não no
   combine (L3 §3.6, `_MSG_UseItem.cpp:140-980`); equipamento padrão `rand()%100`,
    falha só gasta o pó; selado `rand()%115`+fold destrói; celestial/HC e pedras
   arcanas `sanc→0`; brinco +10–14 destrói (L3 §3.6 tabela).
4. Regras de negócio: `SancRate` indexa `sanc+1` (slot 0 morto); Ori trava +6,
    Lac +9, +10 preserva gem e para; selado `sanc≥9` não refina; outros
   `sanc≥6 && Vol==4` não; `EF_NOSANC(126)` bloqueia; `Lactolerium_100 (4141)`
   força Âmago 100% (L3 §3.3–3.6).
5. Fórmulas matemáticas: taxas PO/PL/Âmago por nível (L3 §3.3) CITADAS;
   pity `+{5,5,5,5,4,4,3,3,2,1,0}` por próximo nível; acúmulo 3/4
   (`rand()%4≤2`), Lac sempre, Ori só até o teto (L3 §3.3; E1
   `refine.rs:155-189`, `gameplay.rs:37`).
6. Estruturas de dados: `Powder {Ori, Lac, Amago}`, `RefinePolicy::LEGACY`
   (`allows`, `ori_max/PL_MAX`, E1 `refine.rs:82-145`); `PITY_MAX=20`,
   `REFINE_MAX=15` (E1 `refine.rs:19,155`).
7. Entidades envolvidas: item-alvo, pó-origem (`EF_VOLATILE`), personagem.
8. NPCs e mapas: nenhum (arrastar pó sobre item); NPCs de combine ficam para o
   combine Anct, não este item.
9. Requisitos: SancRate viva por tentativa (existe); gravação em
   `character_items` (existe).
10. Dependências: `Gameplay::use_item/refine` (existe); `UpdateItem` de resposta
    (ausente); piedade persistida (ausente).
11. Fluxos: `UseItem` → classifica pelo `EF_VOLATILE` → `allows` → taxa do
    snapshot → roll → consome pó → sobe/destrói/mantém → grava → (falta)
    `UpdateItem` + `SendItem` (E1 `cpsock.rs:1024+`, L3 §3.6).
12. Comportamento do cliente: arrasta pó sobre item; espera `SendItem (0x0182)`
    / `UseItem (0x0374)` de resultado (L1 §3.2).
13. Comportamento do servidor: lê SancRate por tentativa; recusa sem mutação
    quando a célula falta (`MissingRate`, E1 `gameplay.rs:89-108`).
14. Persistência: gravar item + pó consumido na mesma transação; piedade por
    item (a criar).
15. Protocolos: `0x0373 UseItem`, `0x0182 SendItem`, `0x0374 UseItem` resultado,
    `0x0386 CombineComplete` só para combine (L1 §3.2).
16. Estados: `sanc 0..15`; `cValue = nível+10*pity` (0–9),
    `cValue = base+gem` (10–15: `230/234/238/242/246/250`, L3 §3.3).
17. Restrições: tintura→feijão é decisão de migração, não port (L3 §3.6.1) —
    manter isolada em `refineTintura` antes do `Bootstrap`.
18. Casos especiais: `EF_SANC` em tintura (sem roll, transformação determinística,
    L3 §3.6.1 UNVERIFIED-legado).
19. Bugs conhecidos: B16 (Amunra sem multiplicador `(sanc+10)/10` + promoção
    `9→10` em acessório + isenções + `EF_RUNSPEED` clampado, L5 B16).
20. Vulnerabilidades: roll client-side ignorado; anti-spam 1000 ms DESATIVADO no
    legado (L3 §3.6) — rate-limit do `wyd-net` como compensação.
21. Testes necessários: taxa por passo antes/depois da recarga (padrão
    `config_gameplay.rs`); pity 3/4; travas +6/+9; `MissingRate` sem mutação.
22. Implementação atual: caminho principal no fio com prova de recarga (Fase 1);
    `accessory.rs` com colar/cinto 60%/30% e 75/50/25/10% (E1).
23. Gaps: `UpdateItem` de resposta; piedade persistida; B16 completo
    (`itemAbilityRefined` + thresholds `+25 AC/+40 dano` por cima).
24. Issues: (a criar) "pity persistida + UpdateItem"; depende só de banco/fio.
25. Fontes: L3 §3/§7, E1 `refine.rs`, `accessory.rs`, `gameplay.rs`,
    `config_gameplay.rs`, L5 B16.
26. Pendências: taxas 7662 por passo (seed vs `CompRate/Odin`); P-10 (5% vs 4%).
    Grau: CAMINHO VALIDADO / TAXA POR VERSÃO.

## Item 7 — Trade atômica por par

1. Visão geral: troca jogador↔jogador sem dup, com confirmação bilateral.
2. Objetivo: implementar o swap atômico de itens + gold entre dois `CUser/CMob`.
3. Funcionamento original: `TradingItem` (oferta) → `Trade` (estado espelhado) →
    ambos `MyCheck==1` → swap ATÔMICO (`Item[]`+gold) → `Trade(done)`; `QuitTrade
    (0x0384)` cancela ambos (`RemoveTrade`, L4 §5; L1 `MSG_Trade` 15 itens +
    `InvenPos[15]` + `TradeMoney` + `MyCheck` + `OpponentID`, `Basedef.h:2435`).
4. Regras de negócio: tocar dois inventários numa única transação por par;
   concorrência em thread única no legado, transação por par na stack nova (L4 §5).
5. Fórmulas: nenhuma.
6. Estruturas: `Item[15]`, `InvenPos[15]`, `TradeMoney int`, `MyCheck uchar`,
   `OpponentID ushort` (L1 §3.5); `Trade/QuitTrade/TradingItem/CreateMobTrade/
   SendAutoTrade/ReqTradeList` no catálogo (E1 `opcode_table.rs:97-123`).
7. Entidades: A, B, par (unidade de atomicidade).
8. NPCs/mapas: nenhum (auto-trade usa `CreateMobTrade 0x0363`, fora do escopo).
9. Requisitos: sessão/inventário persistidos (existem).
10. Dependências: `character_items` + `characters.gold` transacionais (existem).
11. Fluxos: oferta → espelho → confirmação bilateral → swap → done; cancela →
    `RemoveTrade(ambos)` (L4 §5).
12. Comportamento do cliente: coloca item/gold, confirma (`MyCheck`), pode
    cancelar (`0x0384`); B11 como padrão de ressincronia em condição ocupada.
13. Comportamento do servidor: HOJE nada (sem handler; `MSG_Trade` fora do
    `payload` por UNVERIFIED, E1 `commerce.rs:3`, `payload/mod.rs:12,293`).
14. Persistência: swap + gold numa transação; log de auditoria (invariante Cold).
15. Protocolos: `Trade/QuitTrade/TradingItem` sem layout portado; `require_ported
    (Trade)` recusa com `Unported` (E1 `payload/mod.rs:293-295`).
16. Estados: oferta → confirmado(A/B) → done/cancelado.
17. Restrições: layout UNVERIFIED (invariante 5) — sem `Trade` tipado até captura.
18. Casos especiais: desconexão no meio (rollback); item vendido/usado durante a
    oferta (revalidar slots na confirmação); gold insuficiente (recusa atômica).
19. Bugs conhecidos: nenhum mapeado no Go para trade; L4 marca dup como ponto
    crítico do par.
20. Vulnerabilidades: dup por race (dois swaps do mesmo slot); gold duplicado por
    replay; `OpponentID` forjado — validar par servidor-side, idempotência por
    oferta.
21. Testes: confirmação bilateral debita/credita uma vez; cancelamento não move;
    concorrência no mesmo slot; desconexão reverte.
22. Implementação atual: catálogo com opcodes, sem payload sem handler (E1).
23. Gaps: layout `MSG_Trade` (captura), handler atômico, `QuitTrade`, testes de
    contenção.
24. Issues: (a criar) "trade atômica por par"; bloqueada em layout (captura 08-6).
25. Fontes: L1 §3.1/§3.5/Apêndice B, L4 §5, E1 `opcode_table.rs`, `commerce.rs`,
    `payload/mod.rs`.
26. Pendências: captura do `MSG_Trade` real 7662. Grau: FLUXO PROVÁVEL /
    LAYOUT UNVERIFIED.

## Item 8 — Catálogo de ameaças B1–B17 + dup

1. Visão geral: transformar o histórico de bugs do cliente real↔Go no catálogo
   de ameaças do nosso Rust.
2. Objetivo: cumprir o §6 do `wyd2.md` (um item por ameaça com causa, impacto,
   prevenção, validação).
3. Funcionamento original: cada linha B1–B17 do L5 com causa-raiz e fix no Go
   (ouro `0xCC` B2; level+1 B5; posição exata B6 = comportamento esperado; loja/
   banco B7 parcial; cena B8/B9; `UpdateEtc` B10; slot ocupado B11; skill-gate B12;
   Elemental B13; durações B14/B15; Amunra B16; Samaritano B17 com gate `IsPlayer`
   + `DoRemoveSamaritano`).
4. Regras de negócio: cada ameaça vira linha com severidade, impacto,
   reprodutibilidade e correção proposta (rito §6 do `wyd2.md`, 10 subitens).
5. Fórmulas: as já citadas nos itens 4–6 (sem nova).
6. Estruturas: `affect 24/14`, `SkillBar`, `Cargo`, `Donate`, timers de summon
   vs buff (`idx≥MAX_USER`, L5 B17).
7. Entidades: player, mob, summon, item, sessão, canal.
8. NPCs/mapas: não aplicável (transversal).
9. Requisitos: checklist §6 hoje ❌; L5 como esqueleto.
10. Dependências: itens 2/4/6 (cena/dano/refino) como correções vinculadas.
11. Fluxos: bug → reprodução headless → teste sem rede → fix → regressão
    (rito `wyd-rust-testing`).
12. Comportamento do cliente: sintomas B1/B8/B9/B11 como oráculo (o que o 7662
    mostra quando o servidor erra).
13. Comportamento do servidor: já protege — senha/PIN Argon2id, sessão única,
    dano server-authoritative, `MissingRate` sem mutação, idempotência de grants.
14. Persistência: `pin_failures/lockout`, `delivery_queue.idempotency_key`,
    `audit_log` (E1 `0030`, `0005/0006`).
15. Protocolos: `MSG_Trade/DropItem/GetItem` (dup), `CharacterLogout` (sessão),
    `Action` (speed-hack: validar velocidade/rota, L4 §3), `Attack` (B12).
16. Estados: sessão presa/liberada; oferta/done; caixa viva/expirada.
17. Restrições: B6 não é bug (não implementar persistência de coordenada exata);
    B1 não reproduzido no cliente real (observar, não mexer).
18. Casos especiais: Samaritano em MOB = tempo de summon (gate `IsPlayer`, B17);
    Elemental só nos índices corretos + clamp 0..100 (B13).
19. Bugs conhecidos: os 17, com status (resolvidos no Go; a checar no Rust um a
    um — sem declarar cobertura sem teste).
20. Vulnerabilidades: dup item/moeda, exploit de troca/inventário/refino/combine,
    quest/XP/drop, PVP/skill, movimento/teleporte, canal/sessão, replay, race,
    timer, persistência, auth (lista §6 do `wyd2.md`, uma linha cada).
21. Testes: matriz B1–B17 contra o Rust (passa/falha/aberto); fuzz de framing
    (`Strict`); carga com bots (item 9).
22. Implementação atual: FASE-06-2 ✅ (DoS/rate-limit); checklist §6 ❌ (sem
    catálogo próprio).
23. Gaps: o catálogo em si; validação linha a linha no Rust.
24. Issues: (a criar) "catálogo de ameaças B1–B17 + §6"; sem bloqueio externo.
25. Fontes: L5 integral, L4 §5, E1 (`session.rs`, `limits/`, `delivery_queue`,
    `audit_log`), `06-matematica.md:382` (LCG).
26. Pendências: reproduzir cada B contra o Rust. Grau: ESQUELETO VALIDADO /
    COBERTURA DESCONHECIDA.

## Item 9 — Harness de replay/paridade

1. Visão geral: reproduzir exatamente o legado (RNG inclusive) fora do jogo,
   antes de qualquer carga ou bot jogável.
2. Objetivo: base objetiva para FASE-10-3 e FASE-15-3 (`crates/wyd-bots`
    inexistente; `03-progresso.md:273` confirma "harness cobre protocolo, não
    sessão jogável").
3. Funcionamento original: `parity-tests.md` do legado — golden por subsistema,
   esquema de fixture + harness de replay (Go), proxy de captura nos 3 links
   reais, paridade EXATA via LCG do MSVC, matriz de cobertura, dimensionamento
   estatístico, plano de captura (L4 README Fase 8).
4. Regras de negócio: determinismo antes de carga (mesma seed → mesma partida;
   rolls gravados → replay idêntico; `determinism.rs`, `06-matematica.md:46,103`).
5. Fórmulas: LCG MSVC como oráculo de `rand()` para drop/crítico/refino
   (`03-progresso.md:382`, `game-rules.md`).
6. Estruturas: `crates/wyd-protocol/tests/fixtures/cliente/*.json` + `captura.mjs --check` (5/5 C→S hoje;
   S→C quando 08-6 liberar); `scenario::{resolve_attack, AttackOutcome}`,
   `TICK_PHASES` (E1 `wyd-sim/src/lib.rs:26`).
7. Entidades: harness headless (sem cliente gráfico); bots com conta dedicada
   depois.
8. NPCs/mapas: `npc_definition` 6100 como massa de spawn para replay.
9. Requisitos: `make check` verde; benches `tick.rs`/`limits.rs`
   (`harness=false`, E1 `03-progresso.md:829,856`).
10. Dependências: itens 4–6 (fórmulas) como casos do harness; captura 08-6 para
    os goldens S→C.
11. Fluxos: fixture → replay → diff byte-exato (divergência = bug, nunca ajuste
    silencioso de golden); carga só após determinismo.
12. Comportamento do cliente: cliente real só abre após harness verde (rito
    `wyd-rust-testing` regra 1).
13. Comportamento do servidor: codec real, nunca reimplementado (`HEADER` 12 B,
    `pKeyWord`, `Strict`); `fetch_add` fora do build default (E1 `wyd-net`).
14. Persistência: nenhuma no harness; sanitizar fixtures (conta/senha → GOLDEN).
15. Protocolos: `captura.mjs --gen/--check`; proxy nos links C↔TM↔DB↔BI.
16. Estados: `TICK_PHASES` (rótulos hoje; `step` drena eventos — Fase 1).
17. Restrições: sem `#[ignore]`/skip; golden novo exige motivo; sem `.pcap`/logs
    versionados.
18. Casos especiais: seed diferente → resultado diferente; rolls gravados →
    replay idêntico (`03-progresso.md:577`).
19. Bugs conhecidos: B12 (config sem `Spells` escondia o caminho quebrado) —
    o harness deve carregar o catálogo de produção.
20. Vulnerabilidades: bot sem conta dedicada/rate-limit é DoS próprio
    (`limits/`, regra 5 do `wyd-rust-testing`).
21. Testes: `cargo test -p wyd-protocol -p wyd-net` (bytes); `cargo test
    --workspace`; `bench` com números; E2E Playwright 3/3 como prova Cold
    (`03-progresso.md:250,301-302`).
22. Implementação atual: harness de protocolo ✅; sessão jogável ❌;
    `wyd-bots` ⬜ (ADR-0010).
23. Gaps: goldens S→C; LCG; matriz de cobertura; bots jogáveis.
24. Issues: (a criar) "harness de replay + LCG"; base de FASE-10-3/15-3.
25. Fontes: `parity-tests.md` (via L4), `06-matematica.md`, `03-progresso.md`,
    E1 (`determinism.rs`, benches, `captura.mjs`).
26. Pendências: captura 08-6; LCG exato. Grau: DESENHO VALIDADO / OURO PENDENTE.

*Branch `develop`; árvore suja intocada; nenhum commit; nenhuma issue criada.*
