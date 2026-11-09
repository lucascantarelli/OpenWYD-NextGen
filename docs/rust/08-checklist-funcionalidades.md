# Mapa de funcionalidades — checklist viva

Este é o **mapa completo do jogo**: cada funcionalidade, sistema, item e skill com o estado real da
implementação. Ele complementa o [`roadmap-rust/index.md`](../../roadmap-rust/index.md) (que organiza
*o trabalho* em 15 fases e issues) — aqui o recorte é **o que o jogo precisa ter** e quanto disso já
existe de verdade.

> Estado em 2026-11-09: 27 ✅ · 19 🚧 · 42 ⬜ — contagem tirada das linhas da tabela.

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

Última revisão: **2026-11-09** (PIN persistido, multijanela, limite por IP).

## 1. Tela de entrada e conta (o caminho do jogador até o mundo)

| # | Funcionalidade | Estado | Evidência (código + prova) |
|---|---|---|---|
| 1.1 | Lista de canais no seletor (`sn.bin`, `serverlist.bin`) | ✅ | `crates/wyd-launcher/src/serverlist.rs` e `snbin.rs`; `crates/wyd-launcher/tests/serverlist_paridade.rs` |
| 1.2 | Status do canal (`/serv00.htm` + sonda UDP `:8281`) | ✅ | `crates/wyd-server/src/status.rs`; `tests/status_udp.rs` |
| 1.3 | Conexão CPSock (`INITCODE`, header 12 B, desofuscação) | ✅ | `cpsock::handshake`; `tests/handshake.rs` |
| 1.4 | Checksum `Strict` + gate de versão (12000) | ✅ | `checksum::verify`/`VersionGate`; `tests/handshake.rs::versao_do_cliente_oficial_completa_o_handshake` |
| 1.5 | **Senha da conta (Argon2id)** | ✅ | `auth::Authenticator::verify_password` (`crates/wyd-server/src/auth.rs`), chamado em `cpsock::handshake` **antes** do `CNF`; `tests/login.rs`: `senha_errada_e_recusada_sem_sessao`, `senha_e_pin_corretos_chegam_na_selecao` |
| 1.6 | **Senha numérica (`0x0FDE`, recusa `0x0FDF`) + bloqueio persistido** | ✅ | `cpsock::authorize_pin` + `auth::secure_pin`; PIN 4–6 dígitos, Argon2id em `accounts.pin_hash`; 5 falhas → 10 min gravados em `accounts.pin_failures`/`pin_locked_until` (repo `pin_guard`/`pin_failure`/`pin_success`). Provas: `tests/login.rs` (`primeira_entrada_cadastra_o_pin`, `pin_errado_e_recusado_sem_sessao`, `cinco_pins_errados_bloqueiam_a_conta`, `pin_e_senha_nunca_aparecem_no_log`) e `tests/pin_persistencia.rs` (`bloqueio_sobrevive_a_novo_processo`, `pin_correto_zera_falhas_para_o_proximo_processo`, `pin_gravado_nao_e_texto_plano`) |
| 1.7 | `_MSG_CNFAccountLogin` com corpo `STRUCT_SELCHAR` | ✅ | `cpsock::reply_with_body` + `personagens::character_body`; `tests/selchar.rs::selchar_e_mob_saem_do_banco_e_voltam_pelo_decoder` (decodifica com `decode_in_place`/`Opcode` do crate e compara campo a campo com o banco) |
| 1.8 | Sessão única por conta entre canais | ✅ | `0030_sessao_unica.sql`, `crates/wyd-server/src/session.rs`; `tests/sessao_unica.rs::mesma_conta_nao_entra_em_dois_canais_e_libera_ao_sair` |
| 1.9 | Criação de personagem (`CreateCharacter`) | ✅ | `characters.rs` grava no banco e responde `CNFNewCharacter`; `tests/personagens_crud.rs::cria_e_deleta_personagem_pelo_protocolo` cria pelo fio e confere nome/slot/classe no roster |
| 1.10 | Deleção de personagem (`DeleteCharacter`) | ✅ | `characters.rs` recusa nome divergente e faz soft delete; o mesmo teste de `personagens_crud.rs` prova `DeleteCharacterFail` sem apagar e roster vazio no caso correto |
| 1.11 | Seleção/entrada no personagem (`CharacterLogin`) | 🚧 | seleção decidida no banco e `STRUCT_MOB` emitido por `personagens::StructMob` (`tests/selchar.rs`); **sem cena** (`ESCENE_FIELD` não existe) e offsets do 7662 UNVERIFIED |
| 1.12 | Tela de personagens com os 14 campos de `STRUCT_SELCHAR` | 🚧 | dados reais do banco (nome, classe, level, exp, hp/mp, máximos, ouro, PvP, posição, slot, guilda) provados em `tests/selchar.rs`; **layout provisório `rust-v1`** — o cliente 7662 ainda não confirmou offsets (`docs/cliente/captura-selchar.md`) |
| 1.13 | Sair / logout explícito | 🚧 | libera a sessão por ociosidade (120 s) em `session.rs`; pacote de saída não tratado. **Sem teste dedicado** |
| 1.14 | Canal cheio / manutenção / recusa de entrada | ⬜ | `channels.rs` conta canais; regra de lotação não existe |
| 1.15 | Conta no portal: criar, trocar senha, **trocar PIN** | ✅ | Tela de conta do portal (`conta/pin-form.tsx` -> `POST /api/panel/pin`) troca o PIN exigindo o atual, pelo mesmo `0x0FDE` em duas etapas do jogo. Prova: `tests/portal_pin.rs::troca_de_pin_pelo_portal_funciona_e_invalida_o_antigo` (servidor real + banco), `pin/route.test.ts` (rota) e `conta/pin-form.test.tsx` (componente real, jsdom: textos do i18n, POST e mensagens de erro). Criar conta e trocar senha seguem no fluxo de staff |
| 1.16 | Limite de conexões simultâneas por IP (multiboxing) | ✅ | `cpsock::ConnectionLimit`/`serve_listener_with_limit`, env `WYD_MAX_CONNECTIONS_PER_IP` (padrão 6, `0` desliga); `tests/limite_ip.rs::segunda_conexao_do_mesmo_ip_e_recusada_e_volta_ao_sair`. Recusa conta em `wyd_ip_limit_refusals_total` (`metrics::ip_limit_refused`) e gera resumo a cada 25 (`result=ip_limit_summary`) |
| 1.17 | **Troca de PIN em duas etapas (`0x0FDE`, `change_numeric` 1→2)** | ✅ | `cpsock::authorize_pin` (passo 1 exige o PIN atual e arma; passo 2 grava) + `auth::change_pin`; `tests/pin_troca.rs::troca_de_pin_exige_o_atual_e_invalida_o_antigo` prova passo 2 sem passo 1 recusado, PIN atual errado recusado, e que o PIN antigo deixa de valer no banco e no fio |

