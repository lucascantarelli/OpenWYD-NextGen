# ⚔️ WYD — nova geração (Rust)

Servidor do *With Your Destiny* (WYD2) reescrito em **Rust** — Hot Path monolítico,
mesmo espaço de memória, sem GC — com portal de administração em **Next.js** no Cold Path.

> 📦 **Transição Go → Rust concluída (2026-11):** o arquivo Go/C++ foi removido do
> repositório. A referência canônica atual é `docs/` + `crates/` + `roadmap-rust/`
> (ver `docs/arquitetura/protocolo.md`, `docs/cliente/` e `docs/rust/`).

## 🏗️ Estrutura do repositório

```
.
├── crates/                # 🦀 workspace Rust (o servidor)
│   ├── wyd-protocol/      # contratos de wire/cliente (CPSock, offsets explícitos)
│   ├── wyd-core/          # engine Hot Path (tick, ECS, spatial grid)
│   ├── wyd-net/           # transporte async (tokio/io_uring, zero-copy)
│   ├── wyd-db/            # persistência async (Postgres sqlx)│   ├── wyd-admin/        # control plane Cold Path (HTTP/JSON p/ o portal)
│   ├── wyd-server/       # monólito de jogo: um processo por canal (CPSock/status/sessão)
│   ├── wyd-sim/           # laboratório matemático (balanceamento)
│   └── wyd-launcher/      # launcher/patcher para o cliente (hash, patch, bootstrap,update,install, anticheat)
├── wyd-portal/                # 📊 UI Next.js (BFF; nunca escreve estado vivo)
├── docs/                  # 📚 fonte canônica da verdade
│   ├── arquitetura/       # 🏛️ protocolo, modelo de banco, C4, estrutura, ADRs
│   ├── rust/              # ⚙️ execução, dependências, progresso, stack docker
│   ├── cliente/           # 🕹️ dossiê do cliente 7662 (inventário, RE, pacotes)
│   ├── manual-usuario/    # 👤 manuais de jogador, launcher e portal
│   └── manual-técnico/    # 🛠️ guia de contribuição, auditoria, mapa de skills
├── roadmap-rust/          # 🗺️ as 15 fases + issues (planejamento vigente)
├── docker/                # 🐳 Dockerfiles (stack e smoke do launcher)
├── .agents/skills/        # 🤖 skills vigentes (wyd-rust-* + boas práticas)
├── Cargo.toml             # workspace virtual (resolver 2)
└── rust-toolchain.toml    # stable pinado + rustfmt/clippy
```

Planejamento e análise completa: [`pre-release-rust.md`](pre-release-rust.md),
[`start-rust.md`](start-rust.md), [`roadmap-rust/index.md`](roadmap-rust/index.md).

## 🚀 Começando

Pré-requisito: `rustup` com toolchain **stable** (ver `rust-toolchain.toml`).
Guia completo para quem chega agora: [`docs/manual-técnico/guia-contribuicao.md`](docs/manual-técnico/guia-contribuicao.md).

```bash
cargo fmt --check                      # formatação
cargo clippy --workspace -- -D warnings # lints como erro
cargo build --workspace                # compila as 8 crates
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
`wyd-rust-toolchain`, `wyd-rust-testing` (+ boas práticas: `rust-code-quality`,
`software-architecture`, `solid-design-patterns`; domínio do cliente:
`wyd-client-reverso`).

## 📚 Documentação

| Preciso de… | Leia |
|---|---|
| 🧭 Visão, estrutura e estado da transição | [`docs/rust/00-visao-estrutura.md`](docs/rust/00-visao-estrutura.md) |
| ⚙️ Ciclo de execução, gates, DoD e doc viva | [`docs/rust/02-execucao.md`](docs/rust/02-execucao.md) |
| 📊 Painel de progresso da execução | [`docs/rust/03-progresso.md`](docs/rust/03-progresso.md) |
| 📦 Dependências Rust (catálogo e decisão offline) | [`docs/rust/01-dependencias.md`](docs/rust/01-dependencias.md) |
| 🏛️ Decisões de arquitetura (ADRs) | [`docs/arquitetura/adr/README.md`](docs/arquitetura/adr/README.md) (índice) |
| 🧱 Estrutura de módulos, nomenclatura e padrões (normativo) | [`docs/arquitetura/estrutura-e-padroes.md`](docs/arquitetura/estrutura-e-padroes.md) |
| 🧭 C4 (contexto, containers, componentes) + fluxos de pacotes | [`docs/arquitetura/c4-pacotes.md`](docs/arquitetura/c4-pacotes.md) |
| 🛠️ Guia de contribuição (build, gates, fluxo de issue) | [`docs/manual-técnico/guia-contribuicao.md`](docs/manual-técnico/guia-contribuicao.md) |
| 👤 Manual do usuário (jogar, launcher, telas do portal) | [`docs/manual-usuario/README.md`](docs/manual-usuario/README.md) |
| 🗂️ Visão de pastas: inventário, responsabilidade e padrões | [`docs/arquitetura/visao-pastas.md`](docs/arquitetura/visao-pastas.md) |
| 🗺️ Planejamento e issues das 15 fases | [`roadmap-rust/index.md`](roadmap-rust/index.md) |
| 🕵️ Auditoria do planejamento legado (Go) | [`docs/manual-técnico/auditoria-roadmap.md`](docs/manual-técnico/auditoria-roadmap.md) |
| 🤖 Mapa de skills Go → Rust | [`docs/manual-técnico/skills.md`](docs/manual-técnico/skills.md) |
| 🔌 Protocolo CPSock e formatos (normativo) | [`docs/arquitetura/protocolo.md`](docs/arquitetura/protocolo.md) |
| 🕹️ Cliente do jogo (inventário, hooks, plano de RE) | [`docs/cliente/`](docs/cliente/) |
| 🗺️ Mapa do cliente (usuário do launcher, lacunas, roadmap) | [`docs/cliente/mapa.md`](docs/cliente/mapa.md) |
| 🧩 Customização e manifesto do launcher (ADR-0003) | [`docs/cliente/customizacao.md`](docs/cliente/customizacao.md) |
| 🖥️ Portal (BFF, telas, sessão) | [`wyd-portal/README.md`](wyd-portal/README.md) |
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
5. Arquivo Go/C++ não existe mais; mudança de árvore/contrato exige ADR.

---

## 📜 Licença

Distribuído sob a **GNU General Public License v2.0** — veja [`LICENSE`](LICENSE).

Os binários e assets do cliente WYD são propriedade dos seus respectivos donos e **não** são
distribuídos por este repositório (`client/` é ignorado pelo git).
