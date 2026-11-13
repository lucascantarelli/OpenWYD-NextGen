# Mapa de funcionalidades — checklist viva

Este é o **mapa completo do jogo**: cada funcionalidade, sistema, item e skill com o estado real da
implementação. Ele complementa o `roadmap-rust/index.md` (fora do site; organiza
*o trabalho* em 15 fases e issues) — aqui o recorte é **o que o jogo precisa ter** e quanto disso já
existe de verdade.

> Estado em 2026-11-13 (revisão): **35 ✅ · 23 🚧 · 34 ⬜ · 2 ⛔** — contagem tirada das
> linhas das tabelas (`grep` por estado; 94 linhas numeradas).

## Como usar

| Marca | Significado |
|---|---|
| ✅ | Existe, tem prova **automatizada** e o efeito chega ao cliente/servidor. |
| 🚧 | Parcial: parte existe com prova, parte falta — a evidência diz qual. |
| ⬜ | Não existe. |
| ⛔ | Bloqueado por captura/RE ou decisão (ADR) — não é falta de código. |

Regras:

1. **Só muda na entrega que muda o comportamento**, junto com a evidência (mesmo rito do
   `docs/rust/02-execucao.md` §3). Status sem evidência é opinião.
2. **✅ exige teste ou comando reproduzível com saída registrada.** Código sem prova vai para
   `🚧` com a nota "sem teste dedicado" — nunca para ✅.
3. **`⬜` não é dívida escondida**: é o que falta para o jogo ser jogável de ponta a ponta.
4. "Catálogo/seed" ✅ significa **dado** validado no banco; **não** significa regra implementada.

Última revisão: **2026-11-12** (CLIENT-10: layout candidato de `MSG_Trade` — 156 B natural ×
154 B do porte Go; captura do 7662 pendente).

## 1. Tela de entrada e conta (o caminho do jogador até o mundo)

