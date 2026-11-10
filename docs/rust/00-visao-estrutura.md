# Visão da transição Go → Rust — planejamento, inventário e estrutura proposta

> **Documento histórico (snapshot de 2026-11-07).** Ele registra o estado da transição
durante o arquivamento do Go (7 crates, 15 skills, Go ainda no PATH) e **não** descreve o
presente. Estado vigente: **8 crates** (inclui `wyd-server`), 27 skills, Go fora da fonte de
build — ver `docs/rust/03-progresso.md`, `docs/arquitetura/estrutura-e-padroes.md` §1/§16 e
`docs/rust/04-revisao-roadmap.md` §6.

> Origem: `start-rust.md` + `pre-release-rust.md`. Escopo deste passe: **esqueleto Rust compilável sem arquivar/mover/apagar Go e sem desinstalar Go**.
> Estado em 07/11/2026 (histórico, não vigente): toolchain `stable 1.99.1` presente (`cargo --version` OK), workspace `Cargo.toml` na raiz com os 7 crates de `crates/`, `rust-toolchain.toml` pinando stable, código mínimo que compila, `cargo build --workspace` + `cargo test --workspace` + `clippy -D warnings` + `fmt --check` passando, `go build ./...` ainda passando, `git status` sem remoções de Go.

## 1. Inventário atual (07/11/2026)

- Branch: `develop` local ativa (commit `b58a607`), **sem `origin/develop`** — `origin/HEAD -> origin/main`.
- Working tree: limpo, exceto `?? start-rust.md` (não commitado).
- Arquivos rastreados: **1499** (`git ls-files | wc -l`).
- `deprecate-go/`: não existe. `Cargo.toml` / `rust-toolchain*`: não existem.
- Toolchains: `go version go1.27.3 windows/amd64` em `D:\Programas\Go`; `cargo`/`rustc`: **não encontrados**.
  - `go env`: `GOROOT=D:\Programas\Go`, `GOPATH=C:\Users\Canta\go`, `GOCACHE=C:\Users\Canta\AppData\Local\go-build`, `GOMODCACHE=C:\Users\Canta\go\pkg\mod`.
- VS Code: extensões instaladas (sem nenhuma de Go — nada a remover aqui):
  `cweijan.vscode-postgresql-client2`, `github.vscode-github-actions`, `ms-azuretools.vscode-containers`,
  `ms-ceintl.vscode-language-pack-pt-br`, `ms-playwright.playwright`, `ms-vscode-remote.*`, `pkief.material-icon-theme`, `vitest.explorer`.
- Roadmap Go: `roadmap/issues/` (30+ arquivos), `roadmap/done/`, `roadmap/test/`, `index.md`, `PLANO-2026-11.md`.
- Skills: 15 pastas em `.agents/skills/` (ex.: `wyd-server-dev`, `wyd-server-ops`, `wyd-database`, `wyd-client-connect`, `wyd-testing-bots`, `wyd-web-admin-ui`, `wyd-economy`, `wyd-game-admin` + 7 genéricas). Exemplo lido: `wyd-server-dev/SKILL.md` — centrada nas 4 invariantes Go (dono único, offset explícito, ordem do RNG, autoridade de cena). Todas as skills `wyd-*` precisarão de sucessoras Rust.
- Referências de wire/gameplay a preservar antes de qualquer desativação (pre-release.md §1.2): `Source/`, `docs/migration/` (protocol-spec, data-formats, game-rules, handlers 58/58, flows, game-messages), `tmserver/content/` (taxas obrigatórias, mapas 17 MiB fora do git, BaseMob/BaseSummon), catálogos no Postgres via dbserver.

## 2. Limpeza do ambiente Windows/Go — o que seria removido (NÃO executado automaticamente)

> Por que não executei a desinstalação direta: remover o Go agora quebra `make build/test`, `make run-local` e o e2e headless enquanto o código Go continua sendo os 1499 arquivos do repo (arquivamento adiado). Desinstalação também exige elevação (admin) e é irreversível sem reinstalação.

Alvos identificados (a partir de `go env`):

1. `D:\Programas\Go` (GOROOT + toolchain 1.27.3) — desinstalar via Apps/winget, não só apagar pasta.
2. `C:\Users\Canta\go\pkg\mod` (GOMODCACHE) — cache de módulos, regerável com `go mod download`.
3. `C:\Users\Canta\AppData\Local\go-build` (GOCACHE) — cache de build, seguro apagar com `go clean -cache`.
4. `C:\Users\Canta\go\bin` (se existir) — binários instalados via `go install`.
5. Extensões VS Code de Go — **nenhuma instalada** (`code --list-extensions` não retorna `golang.go`); nada a fazer.
6. Variáveis de usuário `GOPATH`/`GOROOT`/`GOBIN`, se definidas fora do padrão.

