# Issue 03-10 — Cidadania e canal verde: bônus e papel do canal aplicados

**Fase:** 3 · **Depende de:** 03-7 ⬜ · **Bloqueia:** — · **Esforço:** M

## Objetivo

Fazer `citizen` deixar de ser só um número no `STRUCT_SELCHAR` e virar **regra aplicada**:
bônus de caça/drop do canal de cidadania, recusa de serviço de NPC de outra cidade e papel
verde (sem PvP, loja permitida) como propriedade do processo.

## Contexto / evidência

`docs/wyd/canais-e-cidadania.md` documenta a regra com evidência do legado
(`_MSG_Quest.cpp:2433-2437`, `Server.cpp:2891`, `Language.txt:423/432`): compra na Kibita por
4.000.000, `Citizen = ServerIndex + 1`, `/getout` zera, bônus de drop no canal de cidadania.
Hoje `citizen = canal + 1` é só campo do `STRUCT_SELCHAR` (checklist 2.3 🚧) e o canal 2 não
tem comportamento verde — os dois processos são idênticos.

A migration `0029_canal_de_servidor.sql` e `channels::World` já modelam canal; falta o
**papel** e o efeito.

## Passos

1. `channels.rs`: papel do canal (`normal`/`verde`) lido de `server_channel` ou env, validado
   no boot; expor em `/health` e na métrica `wyd_server_channels_published` (sem nome novo).
2. Bônus de cidadania no cálculo de drop (FASE-10 `formula/drop.rs`), como **entrada**
   (bps) — não como constante: `citizen_bonus_bps` em `game_setting`, default 0 até haver
   número com fonte.
3. Canal verde: PvP desligado e loja pessoal permitida por política do processo — sem tocar
   mapa nem `AttributeMap`; a decisão é do canal, não do jogador.
4. `/getout`: mapear o opcode de sussurro em `wire/generated/opcode_table.rs`; se o layout não
   estiver provado, **tolerar e logar** (invariante 5) até a captura, sem inventar pacote.
5. Testes: drop com cidadania > drop sem (mesma seed); canal verde recusa agressão PvP e
   aceita loja; canal normal faz o inverso; `/getout` não derruba a sessão quando tolerado.

## Arquivos

- Criados: `crates/wyd-core/tests/cidadania.rs`, `crates/wyd-server/tests/canal_verde.rs`.
- Alterados: `crates/wyd-server/src/channels.rs`, `crates/wyd-sim/src/formula/drop.rs`
  (entrada nova, com golden), `crates/wyd-db/migrations/00NN_canal_papel.sql` (só se o papel
  não couber em `server_channel`), `docs/wyd/canais-e-cidadania.md`,
  `docs/rust/08-checklist-funcionalidades.md` (2.3).
- Proibido tocar: compra na Kibita por diálogo de NPC (sem sistema de NPC no fio), guerra
  entre canais, `client/`.

## Testes / verificação

```bash
cargo test -p wyd-core --test cidadania
cargo test -p wyd-sim
cargo test -p wyd-server --test canal_verde
cargo test -p wyd-db --test catalogo   # se houver migration nova
```

Saída esperada: bônus aplicado por dado; canal verde sem PvP; nenhum comportamento inventado
para pacote não capturado.

## Skills Relacionadas do freebuff

- .agents/skills/wyd-rust-core — regra como sistema, sem constante mágica.
- .agents/skills/wyd-rust-sim — fórmula de drop testável e determinística.
- .agents/skills/wyd-economy — bônus como parâmetro de economia, não hardcode.

## Documentação de referencia

- `docs/wyd/canais-e-cidadania.md` §2–§5 (regra e evidência).
- `docs/wyd/sistemas/drops.md` (roll por item) e `docs/wyd/decisoes-servidor.md` §1.
- `docs/rust/05-stack-docker.md` §6 (dois canais, um banco).

## Subtasks

- [x] Codificar issue (papel do canal, `ChannelRules`, bônus no drop, `/getout` tolerado)
- [x] Testar issue (`tests/canal_verde.rs` 4/4 + sussurro em `selchar.rs`; pulam com `SKIP:` sem DSN)
- [x] Revisar issue (nenhum pacote inventado; leitura de dado, não constante no core)
- [x] Documentar issue (checklist 2.3, `canais-e-cidadania.md` §3/§5.4, `05-stack-docker.md` §2)
- [ ] Versionar issue, mergear branch e fechar issue
- [ ] Limpar branch (se aplicável)

## Critérios de aceite

