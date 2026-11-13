# Missão wyd2.md — Fase 1: diagnóstico e imersão (2026-11-10)

> Primeira etapa obrigatória da missão `wyd2.md` (§15): imersão completa antes de
> criar issues. Nada aqui vira regra de código por existir neste arquivo.
> Promoção segue o rito: fonte → `game_setting`/catálogo com `content_source` → teste.
> Níveis de confiança: `docs/wyd/fontes.md` §1 (A oficial · B oficial de servidor ·
> C privado com guia · D comunidade · E dado do repositório).

## 1. O que foi efetivamente investigado (com evidência aberta)

### 1.1 Referências obrigatórias do projeto (§2 do wyd2.md)

| Referência exigida | Estado | Evidência |
|---|---|---|
| `wyd2.md` (2241 linhas) | LIDO integralmente | §§1–16: 33 domínios (4.1–4.33) + cliente/servidor (§5) + bugs (§6) + XP jogadores (§7) + matemática (§8) + bots (§9) + auditoria roadmap (§10) + docs/issues (§11–12) |
| `pre-release-rust.md` (236 linhas) | LIDO (cabeça + §0 autopsia) | Hot Path monolítico, DOD/ECS, thread-per-core, io_uring/tokio, Hot/Cold, tabela de-para C++→Rust |
| `start-rust.md` (48 linhas) | LIDO integralmente | Veredito Go→Rust (GC, cache-miss, microsserviços no Hot Path), limpeza `.deprecated-go`, `develop` como principal |
| `docs/` (185 arquivos) | MAPEADO | `docs/wyd/` (manual Kersef), `docs/cliente/` (dossiê 7662), `docs/arquitetura/` (protocolo, banco, ADRs 0000–0025), `docs/rust/` (execução, progresso, checklist), `docs/manual-técnico/` |
| `client/` (nosso, 7662) | CONFERIDO | `WYD.exe` hash `04c9d2a1…27ddf8` (confere com `docs/cliente/inventario.md` §1); `serverlist.txt` local `192.168.100.3:8281`; `.bin` + `serverlist.bin` com 10+ backups de experimento |
| `C:/Users/Canta/AppData/Local/wyd_launcher` (Global/RaidHunt, 3º) | LISTADO (só referência) | `app-2.7.3/` Electron (Squirrel): `With Your Destiny.exe`, `launcher.json`, `resources.pak`; NÃO confundir versão Global atual com build 7662 |

### 1.2 Projeto legado OpenWYD — GitHub `main`

Pesquisa web 2026-11-10 (fontes abertas, não clonadas):

| Candidato | Veredito | Por que |
|---|---|---|
| `Jean1dev/w2pp-OpenWYD` (`main`, 672 commits, GPL-2.0) | **LEGADO DE REFERÊNCIA** | Reescrita Go big-bang mirando `WYD.exe 7662` sem modificação (protocolo 7640). `Source/` (C++ legado), `Release/` (binários+conteúdo), `tmserver/` (dono do mundo, single-goroutine sem locks), `dbserver/` (Postgres/pgx), `binserver/`, `webserver/`, `docs/migration/` (RE com `Source/...cpp:linha`). É o "OpenWYD" que o wyd2.md cita |
| `venantivs/snalmir` (`master`, 7.54, epoll) | Comparativo de versão | Foco 7.54 ≠ 7662; serve para mecânica qualitativa, nunca para layout |
| `open-wyd/*` (org) | Dados de cliente | `ows-clientdata` (InitItem/ItemList/SkillData/serverlist), `open-wyd-scripts` (Lua) |
| `alanpetry/OpenWyd` (WASM/WebGL) | Preservação de cliente | Porte DirectX→WASM; útil para pipeline de render, não para regra de servidor |
| `antonioducs/open-wyd` (Node/TS) | Descartado como referência | Arquitetura divergente (Gateway+SQS+Mongo); anti-padrão para Hot Path |
| Técnicas (`WYD2EncDec`, `W2PP Basedef.h`, `WYD2Bot`, `WYD-ItemList-Editor`) | Apoio já catalogado | Header 12 B, `0x0FDE/0x0FDF`, `ESCENE_FIELD=30000`; ver `docs/wyd/fontes.md` §2 |

