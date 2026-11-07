# Progresso da execução — roadmap-rust

> Documento vivo. Atualizado a **cada issue** executada (regra em
> `docs/rust/02-execucao.md` §5). Última atualização: 2026-11-07.

## Legenda

✅ concluída · 🚧 em curso · ⬜ pendente · ⛔ bloqueada

## Fase 0 — preparação do ambiente

| Issue | Status | Evidência / observação |
|---|---|---|
| FASE-00-1 limpeza do ambiente | 🚧 | `go clean -cache -modcache -testcache` executado (exit 0). `go` ainda no PATH (`D:\Programas\Go\bin\go.exe`); desinstalação exige ação elevada do dono; `GOPATH=C:\Users\Canta\go` (200 MB) e `GOROOT=D:\Programas\Go` a remover depois da desinstalação. |
| FASE-00-2 VS Code Rust | ✅ | `code --list-extensions` contém `rust-lang.rust-analyzer`, `tamasfe.even-better-toml`, `fill-labs.dependi`, `vadimcn.vscode-lldb`, `ms-playwright.playwright`, `github.vscode-github-actions`; **nenhuma extensão Go**. `.vscode/settings.json` usa `clippy` com `-D warnings` e format on save. |
| FASE-00-3 branch `develop` | ✅ | `develop` existe e é a branch atual (`git branch --show-current`). Fluxo issue→branch→PR documentado em `docs/rust/02-execucao.md` §2. |

## Fase 1 — auditoria

| Issue | Status | Evidência / observação |
|---|---|---|
| FASE-01-1 auditoria de estado | 🚧 | Inventário publicado em `docs/manual-técnico/auditoria-roadmap.md`: 138 arquivos no planejamento Go (62 issues abertas, 54 em `done/`, 13 em `test/`). Verificação de evidência item a item continua (subtarefa aberta). |
| FASE-01-2 varredura de dívidas | ✅ | Go: 0 marcadores. C++ (`Source/`): 16 ocorrências, todas classificadas com issue dona (`portar`/`descartar`/`investigar`) em `docs/manual-técnico/auditoria-roadmap.md` §3.1. |
| FASE-01-3 inventário de skills | ✅ | `docs/manual-técnico/skills.md`: 24 skills classificadas (6 Rust vigentes + 3 de boas práticas + 8 de domínio com destino + 7 genéricas mantidas).

## Fase 2 — processo

| Issue | Status | Evidência / observação |
|---|---|---|
| FASE-02-1 ciclo de execução | ✅ | `docs/rust/02-execucao.md` criado (ciclo, gate, branch, DoD, revisão, doc viva). |
| FASE-02-2 revisão consolidada | ✅ | Checklist e gatilho de 10 issues em `docs/rust/02-execucao.md` §4. |
| FASE-02-3 ordem de priorização | ✅ | Grafo e caminho crítico em `roadmap-rust/index.md` (dependências declaradas por issue). |
| FASE-02-5 visão de pastas + mapa do cliente | ✅ | `docs/arquitetura/visao-pastas.md` (inventário real + tabela-alvo + anti-sprawl) e `docs/cliente/mapa.md` (documentado×lacuna, client↔launcher, plano de desassemble); lacunas registradas (`manual-usuario/` vazia, `tools/dev/` placeholder). |
| FASE-02-4 estrutura e padrões | ✅ | `docs/arquitetura/estrutura-e-padroes.md` + ADR-0002 (árvore, nomenclatura, padrões, SOLID, testes); regra aplicada em `AGENTS.md` §0 e nas skills `software-architecture`, `wyd-rust-net`, `rust-code-quality`. Revisão fria pendente. |

## Fase 3 — engine (`wyd-core`)

| Issue | Status | Evidência / observação |
|---|---|---|
| FASE-03-1 tick | ✅ | `tick/scheduler.rs` (acumulador + clamp) + `ports/clock.rs`; fatiamento 7×286 = 50×40. |
| FASE-03-2 ECS | 🚧 | Dados prontos (`components.rs` + `ComponentStore` SoA); restam `systems/`, ADR e bench. |
| FASE-03-3 spatial grid | ✅ | `spatial/grid.rs`; guerra 5000: 8,6 ms inserts, 79 µs consulta (debug). |
| FASE-03-4 dados estáticos | ⬜ | Depende de FASE-04 (fonte de dados). |
| FASE-03-5 autoridade do servidor | ⬜ | Depende de 03-2, 03-3, 03-4. |

