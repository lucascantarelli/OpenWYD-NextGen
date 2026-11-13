# Visão de pastas do repositório

> **Gate de arquitetura** (PROC-12, 2026-11-07). Regra vigente: **estrutura antes de
> código** (`estrutura-e-padroes.md` §8, ADR-0002). Este documento é o mapa oficial de
> onde cada coisa mora — e de onde nada novo pode nascer sem ADR.
> Nenhuma implementação foi criada para escrevê-lo: é inventário + norma.

## 1. Inventário real — o que existe hoje

### Raiz (somente manifestos e contratos de repo)

| Arquivo | Papel |
|---|---|
| `AGENTS.md` | instruções de operação do repo (fonte para agentes) |
| `README.md`, `LICENSE` | entrada e licença |
| `Cargo.toml`, `Cargo.lock`, `rust-toolchain.toml` | workspace Rust + toolchain pinada |
| `Makefile` | atalhos de build/teste |
| `pre-release-rust.md`, `start-rust.md` | planejamento da transição Go→Rust |
| `.env` | segredos locais — **nunca versionar conteúdo real** |
| `.gitignore`, `.gitattributes` | higiene do git |

**Não existe diretório `server/`**: o servidor é o workspace `crates/`. Não há arquivo
`.rs`/`.mjs`/`.ts`/`.py` solto na raiz — e nunca deve haver (§3).

### Servidor (`crates/` — workspace)

| Crate | Responsabilidade única |
|---|---|
| `wyd-protocol` | contratos de wire/cliente (CPSock, offsets explícitos) |
| `wyd-core` | engine Hot Path (tick, ECS, spatial grid) |
| `wyd-net` | transporte async (tokio/io_uring, zero-copy) |
| `wyd-db` | persistência async (Postgres) |
| `wyd-admin` | control plane Cold Path (HTTP/JSON p/ o portal, ADR-0008) |
| `wyd-sim` | laboratório matemático (balanceamento) |
| `wyd-launcher` | launcher/patcher — **consome o `client/`** (ver §4) |

### Ferramentas (`apps/tools/` + `tools/` — análise e geração, nunca runtime)

| Pasta | Conteúdo real | Papel |
|---|---|---|
| `apps/tools/` | CLI `wyd-tools` + `lib/` + `test/` (ADR-0028) | porta de entrada de RE: `inventario`, `binarios`, `strings`, `pacotes`, `docs` (Markdown medido) e `zip-check` do `client.zip`; parsers de PE/strings reusados de `tools/client/` |
| `tools/client/` | 15 scripts `.mjs`/`.ps1` + `README.md` | **biblioteca** de parse/captura, sem CLI própria de documentação: `pe`, `strings`, `formats`, `keytable`, `wyt`, `serverlist`, `captura`, `captura-live`, `analyse`, `evidence` |
| `tools/protocol/` | `gen-protocol.mjs` | gerador das tabelas de protocolo a partir do C++ arquivado |
| `tools/content/` | 11 scripts + `README.md` | geração/validação de conteúdo: seeds, catálogo do banco, enciclopédia, status de issue, migrations |
| `tools/dev/` | `bench-gate.mjs`, `monitoring-smoke.mjs`, `clean-tmp.mjs` | gates de operação: regressão de bench, smoke do monitoramento, limpeza de `tmp/` |

### Documentação (`docs/`)

| Pasta | Conteúdo real |
|---|---|
| `docs/arquitetura/` | `protocolo.md`, `modelo-banco.md`, `c4-pacotes.md`, `estrutura-e-padroes.md`, `visao-pastas.md` (este), `contrato-transporte-cpsock.md`, `contrato-dispatch-opcode.md`, `contrato-persistencia.md`, `adr/` |
| `docs/cliente/` | 10 docs (`README`, `mapa`, `customizacao`, `inventario`, `superficie-conhecida`, `analise-estatica`, `formatos`, `engenharia-reversa`, `depuracao`, `pacotes`) + `gerado/` (artefatos determinísticos) |
| `docs/rust/` | `00-visao-estrutura`, `01-dependencias`, `02-execucao`, `03-progresso` |
| `docs/manual-técnico/` | `auditoria-roadmap.md`, `skills.md` |
| `docs/manual-usuario/` | **vazia — lacuna registrada**, não criar placeholder sem conteúdo |

### Dados, cliente e resto