Divergência resolvida: "OpenWYD" não é um repositório único — o legado útil para
esta missão é o `w2pp-OpenWYD` (Go + `Source/` C++ + `docs/migration/`). Os demais
são comparativos por versão ou arquitetura.

### 1.3 Skills e docs normativas consultadas

Vigentes usadas nesta fase: `wyd-rust-core`, `wyd-rust-net`, `wyd-rust-db`,
`wyd-rust-sim`, `wyd-rust-obs`, `wyd-rust-toolchain`, `wyd-rust-testing`,
`wyd-launcher`, `wyd-client-reverso` (+ `software-architecture`, `rust-code-quality`
como pano de fundo). Legadas Go (`wyd-server-dev`, `wyd-database`, etc.)
**não usadas como norma** — sucessão em `docs/manual-técnico/skills.md` (PROC-18).

Normas lidas: `docs/arquitetura/estrutura-e-padroes.md` (ADR-0002),
`docs/arquitetura/protocolo.md` (416 linhas), `docs/cliente/pacotes.md` (189),
`docs/cliente/superficie-conhecida.md` (90), `docs/rust/02-execucao.md` (ciclo),
`docs/rust/08-checklist-funcionalidades.md` (85 linhas marcadas).

## 2. Estado real do Rust (o que já existe, com prova)

- **Workspace:** `crates/wyd-{protocol,core,net,db,admin,server,sim,launcher}` +
  `wyd-portal/`; 208 `.rs`; `develop` atual; `make check` = gate.
- **Checklist viva:** 29 ✅ · 18 🚧 · 36 ⬜ · 2 ⛔ (contagem 2026-11-09; recontar após Fase 5).
- **Protocolo:** HEADER 12 B, `INITCODE 0x1F11F311` (dado E, medido), `pKeyWord` 512 B
  byte-idêntica, checksum **não rejeitante** (build já vem com `EB`), versão 12000.
  `CreateItem 0x026E` é **S→C** (binário corrige o `Basedef.h`) — tabela congelada ainda
  declara `Client2Game` (rito próprio pendente).
- **Jogo ponta-a-ponta parcial:** login + PIN Argon2id + `STRUCT_SELCHAR/MOB` do banco,
  `CreateMob 0x0364` / `RemoveMob 0x0165` no fio, `Attack*` → `Engine::attack_at` →
  `gain_exp` + `loot_at_death` (sorteio) + `UseItem`→refino com SancRate viva.
  **Falta materializar:** `ESCENE_FIELD` (GAME-09), caixa no chão (`CreateItem`) + pickup
  (`GetItem`/`CNFGetItem`), `UpdateItem` de resposta, regen/respawn próprio.
- **Conteúdo no banco:** 34–38 migrations conforme a entrega (ver `03-progresso.md`;
  medir com `make banco-verify` antes de citar número novo), `item_catalog` 3.220,
  `npc_template` 1.991, `npc_definition` 6.100, `skill_catalog` 154, 52 regiões.
- **Árvore suja hoje:** `git status` mostra ~30 modificados (canal, banco, protocolo,
  checklist, progresso) — **nada commitado**. Fase 5 deve fechar ou registrar.

## 3. Auditoria de cobertura wyd2.md × roadmap-rust (resumo executivo)

Legenda: ✅ coberto com profundidade · ⚠️ parcial/genérico · ❌ ausente/insuficiente.
"Existe issue" ≠ "sistema coberto" (§10 do wyd2.md).

