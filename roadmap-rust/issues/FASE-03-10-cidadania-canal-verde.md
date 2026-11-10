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

- [ ] Codificar issue
- [ ] Testar issue
- [ ] Revisar issue
- [ ] Documentar issue
- [ ] Versionar issue, mergear branch e fechar issue
- [ ] Limpar branch (se aplicável)

## Critérios de aceite

- [ ] Cidadania altera o drop por parâmetro de conteúdo, com teste determinístico.
- [ ] Canal verde não aplica PvP e permite loja; canal normal mantém a regra completa.
- [ ] Papel do canal é configurável por dado/env, nunca por build.
- [ ] `/getout` tratado sem inventar layout quando não há prova.

## Riscos / rollback

- Risco: bônus de cidadania virar “buff global” por erro de derivação → mitigação: teste com
  dois personagens de canais diferentes na mesma seed.
- Risco: canal verde confundido com “evento de novato” → mitigação: nomenclatura explícita na
  doc e no código (`CanalRole`).
- Rollback: papel default `normal`; nenhum comportamento muda.
