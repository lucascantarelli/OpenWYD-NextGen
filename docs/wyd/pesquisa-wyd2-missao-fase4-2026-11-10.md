# Missão wyd2.md — Fase 4: auditoria do Rust contra os 9 itens (2026-11-10)

> Compara cada item da Fase 3 com o que existe no Rust, avalia implementação,
> identifica gaps, bugs e cobertura documental/de testes. Nada aqui vira regra
> por existir neste arquivo. Níveis: `fontes.md` §1. Nenhuma issue criada,
> nenhum commit, nenhum arquivo sujo tocado (só este artefato é novo).

## 1. Recontagem da checklist (medida, não herdada)

Comando: `grep -E "^\| [0-9]+\.[0-9]+" docs/rust/08-checklist-funcionalidades.md`
(85 linhas numeradas) → estado em `awk -F'|' $4`:
**28 ✅ · 20 🚧 · 32 ⬜ · 1 ⛔ · 4 sem marca** (linhas 1.16/11.7/4.3/5.4-5.6 sem
marca na coluna — texto descritivo no lugar do estado; a Fase 5 deve normalizar
para ✅/🚧/⬜/⛔ antes de qualquer contagem oficial).
Fase 1 citava 29/18/36/2 (2026-11-09): a diferença vem de entregas intermediárias
(motor no processo, entrada no mundo, refino no fio) e das 4 linhas sem marca —
não reinterpretar como regressão sem diff linha a linha.

## 2. Auditoria por item (implementação real, com evidência)

| # | Item | Veredito | Evidência (código + prova) |
|---|---|---|---|
| 1 | CreateItem/GetItem | ❌ ausente no fio | Sorteio existe (`Gameplay::loot_at_death`, `event=death/event=loot`); `CreateItem/GetItem/CNFGetItem/DecayItem` sem handler; tabela com `Flow::Client2Game` oposto ao binário (`opcode_table.rs:881-883`, rito próprio pendente). Payloads de inventário portados: `UseItem` 34 B, `TradingItem` 20 B, `ApplyBonus` 18 B (`payload/inventory.rs`). Gap: emitir caixa, tratar `GetItem`, ids de caixa, `UpdateItem` |
| 2 | ESCENE_FIELD | 🚧 parcial | `CreateMob` 232 B + `RemoveMob` 16 B portados e provados (6100 frames); ataque no fio (`Attack/One/Two`, `combat.rs:164 B full`); cena (broadcast com `ID=30000`, `UpdateEtc/UpdateScore` no abate) incompleta. Checklist 4.8/10.4 ⬜. Gap: ligar cena no ataque/morte/XP |
| 3 | Logout `0x0215` | 🚧 parcial | Handler existe e responde `CNFCharacterLogout` + `leave_world` + `Drop` do `SessionLease` (`cpsock.rs:1008-1023`, texto "era o defeito de cliente fechado continua online" resolvido no código); teste dedicado ausente (checklist 1.13 🚧, texto desatualizado). Gap: teste + golden do frame de saída (captura) |
| 4 | Cadeia de dano | 🚧 hipótese | Único passo provado: `damage_after_absorb` (saturação). `damage_chain`/`DamageInput`/`STAGE_ORDER` provisórios (`damage.rs:1-13`, D-02/D-03/D-04 em `matematica-sim.md` §4); `resolve_attack`/`AttackOutcome` sem a cadeia L3. Gap: goldens L3 + LCG antes de tocar `wyd-core` (GAME-19) |
| 5 | EXP por flags | 🚧 núcleo | `grant_exp` + `exp_reward` com multiplicador (curva LEGACY, teto retido, teste `grant_exp_levels_up`); árvore L3 (party, divisores tier, −40%, bônus, `DOUBLEMODE`, `KefraLive`, ±15%, `Hold`, log diário) ausente por decisão. Gap: flags em `game_setting` sem mudar default |
| 6 | Pity/refino | 🚧 caminho principal | `Gameplay::refine/use_item` no fio com SancRate viva + `MissingRate` sem mutação + prova de recarga (`config_gameplay.rs`); `REFINE_MAX=15`, `PITY_MAX=20`, `allows`, `pity_gain` 3/4 (`refine.rs:19,155,183`); acessório colar/cinto 60/30 e 75/50/25/10 (`accessory.rs`). Falta: piedade persistida, `UpdateItem`, B16 completo |
| 7 | Trade atômica | ❌ ausente | Opcodes no catálogo (`Trade/QuitTrade/TradingItem`, `opcode_table.rs:97-123`); `MSG_Trade` UNVERIFIED fora do payload (`commerce.rs:3`, `payload/mod.rs:12,293`, `require_ported`→`Unported`); `Buy` 22 B/`Sell` 18 B portados (loja NPC, não troca). Gap: layout (captura) + handler atômico + testes de contenção |
| 8 | Ameaças B1–B17 | ❌ sem catálogo próprio | L5 dá o esqueleto; Rust já protege: Argon2id, sessão única (`0030`), dano server-authoritative (B12), `MissingRate`, idempotência (`delivery_queue.idempotency_key`), rate-limit (QA-05). Checklist §6 segue sem matriz própria. Gap: validar B1–B17 um a um no Rust |
| 9 | Harness/paridade | 🚧 protocolo | `cargo test -p wyd-protocol -p wyd-net` (bytes), `fixtures + captura.mjs --check` (5/5 C→S), determinismo (`determinism.rs`), benches `tick/limits`, E2E portal 3/3. Falta: goldens S→C (CLIENT-06), LCG exato, matriz de cobertura, `wyd-bots` (⛔/⬜, ADR-0010) |

Dispatch C→S hoje (`cpsock.rs`): `AccountLogin/Secure`, `Create/Delete/CharacterLogin/Logout`,
`UseItem`, `ApplyBonus`, `Attack/One/Two`, `Action` — todo o resto cai em `outro` tolerado.
Testes servidor: 16 arquivos (`handshake, login, selchar, personagens_crud, pin_*,
sessao_unica, entrada_mundo, config_* …`); protocolo: `golden.rs/roundtrip.rs`.

## 3. Gaps que o roadmap não cobre (confirmados nesta fase)

1. Linhas da checklist sem marca (4) — normalizar antes de contar.
2. Texto desatualizado (1.13 diz "pacote de saída não tratado" quando o handler
   existe — o que falta é o teste).
3. `UpdateItem`/`SendItem` de refresh do refino sem dono explícito.
4. `Hold` de EXP + log diário sem desenho.
5. Catálogo de ameaças §6 sem dono (maior gap documental).
6. `wyd-bots` sem esqueleto (só ADR-0010 + harness de protocolo).

## 4. Fatiamento para a Fase 5 (propostas, NÃO criadas)

Na ordem da Fase 2 (cada uma testável, sem número inventado): caixa+pickup (+rito
tabela), cena autoritativa, logout com teste, cadeia de dano com goldens, EXP por
flags, pity persistida + `UpdateItem`, trade atômica, catálogo B1–B17+§6, harness
LCG. Pré-requisito transversal: normalizar as 4 linhas sem marca e corrigir o
texto da 1.13.

*Branch `develop`; só este artefato é novo; nenhum commit; nenhuma issue criada.*