| Pasta | Conteúdo real |
|---|---|
| `crates/wyd-protocol/tests/fixtures/cliente/` | 5 goldens sintéticos validados (`captura.mjs`) |
| `client/` | **~344 MB, fora do git**: `WYD.exe` 7662 + DLLs + XTrap + 5.361 `.msa` + 350 `.wyt` + catálogos — dossiê em `docs/cliente/` |
| `wyd-portal/` | UI Next.js (BFF; nunca escreve estado vivo) |
| `roadmap-rust/` | `index.md` + `issues/` (fases 00–15, incl. PROC-11/PROC-12 e CLIENT-04…CLIENT-07) |
| `docker/`, `.github/`, `.vscode/` | container (inclui `docker/monitoring/`: Prometheus, Grafana, Loki, Alloy), CI, editor |
| `.agents/skills/` | skills vigentes (`wyd-rust-*`, `wyd-client-reverso`, boas práticas) |
| ~~arquivo Go/C++~~ | **removido em 2026-11** — sem leitura, sem cópia; fonte canônica é `docs/` + `crates/` (mudança exige ADR) |
| `target/` | cache de build (apagado em 2026-11-12 e recriado por `cargo build`), ignorado — `make clean-target` |
| `tmp/` | artefatos locais descartáveis (147 MB): **fonte de captura** em `tmp/captura*.json` (frames reais usados por `wyd-tools pacotes`) + rascunhos; `make clean-tmp` remove só o que o projeto gerou e **preserva** as capturas |
| `client.zip` | **186 MB, fora do git**: pacote do cliente para o launcher; publicado na release `client-7662-r1` (GitHub Releases, ADR-0028) com SHA-256 fixado, verificado por `make tools-zip-check` |

## 1.1 Política de limpeza e artefatos

| Artefato | Regra |
|---|---|
| `target/` | cache do Cargo, ignorado no git (`/target/`); **nunca** é fonte. Pode ser apagado a qualquer momento (`make clean-target`) e o próximo `cargo build`/`make check` recria |
| `tmp/` | descartável e ignorado; `make clean-tmp` remove logs de teste, `tmp/drivers`, `tmp/e2e-janela`, `tmp/instala-teste`, cópias de cliente e **caches `*/target` de probes** (`tmp/hashgen`, `tmp/loginprobe`), **preservando** fontes dos probes, `tmp/captura*`, `tmp/basemob` e PNGs nomeados pelo operador |
| `dist/` | saída de distribuição (launcher), ignorada; regenerada por `tools/launcher/dist-manifest.mjs` |
| `client.zip` | artefato de release (186 MB), ignorado; gerado por `make client-zip`, conferido por `make tools-zip-check` e publicado em `client-7662-r1` com hash fixado no launcher |
| `client/` | dado externo (~344 MB), ignorado; nunca versionar nem redistribuir fora do release do cliente |
| `docs/cliente/inventario-gerado.md` | derivado do cliente real; regenerar com `make tools-inventario` em vez de editar |
| `docs/cliente/re-binarios.md`, `re-strings.md`, `re-pacotes.md` | derivados dos binários/capturas; regenerar com `make tools-re` (nunca editar à mão) |
| `tmp/captura*` | **entrada** de evidência (`wyd-tools pacotes` lê frames e tipos textuais); não é código de ferramenta e não é versionado |

## 2. Tabela-alvo — pasta → responsabilidade → padrão → SOLID → nomenclatura

