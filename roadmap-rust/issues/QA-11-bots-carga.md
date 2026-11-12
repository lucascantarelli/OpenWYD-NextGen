# QA-11 — Bots jogáveis e carga (`crates/wyd-bots`)

> **ID canônico:** `QA-11`
> **ID histórico:** `FASE-15-3`
> **Estado 2026-11-12:** 🟡 · crate `wyd-bots` joga pelo fio; falta a prova de carga/relatório de 200 bots. Ver `roadmap-rust/index.md`.

**Fase:** 15 · **Depende de:** GAME-06, CLIENT-02, QA-05 · **Bloqueia:** QA-10 · **Esforço:** M

## Objetivo

Criar bots que jogam pelo **mesmo protocolo do cliente** — login, movimento, combate e loja —
para QA, carga e economia, com a política testada no laboratório e nenhum código de bot
dentro do processo do servidor.

## Contexto / evidência

A skill de harness existe (`wyd-rust-testing`, sucessora da `wyd-testing-bots`;
`docs/rust/03-progresso.md`), e o crate **`crates/wyd-bots` já existe** (2026-11-10):
`bot/{mod,movimento,combate,comercio,campanha,cena,recusa,config}.rs`, login/PIN/
`CharacterLogin`, `Action`, `AttackOne`, loja e campanha pelo protocolo real;
`tests/sistema/cena_campo.rs` é o harness vivo gated por `WYD_LIVE_CPSOCK`. O que falta é a
**carga de 200 bots**, o relatório P50/P99/rejeições e a prova de política determinística no
laboratório. `pre-release-rust.md` FASE 15 pede "carga de rede: harness contra `wyd-net`
medindo pacotes/s, P99 e rejeições". Fronteiras: `docs/arquitetura/adr/0010-ia-npc-e-bots.md`
(camada B); plano em `docs/rust/07-ia-e-bots.md`.

## Passos

1. `crates/wyd-bots/src/lib.rs`: sessão (conecta, autentica, entra no mundo) usando
   `wyd-protocol` + `wyd-net`, sem atalho interno.
2. `crates/wyd-sim/src/bot/{persona,policy}.rs`: perfil e política pura
   (estado → ação), reusada pelo bot e pelos cenários do laboratório.
3. `src/scenario.rs` + `src/bin/wyd-bots.rs`: CLI (`--bots`, `--perfil`, `--alvo`,
   `--duracao`, `--csv`), carga escalonada e relatório (P50/P99, pacotes/s, rejeições).
4. Telemetria de cliente: RTT, pacotes descartados por rate-limit (`wyd-net`), erros de parse.
5. Testes: dois bots em loopback contra servidor local (login → movimento → saída),
   política determinística por seed, e carga de 200 bots com relatório anexado.

## Arquivos

- Criados (árvore de `estrutura-e-padroes.md` §1/§14, autorizada pela ADR-0010):
  `crates/wyd-bots/{Cargo.toml,src/lib.rs,src/scenario.rs,src/bin/wyd-bots.rs}`,
  `crates/wyd-sim/src/bot/{mod,persona,policy}.rs`.
- Alterados: `Cargo.toml` (workspace), `docs/arquitetura/estrutura-e-padroes.md` §1/§14
  (status: de "planejado" para "executado"), `docs/rust/07-ia-e-bots.md`,
  `roadmap-rust/index.md`, `docs/rust/03-progresso.md`.
- Proibido tocar: código do servidor (`crates/wyd-core/`), `client/`, compose de produção
  (bot não é serviço).

## Testes / verificação

```bash
cargo test -p wyd-bots -p wyd-sim
cargo run -p wyd-bots -- --bots 2 --duracao 10
cargo clippy --workspace --all-targets -- -D warnings
```

Saída esperada: dois bots completam login/movimento/saída; relatório de carga com P99;
nenhuma dependência do servidor em `wyd-bots`.

## Skills Relacionadas do freebuff

- .agents/skills/wyd-rust-testing - harness, carga, goldens.
- .agents/skills/wyd-rust-net - framing, sessão, rate-limit.
- .agents/skills/wyd-rust-sim - política pura e cenários.

## Documentação de referencia

- docs/arquitetura/adr/0010-ia-npc-e-bots.md
- docs/rust/07-ia-e-bots.md §5–§8
- pre-release-rust.md FASE 15 (carga de rede)
- docs/rust/05-stack-docker.md (bot não entra no compose)

## Subtasks

- [x] Sessão de protocolo real (`INITCODE` → `AccountLogin` → `AccountSecure` → `CharacterLogin`) em `crates/wyd-bots`.
- [x] Ações jogáveis: `walk_to`/`action_at`, `attack_at`/`kill`, `open_shop`/`buy`, `quest`/`campaign`.
- [x] Sem dependência de `wyd-core`/`wyd-db` no `Cargo.toml` do bot.
- [ ] Política determinística por seed com golden em `wyd-sim`.
- [ ] Carga de 200 bots + relatório P50/P99/rejeições.
- [ ] Prova viva de dois bots em loopback (login → movimento → saída).

## Critérios de aceite

- [ ] Bot conecta, autentica, move e sai por protocolo real (sem função interna do servidor).
- [ ] Política determinística por seed, com golden em `wyd-sim`.
- [ ] Carga de 200 bots com P99 e rejeições anexados.
- [x] `cargo tree -p wyd-bots` sem dependência de `wyd-core`/`wyd-db`.

## Riscos / rollback

- Risco: bot "trapacear" usando API interna → mitigação: revisão do `cargo tree` no aceite.
- Risco: carga derrubar o ambiente de teste → mitigação: alvo loopback, limite de conexões.
- Rollback: remover as duas entradas do workspace (`wyd-bots`, `sim/src/bot`); nada mais muda.
