# ⚔️ OpenWYD — NextGen (Rust)

Servidor do *With Your Destiny* (WYD2) reescrito em **Rust** — Hot Path monolítico,
mesmo espaço de memória, sem GC — com portal de administração em **Next.js** no Cold Path.

> 📦 **Transição Go → Rust:** o código Go legado (tmserver, dbserver, binserver,
> webserver, `internal/`, `api/`, `roadmap/`, `docs/` históricos) está arquivado em
> [`.deprecated-go/`](.deprecated-go/) como **referência de engenharia reversa** —
> não é mais compilado nem evoluído. O desenvolvimento acontece no workspace Rust.

## 🏗️ Estrutura do repositório

```
.
├── crates/                # 🦀 workspace Rust (o servidor)
│   ├── wyd-protocol/      # contratos de wire/cliente (CPSock, offsets explícitos)
│   ├── wyd-core/          # engine Hot Path (tick, ECS, spatial grid)
│   ├── wyd-net/           # transporte async (tokio/io_uring, zero-copy)
│   ├── wyd-db/            # persistência async (Postgres sqlx)
│   ├── wyd-admin/         # control plane Cold Path (gRPC/REST p/ o portal)
│   ├── wyd-sim/           # laboratório matemático (balanceamento)
│   └── wyd-launcher/      # launcher/patcher (hash, patch, bootstrap)
├── portal/                # 📊 UI Next.js (BFF; nunca escreve estado vivo)
├── docs/                  # 📚 documentação (arquitetura, ADRs, boas práticas, testes, CI/CD)
│   ├── manual-usuario/    # documentação de usuário final (portal, launcher,manuais do jogo)
│   ├── manual-técnico/    # documentação técnica (arquitetura, implementação, padrões, boas práticas, testes, CI/CD)
│   └── arquitetura/       # documentação da arquitetura (componentes, módulos, diagramas, fluxos, protocolos, padrões, ADRs)
├── roadmap-rust/          # 🗺️ as 15 fases + issues (planejamento vigente)
├── docker/                # 🐳 Arquivos de docker
├── .agents/skills/        # 🤖 skills Go (legado) + wyd-rust-* (vigentes)
├── .deprecated-go/        # 📦 arquivo do Go (referência, fora do build)
├── Cargo.toml             # workspace virtual (resolver 2)
└── rust-toolchain.toml    # stable pinado + rustfmt/clippy
```

Planejamento e análise completa: [`pre-release-rust.md`](pre-release-rust.md),
[`start-rust.md`](start-rust.md), [`roadmap-rust/index.md`](roadmap-rust/index.md).

## 🚀 Começando

Pré-requisito: `rustup` com toolchain **stable** (ver `rust-toolchain.toml`).

```bash
cargo fmt --check                      # formatação
cargo clippy --workspace -- -D warnings # lints como erro
cargo build --workspace                # compila as 7 crates
cargo test --workspace                 # testes unitários (cada crate nasce com teste)
```

O CI ([`.github/workflows/rust.yml`](.github/workflows/rust.yml)) roda exatamente
esses quatro gates em todo push/PR que toque no workspace.

## 🧭 Invariantes do núcleo Rust

1. **Hot Path monolítico** — sem gRPC/HTTP no tick, sem GC, sem mutex global.
2. **DOD/ECS** — dados contíguos, varreduras em lote; nada de OOP espalhada no heap.
3. **`unsafe` só em interop comprovada**; erros tipados (`Result`/`Option`), sem `panic!` em input de rede.
4. **Servidor é autoridade** — física, dano, skills, inventário; `ESCENE_FIELD`
   (30000) segue como pacote autoritativo de cena.
5. **Determinismo** — tick em frequência fixa; ordem de RNG/eventos é contrato.

Fonte autoritativa para agentes: [`AGENTS.md`](AGENTS.md) §0. Skills vigentes:
`wyd-rust-core`, `wyd-rust-net`, `wyd-rust-db`, `wyd-rust-sim`, `wyd-rust-obs`,
`wyd-rust-toolchain` (+ boas práticas: `rust-code-quality`, `software-architecture`,
`solid-design-patterns`).

## 📚 Documentação

| Preciso de… | Leia |
|---|---|
| 🧭 Visão, estrutura e estado da transição | [`docs/rust/00-visao-estrutura.md`](docs/rust/00-visao-estrutura.md) |
| ⚙️ Ciclo de execução, gates, DoD e doc viva | [`docs/rust/02-execucao.md`](docs/rust/02-execucao.md) |
| 📊 Painel de progresso da execução | [`docs/rust/03-progresso.md`](docs/rust/03-progresso.md) |
| 📦 Dependências Rust (catálogo e decisão offline) | [`docs/rust/01-dependencias.md`](docs/rust/01-dependencias.md) |
| 🏛️ Decisões de arquitetura (ADRs) | [`docs/arquitetura/adr/`](docs/arquitetura/adr/) |
| 🗺️ Planejamento e issues das 15 fases | [`roadmap-rust/index.md`](roadmap-rust/index.md) |
| 🕵️ Auditoria do planejamento legado (Go) | [`docs/manual-técnico/auditoria-roadmap.md`](docs/manual-técnico/auditoria-roadmap.md) |
| 🤖 Mapa de skills Go → Rust | [`docs/manual-técnico/skills.md`](docs/manual-técnico/skills.md) |
| 🔌 Protocolo CPSock e formatos (referência) | [`.deprecated-go/docs/migration/`](.deprecated-go/docs/migration/) |
| 🖥️ Portal (BFF, telas, sessão) | [`portal/README.md`](portal/README.md) |
| 🤖 Convenções para agentes de IA | [`AGENTS.md`](AGENTS.md) |

## 🔐 Segurança

- Senhas e PINs sempre com função de hash resistente (argon2id no legado;
  equivalente ao adotar autenticação no `wyd-db`); texto plano é proibido, inclusive em log.
- Segredos ficam fora do git (`.env` local ou gerenciador de secrets); gitleaks no CI do legado.
- Input de rede nunca causa `panic!`; handshake com rate limiting (ver FASE-06).

## 🤝 Contribuindo

1. Leia [`AGENTS.md`](AGENTS.md) §0 e a issue da fase em [`roadmap-rust/`](roadmap-rust/).
2. Código Rust novo nasce com teste; gates (`fmt`, `clippy -D warnings`, `build`, `test`) verdes.
3. Commits pequenos e descritivos, em português; docs e UI também em português.
4. Nunca versione segredos, `client/`, `serverlist.bin` com IP privado ou dumps de conta.
5. Nada sai de `.deprecated-go/` de volta ao build sem ADR.

---

## 📜 Licença

Distribuído sob a **GNU General Public License v2.0** — veja [`LICENSE`](LICENSE).

Os binários e assets do cliente WYD são propriedade dos seus respectivos donos e **não** são
distribuídos por este repositório (`client/` é ignorado pelo git).
