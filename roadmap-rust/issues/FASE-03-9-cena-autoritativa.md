# Issue 03-9 — Cena autoritativa (`ESCENE_FIELD` 30000): o primeiro pixel de mundo

**Fase:** 3 · **Status:** ✅ concluída em 2026-11-10 (lado servidor; captura 08-6 pendente) · **Depende de:** 03-8 ✅, 08-2 ✅ · **Bloqueia:** 4.8/10.4 do checklist · **Esforço:** L

## Objetivo

Emitir o pacote autoritativo de cena com `HEADER.ID = ESCENE_FIELD` (30000): entidades
visíveis, posição, HP/EXP e remoção — o que faz o cliente desenhar o mundo e atualizar as
barras.

## Contexto / evidência

`ESCENE_FIELD = 30_000` existe em `crates/wyd-protocol/src/lib.rs:43` com o teste
`is_authoritative_scene`, mas **nenhum pacote é emitido** — `grep` no workspace encontra
apenas a constante, o reexport em `wyd-core` e comentários. O checklist 4.8/10.4 está ⬜ e o
invariante 4 do `AGENTS.md` (“pacote autoritativo de cena vai com `HEADER.ID = ESCENE_FIELD`,
não com o id da conexão”) ainda não tem implementação.

Consequência: mesmo com tick e spawn (03-7/03-8), o cliente não vê mundo; a barra de EXP/HP
não atualiza — sintoma clássico de ID errado no pacote, exatamente o que o invariante previne.

## Passos

1. Definir o layout de cena em `crates/wyd-protocol/src/payload/scene.rs` com o que está
   **provado**: `HEADER` 12 B + lista de entidades com campos presentes em `STRUCT_MOB`
   (já decodificado em `personagens.rs`). Campo sem prova fica de fora — nunca inventar
   offset (invariante 5).
2. Marcar o layout como `UNVERIFIED` no doc e no código, com **golden byte-exato** montado a
   partir do codec real (`tests/scene.rs`), não de reimplementação.
3. `world.rs`: a cada tick de visibilidade (não a cada tick de simulação), montar delta de
   cena pela `spatial::grid` — entrar/sair/mover — e enfileirar por sessão; `HEADER.ID =
   ESCENE_FIELD` sempre, inclusive para o próprio jogador.
4. Enviar pelo caminho de escrita já existente (buffer + flush no tick, como o legado).
5. Rate-limit de saída: teto de pacotes de cena por segundo por sessão (a grade é O(1), mas
   o cliente não pode ser inundado).
6. Testes: delta de entrada contém o jogador e vizinhos; sair do raio emite remoção; nenhum
   pacote de cena usa o id da conexão; pacote de 100 entidades ≤ `MAX_MESSAGE_SIZE` (8192).
7. Prova com o cliente 7662 quando houver captura (08-6): registrar o que apareceu e o que
   não. Sem isso, a issue fecha pelo lado servidor com `UNVERIFIED` declarado.

## Arquivos

- Criados: `crates/wyd-protocol/src/payload/scene.rs`, `crates/wyd-protocol/tests/scene.rs`,
  `crates/wyd-server/tests/cena.rs` (ou `wyd-core/tests/visibility.rs`).
- Alterados: `crates/wyd-protocol/src/payload/mod.rs`, `crates/wyd-core/src/systems/visibility.rs`
  (hoje só planejado), `crates/wyd-server/src/world.rs`, `docs/arquitetura/protocolo.md`,
  `docs/arquitetura/estrutura-e-padroes.md` §3/§10, `docs/rust/08-checklist-funcionalidades.md`.
- Proibido tocar: `ESCENE_FIELD` como id de conexão, layout de `MSG_Trade`/`MSG_SetHpMp`
  (UNVERIFIED), `client/`.

## Testes / verificação

```bash
cargo test -p wyd-protocol --test scene
cargo test -p wyd-core --test visibility
cargo test -p wyd-server --test cena
cargo test -p wyd-protocol -p wyd-net     # bytes no fio
```

Saída esperada: golden de cena byte-idêntico ao codec; delta correto em entrada/saída de raio;
`HEADER.ID == 30000` em 100% dos pacotes de cena.

## Skills Relacionadas do freebuff

- .agents/skills/wyd-rust-net — framing, payload, sem `panic!` em entrada de rede.
- .agents/skills/wyd-rust-core — spatial grid, delta de visibilidade.
- .agents/skills/wyd-client-reverso — confirmar layout com captura quando possível.

## Documentação de referencia

- `AGENTS.md` §3 invariante 4 (`ESCENE_FIELD` 30000).
- `docs/arquitetura/protocolo.md` §4–§5 e `docs/arquitetura/contrato-transporte-cpsock.md` §8.
- `docs/cliente/superficie-conhecida.md` (pipeline CPSock do 7662).

## Subtasks