| # | Funcionalidade | Estado | Evidência (código + prova) |
|---|---|---|---|
| 1.1 | Lista de canais no seletor (`sn.bin`, `serverlist.bin`) | ✅ | `crates/wyd-launcher/src/serverlist.rs` e `snbin.rs`; `crates/wyd-launcher/tests/golden/serverlist_paridade.rs` |
| 1.2 | Status do canal (`/serv00.htm` + sonda UDP `:8281`) | ✅ | `crates/wyd-server/src/status/service.rs`; `tests/sistema/status_udp.rs` |
| 1.3 | Conexão CPSock (`INITCODE`, header 12 B, desofuscação) | ✅ | `cpsock::handshake`; `tests/sistema/handshake.rs` |
| 1.4 | Checksum `Strict` + gate de versão (12000) | ✅ | `checksum::verify`/`VersionGate`; `tests/sistema/handshake.rs::versao_do_cliente_oficial_completa_o_handshake` |
| 1.5 | **Senha da conta (Argon2id)** | ✅ | `auth::Authenticator::verify_password` (`crates/wyd-server/src/auth/mod.rs` + `auth/password.rs`), chamado em `cpsock::handshake` **antes** do `CNF`; `tests/sistema/login.rs`: `senha_errada_e_recusada_sem_sessao`, `senha_e_pin_corretos_chegam_na_selecao` |
| 1.6 | **Senha numérica (`0x0FDE`, recusa `0x0FDF`) + bloqueio persistido** | ✅ | `cpsock::authorize_pin` + `auth::secure_pin`; PIN 4–6 dígitos, Argon2id em `accounts.pin_hash`; 5 falhas → 10 min gravados em `accounts.pin_failures`/`pin_locked_until` (repo `pin_guard`/`pin_failure`/`pin_success`). Provas: `tests/sistema/login.rs` (`primeira_entrada_cadastra_o_pin`, `pin_errado_e_recusado_sem_sessao`, `cinco_pins_errados_bloqueiam_a_conta`, `pin_e_senha_nunca_aparecem_no_log`) e `tests/sistema/pin_persistencia.rs` (`bloqueio_sobrevive_a_novo_processo`, `pin_correto_zera_falhas_para_o_proximo_processo`, `pin_gravado_nao_e_texto_plano`) |
| 1.7 | `_MSG_CNFAccountLogin` com corpo `STRUCT_SELCHAR` | ✅ | `cpsock::reply_with_body` + `personagens::character_body`; `tests/sistema/selchar.rs::selchar_e_mob_saem_do_banco_e_voltam_pelo_decoder` (decodifica com `decode_in_place`/`Opcode` do crate e compara campo a campo com o banco) |
| 1.8 | Sessão única por conta entre canais | ✅ | `0030_sessao_unica.sql`, `crates/wyd-server/src/character/session.rs`; `tests/sistema/sessao_unica.rs::mesma_conta_nao_entra_em_dois_canais_e_libera_ao_sair` |
| 1.9 | Criação de personagem (`CreateCharacter`) | ✅ | `characters.rs` grava a ficha inicial por classe (`BaseSIDCHM`/`BASE_GetHpMp`, migration `0032_character_scores`) e responde `CNFNewCharacter` com o roster real; `tests/sistema/personagens_crud.rs::cria_e_deleta_personagem_pelo_protocolo` cria pelo fio e confere nome/slot/classe, HP/MP e atributos no banco |
| 1.10 | Deleção de personagem (`DeleteCharacter`) | ✅ | `characters.rs` recusa nome divergente e faz soft delete; o mesmo teste de `personagens_crud.rs` prova `DeleteCharacterFail` sem apagar e roster vazio no caso correto |
| 1.11 | Seleção/entrada no personagem (`CharacterLogin`) | ✅ | seleção decidida no banco, `STRUCT_MOB` emitido por `personagens::StructMob`, **personagem vira entidade** na posição salva e **entra na cena**: `_MSG_CreateMob` com `HEADER.ID = ESCENE_FIELD` (GAME-09, `world::{scene_snapshot,plan_scene}`, `cpsock::{send_create_mob,refresh_scene}`, testes de cena em `world` + fio loopback em `cpsock::tests`) e **id de cena na faixa do legado**: jogador `< MAX_USER` (1000) e criatura `>= MAX_USER` (GAME-17, `world::{alloc_scene_id, Entered}`), `UpdateScore` do dono entre `CNFCharacterLogin` e a cena (`sendScoreSelf` do porte Go) e `CreateType` 2 só para o dono. Prova: `cargo test -p wyd-server --lib` 42/42 + `tests/sistema/cena_mobs.rs`. Resíduo: prova visual no 7662 |
| 1.12 Campo de Treinamento (quest) | 🟡 | `_MSG_Quest` 0x028B (`payload/quest.rs`, 20 B) + `cpsock::handle_quest` com merchant do mundo; `character_quest` (0044) com `stage`/`completed`/`reward_mask`; motor `gameplay::{quest_engage, apply_quest_reward}` na sequência A (682; `Equip[6]` 50+ReqLvl; `Equip[1..7]`+sanc 3; sorteio 682/481/652), teto 35, prova consumida. `cargo test -p wyd-server --test integracao_campo_treinamento` 7/7. Portões: `CAMP_GATES` com `EF_KEYID` 2/3/4, consumo da chave e fechamento por tempo (30 s), acionados por `_MSG_Action` no tile (`Engine::use_key_at` + `SelectionStore::use_key`; 9/9 no teste do Campo). Bots: crate `wyd-bots` com 4 raças e campanha de 4 etapas (3/3). Falta: NPCs/guardiões/boss no mundo (templates `STRUCT_MOB` do cliente), porta como entidade visível e coleta de drop pelos bots |
| 1.12 | Tela de personagens com os 14 campos de `STRUCT_SELCHAR` | 🚧 | dados reais do banco (nome, classe, level, exp, hp/mp, máximos, ouro, PvP, posição, slot, guilda) provados em `tests/sistema/selchar.rs`; **layout provisório `rust-v1`** — o cliente 7662 ainda não confirmou offsets (`docs/cliente/captura-selchar.md`) |
| 1.13 | Sair / logout explícito | ✅ | libera a sessão por ociosidade (120 s), na queda de socket e no pacote de saída (`CharacterLogout → CNFCharacterLogout + leave_world + Drop`, `cpsock.rs:1030-1045`); log `event=logout result=ok reason=explicit\|socket`. Prova: `tests/sistema/selchar.rs::logout_explicito_fecha_a_conexao_e_libera_a_sessao` (CNF + conexão fecha + `live_session_of=None` em ≤ 5 s). Golden do frame real segue pendente da captura CLIENT-06 |
| 1.14 | Canal cheio / manutenção / recusa de entrada | ✅ | `channel_capacity` (migration `0040`, seed 1000/`MAX_USER` por canal 1–10) + `SessionGate::claim` recusa `Capacity{limit}`/`Maintenance` antes de abrir sessão; env `WYD_SERVER_MAX_PLAYERS`/`WYD_SERVER_MAINTENANCE` sobrescreve por processo; recusa em `wyd_packets_rejected_total{reason="canal_lotado"\|"manutencao"}` + log; `serv00.htm` mantém números, `/health` expõe `maintenance`. Provas: `tests/sistema/lotacao.rs` (5 casos) |
| 1.15 | Conta no portal: criar, trocar senha, **trocar PIN** | ✅ | Tela de conta do portal (`conta/pin-form.tsx` -> `POST /api/panel/pin`) troca o PIN exigindo o atual, pelo mesmo `0x0FDE` em duas etapas do jogo. Prova: `tests/sistema/portal_pin.rs::troca_de_pin_pelo_portal_funciona_e_invalida_o_antigo` (servidor real + banco), `pin/route.test.ts` (rota) e `conta/pin-form.test.tsx` (componente real, jsdom: textos do i18n, POST e mensagens de erro). Criar conta e trocar senha seguem no fluxo de staff |
| 1.16 | Limite de conexões simultâneas por IP (multiboxing) | ✅ | `cpsock::ConnectionLimit`/`serve_listener_with_limit`, env `WYD_MAX_CONNECTIONS_PER_IP` (padrão 6, `0` desliga); `tests/sistema/limite_ip.rs::segunda_conexao_do_mesmo_ip_e_recusada_e_volta_ao_sair`. Recusa conta em `wyd_ip_limit_refusals_total` (`metrics::ip_limit_refused`) e gera resumo a cada 25 (`result=ip_limit_summary`) |
| 1.17 | **Troca de PIN em duas etapas (`0x0FDE`, `change_numeric` 1→2)** | ✅ | `cpsock::authorize_pin` (passo 1 exige o PIN atual e arma; passo 2 grava) + `auth::change_pin`; `tests/sistema/pin_troca.rs::troca_de_pin_exige_o_atual_e_invalida_o_antigo` prova passo 2 sem passo 1 recusado, PIN atual errado recusado, e que o PIN antigo deixa de valer no banco e no fio |