| Pasta | Responsabilidade única | Padrão adotado | SOLID aplicado | Nomenclatura | Pode morar ali | Nunca pode morar ali |
|---|---|---|---|---|---|---|
| `crates/wyd-protocol/` | contrato de wire | Facade + Newtype + `enum`+`match` exaustivo | SRP (só fio), OCP (novo opcode = variante, não `if`) | `wire/`, `codec/`, `payload/`, `error.rs`, `tests/` (ADR-0002) | tipos de pacote, tabelas geradas | lógica de jogo, I/O, SQL |
| `crates/wyd-core/` | simulação Hot Path | ECS/DOD, State machine de tick | SRP (tick puro), DIP (dados via componentes) | sistemas por domínio, sem `utils.rs` | tick, componentes, spatial grid | rede, banco, HTTP |
| `crates/wyd-net/` | transporte | Adapter (socket→frame), validação em camadas | SRP (bytes→frames), LSP (erros tipados) | `session/`, `transport/`, `dispatch/`, `limits/`, `telemetry/`, `error.rs` (estrutura §11) | framing, limites, rate limit | regra de jogo |
| `crates/wyd-db/` | persistência | Repository + porta | DIP (core depende de trait, não de SQL) | `model/`, `repo/`, `pool/`, `transfer/`, `migrations/` (estrutura §12) | SQL, pool, seeds | estado vivo |
| `crates/wyd-admin/` | control plane | Facade HTTP/JSON p/ o portal (ADR-0008) | ISP (contratos mínimos por tela) | `api/`, `auth/`, `audit/`, `service/` (estrutura §13) | leitura e intenções auditadas | mutação de estado vivo |
| `crates/wyd-sim/` | matemática offline | Strategy de fórmulas, laboratório em lote | OCP (nova fórmula = módulo) | `formula/`, `scenario/`, `report/` (estrutura §14) | equações, benches | rede, cliente |
| `crates/wyd-launcher/` | verificar/patch/iniciar o jogo | Facade (`lib.rs`) + erros tipados + CLI fina (check/launch/update) | SRP (só distribuição) | `manifest.rs`, `patch.rs`, `bootstrap.rs`, `sha256.rs`, `distribuicao.rs`, `lancamento.rs` (estrutura §9) | hash, patch atômico, bootstrap | lógica de jogo, segredo |
| `apps/launcher-ui/` | janela do launcher (Tauri v2) | MVP de apresentação sobre o crate | DIP (consome `wyd-launcher`, não reescreve disco) | `ui/`, `capabilities/`, `src/main.rs` (estrutura §9.1) | janela, IPC `pagina`/`jogar`, cópia do `.exe` | regra de patch/hash, segredo, lógica de jogo |
| `tools/client/` | análise do cliente | pipeline de funções puras + CLI fina | SRP (um módulo = uma etapa) | verbo + domínio: `pe.mjs`, `evidence.mjs`, `captura.mjs` | leitura de binário, relatório | código de jogo/servidor |
| `tools/protocol/` | geração de tabelas | gerador versionado com fonte canônica | — (determinístico, `cargo fmt` após gerar) | `gen-*.mjs` + `README` de reprodução | geradores | estado, credencial |
| `docs/*` | conhecimento | um assunto por arquivo, nome do domínio | — | `substantivo-do-dominio.md`, minúsculas | narrativa, tabelas, evidência | binário, segredo, código |
| `crates/wyd-protocol/tests/fixtures/cliente/` | goldens de pacote | JSON com esquema fixo + validador | — | `<fase>-<opcode>-<n>.json` | bytes sintéticos/higienizados | dado real de conta, `.pcap` |
| `wyd-portal/` | BFF e telas | BFF puro sobre `wyd-admin` | DIP (fala com contrato, não com banco) | rotas por tela | UI, chamadas admin auditadas | chamada direta ao backend fora do `adminCall`, segredo no browser |
| `client/` | artefato fechado | **nenhum padrão de código**: é dado | — | (não versionado) | binários/assets locais | qualquer arquivo versionado novo |
| `roadmap-rust/issues/` | uma entrega = uma issue | template Fase/Depende/Bloqueia/Esforço + DoD | — | `FASE-NN-slug.md` | plano e aceite | implementação |

## 3. Regras anti-sprawl da raiz

1. Na raiz só vivem manifestos, contratos de repo e portas de entrada. Implementação nova
   nasce em `crates/` (runtime) ou `tools/` (análise/geração) — nunca na raiz.
2. Pasta nova exige `README.md` de propósito + entrada nesta tabela (via issue, nunca direto).
3. `tools/dev/` e `docs/manual-usuario/` continuam como estão até terem propósito real e
   documentado — placeholder sem conteúdo é dívida, não organização.
4. `client/` é artefato: nenhum script o escreve sem backup, nenhum doc o redistribui, nada
   dele entra no git.
5. Mudança de árvore exige ADR; este documento é atualizado na mesma entrega que a move.

## 4. O cliente no mapa

`client/` é o **usuário do `wyd-launcher`**: o launcher valida (`inventario.md`), atualiza e
inicia o `WYD.exe`, e nunca o contrário. Mapa completo — inventário, lacunas, roadmap de
modernização e plano de desassemble — em [`docs/cliente/mapa.md`](../cliente/mapa.md).