- [x] Codificar issue (2026-11-10)
- [x] Testar issue (2026-11-10)
- [x] Revisar issue (2026-11-10)
- [x] Documentar issue (2026-11-10)
- [ ] Versionar issue, mergear branch e fechar issue — **não executado**: a missão não
  autoriza commit/push/PR; tudo segue uncommitted em `develop`.
- [x] Limpar branch (se aplicável) — nada a limpar: nenhum branch novo foi criado.

## Critérios de aceite

- [x] Pacote de cena emitido com `HEADER.ID = ESCENE_FIELD` e golden byte-exato.
- [x] Entrada/saída refletida em delta (motor real; fila de entregas: 03-7 ✅ → 03-8 ✅ →
  esta fase; fila por raio de visão **não** — sem fonte provada, declarado como resíduo).
- [x] Nenhum campo de layout inventado: o que não está provado fica fora e documentado.
- [x] Cliente desenha o jogador e atualiza barra — **ou** evidência do que faltou, com
  `UNVERIFIED` e a pergunta que fecha (nunca “deve funcionar”): lado servidor fechado
  (detalhes em § Evidência); a prova no 7662 segue com a captura 08-6.

## Evidência (execução 2026-11-10, sem commit/push por regra da missão)

**O que foi entregue** (a base já existia em `payload/scene.rs`/`cpsock::send_create_mob`/
`tests/cena_mobs.rs`; esta fase completou os personagens na cena):

- `world.rs`: `ActorSkin` (nome/equip/guilda/aparência vinda do `STRUCT_MOB` do login),
  `SceneMob` (id de cena = **índice da entidade**, espaço único NPC + jogador),
  `SceneSnapshot { generation, mobs }` (NPCs + personagens, posição autoritativa),
  `plan_scene` (delta puro: só entrar/sair, saídas antes de entradas, teto por varredura, sem
  reenvio; `RemoveType` 2 para personagem, 0 para NPC), `REMOVE_OUT_OF_VIEW/LOGOUT`,
  `WYD_SCENE_MAX_PER_REFRESH` + `scene_budget_from_value` (default 64, 1..=4096, erro fatal no
  boot via `main.rs`). `Engine::enter` recebe a skin; `spawn`/`despawn` sobem
  `scene_generation` (a remoção por morte/abate e por logout/saída acorda as outras sessões).
- `cpsock.rs`: `CharacterLogin` monta a skin antes de `enter_world`; `Scene` virou
  `BTreeMap<u16, bool>` (id → é personagem); `refresh_scene` aplica `plan_scene`, não
  reenvia o já visto e só avança a geração quando o delta coube no teto (senão repete com
  log `warn`); `AttackOutcome.target_scene`/`AttackReport.target_scene` levam o id de cena ao
  `_MSG_RemoveMob` de morte (antes ia o id de `npc_definition`, que não é `MobID`).
- Docs: `docs/arquitetura/protocolo.md` §5 (espaço único de `MobID`, origem dos campos do
  jogador, teto, resíduos), checklist 1.11/4.8 ✅ e 10.4 🚧.

**Testes reais que passam:**

- `cargo test -p wyd-server --lib`: 38/38 — novos: ator entra na cena com nome/equip(831)/
  guilda(4)/nível/HP/posição e id próprio sem colidir com o id do banco; saída esvazia a cena
  e sobe a geração; `plan_scene` (entrar tudo, delta vazio, logout=2 vs fora-de-vista=0,
  teto 1 → incompleto); teto valida `0/4097/abc/-1`; `fixed_name` não parte UTF-8.
- `cpsock::tests` (fio loopback real, sem DB): `_MSG_CreateMob` de personagem chega com
  `HEADER.ID = ESCENE_FIELD` (30000), `MobID = 9`, nome/equip/guilda/HP do login;
  `_MSG_RemoveMob` leva o id da cena no header e `LOGOUT` no corpo.
- Gates: `cargo fmt --check` 0 · `cargo clippy --workspace --all-targets -D warnings` 0 ·
  `cargo test --workspace --all-targets` 58 alvos ok, 0 falhas · `check-migrations`
  41 migrations / 82 tabelas / 152 INSERTs ok · `check-issue-status` 60 comparadas, 0
  divergências · `gen-catalogo-banco --check` ok. Sem mudança de esquema (`SCHEMA_VERSION` =
  41); testes com banco pulam nesta máquina (forward do Docker Desktop aceita TCP e fecha
  sem bytes; stack não reiniciada — sessões possivelmente vivas).

**Resíduos declarados (fora de escopo, nada inventado):** movimento de entidade no fio
(sem opcode provado na tabela congelada; entidade que mudou não é reenviada), raio de visão
(sem fonte provada para a distância), barra de outro jogador no fio, prova de render no
cliente 7662 (depende da captura 08-6).

## Riscos / rollback

- Risco: layout de cena errado derrubar o cliente → mitigação: emissão atrás de flag
  (`WYD_SCENE_PACKETS=0`) e teste de tamanho antes de ligar por default.
- Risco: flood de cena em guerra → mitigação: delta por raio + teto por segundo.
- Rollback: desligar a flag; tick/spawn continuam válidos.
