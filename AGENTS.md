# AGENTS.md — Dolorestec WYD (Rust)

Instruções para agentes de IA que trabalham neste repositório. Leia antes de qualquer tarefa:
define o que é o projeto, como subir a stack, quais invariantes não podem ser quebradas e onde
está o conhecimento já produzido.

## 0. Projeto 100% Rust (arquivo Go/C++ removido em 2026-11)

O projeto concluou a transição de Go para Rust (`start-rust.md`, `pre-release-rust.md` como
histórico de planejamento). Regras vigentes:

- **Arquivo Go/C++ removido:** não existe mais `.deprecated-go/`, `tmserver/`, `internal/`,
  `Source/`, `roadmap/` (Go) ou `docs/migration/` na árvore. Nada de `go build`, `go test`
  ou referência legada como fonte. Fonte canônica atual é `docs/` + `crates/` + `roadmap-rust/`
  (ver `docs/arquitetura/protocolo.md`, `docs/cliente/`, `docs/rust/`).
- **Trabalho Rust vive em:** `crates/wyd-{protocol,core,net,db,admin,sim,launcher}/`,
  `Cargo.toml` (workspace), `rust-toolchain.toml`, `docs/arquitetura/`, `docs/rust/`,
  `docs/cliente/`, `roadmap-rust/`, `wyd-portal/` (Cold Path), `tools/` (análise/geração),
  `fixtures/cliente/` (goldens).
- **Skills Rust vigentes** (use estas para código Rust; não use skills Go legadas como norma):
  `wyd-rust-core` (engine/Hot Path), `wyd-rust-net` (transporte/protocolo),
  `wyd-rust-db` (persistência), `wyd-rust-sim` (laboratório matemático),
  `wyd-rust-obs` (observabilidade/segurança), `wyd-rust-toolchain` (build/CI),
  `wyd-rust-testing` (harness/goldens/carga), `wyd-launcher` (distribuição/patch). Para o cliente fechado, use `wyd-client-reverso`
  (inventário, RE, manifesto do launcher).
- **Boas práticas vigentes** (valem para todo código novo): `rust-code-quality`
  (ownership, erros tipados, clippy/fmt, testes), `software-architecture`
  (Hot/Cold, DOD/ECS, fronteiras, ADRs), `solid-design-patterns`
  (SOLID e patterns idiomáticos em Rust).
- **Skills legadas Go** (`wyd-server-dev`, `wyd-database`, `wyd-client-connect`,
  `wyd-testing-bots`, `wyd-game-admin`, `wyd-economy`, `wyd-web-admin-ui` legado):
  resolvidas em FASE-09-2 (ver `docs/manual-técnico/skills.md`). Não as use como norma de implementação;
  se precisar de contexto histórico, cite `docs/manual-técnico/auditoria-roadmap.md`
  e proponha a sucessora Rust.
- **Invariantes do núcleo Rust:** Hot Path monolítico no mesmo espaço de memória
  (sem gRPC/HTTP no tick, sem GC, sem mutex global); DOD/ECS com dados contíguos;
  `unsafe` só em interop comprovada; erros tipados (`Result`/`Option`, um erro por
  fronteira); servidor é autoridade (física, dano, skills, inventário);
  `ESCENE_FIELD` (30000) segue como pacote autoritativo de cena;
  `clippy -D warnings` e `rustfmt` limpos; entrada de rede nunca causa `panic!`.
- **Estrutura antes de código:** a árvore de módulos, a nomenclatura e os padrões
  estão em `docs/arquitetura/estrutura-e-padroes.md` (ADR-0002) — árvores
  arquivo-a-arquivo por crate nos §3 (protocol) e §9–§15 (demais + `tools/`).
  Nenhum arquivo novo nas `crates/` fora dessa árvore; mudança de árvore exige ADR
  em `docs/arquitetura/adr/` (rito FASE-13-1). Tabelas de protocolo em `generated/`
  são congeladas (fonte histórica Basedef.h/CPSock.cpp v7662).