## Fase 4 — persistência (`wyd-db`)

| Issue | Status | Evidência / observação |
|---|---|---|
| FASE-04-1 modelagem | ✅ | `docs/arquitetura/modelo-banco.md` (ER, 8 tabelas, 6 decisões; arquivo legado vazio — registrado). |
| FASE-04-2 migrations | ✅ | 6 migrations reversíveis + `seeds/dev.sql`; prova ao vivo zero→última→revert→última em PG17 descartável. |
| FASE-04-3 persistência async | 🚧 | Base pronta (traits, `PoolConfig`, `DeliveryQueue`); resta impl `pg` + pool (rede + ADR). |

## Fases 5–15

| Issue | Status | Observação |
|---|---|---|
| FASE-05-1 pipeline CI | ⬜ | Base: `.github/workflows/rust.yml` + `Makefile` (`check`, `ci`, `audit`, `bench`). |
| FASE-05-2 release/docker | ⬜ | Base: `docker/Dockerfile.rust`. |
| FASE-06-1 telemetria | ⬜ | Depende de FASE-03. |
| FASE-06-2 hardening | ✅ | `limits/rate.rs` (balde + porte de handshake) + `tests/flood.rs`; flood 100/10000, fuzz 5000 sem panic; legit preservado. |
| FASE-07-1 gateway | ✅ | Backend do contrato (`AdminError`→HTTP, staff/auditoria/grants idempotentes); 11 lib + 4 integração; fiação do portal aguarda transporte. |
| FASE-07-2 painel usuário | ⬜ | Depende de 07-1. |
| FASE-07-3 painel admin | ⬜ | Depende de 07-1. |
| FASE-07-4 E2E Playwright | ⬜ | Depende de 07-2, 07-3. |
| FASE-08-1 tabela do protocolo | ✅ | Reorganizado na árvore §3 (`wire/`, `codec/`, `payload/`, `error.rs`, `tests/`); `protocolo.md` publicado (HEADER, ofuscação, 198 opcodes gerados, 16 payloads, UNVERIFIED); gate verde (32 lib + 4 integração). |
| FASE-08-2 parsers | ✅ | `repr(C)` + offsets provados; `inspect` (tamanho → opcode) + `require_ported` (gate UNVERIFIED); 9 goldens contra `fixtures/cliente/*.json` (ida-volta byte-idêntica); zero `unsafe`; sem deps novas (zerocopy/nom diferidos p/ 13-2). |
| FASE-08-3 distribuição | ✅ | `distribuicao.rs` + `update`/`--dry-run`; 27/27 testes; gate verde; e2e em tmp (aplica+backup+check verde, adulterado recusado exit 1). |
| FASE-08-4 dossiê do cliente | ✅ | `docs/cliente/` publicado (README, inventário, superfície conhecida, plano de RE); baseline de `WYD.exe` 7662 com SHA-256 `04c9d2a1…` e PE32/i386; skill `wyd-client-reverso`. Desassemble é 08-5/08-6. |
| FASE-08-5 RE estático | ✅ | `tools/client/` (7 módulos) + `docs/cliente/gerado/` (PE, strings, evidência, formatos); **6/6 âncoras** e **3/3 patches** confirmados por bytes + objdump; os 3 sites de checksum já estão `EB` no disco; `.wyt`/`.msa`/`sn.bin` com estrutura inicial; `serverlist.bin` opaco. T2 (structs) movido para 08-6. |
| FASE-08-6 RE dinâmico | 🚧 | Roteiro em `docs/cliente/depuracao.md` + `captura.mjs` (`--gen`/`--check`, 5/5 goldens gerados C→S, determinístico). Captura ao vivo bloqueada: exige servidor executável + VM. |
| FASE-08-7 customização | ✅ | `manifest/patch/bootstrap/sha256.rs` + CLI `check`/`launch`, 19/19 testes, `check` real 6/6 OK; spawn real pendente de ambiente com jogo. |
| FASE-09-1 auditoria de skills | ⬜ | Insumo: FASE-01-3. |
| FASE-09-2 skills sucessoras | ⬜ | Depende de 09-1. |
| FASE-10-1 equações | ⬜ | Depende de 03-5. |
| FASE-10-2 velocidade | ⬜ | Depende de 10-1. |
| FASE-10-3 laboratório | ⬜ | Depende de 10-1, 10-2. |
| FASE-11-1 guia + manuais | ⬜ | Pastas `docs/manual-*/` criadas. |
| FASE-11-2 C4 + pacotes | ⬜ | Depende de 08-1. |
| FASE-12-1 dashboards | ⬜ | Depende de 06-1. |
| FASE-13-1 rito ADR | ✅ | `docs/arquitetura/adr/0000-template.md` criado (estado, contexto, decisão, consequências, alternativas, rastreabilidade); `0001` promovida a **aceita** no formato do template. |
| FASE-13-2 primeiras ADRs | ✅ | ADR-0004 (SoA manual), 0005 (LE+`repr(C)`), 0006 (std até rede/vendor); 7 arquivos 0000–0006 sem lacuna, seções conformes; consumidoras vinculadas. |
| FASE-14-1 pesquisa | ⬜ | Catálogo: `docs/rust/01-dependencias.md`. |
| FASE-15-1 suite + concorrência | ⬜ | Depende de 05-1, 10-3. |
| FASE-15-2 carga + banco + release | ⬜ | Depende de 15-1, 12-1. |

