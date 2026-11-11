# FASE-03-17 — Entrada no mundo fiel ao legado (id de jogador, `UpdateScore`, `CreateType`)

**Fase:** 3 · **Depende de:** FASE-03-8 ✅, FASE-03-9 ✅ · **Bloqueia:** FASE-03-18 (Campo de Treinamento), FASE-15-4 (bots) · **Esforço:** M

## Objetivo

Fazer o personagem **entrar no mundo no cliente 7662** — hoje o login/criação funcionam, mas a
entrada mostra “Falha ao conectar”. Alinhar a rajada de entrada ao que o servidor legado faz
depois do `_MSG_CharacterLogin`, sem inventar pacote.

## Contexto / evidência

O porte Go do mesmo alvo (`main:tmserver/internal/handler/character.go`,
`completeCharacterLogin` → `enterWorldView`) prova três coisas que faltavam no canal Rust:

1. **Jogador e criatura vivem em faixas de id diferentes.** `world.IsPlayer(id) = id < MaxUser`
   (`MaxUser = 1000`, `MaxMob = 25000`; `world.go:25-27`). O cliente decide “é personagem”
   pela faixa do `MobID`/`ClientID`.
2. **`UpdateScore` do próprio personagem logo depois do `CNFCharacterLogin`**
   (`sendScoreSelf`, antes do `CreateMob`) — é o que pinta HP/MP/EXP na entrada.
3. **`CreateMob` com `CreateType = 2` só para quem acabou de entrar** (`createMobFrom(self, 2)`);
   os demais entram com `0`.

No canal Rust, `world::scene_snapshot` usava o **índice do armazém** como id de cena. Como os
6 100 NPCs nascem primeiro, o primeiro jogador recebia `ClientID > 6100` — um id de criatura —
e não recebia `UpdateScore` nenhum. `send_create_mob` ainda mandava `CREATE_JUST_ENTERED` para
todos, NPCs inclusive.

Pesquisa: `docs/wyd/pesquisa-campo-de-treinamento-2026-11-10.md` §5.

## Passos

1. Espaço de id de cena em duas faixas (`world::{MAX_USER, MAX_MOB}`): jogador `1..1000`,
   criatura `1000..25000`, com alocação/liberação por entidade (`alloc_scene_id`/
   `release_scene_id`) e recusa tipada `EnterWorldError::SceneFull`.
2. `Engine::enter` devolve `Entered { entity, scene_id }`; a borda usa `scene_id` em
   `ClientID`, `HEADER.ID` de `UpdateEtc`/`UpdateScore`/`SendItem` e no `MobID` do próprio
   `CreateMob`. `attack_at` usa o id de cena no `RemoveMob` de morte.
3. `cpsock`: enviar `UpdateScore` (corpo de `personagens::update_score_body`) logo depois do
   `CNFCharacterLogin`, antes de `refresh_scene`.
4. `send_create_mob(..., create_type)`: `2` apenas quando `mob.id == client_id`, `0` nos demais.
5. Atualizar o teste vivo `tests/cena_mobs.rs` para exigir `UpdateScore`, `ClientID < 1000`,
   criaturas `>= 1000` e `CreateType` correto.

## Arquivos

- Alterados: `crates/wyd-server/src/world.rs`, `crates/wyd-server/src/cpsock.rs`,
  `crates/wyd-server/src/characters.rs`, `crates/wyd-protocol/src/payload/scene.rs`
  (`CREATE_NORMAL`), `crates/wyd-server/tests/{cena_mobs,entrada_mundo}.rs`,
  `docs/rust/08-checklist-funcionalidades.md` (1.11), `docs/rust/03-progresso.md`.
- Criados: este arquivo, `docs/wyd/pesquisa-campo-de-treinamento-2026-11-10.md`.
- Proibido: inventar `_MSG_PKInfo`/movimento sem layout provado (invariante 5).

## Testes / verificação

```bash
cargo fmt --all --check
cargo clippy --workspace --all-targets -- -D warnings
cargo test -p wyd-server --lib
cargo test -p wyd-protocol
# com o canal no ar e conta real:
WYD_LIVE_CPSOCK=... WYD_LIVE_ACCOUNT=... WYD_LIVE_PASSWORD=... \
  cargo test -p wyd-server --test cena_mobs
```