- **Qualidade/planejamento:** `roadmap-rust/` documenta as 15 fases; código Rust
  novo nasce com teste (`cargo test --workspace`); ciclo obrigatório em
  `docs/rust/02-execucao.md`; painel vivo em `docs/rust/03-progresso.md`.

## 1. O que é este projeto

Servidor do WYD2 (With Your Destiny) reescrito em Rust (Hot Path monolítico, sem GC),
portal administrativo em Next.js no Cold Path e cliente Windows 7662 como artefato
externo — tudo num único repositório.

| Pasta | O que é | Stack |
|-------|---------|-------|
| `crates/wyd-protocol/` | Contratos de wire/cliente (CPSock, offsets explícitos, checksum) | Rust, sem deps externas |
| `crates/wyd-core/` | Engine Hot Path (tick, ECS SoA, spatial grid) | Rust |
| `crates/wyd-net/` | Transporte async (framing, sessão, rate-limit) | Rust (tokio/io_uring via ADR) |
| `crates/wyd-db/` | Persistência async (Postgres, migrations, delivery_queue) | Rust + sqlx (via ADR) |
| `crates/wyd-admin/` | Control plane Cold Path (auth, grants, auditoria p/ o portal) | Rust |
| `crates/wyd-sim/` | Laboratório matemático (fórmulas puras, cenários, goldens) | Rust |
| `crates/wyd-launcher/` | Launcher/patcher (manifest, patch atômico, bootstrap, update) | Rust std puro |
| `wyd-portal/` | UI de administração: BFF puro sobre `wyd-admin`, sem tocar estado vivo | Next.js 15, TypeScript |
| `client/` | Cliente Windows 7662 (PE32/i386) — usuário do `wyd-launcher`; dossiê em `docs/cliente/`, **ignorado no git** | Binários fechados |
| `roadmap-rust/` | Planejamento vigente: 15 fases + issues por entrega | Markdown |
| `docs/arquitetura/` | Norma técnica (estrutura, protocolo, banco, visao-pastas, ADRs) | Markdown |
| `docs/cliente/` | Dossiê do cliente (inventário, RE, pacotes, depuração) | Markdown + gerados |
| `docs/rust/` | Execução Rust (visão, dependências, ciclo, progresso, stack-docker) | Markdown |
| `tools/` | Análise e geração (client/protocol), nunca runtime | Node |
| `fixtures/cliente/` | Goldens sintéticos de pacote (validados por `captura.mjs`) | JSON |
| `.agents/skills/` | Skills vigentes `wyd-rust-*` + boas práticas + cliente | Markdown |

O objetivo: **servidor WYD2 em Rust operacional, com login pelo cliente 7662,
personagens, batalha, comércio e quests funcionando**, mais portal auditável e
automação por IA para administrar e evoluir o servidor.

`roadmap-rust/index.md` é a fonte da verdade do planejamento. Este arquivo cobre o
repositório inteiro (servidor + portal + cliente + roadmap + skills).

## 2. Caminho feliz (bring-up do zero)

```bash
# 0. Toolchain Rust pinada
cargo --version && rustc --version   # stable via rust-toolchain.toml

# 1. Gates Rust (obrigatório em toda entrega)
make check          # fmt --check + clippy -D warnings + build + test

# 2. Stack atual (Postgres + portal; tmserver/gateway entram com os binários das FASEs)
make stack-config   # valida compose base+prod sem subir
make stack-up       # db + portal em background
docker compose ps   # Up != pronto: ver sinais em docs/rust/05-stack-docker.md
curl -s localhost:3000/login | head -3
curl -s localhost:3000/api/health   # 502 esperado sem gateway (FASE-07-1), não é bug

# 3. Portal em modo desenvolvimento (hot reload), sem container:
make portal-install     # npm ci
make portal-dev         # http://127.0.0.1:3000

# 4. Launcher (lado do cliente, fora do compose)
cargo run -p wyd-launcher -- check /caminho/do/jogo --manifest manifest.toml
```