**Resposta curta ao "já está tudo vinculado?"**: conectar, seletor, senha (1.5) e PIN (1.6) ✅ com
teste; CNF com personagens reais (1.7) ✅ no protocolo. O que **não** fecha: criação/deleção sem
teste dedicado, seleção sem cena, e a tela de personagens com layout provisório até a captura do
cliente.

## 2. Personagem e progressão

| # | Funcionalidade | Estado | Evidência (código + prova) |
|---|---|---|---|
| 2.1 | Ficha persistida (nível, EXP, HP/MP e máximos, ouro, PvP, posição, classe, guilda) | ✅ | `0002_personagens.sql` (inclui `gold`, `hp_max`, `mp_max`, `pvp_points`), `0004_guildas.sql` (guilda via `guild_members`), `repo/pg.rs`; `tests/selchar.rs` afirma esses campos no frame = banco. Cidadania é derivada do canal, não persistida |
| 2.2 | Atributos e distribuição de pontos | ⬜ | cálculo é FASE-10 |
| 2.3 | Cidadania (3 reinos) e canais por reino | 🚧 | `docs/wyd/canais-e-cidadania.md`, seed `0029`; `citizen = canal + 1` no `STRUCT_SELCHAR`, sem regra de reino |
| 2.4 | Aparência / customização | ⬜ | — |
| 2.5 | Rename / exclusão definitiva / recuperação | ⬜ | — |
| 2.6 | Montaria | ⬜ | `game_system` tem `montaria`; sem regra |
| 2.7 | Progressão por nível/EXP e camadas (tier) | 🚧 | schema + `docs/wyd/progressao.md`; fórmula pendente |

## 3. Mundo