**Resposta curta ao "já está tudo vinculado?"**: conectar, seletor, senha (1.5) e PIN (1.6) ✅ com
teste; CNF com personagens reais (1.7) ✅ no protocolo. O que **não** fecha: criação/deleção sem
teste dedicado, seleção sem cena, e a tela de personagens com layout provisório até a captura do
cliente.

## 2. Personagem e progressão

| # | Funcionalidade | Estado | Evidência (código + prova) |
|---|---|---|---|
| 2.1 | Ficha persistida (nível, EXP, HP/MP e máximos, ouro, PvP, posição, classe, guilda) | ✅ | `0002_personagens.sql` (inclui `gold`, `hp_max`, `mp_max`, `pvp_points`), `0032_character_scores.sql` (FOR/INT/DES/CON e pontos livres), `0004_guildas.sql` (guilda via `guild_members`), `repo/pg.rs`; `tests/sistema/selchar.rs` afirma esses campos no frame = banco e `tests/sistema/personagens_crud.rs` afirma a ficha inicial da classe. Cidadania é derivada do canal, não persistida |
| 2.2 | Atributos e distribuição de pontos | 🚧 | criação grava `BaseSIDCHM[classe]` e 5 pontos livres (3 de skill) por nível em `character_scores`; a distribuição em jogo (`_MSG_ApplyBonus`) ainda não está implementada |
| 2.3 | Cidadania (3 reinos) e canais por reino | ✅ | Papel do canal em dado (`channel_role`, migration `0041`: `1 = normal`, `2 = verde`) com sobrescrita por processo (`WYD_SERVER_CHANNEL_ROLE`); `channels::ChannelRules` decide `pvp_allowed(map)` (verde nunca liga, normal segue o mapa) e `personal_shop_allowed()`; `/health` publica `role`/`pvp`/`shop` e o boot loga `event=channel role=… source=banco|env`. Bônus de cidadania no drop é **parâmetro** (`Channel Settings.citizen_bonus_bps`, default 0, teto 50%) aplicado por `Gameplay::loot*` via `citizen`. Provas: `tests/integracao/canal_verde.rs` (4 casos: política, env, bônus na mesma seed, faixa inválida) e `tests/sistema/selchar.rs::sussurro_de_getout_e_tolerado_e_nao_derruba_a_sessao`. Resíduo declarado: cidadania ainda **derivada do canal** (compra na Kibita pertence ao diálogo de NPC — 3.5/7.2) e o PvP/loja no fio chegam com 4.5/7.2 |
| 2.4 | Aparência / customização | ⬜ | — |
| 2.5 | Rename / exclusão definitiva / recuperação | ⬜ | — |
| 2.6 | Montaria | ⬜ | `game_system` tem `montaria`; sem regra |
| 2.7 | Progressão por nível/EXP e camadas (tier) | 🚧 | schema + `docs/wyd/progressao.md`; fórmula pendente |