Detalhes, diagnóstico e alternativas: skills `wyd-server-ops` (stack),
`wyd-rust-toolchain` (gates), `wyd-client-reverso` + `docs/cliente/` (cliente).

Não reinicie serviços no meio de um teste do usuário: derruba save e gera falhas
transitórias que parecem bug. Faça o deploy, **espere** o teste humano terminar.

## 3. Invariantes que não podem ser quebradas

1. **Hot Path monolítico, sem I/O no tick.** Tick, física, combate e visibilidade vivem
   juntos em `wyd-core` no mesmo espaço de memória. Nada de gRPC/HTTP/SQL no loop,
   nada de mutex global, nada de alocação quente. Persistência entra por portas
   (traits) e canais async fora do tick (ver ADR-0001/0004/0006).
2. **Layout binário é por offset explícito.** HEADER 12 bytes LE (`Size/KeyWord/CheckSum/
   Type/ID/ClientTick`), `Size` em [12, 8192], `INITCODE = 0x1F11F311`, transformação
   a partir do offset 4 com `pKeyWord[512]`. Não confie em `repr(Rust)`; `repr(C)` +
   offsets provados por teste. Entrada de rede: política `Strict` (divergência é erro
   tipado); `Tolerant` só em captura/diagnóstico.
3. **Servidor é autoridade.** Cliente nunca dita posição/dano/cooldown/inventário.
   Fórmulas aplicadas vêm de FASE-10 aprovada, não inventadas no core.
4. **Pacote autoritativo de cena vai com `HEADER.ID = ESCENE_FIELD`** (30000), não com o id da
   conexão do dono — senão barra de EXP/HP não atualiza no cliente.
5. **Campo UNVERIFIED não é gate nem vira tipo.** Para comportamento/pacote não capturado:
   tolerar, logar e seguir pelo caminho conservador (`require_ported`/`Unported`).
   Nunca rejeitar ação server-authoritative por sentinela não confirmado, nem inventar
   layout (`MSG_Trade`, `MSG_SetHpMp`, `_AUTH_GAME` fora de `payload/` até FASE-08-2/08-6).
6. **Senhas e PINs com função de hash resistente** (argon2id no legado; equivalente ao
   adotar autenticação em `wyd-db`). Texto plano é proibido, inclusive em log. DSN real
   e segredos nunca versionados.
7. **A web nunca escreve estado de personagem vivo.** Grants vão por `delivery_queue`
   (ver `docs/arquitetura/modelo-banco.md`) que o servidor drena. O portal é BFF: só lê
   e envia intenções auditadas via `wyd-admin`.
8. **`client/` não é versionado.** ~344 MB de binários proprietários; fica no `.gitignore`.
   Nunca commite, nunca inclua no contexto de build do Docker, nunca redistribua.
   Backup antes de escrever; patch só por ferramenta (`wyd-launcher`).

### Regras do portal

- **Toda ação admin passa por `adminCall`** (`wyd-portal/src/server/admin.ts`): sessão de staff,
  `moderator_id` vindo da sessão (nunca do corpo), deadline, mapeamento `AdminResult` → HTTP e
  auditoria. Page/action não chama backend direto sem esse envelope.
- O browser nunca fala com banco nem vê DSN ou stack interna: `toResponse` devolve só `{ error }`
  com `cache-control: no-store`.
- `SESSION_SECRET` é obrigatório fora de desenvolvimento (>= 32 caracteres). O default do compose
  é **apenas para dev**.
- O portal não escreve em tabelas de personagem vivo; overlays têm auditoria; guias citam
  módulos Rust (`*.rs` em `crates/wyd-protocol/src/payload/`), não handlers Go.

## 4. Fontes canônicas (onde está a verdade)