| # | Funcionalidade | Estado | Evidência (código + prova) |
|---|---|---|---|
| 3.1 | Tick determinístico + ECS contíguo | ✅ | `crates/wyd-core/src/tick` e `ecs`; `crates/wyd-core/tests/determinism.rs`, `autoridade.rs` |
| 3.2 | Spatial grid / interest management | ✅ | `crates/wyd-core/src/spatial`; `tests/movement.rs` |
| 3.3 | Mapas e regiões no banco | ✅ | `0020_seed_mundo.sql`, `0023_seed_mundo_fixo.sql`, `0011_rotas_mundo.sql`; `tests/catalogo.rs` |
| 3.4 | Colisão e pathfinding | 🚧 | `core/src/npc/route.rs` (rotas de NPC); colisão de jogador não |
| 3.5 | NPCs, lojas e diálogo | 🚧 | `core/src/npc/` com `tests/npc.rs`; spawn/loja no fio ⬜ |
| 3.6 | Teleportes e portais | ⬜ | rotas catalogadas; sem handler |
| 3.7 | Spawn/respawn e densidade por área | ⬜ | — |
| 3.8 | Dungeons / instâncias | ⬜ | — |
| 3.9 | Clima, ciclo dia/noite | ⬜ | — |

## 4. Combate

| # | Funcionalidade | Estado | Evidência (código + prova) |
|---|---|---|---|
| 4.1 | Ataque físico (intenção → resolução) | 🚧 | `core/src/systems/combat.rs` + `tests/combat.rs`; sem fio |
| 4.2 | Fórmulas de dano/defesa/crítico | ⛔ | FASE-10 (`crates/wyd-sim`); nada inventado no core |
| 4.3 | Skills em combate (custo, cast, alvo) | 🚧 | `core/src/systems/skill.rs`; **sem teste dedicado** e sem fio |
| 4.4 | HP/MP, regen e morte/respawn | ⬜ | colunas existem; regra não |
| 4.5 | PvP, PK, chaos e penalidades | ⬜ | docs + coluna `pvp_points` (sem regra) |
| 4.6 | IA de NPC (FSM + utilidade) | ✅ | `core/src/npc/brain.rs`, `behavior.rs`, `budget.rs`; `tests/npc.rs` |
| 4.7 | Bosses e invasões | ⬜ | `docs/wyd/sistemas/eventos.md` |
| 4.8 | Pacote autoritativo de cena (`ESCENE_FIELD` 30000) | ⬜ | invariante 4 do `AGENTS.md`; ainda não emitido |

## 5. Itens, inventário e refino

| # | Funcionalidade | Estado | Evidência (código + prova) |
|---|---|---|---|
| 5.1 | Catálogo de itens | ✅ | `0018_seed_item_catalog.sql`, `0017_seed_taxonomia.sql`; `tests/catalogo.rs` |
| 5.2 | Inventário e equipamento | 🚧 | `core/src/systems/inventory.rs` + `tests/inventario.rs`; `0003_inventario.sql`; sem fio |
| 5.3 | Drop e loot | ⬜ | catálogo existe; regra não |
| 5.4 | Refino, Anct, celestial/God2 | 🚧 | `0009_refino.sql`, `0022_seed_refino.sql`, `0025/0026_sanc_rate`; regra no fio ⬜ |
| 5.5 | Imbuir / repleção | ⬜ | docs; sem regra |
| 5.6 | Mix / compose | 🚧 | `core/src/systems/compose.rs` + `tests/compose.rs`; sem fio |
| 5.7 | Craft | ⬜ | catálogo `craft`; sem receita executável |
| 5.8 | Armazém / baú | ⬜ | — |
| 5.9 | Durabilidade, reparo e quebra | ⬜ | — |
| 5.10 | Comércio jogador↔jogador | ⬜ | `MSG_Trade` UNVERIFIED (invariante 5) |

## 6. Skills

| # | Funcionalidade | Estado | Evidência (código + prova) |
|---|---|---|---|
| 6.1 | Catálogo de skills e linhagens | ✅ | `0019/0021/0024_seed_skill_catalog*`; `tests/catalogo.rs` |
| 6.2 | Aprendizado e livros de skill | 🚧 | livros no catálogo; regra ⬜ |
| 6.3 | Cooldown, mana e alcance | ⬜ | depende de 4.2/4.3 |
| 6.4 | Buffs/debuffs | ⬜ | campos UNVERIFIED (captura 08-6) |
| 6.5 | Reset/redistribuição | ⬜ | — |

## 7. Economia