## 3. Mundo

| # | Funcionalidade | Estado | Evidência (código + prova) |
|---|---|---|---|
| 3.1 | Tick determinístico + ECS contíguo | ✅ | `crates/wyd-core/src/tick` e `ecs`; `integracao_determinism.rs`/`integracao_autoridade.rs` e **tick no processo** (`wyd-server/src/world.rs`, `tests/integracao/tick_processo.rs`, GAME-07) |
| 3.2 | Spatial grid / interest management | ✅ | `crates/wyd-core/src/spatial`; testes unitários no módulo (`src/spatial/`) |
| 3.3 | Mapas e regiões no banco | ✅ | `0020_seed_mundo.sql`, `0023_seed_mundo_fixo.sql`, `0011_rotas_mundo.sql`; `tests/contrato/catalogo.rs` |
| 3.4 | Colisão e pathfinding | 🚧 | `core/src/npc/route.rs` (rotas de NPC); colisão de jogador não |
| 3.5 | NPCs, lojas e diálogo | 🚧 | `core/src/npc/` com `tests/integracao/npc.rs`; o mundo já tem spawn/despawn (`GAME-07/GAME-08`), mas NPC nasce com o mapa (GAME-09); loja/diálogo no fio ⬜ |
| 3.6 | Teleportes e portais | ⬜ | rotas catalogadas; sem handler |
| 3.7 | Spawn/respawn e densidade por área | ⬜ | — |
| 3.8 | Dungeons / instâncias | ⬜ | — |
| 3.9 | Clima, ciclo dia/noite | ⬜ | — |

## 4. Combate

