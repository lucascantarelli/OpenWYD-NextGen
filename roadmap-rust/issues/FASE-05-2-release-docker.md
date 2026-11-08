# Issue 05-2 — Release LTO + imagem mínima + proveniência

**Fase:** 5 · **Depende de:** 05-1 · **Bloqueia:** FASE-15 · **Esforço:** S
**Estado:** ✅ entregue em 2026-11-08 (ADR-0017)

## Objetivo

Release otimizado (LTO) empacotado em imagem mínima multi-stage, com build
reproduzível e smoke do binário.

## O que foi entregue

| Peça | Conteúdo |
|---|---|
| `Cargo.toml` | `[profile.release]` do workspace: `lto = "thin"`, `codegen-units = 1`, `strip = "symbols"`, `panic = "abort"`. Sem `[profile.bench]`: o Cargo ignora `panic` nesse perfil (avisava no manifesto) |
| `docker/Dockerfile.launcher` | Multi-stage real: builder `rust:1-bookworm` (cache BuildKit de registry e `target/`) e runtime `gcr.io/distroless/cc-debian12:nonroot` — glibc para binário gnu, sem shell/apt/toolchain, `USER nonroot:nonroot`, `ENTRYPOINT`, `CMD ["--help"]` e labels OCI (`version`, `revision`, `base.name`) |
| `.dockerignore` | Contexto mínimo: fora `target/`, `client/`, `.git`, `node_modules`, `.next`, logs e temporários |
| `Makefile` | `release` (`cargo build --release --workspace`), `docker` (build com `WYD_VERSION`/`WYD_COMMIT`) e `docker-smoke` (tamanho + `check` dentro do container) |
| `.github/workflows/rust.yml` | Job `release`: `cargo build --release --workspace`, `docker build` do launcher, `docker image inspect`, `docker build --check` do portal; `docker/**` entrou no gatilho |
| `docs/arquitetura/adr/0017-*` | Decisão registrada (thin vs fat, distroless cc vs static/alpine, proveniência por build-arg) |

`deny.toml` **não** mudou: a imagem não adiciona dependência de crate (o estágio final só copia
o binário e a libc da base).

## Medições (árvore final, 2026-11-08)

### Perfil de release — mesmo binário, antes e depois

| Item | Antes (default do Cargo) | Depois (`[profile.release]`) |
|---|---|---|
| `target/release/wyd-launcher` | 323.072 bytes | **296.448 bytes** (−26.624 / −8,2%) |
| flags | sem LTO, 16 codegen-units, símbolos, unwind | `lto=thin`, `codegen-units=1`, `strip=symbols`, `panic=abort` |
| `cargo build --release --workspace` | exit 0 | **exit 0** (rebuild completo 13,89 s no host) |
| execução | `check` exit 0 | **`check` exit 0** (`OK jogo.txt` / `1 íntegros`) |

### Imagem

```text
docker build -f docker/Dockerfile.launcher -t wyd/launcher:check .   → exit 0
docker image inspect  → size=37.847.971 bytes (36,1 MiB), 20 camadas, User=nonroot:nonroot
labels → version=0.1.0 revision=2f8b033 base=gcr.io/distroless/cc-debian12:nonroot
```

### Smoke dentro do container (execução real)

```text
docker create wyd/launcher:check check /jogo /jogo/manifest.toml
docker cp target/smoke/. <cid>:/jogo/
docker start -a <cid>
  OK      jogo.txt
  manifesto build 1: 1 íntegros, 0 com problema        → exit 0
docker run --rm wyd/launcher:check --help              → imprime uso, exit 2 (contrato do binário)
```

Conteúdo da imagem (`docker export` + `tar -t`): 1.792 arquivos, `usr/local/bin/wyd-launcher`
como único binário de aplicação, **0** ocorrências de `/bin/sh`, `bash`, `rustc`, `cargo`,
`gcc`, `apt` ou `dpkg` — runtime mínimo comprovado, não só declarado.