| Preciso de… | Leia (normativo) |
|-------------|------------------|
| Estrutura de módulos, nomenclatura, padrões, SOLID | `docs/arquitetura/estrutura-e-padroes.md` (ADR-0002) |
| Protocolo CPSock (HEADER, ofuscação, catálogo, payloads, UNVERIFIED) | `docs/arquitetura/protocolo.md` + `crates/wyd-protocol/` |
| Tabelas congeladas (opcodes, keyword) | `crates/wyd-protocol/src/wire/generated/`, `codec/generated/` |
| Banco (ER, CHECKs, migrations, delivery_queue, seeds) | `docs/arquitetura/modelo-banco.md` + `crates/wyd-db/` |
| Visão de pastas (inventário, responsabilidade, anti-sprawl, cliente↔launcher) | `docs/arquitetura/visao-pastas.md` |
| ADRs (monolito, ECS, binário, async, launcher) | `docs/arquitetura/adr/` (rito FASE-13-1) |
| Cliente: inventário, superfície, RE, pacotes, depuração, customização | `docs/cliente/` + `tools/client/` + `fixtures/cliente/` |
| Execução Rust (ciclo, gates, DoD, revisão, doc viva) | `docs/rust/02-execucao.md` |
| Dependências Rust (catálogo, decisão offline, como adicionar) | `docs/rust/01-dependencias.md` |
| Stack Docker (crates→deployáveis, o que sobe, proibições) | `docs/rust/05-stack-docker.md` |
| Progresso e evidências | `docs/rust/03-progresso.md` + `roadmap-rust/index.md` |
| IA de NPC, bots de carga e LLM (o que existe × planejado) | `docs/rust/07-ia-e-bots.md` + `docs/arquitetura/adr/0010-ia-npc-e-bots.md` |
| Contrato de um opcode | `docs/arquitetura/protocolo.md` §4–§5 + `crates/wyd-protocol/src/payload/<domínio>.rs` |
| Fluxos ponta-a-ponta | `docs/cliente/pacotes.md` + `crates/wyd-net/tests/` |
| Fórmulas/balanceamento | `crates/wyd-sim/` + issues FASE-10 |
| Lore, classes, skills, sistemas e itens | `docs/wyd/` (base de conhecimento) + `docs/wyd/itens.md` (classificação de item) |
| Portal: BFF, telas, sessão, guias | `wyd-portal/README.md` + skill `wyd-web-admin-ui` (via FASE-09) |
| Plano vigente (15 fases, dependências, status) | `roadmap-rust/index.md` + `roadmap-rust/issues/FASE-*.md` |
| Auditoria do planejamento histórico | `docs/manual-técnico/auditoria-roadmap.md` |
| Mapa de skills Go→Rust | `docs/manual-técnico/skills.md` + FASE-09-1/09-2 |

Antes de mexer em wire/format/gameplay, leia o doc correspondente. O que não está
documentado é **hipótese/UNVERIFIED**: não vira tipo, não vira gate, não vira regra.

## 5. Verificação (escada mínima, do mais barato ao mais caro)

```bash
make check          # gate único: fmt --check + clippy -D warnings + build + test
cargo test --workspace           # unitários + integração (goldens de protocolo, determinismo do core)
cargo test -p wyd-protocol -p wyd-net   # bytes no fio quando mexeu em protocolo/rede
make stack-config   # compose base+prod válidos sem subir nada
make portal-test    # tsc --noEmit + vitest (portal)
make portal-build   # next build (valida rotas e standalone)
```

Sem `make` no host Windows (Git Bash), os quatro comandos `cargo` equivalentes valem
como gate (registrar qual foi usado na evidência). Bench/regressão (`make bench`)
quando a issue exigir números (FASE-05/10/15).

## 6. Skills disponíveis (`.agents/skills/`)

Vigentes (use estas):