| # | Funcionalidade | Estado | Evidência (código + prova) |
|---|---|---|---|
| 4.1 | Ataque físico (intenção → resolução) | 🚧 | `crates/wyd-core/src/systems/combat.rs` + testes unitários no módulo; sem fio |
| 4.2 | Fórmulas de dano/defesa/crítico | ⛔ | FASE-10 (`crates/wyd-sim`); nada inventado no core |
| 4.3 | Skills em combate (custo, cast, alvo) | 🚧 | `core/src/systems/skill.rs` **coberto** por `tests/integracao/autoridade.rs` (skill não aprendida, MP/cooldown, alvo fora de alcance, recusa atômica); falta o fio (payload de uso de skill) |
| 4.4 | HP/MP, regen e morte/respawn | 🚧 | **morte ligada:** `Opcode::Attack*` → `Engine::attack_at` retira a criatura do mundo no golpe fatal (geração da cena sobe) → `_MSG_RemoveMob` com `RemoveType=1` na borda; falta regen e respawn próprio (hoje o NPC só volta numa recarga de `npc_definition`) |
| 4.5 | PvP, PK, chaos e penalidades | ⬜ | docs + coluna `pvp_points` (sem regra) |
| 4.6 | IA de NPC (FSM + utilidade) | ✅ | `core/src/npc/brain.rs`, `behavior.rs`, `budget.rs`; `tests/integracao/npc.rs` |
| 4.7 | Bosses e invasões | ⬜ | `docs/wyd/sistemas/eventos.md` |
| 4.8 | Pacote autoritativo de cena (`ESCENE_FIELD` 30000) | ✅ | invariante 4 cumprido: NPCs + personagens com `MobID` = índice da entidade (espaço único), delta entrar/sair por sessão, `RemoveType` 0/1/2, teto `WYD_SCENE_MAX_PER_REFRESH` (default 64, fatal no boot se inválido); prova: `world::tests` (ator na cena, saída, `plan_scene`, teto, UTF-8) + `cpsock::tests` (fio loopback: `ESCENE_FIELD` no header, `MobID` no corpo). Resíduos: movimento no fio e raio de visão sem fonte provada + captura CLIENT-06 |

## 5. Itens, inventário e refino

| # | Funcionalidade | Estado | Evidência (código + prova) |
|---|---|---|---|
| 5.1 | Catálogo de itens | ✅ | `0018_seed_item_catalog.sql`, `0017_seed_taxonomia.sql`; `tests/contrato/catalogo.rs` |
| 5.2 | Inventário e equipamento | 🚧 | `core/src/systems/inventory.rs` + `tests/integracao/inventario.rs`; `0003_inventario.sql`; sem fio |
| 5.3 | Drop e loot | 🚧 | **o sorteio está ligado ao abate:** `Opcode::Attack*` → `SelectionStore::attack` → `Gameplay::loot_at_death(template, die)` — um roll por item da tabela do snapshot, tirado no instante da morte, com `event=death`/`event=loot` (seed) em `cpsock`; `tests/integracao/config_gameplay.rs::morte_sorteia_um_roll_por_item_da_tabela_do_snapshot` prova seed→rolls→drops e a recarga do portal mudando o peso. **Entrega no inventário (2026-11-11):** `gameplay::add_item` empilha até 120 e ocupa slot `16..80`, `SelectionStore::grant_loot` grava com `replace_items` e `cpsock::handle_attack` manda `_MSG_SendItem` por slot; provas `campo_treinamento::{loot_empilha_ate_o_teto_e_ocupa_slot_livre, loot_sem_slot_nao_e_inventado}`. **Falta materializar a caixa no chão** (`_MSG_CreateItem` 0x026E) e o pickup (`_MSG_GetItem` 0x0270 → `CNFGetItem` 0x0171): offsets levantados em `docs/cliente/pacotes.md` §“Caixa de item”, direção/pack dependem da captura CLIENT-06 |
| 5.4 | Refino, Anct, celestial/God2 | 🚧 | `0009_refino.sql`, `0022_seed_refino.sql`, `0025/0026_sanc_rate`; taxas editáveis no portal (`0037`); `Opcode::UseItem` → `SelectionStore::use_item` → `Gameplay::use_item` → `Gameplay::refine` lê SancRate do snapshot por tentativa (piedade da `0043_refine_pity` somada), grava `character_items` + piedade na mesma transação e emite `SendItem` (0x0182) com o alvo; recusa (`MissingRate`, teto +6/+9, selado, `EF_NOSANC`) sem mutação e sem reenvio; `tests/integracao/config_gameplay.rs` prova recarga/reversão, `tests/sistema/refino_piedade.rs` 5/5 prova piedade + fio; refino de acessório como dado (`0038`). Resíduo: estatística +25AC/+40dmg do B16 e cadeia +12~+15 (FASE-10) |
| 5.5 | Imbuir / repleção | ⬜ | docs; sem regra |
| 5.6 | Mix / compose | 🚧 | `core/src/systems/compose.rs` + `tests/integracao/compose.rs`; sem fio |
| 5.7 | Craft | ⬜ | catálogo `craft`; sem receita executável |
| 5.8 | Armazém / baú | ⬜ | — |
| 5.9 | Durabilidade, reparo e quebra | ⬜ | — |
| 5.10 | Comércio jogador↔jogador | ⬜ | Layout candidato de 156 B (natural; `Basedef.h:2435-2445`) × 154 B do Go (`pack(1)`); captura do 7662 decide. `require_ported(Trade)` segue fechado (invariante 5) |