### Bench contra a baseline da 05-1 (sem regressão)

| Bench | Baseline (05-1) | Árvore final (perfil LTO) |
|---|---|---|
| IA de NPC (`wyd-core/tick`) | 205,86 µs/tick, p99 420,10 | **85,6–189,7 µs/tick, p99 139,9–397,5** (todas acima da baseline) |
| Piso sem jogador | 29,14 µs | 25,05 µs |
| `TokenBucket` (`wyd-net/limits`) | 8,1 ns/op | **2,7–2,8 ns/op** |
| `ConnectionGate` | 234,3 ns/handshake | **100,1–100,8 ns/handshake** |

### Gates re-rodados na árvore final

```text
cargo test --workspace                         exit 0 — 30 alvos, 0 falhas
cargo clippy --workspace --all-targets -D warnings  exit 0 — 0 erro, 0 warning
cargo fmt --all --check                        exit 0
node tools/content/check-migrations.mjs        exit 0 — 24 migrations, 59 tabelas, 140 INSERTs
node tools/content/gen-catalogo-banco.mjs --check   exit 0 — catálogo em dia
node tools/content/verify-schema.mjs           exit 0 — 59 tabelas, 457 colunas, 59 PKs
node tools/content/migrate-cycle.mjs           exit 0 — up/down/up idêntico, 49,3 s
cargo audit                                    exit 0 — 7 crate dependencies
cargo deny check                               exit 0 — advisories/bans/licenses/sources ok
```

## O que ficou sem prova / segue aberto

- **Paridade de runner:** o build da imagem e o smoke foram executados no host de
  desenvolvimento (Docker 29.9.2, linux/amd64). O job de CI roda em `ubuntu-24.04`; não houve
  execução do workflow em runner nesta passada.
- **Imagem do portal:** apenas sintaxe validada (`docker build --check` → `Check complete, no
  warnings found`). O build completo exige `npm ci` + `next build` e é do artefato do portal,
  fora do alvo desta issue.
- **Proveniência criptográfica:** labels OCI com versão/commit existem, mas SBOM/attestation e
  assinatura de imagem não; `WYD_VERSION`/`WYD_COMMIT` ainda são build-args locais (sem tag de
  release automatizada — FASE-15).
- **`--help` do launcher sai 2** (contrato de uso do binário). O aceite foi cumprido com um
  caminho de sucesso real (`check` de manifesto), não alterando comportamento de crate — a
  issue proíbe tocar código das crates.

## Skills Relacionadas do freebuff

- .agents/skills/wyd-rust-toolchain - release e proveniência.
- .agents/skills/github-actions-docs - job de release.
- .agents/skills/wyd-launcher - distribuição/patch do cliente.

## Documentação de referencia

- pre-release-rust.md (FASE 5 §5.2).
- docs/arquitetura/adr/0017-release-e-imagem-minima.md
- docs/rust/05-stack-docker.md · docs/rust/02-execucao.md §6

## Subtasks

- [x] Codificar issue
- [x] Testar issue
- [x] Revisar issue
- [x] Documentar issue
- [ ] Versionar issue, mergear branch e fechar issue (árvore só com alterações locais, por instrução)
- [ ] Limpar branch (se aplicável)

## Critérios de aceite

- [x] LTO ativo no release; imagem mínima builda e roda `wyd-launcher` (smoke `check` exit 0).
- [x] Proveniência documentada (versão/commit/toolchain nas labels e neste arquivo).

## Riscos / rollback

- Risco: LTO estourar tempo/memória do CI → mitigação: `lto = "thin"` (ADR-0017) e build
  incremental com cache BuildKit.
- Rollback: reverter `[profile.release]` em `Cargo.toml`, `docker/Dockerfile.launcher` e
  `.dockerignore`; nenhum código de crate mudou.