## Skills Relacionadas do freebuff

- `.agents/skills/wyd-rust-net` — contrato CPSock e ordem de pacotes.
- `.agents/skills/wyd-rust-core` — cena/id de entidade sem mutex global.
- `.agents/skills/wyd-rust-testing` — teste vivo de fio.
- `.agents/skills/wyd-client-reverso` — faixa de id exigida pelo cliente 7662.

## Documentação de referencia

- `docs/wyd/pesquisa-campo-de-treinamento-2026-11-10.md` §5.
- `main:tmserver/internal/handler/character.go` (`enterWorldView`, `createMobFrom`,
  `sendScoreSelf`) e `main:tmserver/internal/world/world.go` (`MaxUser`/`MaxMob`).
- `docs/cliente/pacotes.md` (S→C) e `docs/arquitetura/protocolo.md` (`ESCENE_FIELD`).

## Subtasks

- [x] Codificar issue
- [x] Testar issue
- [x] Revisar issue
- [x] Documentar issue
- [x] Provar no fio com o canal real (`cena_mobs` vivo: 1281 `CreateMob`, `UpdateScore`, `client_id=1`)
- [ ] Provar com o cliente 7662 aberto (render da entrada)
- [ ] Versionar issue, mergear branch e fechar issue

## Critérios de aceite

- [x] `ClientID`/`MobID` do jogador em `1..MAX_USER` e das criaturas em `MAX_USER..MAX_MOB`.
- [x] `UpdateScore` do dono sai depois do `CNF` e antes da cena.
- [x] `CreateType` 2 só para o dono da sessão.
- [x] Id liberado na saída e reusado no mesmo espaço (sem esgotar após 999 logins).
- [x] Canal real entrega o jogador com `client_id` na faixa `< MAX_USER` (log: `entity=6100 client_id=1`).
- [ ] Cliente 7662 entra no mundo sem “Falha ao conectar” (prova visual pendente).

## Evidência (2026-11-10, sem commit)

- `cargo fmt --all --check` 0 · `cargo clippy --workspace --all-targets -D warnings` 0 ·
  `cargo check --workspace --all-targets` 0.
- `cargo test -p wyd-server --lib` **42/42** (`personagem_entra_na_cena_com_nome_equip_e_id_proprio`
  agora exige `< MAX_USER` para o ator e `>= MAX_USER` para criatura;
  `create_mob_de_personagem...` exige `CreateType` 2; novo `create_mob_de_npc_vai_como_entrada_normal`).
- `cargo test -p wyd-protocol` 0 falhas.
- `tests/cena_mobs.rs` passou a exigir `UpdateScore` de entrada, `ClientID < 1000`, self
  `CreateMob` e `CreateType` por dono (roda contra o canal vivo quando `WYD_LIVE_CPSOCK` existe).
- **Prova viva (2026-11-10, canal reconstruído com o código novo):** conta `qavivo01`
  (`tests/conta_de_teste.rs`) → `handshake accepted` → `login password_ok` → `session open` →
  `character ok action=select ... entity=6100 client_id=1` → **1281 `CreateMob` de 232 B** +
  `UpdateScore`; `_MSG_Action` respondeu `Guarda_Carga` (npc 1212, merchant 2).
  `WYD_LIVE_CPSOCK=127.0.0.1:8281 ... cargo test -p wyd-server --test cena_mobs` → 1 passed.
  Ajuste no teste: o alvo do diálogo passou a ser NPC (`mob_id >= 1000`), porque o jogador
  agora ocupa `1..999`.

## Riscos / rollback

- Risco: reuso de id confundir o delta de cena → mitigação: o id só volta ao espaço depois do
  `RemoveMob`, e a geração do motor sobe em toda saída.
- Risco: `UpdateScore` extra atrapalhar clientes antigos → mitigação: o legado faz o mesmo na
  mesma posição da rajada.
- Rollback: desligar o `UpdateScore` de entrada e voltar o id ao índice da entidade (uma linha
  em `scene_snapshot`), mantendo o teste como especificação.