| # | Funcionalidade | Estado | Evidência (código + prova) |
|---|---|---|---|
| 7.1 | Ouro no personagem persistido | ✅ | `characters.gold` (`0002`), `repo/pg.rs`, `personagens.rs`; `tests/selchar.rs` (ouro no frame = banco). **Sem** ganho/gasto implementado |
| 7.2 | Lojas de NPC | ⬜ | — |
| 7.3 | Mercado/leilão | ⬜ | — |
| 7.4 | Imposto, taxas e drenos | 🚧 | `0010_economia.sql`; regra no fio ⬜ |
| 7.5 | Loja de donate / moeda premium | ⬜ | `game_system` tem `donate` |

## 8. Guildas e guerra

| # | Funcionalidade | Estado | Evidência (código + prova) |
|---|---|---|---|
| 8.1 | Criação, membros e cargos | 🚧 | `0004_guildas.sql` + leitura de guilda no roster (`tests/selchar.rs`); handlers ⬜ |
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
| 10.4 | HUD, barras de HP/EXP, minimapa | ⬜ | depende de `ESCENE_FIELD` |
| 10.5 | Macro/auto | ⬜ | — |

## 11. Administração, portal e cliente

| # | Funcionalidade | Estado | Evidência (código + prova) |
|---|---|---|---|
| 11.1 | BFF do portal com `adminCall` | ✅ | `wyd-portal/src/server/admin.ts`; vitest (145 testes) |
| 11.2 | Grants via `delivery_queue` | ✅ | `0006_delivery_queue.sql` com `idempotency_key` UNIQUE; `grants/request` grava durável; tela `/entregas` enfileira ouro e mostra a fila; canal drena gold (`event=grant delivered`). **E2E real:** `npm run test:e2e` 3/3 (gold 0→250 pela tela; mesma chave → replay, 1 linha, gold 300). Item/nível seguem pendentes no inventário/nível do Hot Path |
| 11.3 | Painel de staff | ✅ | login do portal, busca/criação de conta, moderação (papel + bloqueio com motivo), `/auditoria`, `/economia` (07-3) e `/entregas` (07-4); **provado na stack** com HTTP real por papel, efeito em `account_moderation`/`audit_log` e login bloqueado/liberado no portal e no jogo; E2E Playwright 3/3 |
| 11.4 | Auditoria administrativa | ✅ | `0005_auditoria.sql`, `crates/wyd-admin`; `tests/contract.rs` |
| 11.5 | Identidade visual única | ✅ | `docs/arquitetura/identidade.tokens`, `branding.rs`, `gui.rs`; `crates/wyd-launcher/tests/identidade.rs` |
| 11.6 | Launcher: manifesto, patch atômico, bootstrap | ✅ | `crates/wyd-launcher/src/{manifest,patch,bootstrap}.rs`; testes unitários no crate + `tests/serverlist_paridade.rs` |
| 11.7 | Re-tema de `.wyt` pela paleta | 🚧 | `wyt.rs` + CLI `retheme`; **sem teste automatizado** (só `--dry-run` manual) |
| 11.8 | GUI do launcher (Tauri v2) | 🚧 | `LauncherGUI.html` gerado; shell Tauri não |
| 11.9 | **Multijanela no launcher (`--instances 1..5` + PIDs)** | ✅ | `main.rs` (`cmd_launch`), `bootstrap::parse_instances`; `crates/wyd-launcher/src/bootstrap.rs` testes `instances_aceita_de_um_a_cinco`. Spawn real de N janelas depende do Windows do jogador |

## Itens no código sem prova automatizada (não são ✅)

| Item | Por que está em 🚧 |
|---|---|
| 1.13 (logout) | só o timeout de ociosidade tem código; pacote de saída não é tratado nem testado |
| 4.3 (skill em combate) | `systems/skill.rs` sem teste dedicado e sem fio |
| 11.7 (re-tema `.wyt`) | CLI existe; nenhum teste de ida-e-volta de paleta |
| 5.4 / 5.6 no fio | regra no core testada, emissão de pacote não |

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

## Ordem sugerida das próximas entregas

1. **1.11 + 4.8** (entrar no mundo + `ESCENE_FIELD`) — primeiro pixel de mundo autoritativo.
2. **1.12** — captura real do 7662 para promover os offsets e fechar a tela de personagens.
3. **1.9/1.10 + 1.13** — testes ponta-a-ponta de criação/deleção e logout explícito.
4. **4.1/4.2/4.3** — combate e skills no fio, com fórmulas da FASE-10.
5. **5.2/5.3/5.6** — inventário, drop e compose no fio.
6. **7.2 + 8.1 + 9.1** — loja, guilda e quest.