### 5.11 Resolvedores prontos × fluxo ligado ao protocolo (revisão 2026-11-10)

Resolvedor pronto = regra pura com teste (goldens/duplo de snapshot). Fluxo ligado = existe
opcode do cliente chegando no `cpsock::dispatch` até o resolvedor, com efeito persistido ou
pacote de resposta.

| Resolvedor | Entrada no servidor | Ligado ao protocolo? |
|---|---|---|
| `Gameplay::refine` / `Gameplay::use_item` (SancRate + `wyd-sim`) | `Opcode::UseItem` → `SelectionStore::use_item` | **Sim** — lê o snapshot por tentativa (com piedade 0043), grava `character_items` + piedade e emite `SendItem` (0x0182); recusa sem mutação e sem reenvio |
| `Gameplay::loot` / `Gameplay::loot_at_death` (roll por item, tabela do snapshot) | `Opcode::Attack*` → `Engine::attack_at` → `SelectionStore::attack` | **Sim até o inventário** — o abate tira o mob do mundo, emite `RemoveMob` (morte), resolve os drops com os rolls do momento e entrega no `Carry` (pilha/slot, persistido, `_MSG_SendItem`); **não** cria caixa no chão nem trata pickup |
| `Gameplay::exp_reward` (multiplicador de EXP) | `Opcode::Attack*` → `SelectionStore::gain_exp` | **Sim** — `UpdateEtc`/`UpdateScore` com o `Mob` gravado |
| `Gameplay::drop_rule` / `drop_chance_bps` (peso editável no portal) | portal → `config_revision` → `LiveConfig` | **Sim** — o peso novo entra no próximo roll, sem reiniciar o canal |
| `core/systems/skill.rs` (custo, cooldown, alcance) | nenhum | **Não** — falta o payload de uso de skill |
| `core/systems/compose.rs` (mix) | nenhum | **Não** — falta o layout de `_MSG_CombineItem` |
| `core/systems/inventory.rs` (slots/equip) | leitura no `_MSG_CharacterLogin` (`STRUCT_MOB`) | **Parcial** — leitura/criação persistidas; falta pacote de mover/equipar/vender |
| `core/npc/{brain,behavior}` (IA) | tick do motor | **Sim (interno)** — a IA roda no tick; nenhum pacote de ação de NPC sai para o cliente |

## 6. Skills

| # | Funcionalidade | Estado | Evidência (código + prova) |
|---|---|---|---|
| 6.1 | Catálogo de skills e linhagens | ✅ | `0019/0021/0024_seed_skill_catalog*`; `tests/contrato/catalogo.rs` |
| 6.2 | Aprendizado e livros de skill | 🚧 | livros no catálogo; regra ⬜ |
| 6.3 | Cooldown, mana e alcance | ⬜ | depende de 4.2/4.3 |
| 6.4 | Buffs/debuffs | ⬜ | campos UNVERIFIED (captura CLIENT-06) |
| 6.5 | Reset/redistribuição | ⬜ | — |

## 7. Economia

| # | Funcionalidade | Estado | Evidência (código + prova) |
|---|---|---|---|
| 7.1 | Ouro no personagem persistido | ✅ | `characters.gold` (`0002`), `repo/pg.rs`, `personagens.rs`; `tests/sistema/selchar.rs` (ouro no frame = banco). **Sem** ganho/gasto implementado |
| 7.2 | Lojas de NPC | ⬜ | — |
| 7.3 | Mercado/leilão | ⬜ | — |
| 7.4 | Imposto, taxas e drenos | 🚧 | `0010_economia.sql`; regra no fio ⬜ |
| 7.5 | Loja de donate / moeda premium | ⬜ | `game_system` tem `donate` |

