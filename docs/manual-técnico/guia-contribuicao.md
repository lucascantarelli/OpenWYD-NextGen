# Guia de contribuição (Rust)

> **PROC-19** (2026-11-07). Público: quem chega agora e precisa compilar, testar e
> abrir a primeira contribuição sem ler o repo inteiro. Norma de ciclo e DoD:
> `docs/rust/02-execucao.md`; invariantes: `AGENTS.md` §3. Este guia não duplica
> nenhum dos dois — aponta e resume o caminho.

## 1. Pré-requisitos

| Ferramenta | Versão / observação |
|---|---|
| `rustup` + toolchain stable | pinada em `rust-toolchain.toml` (inclui `rustfmt` e `clippy`) |
| Node.js (só portal/ferramentas) | `npm ci` em `wyd-portal/`; scripts em `tools/` rodam com `node` |
| Docker (opcional) | só para a stack `db` + `portal` (`docs/rust/05-stack-docker.md`) |
| `make` (opcional) | Git Bash no Windows não traz; os comandos `cargo` equivalentes valem |

`client/` (~344 MB de binários proprietários) **não é versionado** e não é necessário
para compilar nem testar o servidor.

## 2. Primeiro build e primeiro teste (5 comandos)

```bash
cargo build --workspace                 # 7 crates do workspace
cargo test --workspace                  # todo crate nasce com teste
cargo fmt --all -- --check              # formatação
cargo clippy --workspace --all-targets -- -D warnings   # lints como erro
```

Gate único da entrega (equivale aos quatro acima + build):

```bash
make check
```

Portal (quando a mudança toca `wyd-portal/`):

```bash
make portal-install     # npm ci
make portal-test        # tsc --noEmit + vitest
make portal-dev         # http://127.0.0.1:3000 (hot reload)
```

Ferramentas (só quando a issue pedir):

```bash
node tools/client/captura.mjs --check   # goldens de pacote (5 fixtures)
node tools/protocol/gen-protocol.mjs    # congelado: não regenera nada por design
docker compose -f docker-compose.yml -f docker-compose.prod.yml config --quiet
```

## 3. Fluxo de uma issue

1. Escolha a issue em `roadmap-rust/index.md` (a linha diz fase, título e status) e leia
   `roadmap-rust/issues/FASE-NN-M-*.md` **inteira**, incluindo "Documentação de referência".
2. Ler as docs normativas citadas: `docs/arquitetura/protocolo.md` (fio),
   `docs/arquitetura/estrutura-e-padroes.md` (árvore de módulos), `docs/rust/` (execução,
   dependências, stack).
3. Branch `fase/NN-slug` a partir de `develop`.
4. Mudança mínima nos arquivos que a issue lista. Arquivo novo fora da árvore de
   `estrutura-e-padroes.md` exige **ADR** em `docs/arquitetura/adr/` (rito PROC-21).
5. Teste do próprio crate/issue + `make check`.
6. Atualizar na mesma entrega: `docs/rust/03-progresso.md`, o status em
   `roadmap-rust/index.md` e qualquer doc invalidado.
7. Commit pequeno em português, merge `--no-ff` em `develop`, issue fechada com a
   evidência (saída dos gates) anexada.

## 4. Regras que reprovam a contribuição

- `panic!`/`unwrap()` em caminho de entrada de rede (`AGENTS.md` §3.2).
- I/O, SQL ou `gRPC` dentro do tick do Hot Path; mutex global; alocação quente.
- Divergência de layout binário sem tabela em `crates/wyd-protocol/src/*/generated/`;
  `repr(Rust)` para wire.
- Segredo, DSN real, `client/`, `target/` ou dump de conta no commit.
- `clippy` com warning, teste removido/afrouxado para "passar", doc dizendo o contrário
  do código.
- Web escrevendo estado de personagem vivo fora de `delivery_queue`.

## 5. Onde pedir contexto

| Assunto | Documento |
|---|---|
| Invariantes e convenções | `AGENTS.md` §0–§7 |
| Ciclo, gates, DoD | `docs/rust/02-execucao.md` |
| Árvore de módulos e nomenclatura | `docs/arquitetura/estrutura-e-padroes.md` (ADR-0002) |
| Protocolo e payloads | `docs/arquitetura/protocolo.md` + `crates/wyd-protocol/` |
| Banco e migrations | `docs/arquitetura/modelo-banco.md` + `crates/wyd-db/` |
| Componentes e fluxos (C4) | `docs/arquitetura/c4-pacotes.md` |
| Cliente 7662 | `docs/cliente/` |
| Skills por domínio | `.agents/skills/` (`wyd-rust-*`) + `docs/manual-técnico/skills.md` |

## 6. Revisão fria (checklist do próprio autor)

- [ ] Um leitor novo, só com §1–§2 deste guia, compila e testa sem perguntar nada.
- [ ] Cada afirmação técnica do diff tem código ou doc que a sustenta (sem "hipótese"
      apresentada como fato).
- [ ] Link adicionado resolve (`test -e` no caminho).
- [ ] Nenhuma referência à árvore legada removida (o servidor é só o workspace `crates/`).