## Registro de execução

| Data | Issue | O que mudou | Evidência |
|---|---|---|---|
| 2026-11-07 | FASE-00-1 | Caches Go limpos (`go clean` exit 0) | `go clean -cache -modcache -testcache` |
| 2026-11-07 | FASE-00-2 | VS Code verificado: rust-analyzer + clippy; zero extensão Go | `code --list-extensions` |
| 2026-11-07 | FASE-00-3 | `develop` confirmada como branch atual | `git branch --show-current` |
| 2026-11-07 | FASE-01-1 | Planejamento Go restaurado do histórico (`roadmap/` estava vazio no arquivo) e inventariado | `git archive HEAD roadmap \| tar -x -C .deprecated-go` → 138 arquivos |
| 2026-11-07 | FASE-02-1/2/3 | Ciclo, checklist de revisão e grafo de dependências documentados | `docs/rust/02-execucao.md`, `roadmap-rust/index.md` |
| 2026-11-07 | FASE-01-3 | Mapa de sucessão das 24 skills publicado | `docs/manual-técnico/skills.md` |
| 2026-11-07 | FASE-13-1 | Template de ADR + rito; `0001` promovida a aceita | `docs/arquitetura/adr/0000-template.md`, `0001-hot-path-monolitico.md` |
| 2026-11-07 | FASE-01-2 | 16 débitos C++ classificados e destinados (Go: 0 marcadores) | `docs/manual-técnico/auditoria-roadmap.md` §3.1 |
| 2026-11-07 | FASE-02-4 | Correção de processo: estrutura/nomenclatura/padrões publicados antes de mais código; protótipo de `wyd-protocol` congelado | `docs/arquitetura/estrutura-e-padroes.md`, `adr/0002-estrutura-modulos.md` |
| 2026-11-07 | FASE-08-1 | Reaberta como 🚧: protótipo existe mas viola a árvore planejada (arquivos de domínio na raiz, `payload.rs` misto, sem `tests/`) | `crates/wyd-protocol/src/` |
| 2026-11-07 | FASE-08-4 | Dossiê do cliente criado antes de qualquer RE: inventário com hashes/PE, superfície por endereço, plano T1–T8/M0–M6 e 3 issues de execução | `docs/cliente/`, `sha256sum client/WYD.exe` |
| 2026-11-07 | FASE-08-5 | Pipeline estático em `tools/client/` (Node puro + objdump) e relatório: 6/6 âncoras confirmadas como instruções internas (só `SendPacket` é função), 3/3 sites de checksum já `EB`, formatos `.wyt`/`.msa`/`sn.bin` mapeados; artefatos determinísticos | `node tools/client/analyse.mjs`, `docs/cliente/analise-estatica.md` |
| 2026-11-07 | FASE-08-6 (parcial) | Roteiro de trace dinâmico + 5 goldens sintéticos C→S validados; captura ao vivo bloqueada por falta de servidor executável e VM | `node tools/client/captura.mjs --check`, `docs/cliente/depuracao.md` |
| 2026-11-07 | FASE-02-5 | Gate de arquitetura: inventário real + tabela-alvo de pastas + mapa client↔launcher; zero implementação nova | `docs/arquitetura/visao-pastas.md`, `docs/cliente/mapa.md` |
| 2026-11-07 | FASE-08-7 (parcial) | ADR-0003 + especificação de customização e manifesto; código do launcher pendente de autorização | `docs/arquitetura/adr/0003-estrategia-patch-cliente.md`, `docs/cliente/customizacao.md` |
| 2026-11-07 | FASE-08-7 | Launcher implementado: 19/19 testes, `check` real 6/6 OK, dry-run sem executar o jogo | `crates/wyd-launcher/src/`, `cargo test --workspace` |
| 2026-11-07 | FASE-08-3 | Distribuição implementada e com gate verde: HMAC-SHA256, resume, `update`; vetor de teste corrigido contra OpenSSL (código estava certo) | `crates/wyd-launcher/src/distribuicao.rs`, `cargo test --workspace` |
| 2026-11-07 | FASE-08-1 | Protocolo reorganizado na árvore §3 sem mudar comportamento; `protocolo.md` com 198 opcodes gerados; checksum provado função-da-chave | `crates/wyd-protocol/`, `docs/arquitetura/protocolo.md` |
| 2026-11-07 | FASE-08-2 | Parsers: `repr(C)` + `inspect` + `require_ported`; 9 goldens byte-idênticos; zero `unsafe`; sem deps novas | `crates/wyd-protocol/src/{wire,lib.rs,payload/}`, `crates/wyd-protocol/tests/golden.rs` |
| 2026-11-07 | FASE-03 | Engine: `CoreError`, SoA, scheduler 20–50 Hz, grade O(1), relógio injetável; 12 lib + 3 determinismo; guerra 8,6 ms/79 µs | `crates/wyd-core/src/{error,ecs,tick,spatial,ports}/`, `crates/wyd-core/tests/determinism.rs` |
| 2026-11-07 | FASE-04 | Banco: 6 migrations reversíveis + modelo + traits + fila; prova ao vivo PG17 (FK/CHECK falham, revert limpa, seed OK); 8 lib + 2 constraints | `crates/wyd-db/{migrations,seeds,src/{error,model,repo,pool,transfer}}`, `docs/arquitetura/modelo-banco.md` |
| 2026-11-07 | FASE-07 (núcleo) | Control plane: `AdminError`→HTTP, staff/auditoria, grants idempotentes; 11 lib + 4 integração (401/403/replay) | `crates/wyd-admin/src/{error,auth,audit,api,service}/`, `crates/wyd-admin/tests/unauthorized.rs` |
| 2026-11-07 | FASE-06-2 | Hardening: balde de tokens + porte de handshake + pipeline estrito; flood contido, fuzz sem panic; 4 lib + 6 flood | `crates/wyd-net/src/limits/`, `crates/wyd-net/tests/flood.rs` |
| 2026-11-07 | FASE-13-2 | ADRs 0004/0005/0006 aceitas (ECS/serialização/async diferidos com gate de revisão); 7/7 conformes ao template | `docs/arquitetura/adr/000{4,5,6}-*.md` + links nas consumidoras |

## Pendências conhecidas

- Desinstalar Go e remover `GOPATH`/`GOROOT` (ação elevada do dono).
- Verificar evidência de implementação das 54 issues em `done/` (Go).
- Revisão fria dos goldens 08-2 contra captura real quando a FASE-08-6 liberar capturas (fixtures atuais são sintéticos).
- Revisão fria do `estrutura-e-padroes.md` por outro agente/humano (subtarefa da FASE-02-4).
- Captura ao vivo (08-6): structs S→C, offsets de campo do header e causa do freeze — exige
  `wyd-net`/`wyd-core` executáveis + VM com o jogo.
- Validar spawn real em VM com o jogo (complemento da 08-7).