| Skill | Use quando |
|-------|-----------|
| `wyd-rust-core` | engine Hot Path (tick, ECS, spatial grid) |
| `wyd-rust-net` | transporte/protocolo (framing, sessão, rate-limit, parsers) |
| `wyd-rust-db` | persistência (model, repo, migrations, delivery_queue) |
| `wyd-rust-sim` | matemática/balanceamento (fórmulas puras, cenários, goldens) |
| `wyd-rust-obs` | observabilidade/segurança (tracing, métricas, hardening) |
| `wyd-rust-toolchain` | toolchain/build/CI (fmt, clippy, deps, gates) |
| `wyd-rust-testing` | harness, goldens, carga e bots (sem cliente gráfico) |
| `wyd-launcher` | distribuição/patch do cliente (manifest, patch, bootstrap) |
| `wyd-client-reverso` | RE do `client/` (inventário, offsets, manifesto do launcher) |
| `wyd-server-ops` | operação da stack Docker (subir, logs, banco, diagnóstico) |
| `rust-code-quality` | qualidade de código Rust (ownership, erros, testes) |
| `software-architecture` | arquitetura (Hot/Cold, DOD/ECS, fronteiras, ADRs) |
| `solid-design-patterns` | SOLID/patterns idiomáticos em Rust |
| `github-actions-docs` | sintaxe de workflows (CI Rust) |
| `documentation-writer` | relatórios/guias (Diátaxis) |
| `frontend-design`, `ui-ux-practices` | direção visual e revisão de UI/UX do portal |

Legadas Go (não usar como norma; destino em FASE-09-1/09-2): `wyd-server-dev`,
`wyd-database`, `wyd-client-connect`, `wyd-testing-bots`, `wyd-game-admin`,
`wyd-economy`, `wyd-web-admin-ui` (legado).

## 7. Convenções de trabalho

- **Estrutura antes de código** (`estrutura-e-padroes.md` §8 + ADR-0002): liste arquivos
  criados/alterados na issue antes de criar; nenhum `utils.rs`/`helpers.rs`, nenhum
  `mod.rs` com lógica, nenhuma tabela fora de `generated/`, nenhum teste fora de
  `tests/` (integração) ou `#[cfg(test)]` (unidade).
- **Uma issue por vez** (`docs/rust/02-execucao.md` §1): Análise → Design (ADR se
  estrutural) → Código mínimo → Teste/gate → Docs (progresso + status + docs
  invalidados) → Fechamento em português.
- **Mudança de contrato/árvore exige ADR** antes do código; doc que descreve
  comportamento muda na entrega que muda o comportamento.
- Prefira editar o que existe a criar arquivo novo; o roadmap já diz onde cada coisa mora.
- Docs, UI e mensagens em português; commits pequenos e descritivos em português;
  `develop` como branch principal (fluxo em `docs/rust/02-execucao.md` §2).
- Nunca versione segredos, `serverlist.bin` com IP privado, dumps de conta, binários do
  cliente (`client/`), `target/` ou project-id do Freebuff (`.freebuff/`).
- Ao concluir uma issue, atualize o status em `roadmap-rust/index.md`, o painel em
  `docs/rust/03-progresso.md` e qualquer doc invalidado na mesma entrega.
- **Definition of Done** por tipo em `docs/rust/02-execucao.md` §3; revisão consolidada
  a cada 10 issues (§4).

## Comandos

```bash
make check          # gate completo (fmt + clippy + build + test)
make fmt/clippy/build/test  # gates isolados
make audit          # supply-chain (cargo-audit; exige binário instalado)
make bench          # benchmarks por crate crítica
make ci             # espelho local do .github/workflows/rust.yml

make stack-config   # valida compose (base+prod) sem subir
make stack-up       # sobe db + portal (dev)
make stack-down     # derruba mantendo pgdata
make stack-logs     # segue logs

make portal-install # npm ci em wyd-portal/
make portal-dev     # portal em http://127.0.0.1:3000 (hot reload)
make portal-test    # tsc --noEmit + vitest
make portal-build   # next build (standalone)
