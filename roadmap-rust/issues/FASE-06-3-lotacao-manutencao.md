# Issue 06-3 — Lotação e manutenção do canal: recusa de entrada tipada

**Fase:** 6 · **Depende de:** 06-2 ✅ · **Bloqueia:** — · **Esforço:** S

## Objetivo

Fechar o checklist 1.14 (“canal cheio / manutenção / recusa de entrada”): limite de jogadores
por canal e modo manutenção, com recusa antes de criar sessão, log e métrica.

## Contexto / evidência

`crates/wyd-server/src/channels.rs` conta canais habilitados (`World`), mas **não há regra de
lotação nem de manutenção** — o checklist 1.14 está ⬜ e a ADR-0022 §7 só define o papel
normal/verde. O limite por IP (`ConnectionLimit`, FASE-06-2) existe e é outra coisa: ele conta
conexões TCP, não jogadores.

Sem essa regra, um canal aceita conexões até o SO recusar, e não há como tirar um canal do ar
para manutenção sem parar o container.

## Passos

1. `channels.rs`: `max_players` (default 1000, o `MAX_USER` do legado) e `maintenance`
   (bool) por canal; fonte: `server_channel` (migration nova só se faltar coluna) ou env
   `WYD_SERVER_MAX_PLAYERS`/`WYD_SERVER_MAINTENANCE` com validação no boot.
2. Contagem de **sessões vivas** (não conexões): entrar conta, sair desconta; o handshake
   recusa quando `>= max_players`.
3. Manutenção: recusa com motivo claro no log e no status (`/serv00.htm`), sem derrubar
   sessões existentes; novo container não é necessário.
4. Métrica: usar nome já declarado (`wyd_sessions_open`, `wyd_server_channels_published`);
   recusa entra em `wyd_packets_rejected_total{reason="canal_lotado"|"manutencao"}` — se o
   nome de label não existir, declarar em `docs/rust/05-stack-docker.md` §4 antes de publicar.
5. Testes: `max_players=1` recusa a segunda **sessão** e libera ao sair; manutenção recusa sem
   afetar sessão viva; `serv00.htm` reflete o estado.

## Arquivos

- Criados: `crates/wyd-server/tests/lotacao.rs`.
- Alterados: `crates/wyd-server/src/{channels,status,cpsock}.rs`, `docker-compose.yml`
  (env nova, se houver), `docs/rust/05-stack-docker.md` §4, `docs/rust/08-checklist-funcionalidades.md` (1.14).
- Proibido tocar: `client/`, regra de cidadania (03-10), compose de produção além de env.

## Testes / verificação

```bash
cargo test -p wyd-server --test lotacao
cargo test -p wyd-server           # suíte do processo
cargo clippy --workspace --all-targets -- -D warnings
```

Saída esperada: segunda sessão recusada com motivo; `serv00.htm`/`/health` mostram o estado;
métrica de recusa incrementa.

## Skills Relacionadas do freebuff

- .agents/skills/wyd-rust-obs — métrica e log sem segredo.
- .agents/skills/wyd-rust-net — recusa no handshake, sem `panic!`.
- .agents/skills/wyd-server-ops — operação de canal e manutenção.

## Documentação de referencia

- `docs/rust/05-stack-docker.md` §6 (canais) e §4 (métricas declaradas).
- `docs/wyd/canais-e-cidadania.md` §1/§5.1 (canal × servidor; limite 10 canais).
- `docs/arquitetura/contrato-transporte-cpsock.md` §8 (paridade do handshake).

## Subtasks

- [x] Codificar issue
- [x] Testar issue
- [x] Revisar issue
- [x] Documentar issue
- [ ] Versionar issue, mergear branch e fechar issue
- [ ] Limpar branch (se aplicável)

## Critérios de aceite

- [x] Lotação conta sessão viva, não conexão TCP.
- [x] Manutenção recusa entrada nova e preserva sessão em andamento.
- [x] Recusa é observável (log estruturado + métrica declarada).
- [x] Nenhum teste depende de `sleep` arbitrário.

## Evidência (2026-11-10)

- Implementação: `0040_capacidade_canal.sql` (tabela + seed 1000 por canal 1–10,
  a série proíbe `ALTER TABLE` no `up`), `ChannelCapacity` + `Sessions::channel_capacity`
  (`wyd-db`, com `FakeSessions` permissivo), `ChannelPolicy`/`PolicyOverrides` +
  gates em `SessionGate::claim` (`wyd-server/src/session.rs`), motivos
  `canal_lotado`/`manutencao` no `cpsock` e em `wyd-net/src/metrics.rs`, env
  `WYD_SERVER_MAX_PLAYERS`/`WYD_SERVER_MAINTENANCE` (vazio = banco) com aviso de boot
  quando o teto < 10, `maintenance` no `/health` (`serv00.htm` mantém os números do
  contrato `sscanf`), env nos dois composes do canal.
- `cargo test -p wyd-server --test lotacao`: 5/5 ok (seed 1000, teto 1 recusa a 2ª e
  libera ao sair, manutenção recusa sem derrubar a viva, env, métricas).
- Gates: `cargo fmt --check` 0 · `cargo clippy -D warnings` 0 ·
  `cargo test --workspace --all-targets` 0 · `check-migrations` ok (40/81) ·
  `check-issue-status` ok.

## Riscos / rollback

- Risco: limite configurado baixo por engano travar teste do dono → mitigação: default 1000 e
  aviso no boot quando `max_players < 10`.
- Rollback: remover a checagem; `ConnectionLimit` por IP continua valendo.