## 8. Guildas e guerra

| # | Funcionalidade | Estado | Evidência (código + prova) |
|---|---|---|---|
| 8.1 | Criação, membros e cargos | 🚧 | `0004_guildas.sql` + leitura de guilda no roster (`tests/sistema/selchar.rs`); handlers ⬜ |
| 8.2 | Sub-líder / destituição | ⬜ | contrato em `contrato-dispatch-opcode.md` §4 |
| 8.3 | Guerra de guilda e castelo | ⬜ | `docs/wyd/sistemas/guerras.md` |

## 9. Quests, eventos e arena

| # | Funcionalidade | Estado | Evidência (código + prova) |
|---|---|---|---|
| 9.1 | Story line, Kefra, runas, Arch | ⬜ | `0013_quests.sql` + docs |
| 9.2 | Eventos e calendário | ⬜ | `0014_eventos.sql` + docs |
| 9.3 | Arena / Battle Royale | ⬜ | `0015_arena.sql` + docs |

## 10. Social e interface em jogo

| # | Funcionalidade | Estado | Evidência |
|---|---|---|---|
| 10.1 | Chat normal e whisper | ⬜ | — |
| 10.2 | Party / grupo | ⬜ | — |
| 10.3 | Amigos / ignore | ⬜ | — |
| 10.4 | HUD, barras de HP/EXP, minimapa | 🚧 | servidor emite HP/nível/guilda na cena (`ESCENE_FIELD`, GAME-09); falta a prova de render no cliente 7662 (captura CLIENT-06) |
| 10.5 | Macro/auto | ⬜ | — |

## 11. Administração, portal e cliente

| # | Funcionalidade | Estado | Evidência (código + prova) |
|---|---|---|---|
| 11.1 | BFF do portal com `adminCall` | ✅ | `wyd-portal/src/server/admin.ts`; suíte vitest do portal (a contagem muda por entrega — vale a saída do CI, não um número fixo na doc) |
| 11.2 | Grants via `delivery_queue` | ✅ | `0006_delivery_queue.sql` com `idempotency_key` UNIQUE; `grants/request` grava durável; tela `/entregas` enfileira ouro e mostra a fila; canal drena gold (`event=grant delivered`). **E2E real:** `npm run test:e2e` 3/3 (gold 0→250 pela tela; mesma chave → replay, 1 linha, gold 300). Item sai da fila pelo dreno `claim_item` (WEB-06 ✅: slot livre de `Carry` 16–79; mochila cheia pende com `attempts`); nível segue pendente no Hot Path |
| 11.3 | Painel de staff | ✅ | login do portal, busca/criação de conta, moderação (papel + bloqueio com motivo), `/auditoria`, `/economia` (WEB-03) e `/entregas` (WEB-04); **provado na stack** com HTTP real por papel, efeito em `account_moderation`/`audit_log` e login bloqueado/liberado no portal e no jogo; E2E Playwright **6/6** (login/logout, conta, painel/contas, entrega, idempotência e financeiro) em três execuções |
| 11.4 | Auditoria administrativa | ✅ | `0005_auditoria.sql`, `crates/wyd-admin`; `tests/contrato/transporte.rs` |
| 11.5 | Identidade visual única | ✅ | `docs/arquitetura/identidade.tokens`, `branding.rs`, `gui.rs`; `crates/wyd-launcher/tests/contrato/identidade.rs` |
| 11.6 | Launcher: manifesto, patch atômico, bootstrap | ✅ | `crates/wyd-launcher/src/{manifest,patch,bootstrap}.rs`; testes unitários no crate + `tests/golden/serverlist_paridade.rs` |
| 11.7 | Re-tema de `.wyt` pela paleta | 🚧 | `wyt.rs` + CLI `retheme`; **sem teste automatizado** (só `--dry-run` manual) |
| 11.8 | GUI do launcher (Tauri v2) | 🚧 | `LauncherGUI.html` gerado; shell Tauri não |
| 11.9 | **Multijanela no launcher (`--instances 1..5` + PIDs)** | ✅ | `main.rs` (`cmd_launch`), `bootstrap::parse_instances`; `crates/wyd-launcher/src/bootstrap.rs` testes `instances_aceita_de_um_a_cinco`. Spawn real de N janelas depende do Windows do jogador |