| Domínio wyd2.md | Roadmap/docs | Veredito |
|---|---|---|
| 4.1 atributos/pontos | GAME-05, GAME-19; `classes.md`, checklist 2.2 🚧 | ⚠️ base existe, distribuição em jogo (`ApplyBonus`) pendente |
| 4.2 combate (vel/crit/evasão/imun/def/HP-MP) | FASE-03 + GAME-19/GAME-20; `wyd-sim` | ⚠️ ordem de modificadores e anomalia 200% em aberto (único ⛔ matemático) |
| 4.3 EXP/progressão | `progressao.md` + `grant_exp` + `game_setting` | ⚠️ curva LEGACY portada; coluna skill/nível (P-1) lacuna doc |
| 4.4 Mortal/Arch/Celestial/Sub | `classes.md`, `progressao.md`, pesquisa 2026-11-10 §2.1 (números novos) | ⚠️ mecânica mapeada; números de bônus Cythera/pontos ainda D/fonte única |
| 4.5 cidadania/Kibita/reinos | GAME-10 ⬜; `canais-e-cidadania.md` | ⚠️ cidadania = `canal+1`, sem regra de reino/bônus |
| 4.6 mapas/teleportes | 52 regiões seed; `sistemas/mundo.md`; checklist 3.3 ✅ / 3.6 ⬜ | ⚠️ dado existe, handler de teleporte não |
| 4.7 monstros/bosses/IA | GAME-06 ✅; 6100 `npc_definition` spawna sem duplicar | ✅ IA no tick; ❌ respawn próprio/loot→caixa |
| 4.8–4.9 itens/drop/catálogo | `itens.md`, `sistemas/drops.md`, `aglutinacao.md`; loot até sorteio | ⚠️ catálogo ✅ como dado; drop→caixa→pickup ❌ no fio |
| 4.10 refino/composição (+10…+15, Anct, gemas) | `sistemas/refino.md` + auditoria; `Gameplay::refine` no fio | ✅ melhor cobertura da missão; falta `UpdateItem` + piedade persistida |
| 4.11 montarias/crias | `multiconta.md`? doc em pesquisa §2.6; checklist 2.6 ⬜ | ❌ sem regra |
| 4.12 capas/pinturas | quest capa em Arch; sem regra de refino visual | ❌ |
| 4.13 skills/Sephira/Soul/evocação/transformação | `skills.md`, `classes.md` (9ª–12ª CONFIRMADO B); `skill.rs` sem fio | ⚠️ catálogo ✅; custo/cast/buff no fio ❌ |
| 4.14 quests/narrativa | `sistemas/quests.md` (Kefra, Runas sala a sala, Arch) + `0013_quests` | ⚠️ doc forte; engine de quest ❌ |
| 4.15 NPCs/lojas/diálogo | `core/npc/` + `npc_shop_item` 6.706; loja no fio ⬜ | ⚠️ dado ✅; diálogo/loja ❌ |
| 4.16 consumíveis | `itens.md` (uso de cada consumível) | ⚠️ doc; efeito no fio parcial (pó de refino ✅) |
| 4.17 guilds/fame/guerras | `sistemas/guerras.md`; `0004_guildas` + leitura no roster | ⚠️ doc + dado; handlers ❌ |
| 4.18 eventos | `sistemas/eventos.md` + `0014_eventos` + `game_setting ev*` | ⚠️ doc + config; scheduler ❌ |
| 4.19 Kefra/especiais (Pesadelos/LAN/Água/Coliseu) | `sistemas/eventos.md`, `mund o.md` | ⚠️ doc; instância/tempo ❌ |
| 4.20 cidades/impostos | `sistemas/guerras.md`; castelo 40%, portal 700 (B) | ⚠️ número B isolado; regra ❌ |
| 4.21 economia | `economia.md`, `decisoes-servidor.md`; ouro persistido sem fluxo | ⚠️ mapa de fluxo pendente (gold só entra via grant/delivery) |
| 4.22 trocas/grupo/loja pessoal | `MSG_Trade` UNVERIFIED; checklist 5.10 ⬜ / 10.2 ⬜ | ❌ |
| 4.23 PVP/PK/chaos/Coliseu/arena | `sistemas/guerras.md` + `battle-royale.md`; `pvp_points` sem regra | ❌ regra |
| 4.24 chat/Trombeta/eros | pesquisa §2.9 (chat/Trombeta) nova; checklist 10.1 ⬜ | ❌ fio |
| 4.25 cash/premium-mail/rastreabilidade | pesquisa §2.10; `donate` só no portal (invariante 7) | ⚠️ decisão correta; rastreabilidade ❌ |
| 4.26 canais/troca de canal | hipótese 8 passos no wyd2.md; sessão única ✅ | ⚠️ hipótese plausível, sem transferência viva |
| 4.27 login/logout/sessões | CLIENT-09 ⬜; sessão única + idle 120 s | ⚠️ logout explícito é a próxima entrega S |
| 4.28 tela personagem | `STRUCT_SELCHAR` 844 B do banco; layout `rust-v1` UNVERIFIED | ⚠️ dado ✅; layout só fecha com captura CLIENT-06 |
| 4.29 barra de skills/persistência | `ShortSkill[16]` persistido ✅ | ✅ persistência; ❌ edição em jogo |
| 4.30 efeitos/aparência | `Effect/*.wys` inventariado; sem vínculo servidor→cliente | ❌ |
| 4.31 itens iniciais/experiência inicial | ficha por classe (`BaseSIDCHM`) ✅ | ✅ criação; tutorial ❌ |
| 4.32 buffs/scripts/temporizadores | campos UNVERIFIED; temporizador = `LiveConfig` poll 1 s | ⚠️ config viva ✅; buff no fio ❌ |
| 4.33 tempos/resets/ciclos | `game_setting` + calendário doc; scheduler ❌ | ❌ |
| §5 funcionalidades básicas | checklist §§1–11 cobre caminho login→mundo parcial | ⚠️ sem cena não há "personagem funcional" |
| §6 bugs/hacks/vulns | QA-05 ✅ (DoS/rate-limit) + `ingame-bugs` arquivado; catálogo WYD ❌ | ❌ catálogo de ameaças é gap puro |
| §7 XP dos jogadores | `referencia-externa` §7 + pesquisa §6 pendente | ❌ sem levantamento próprio |
| §8 matemática central | GAME-19 🚧; ordem base→…→resultado como contrato | ⚠️ contrato definido, equações em aberto |
| §9 bots + LLM + stress | QA-11 ⬜ (`crates/wyd-bots` não existe); ADR-0010 | ❌ maior gap estrutural |