- [x] Cidadania altera o drop por parâmetro de conteúdo, com teste determinístico.
- [x] Canal verde não aplica PvP e permite loja; canal normal mantém a regra completa.
- [x] Papel do canal é configurável por dado/env, nunca por build.
- [x] `/getout` tratado sem inventar layout quando não há prova.

## Evidência (2026-11-10)

**Arquivos (o que foi feito, não o plano):**

- `crates/wyd-db/migrations/0041_papel_do_canal.sql`: tabela `channel_role` (`normal`/`verde`,
  FK em `server_channel`) + seed `1 = normal`, `2 = verde`; linha `game_setting`
  `Channel Settings.citizen_bonus_bps = 0` (down apaga a linha e a tabela).
- `crates/wyd-db/src/repo/{content,sessions,pg}.rs` + `lib.rs`: `ChannelRole` (`parse`/`as_db`),
  `Sessions::channel_role` (linha ausente/valor corrompido = `None`, sem `panic!`),
  `SCHEMA_VERSION` 40 → 41.
- `crates/wyd-server/src/channels.rs`: `ROLE_ENV`, `role_overrides_from_env`,
  `effective_role` e `ChannelRules` (`pvp_allowed`, `personal_shop_allowed`, `as_str`).
- `crates/wyd-server/src/config.rs`: `citizen_bonus_bps()` + `CITIZEN_BONUS_SECTION/KEY`, faixa
  0..=50% validada no snapshot (fora dele recusa a foto, nunca vira taxa).
- `crates/wyd-server/src/gameplay.rs`: `citizen` como entrada de `loot`/`loot_at_death`;
  bônus aplicado no peso efetivo por `effective_rate_with` (soma com outros bônus).
- `crates/wyd-sim/src/formula/drop.rs`: `CITIZEN_BONUS_MAX` (50%), `sum_bonus`,
  `effective_rate_with` + teste de composição/teto.
- `crates/wyd-server/src/main.rs` e `status.rs`: papel resolvido no boot (banco → env), log
  `event=channel`, `/health` com `role`/`pvp`/`shop`.
- `crates/wyd-server/src/cpsock.rs`: `_MSG_MessageWhisper` (0x0334) com ramo próprio que loga
  `action=getout` e mantém a sessão — **sem ler o corpo** (layout UNVERIFIED).

**Desvio do plano da issue:** o teste de política ficou em `crates/wyd-server/tests/canal_verde.rs`
(e não em `crates/wyd-core/tests/cidadania.rs`): o papel e a política moram em `wyd-server`
(mesmo lugar de `channels::World`), então um teste em `wyd-core` não teria acesso ao tipo — e
criar arquivo novo em `wyd-core` fora da árvore do ADR-0002 exigiria ADR. Nenhum código do core
mudou nesta fase.

**Provas:** `cargo test -p wyd-server --test canal_verde` 4/4 (`canal_verde_recusa_pvp_e_mantem_loja`,
`papel_do_canal_e_configuravel_e_invalido_cai_no_default`, `bonus_de_cidadania_muda_o_drop_na_mesma_seed`,
`bonus_fora_da_faixa_recusa_o_snapshot`); `cargo test -p wyd-sim --lib` 95/95;
`cargo test -p wyd-db --test catalogo` 3/3; `cargo test -p wyd-server` 23 alvos ok;
`cargo fmt --check` 0; `cargo clippy --workspace --all-targets -D warnings` 0;
`node tools/content/check-migrations.mjs` 41/82/152 ok; `gen-catalogo-banco --check` ok;
`check-issue-status` sem divergência. Os testes com banco (sussurro, login) **pulam** nesta
máquina porque o forward host→Postgres está fora (TCP aceito e fechado sem bytes); em ambiente
com banco rodam como os demais.

**Resíduo declarado (não é ✅ escondido):** cidadania continua derivada do canal — a compra na
Kibita por 4.000.000 e o `/getout` que zera `MobExtra.Citizen` precisam do sistema de diálogo de
NPC (checklist 3.5) e de cidadania persistida; a **aplicação** do bônus já é por parâmetro e não
muda nada com o default `0`. PvP e loja pessoal no fio dependem de 4.5 e 7.2; até lá a política
é a regra testada, sem pacote inventado (invariante 5).

## Riscos / rollback

- Risco: bônus de cidadania virar “buff global” por erro de derivação → mitigação: teste com
  dois personagens de canais diferentes na mesma seed.
- Risco: canal verde confundido com “evento de novato” → mitigação: nomenclatura explícita na
  doc e no código (`CanalRole`).
- Rollback: papel default `normal`; nenhum comportamento muda.