Script revisável (rodar em Git Bash / PowerShell elevado, **após** commit + backup + arquivamento):

```bash
# 1) Inventário (somente leitura)
go env GOROOT GOPATH GOCACHE GOMODCACHE
code --list-extensions | grep -i -E "go|golang|tinygo" || echo "sem extensoes Go"
# 2) Limpeza de caches (reversível via re-download)
go clean -cache -modcache
# 3) Remover extensões Go (se houver; hoje não há)
# code --uninstall-extension golang.go
# 4) Desinstalar toolchain (Windows, elevado — escolha UMA):
# winget uninstall --id GoLang.Go
# ou: Painel de Controle > Programas > Go > Desinstalar
# 5) Conferir
go version; cargo --version; rustc --version
```

Recomendação: só rode o passo 4 depois que `deprecate-go/` existir, `develop` estiver pusheada e um backup (zip ou branch `archive-go`) estiver feito.

## 3. Estrutura Rust proposta (pastas vazias criadas com `.gitkeep`, sem workspace ainda)

Princípio do `pre-release-rust.md`: **Hot Path monolítico em Rust por nó de mundo** (mesmo espaço de memória, ECS, thread-per-core, arena allocators) + **Cold Path periférico** (auth, billing, portal, matchmaking) via HTTP/JSON assíncrono (ADR-0008). Nada de microsserviços no tick loop, nada de GC no core, nada de OOP espalhada no heap.

```
crates/
  wyd-protocol/   # contratos de wire/cliente: #[repr(C)], zerocopy/rkyv, parsers nom, CPSock
  wyd-core/       # engine: tick determinístico 20-50/s, ECS (hecs/bevy_ecs), spatial grid, fixed-point
  wyd-net/        # transporte async: tokio (+ tokio-uring/io_uring no Linux), zero-copy, rate-limit/DoS
  wyd-db/         # persistência async: Postgres sqlx + Redis cache, migrations versionadas
  wyd-admin/      # control plane Cold Path: HTTP/JSON p/ portal Next.js (ADR-0008; substitui binserver/webserver)
  wyd-sim/        # laboratório matemático: simulação de combate/balanceamento, milhões de iterações
  wyd-launcher/   # launcher/patcher moderno (Tauri/CLI): hash, patch, bootstrap (FASE 8)
tools/dev/        # scripts de dev/bench (futuro; hoje vazio)
docs/rust/        # este doc + futuros ADRs, spec de pacotes, guias (portal Next.js continua em wyd-portal/)
```

O portal Next.js (`wyd-portal/`) é **mantido** como gateway Cold Path (FASE 7) — não entra no `crates/`.
`Source/`, `docs/migration/` e `tmserver/content/` ficam como **referência de engenharia reversa** até a extração do conhecimento (FASE 10) estar documentada; só então desativar (pre-release.md §1.2).

`Cargo.toml` workspace, `rust-toolchain.toml`, CI (`fmt/clippy/test/bench/audit/deny`), Docker multi-stage distroless e `tracing`+Prometheus/Grafana ficam para **depois** que o roadmap Rust estiver aprovado — criar agora violaria o "sem código/configs ainda".

## 4. Levantamento de stack Rust (para instalações futuras, NÃO instalar agora)

| Necessidade (legado → destino) | Candidatos (pre-release-rust) | Observação |
|---|---|---|
| Engine ECS/DOD | `hecs` ou `bevy_ecs`, `bumpalo` (arena), fixed-point (`fixed`/`decorum` a avaliar) | Localidade de cache + zero alloc no tick |
| Concorrência | `crossbeam-channel` / ring buffers lock-free, thread-per-core + spatial partition | Sem mutex global no mundo |
| Rede async | `tokio` (+ `tokio-uring` no Linux), `nom`, `zerocopy`/`rkyv` | Substitui select/poll; valida limites |
| Persistência | `sqlx` (Postgres async) + `refinery`/`sqlx-cli`, Redis transacional | Substitui `.dat` + dbServer em Go |
| Scripting seguro | `wasmtime` (sandbox) ou sistemas compilados | Substitui Lua acoplada |
| Observabilidade | `tracing`, `prometheus`, Grafana | Tick P50/P95/P99, filas, entidades, memória |
| Qualidade/CI | `rustfmt`, `clippy -D warnings`, `cargo test --all-targets`, `cargo bench`, `cargo audit`/`deny`, `cargo-tarpaulin`, `miri` | Gates da FASE 5/15 |
| Launcher | Tauri ou CLI Rust | Hash/patch/bootstrap |
| Portal (mantido) | Next.js App Router já existente + Playwright E2E | Gateway Cold Path, BFF |
| Docs | MkDocs existente + ADRs (docs/arquitetura/adr/) | Uma ADR por decisão grande (FASE 13) |