**Gaps que o roadmap não cobre hoje:** catálogo de ameaças/exploits (§6),
levantamento de XP de jogadores (§7), economia-fluxo ponta a ponta, troca/grupo/loja,
PVP/PK/chaos, chat/Trombeta, buffs/debuffs no fio, scheduler de eventos/resets,
efeitos visuais vinculados, e o programa de bots jogáveis (§9 → `wyd-bots`).

## 4. Conferência de clientes (verdades)

- **Nosso 7662:** hash confere; CPSock (ofuscação mod-4, tabela 512 B, checksum
  desativado, limite 8192, versão 12000) **CONFIRMADO por paridade/binário**.
  Direção `CreateItem S→C` provada no `.text` (`cmp 0x26e` no roteador de recepção;
  `GetItem` montado para `SendPacket`) — diverge do `Basedef.h`/tabela congelada.
- **Global/RaidHunt (externo):** cliente Electron atual — serve para comparar
  interface/mecânica visível (ex.: cidadania por canal, guerra entre canais, custos
  +10 50kk vs 5kk PT/EN), **nunca** para layout binário do 7662.
- **Regra de versão:** Global atual ≠ 7662. Divergência é esperada; registrar
  versão sempre (P-3: 2 canais no plano vs 8 do Global; P-5: teto 400/camada).

## 5. Pendências e perguntas que travam promoção a regra (não travam código)

P-1…P-8 de `referencia-externa-2026-11.md` §9 continuam abertas (skill/nível, custo
+10, nº de canais, bônus cidadania, teto de nível, Arch no primeiro corte, layout
provisório, donate). Somam-se P-9…P-14 da pesquisa 2026-11-10 e a lista de
**fontes descartadas** (§7.2) — consultar antes de re-pesquisar.

## 6. Próximo passo (Fase 2–5, sem criar issues ainda)

1. Fase 2: abrir `Jean1dev/w2pp-OpenWYD` (`Source/`, `docs/migration/`) e cruzar
   `STRUCT_*`/fluxos com nosso `wyd-protocol`; extrair `SkillData`/curva EXP do 7662.
2. Fase 3: para cada domínio 4.1–4.33, preencher os 26 campos (§11.2) só onde houver
   evidência; o resto fica `DESCONHECIDO` com fonte pendente.
3. Fase 4: recontar checklist (grep por estado) e reconciliar com `index.md` +
   `03-progresso.md` (rito PROC-13).
4. Fase 5: propor issues fatiadas (uma por entrega testável) para os gaps do §3 —
   sem duplicar as existentes, sem remover histórico.

*Branch `develop`, árvore suja preservada, nenhum commit feito nesta fase.*