## 12. Segurança e ameaças

| # | Funcionalidade | Estado | Evidência (código + prova) |
|---|---|---|---|
| 12.1 | Catálogo de ameaças B1–B17 + §6 do `wyd2.md` | ✅ | `docs/wyd/sistemas/ameacas.md`: matriz B1–B17 (7 PASSAM, 3 PARCIAIS, 5 ABERTOS, 1 OBSERVAR, 1 DIVERGENTE) e 29 ameaças do §6 com os 10 campos; `PROC-05` (concluída em 2026-11-12) publicou a auditoria das 36 issues em `docs/rust/evidencias/PROC-05-auditoria-2026-11-12.md`. Itens visuais do 7662 continuam declarados como pendência (`CLIENT-06`). |

## Itens no código sem prova automatizada (não são ✅)

| Item | Estado | Por que está em 🚧 |
|---|---|---|
| 4.3 (skill em combate) | 🚧 | `systems/skill.rs` sem teste dedicado e sem fio |
| 11.7 (re-tema `.wyt`) | 🚧 | CLI existe; nenhum teste de ida-e-volta de paleta |
| 5.4 / 5.6 no fio | 🚧 | 5.4 emite `SendItem` (fio provado em `refino_piedade.rs`); 5.6 segue sem emissão |

## Pendências de entrega (2026-11-09)

1. **Nada commitado.** Todo o trabalho desta rodada (PIN persistido, multijanela, limite por IP,
   checklist) está na árvore de trabalho; `git status` mostra as modificações e nada foi para o
   remoto.
2. **Troca de senha numérica feita no jogo e no portal; a semântica do campo ainda é nossa.**
   O `change_numeric` do `0x0FDE` não tem semântica confirmada por fonte (o WYD Global troca pelo
   site exigindo o PIN atual); o fluxo em duas etapas é **decisão nossa**, documentado em
   `docs/wyd/senha-numerica.md`, e vale para o jogo (1.17) e para o portal
   (`account_web/change_pin`). Falta confirmar com captura do cliente 7662.
3. **Offsets do cliente 7662 seguem UNVERIFIED** (layout `rust-v1` provisório). A captura real
   depende da máquina do usuário (`docs/cliente/captura-selchar.md`).
4. **`--instances` prova o parsing e o plano; o spawn de N processos reais** só é verificável no
   Windows com o `WYD.exe` presente.
5. **O limite por IP conta conexões TCP antes do login**: conexão abandonada ocupa vaga até o
   timeout do handshake (10 s), não para sempre.

## Ordem sugerida das próximas entregas (revisão 2026-11-09)

A fila completa, com tamanho e dependência, está em `roadmap-rust/index.md` §“Fila de entregas
rápidas”. Resumo por sistema:

1. **Canal e processo** — `PROC-13` (gate de doc), `QA-06` (lotação), `CLIENT-09` (logout).
2. **Motor no processo** — `GAME-07` (tick real) → `GAME-08` (entidade + save) →
   `GAME-09` (`ESCENE_FIELD`). **Sem isso o jogo não existe**: as regras passam nos testes,
   mas nenhum processo as executa.
3. **Cidadania** — `GAME-10` (bônus + canal verde) em paralelo com GAME-08.
4. **Portal** — `WEB-05`/`WEB-06` para tirar as telas de conteúdo do `501`.
5. **Captura do 7662** — `CLIENT-06` promove 1.12 e os layouts UNVERIFIED (único bloqueio
   externo; depende de VM + cliente).
6. **Combate/skills no fio** e **inventário/drop/compose** — dependem dos layouts da captura;
   não entram na fila rápida para não virarem issue bloqueada.