Pré-requisito de host: `rustup` + toolchain stable + `clippy` + `rustfmt` (ausentes hoje — instalar só após aprovação do roadmap).

## 5. Roadmap Rust e skills (implementado neste passe, sem reescrever `roadmap/` Go)

- `roadmap-rust/index.md` + `roadmap-rust/issues/FASE-01.md`…`FASE-15.md` criados a partir de `pre-release-rust.md` (15 fases + §16); `roadmap/` Go preservado intacto.
- Skills criadas: `wyd-rust-core`, `wyd-rust-net`, `wyd-rust-db`, `wyd-rust-toolchain` (ver `roadmap-rust/issues/FASE-09.md` para o restante do mapeamento: `wyd-testing-bots` → sim Rust, `wyd-server-ops`/`wyd-web-admin-ui` atualizadas depois, `wyd-rust-obs` futura). Nenhuma skill Go apagada.

## 6. VS Code, CI, Docker (implementado neste passe)

- `.vscode/extensions.json` (rust-analyzer, Even Better TOML, crates, CodeLLDB + mantidas) e `.vscode/settings.json` (clippy como check, format on save) criados; nenhuma extensão desinstalada (não havia Go instalado no editor).
- `.github/workflows/rust.yml` (fmt → clippy `-D warnings` → build → test) e Dockerfiles por serviço em `docker/` (`Dockerfile.launcher`, `Dockerfile.portal`) criados; workflows Go/portal intocados.
- Dependências reais catalogadas em `docs/rust/01-dependencias.md`; workspace segue sem deps externas para build offline-verificável (ver §7).

## 7. Estado efetivo deste passe (esqueleto compilável, Go intacto)

- Toolchain: `rustup` já existia em `C:\Users\Canta\.cargo\bin` sem default e com manifesto parcial; reparado com `rustup default stable` + `rustup update stable`. Prova: `cargo 1.99.1`, `rustc 1.99.1`, `rustup show active-toolchain` = `stable-x86_64-pc-windows-msvc (default)`.
- Workspace raiz (`Cargo.toml`, virtual, `resolver = "2"`): membros `crates/wyd-protocol`, `wyd-core`, `wyd-net`, `wyd-db`, `wyd-admin`, `wyd-sim`, `wyd-launcher`. Sem dependências externas para permitir build offline; `wyd-core`/`wyd-net` dependem de `wyd-protocol` via `path`.
- Manifestos + código mínimo por crate (cada um com 1 teste unitário): `wyd-protocol` (PROTOCOL_VERSION=12000, ESCENE_FIELD=30000), `wyd-core` (TICK_HZ=20), `wyd-net` (MAX_PACKET_BYTES=8192, que é **alias** de `MAX_MESSAGE_SIZE` — a fonte única), `wyd-db` (scheme postgres://), `wyd-admin` (/admin), `wyd-sim` (saturating_sub), `wyd-launcher` (lib + bin mínimo).
- Verificação: `cargo build --workspace` OK (7 crates), `cargo test --workspace` OK (7 testes lib + doc-tests 0), `cargo clippy --workspace -- -D warnings` OK, `cargo fmt --check` OK, `go build ./...` OK (Go 1.27.3 intacto).
- `.gitignore`: acrescida seção Rust (`/target/`, `**/*.rs.bk`) — nenhuma regra Go removida.
- Neste passe (não-destrutivo): `roadmap-rust/` + 15 issues, 4 skills Rust, `rust.yml`, `docker/Dockerfile.launcher` (+ `Dockerfile.portal`), `.vscode/`, `docs/rust/01-dependencias.md`, `docs/arquitetura/adr/0001-hot-path-monolitico.md`; workspace mantido sem deps externas (build offline).

## 8. Arquivamento executado e pendências restantes

1. [x] Arquivamento executado em `arquivo Go removido (2026-11)` (nome confirmado pelo dono):
   Go legado movido com histórico preservado (rename); `wyd-portal/`, `.agents/`,
   workspace Rust, `docs/rust/`, `roadmap-rust/` e `README.md`/`AGENTS.md` ficaram
   fora; `docs/migration/` e `Source/` (referência) estão em
   `docs/` e `docs/cliente/ + tools/client/ (fontes C++ removidas com o arquivo Go)`; regra no
   `.gitignore`; `README.md` reescrito para a nova infra.
2. Limpeza/desinstalação do Go no Windows — pendente, destrutivo, exige ação
   elevada do usuário (ver §2).
3. Push/commit/merge (`develop` → `origin`) — pendente, vedado sem ordem.
4. Dependências reais (`tokio`, ECS, `sqlx`, etc.) — exigem rede/vendor + ADR;
   catalogadas em `01-dependencias.md`.
