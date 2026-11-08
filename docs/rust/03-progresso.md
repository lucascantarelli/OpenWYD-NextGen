# Progresso da execução — roadmap-rust

> Documento vivo. Atualizado a **cada issue** executada (regra em
> `docs/rust/02-execucao.md` §5). Última atualização: 2026-11-07 (remoção do arquivo Go + 09-1/14-1).

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
| FASE-05-2 release/docker | ⬜ | Base: `docker/Dockerfile.launcher` + `Dockerfile.portal`. |
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
| FASE-09-1 auditoria de skills | ✅ | Relatório em `roadmap-rust/issues/FASE-09-1-auditoria-skills.md` (7 skills com destino + trechos obsoletos + regras atemporais; plano da 09-2). |
| FASE-09-2 skills sucessoras | ⬜ | Depende de 09-1. |
| FASE-10-1 equações | ⬜ | Depende de 03-5. |
| FASE-10-2 velocidade | ⬜ | Depende de 10-1. |
| FASE-10-3 laboratório | ⬜ | Depende de 10-1, 10-2. |
| FASE-11-1 guia + manuais | ⬜ | Pastas `docs/manual-*/` criadas. |
| FASE-11-2 C4 + pacotes | ⬜ | Depende de 08-1. |
| FASE-12-1 dashboards | ⬜ | Depende de 06-1. |
| FASE-13-1 rito ADR | ✅ | `docs/arquitetura/adr/0000-template.md` criado (estado, contexto, decisão, consequências, alternativas, rastreabilidade); `0001` promovida a **aceita** no formato do template. |
| FASE-13-2 primeiras ADRs | ✅ | ADR-0004 (SoA manual), 0005 (LE+`repr(C)`), 0006 (std até rede/vendor); 7 arquivos 0000–0006 sem lacuna, seções conformes; consumidoras vinculadas. |
| FASE-14-1 pesquisa | ✅ | Notas datadas em `docs/rust/01-dependencias.md` (manter/descartar por candidato + PoC fora de `crates/`; sem diff no workspace). |
| FASE-15-1 suite + concorrência | ⬜ | Depende de 05-1, 10-3. |
| FASE-15-2 carga + banco + release | ⬜ | Depende de 15-1, 12-1. |

## Registro de execução

| Data | Issue | O que mudou | Evidência |
|---|---|---|---|
| 2026-11-07 | FASE-00-1 | Caches Go limpos (`go clean` exit 0) | `go clean -cache -modcache -testcache` |
| 2026-11-07 | FASE-00-2 | VS Code verificado: rust-analyzer + clippy; zero extensão Go | `code --list-extensions` |
| 2026-11-07 | FASE-00-3 | `develop` confirmada como branch atual | `git branch --show-current` |
| 2026-11-07 | FASE-01-1 | Planejamento Go restaurado do histórico (`roadmap/` estava vazio no arquivo) e inventariado | `git archive HEAD roadmap \| tar -x -C arquivo-Go-removido` → 138 arquivos |
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

## 2026-11-07 — Revisão total + reparos do arquivamento (este turno)
- Inventário: 60/60 issues referenciadas no index (100% planejadas); painéis 23✅/5🚧 coerentes (antes da FASE-08-8).
- Revisão registrada em `docs/rust/04-revisao-roadmap.md` (cobertura, conformidade skills, reparos, prova, bloqueadores).
- Reparos: `api/web/v1/web.proto` restaurado do git (b0a0394^, 2046 linhas) + README; 63 handlers Go repovoados no arquivo; `guias.test.ts` aponta p/ arquivo; `wyd-portal/Dockerfile` resiliente ao proto; paths `../api` corrigidos nos .env.example.
- Prova: portal tsc 0 + vitest 118/118; cargo fmt+clippy+test verdes; compose config verde (base e prod).

## 2026-11-07 — Rename para WYD + GUI Tauri decidida (este turno)
- Nome: dono manteve "WYD"; repo GitHub já renomeado para `lucascantarelli/wyd`; remote local ajustado e verificado (`git ls-remote` resolve HEAD).
- Rename aplicado em 49 arquivos (skills, compose, portal, docs, issues, Cargo descriptions, proto `go_package`); crates já eram `wyd-*` (sem mudança de código). Restos zero fora de paths reais Windows (pasta local mantida) e `arquivo Go removido (2026-11)` + `client/` intactos.
- Containers e canais dos guias renomeados para o padrão `wyd-*` / `WYD-1/2`.
- Docs: `docs/cliente/launcher.md` (análise legado + referência Electron/Squirrel + proposta), skill nova `.agents/skills/wyd-launcher`, ADR-0007 (Tauri v2 aceita), issue nova FASE-08-8 (GUI launcher, 🚧).
- Inventário: 61/61 issues no index (100% planejadas); painéis 23✅/6🚧 coerentes.
- Prova: compose config verde (base e prod); portal vitest 118/118; `cargo test --workspace` sem falhas.

## 2026-11-07 — Revisão docker completa + rename portal/ → wyd-portal/ (este turno)
- Resíduo openwyd/NextGen zerado fora de paths reais Windows e `arquivo Go removido (2026-11)`/`client/` (o `grep` anterior só via arquivos tracked — compose, `docker/`, `api/`, `.env` tinham ficado de fora; corrigido com varredura total).
- `portal/` → `wyd-portal/` (pasta renomeada; `portal/Dockerfile` → `docker/Dockerfile.portal` — todo Dockerfile agora em `docker/`); `docker-compose.yml` aponta para o novo caminho; `.gitignore` + 33 arquivos de refs atualizados.
- `docs/rust/05-stack-docker.md` criado: mapa crates → deployáveis (só tmserver/gateway/portal/db viram containers; libs e launcher fora), integração de observabilidade (06-1 → 12-1) e regras (arquivo Go morto, docs só em docs/).
- Issues docker resolvidas: FASE-05, 05-2, 15, 15-2 sem refs a `Dockerfile.rust` (substituído); `Makefile` ganha `portal-install/dev/test/build` (documentados no AGENTS mas inexistentes).
- Prova: compose config verde (base e prod); vitest 118/118 em `wyd-portal/`; tsc OK; `cargo test --workspace` 21 suites ok.
- Painel: 61/61 planejadas, 23✅/6🚧 (sem mudança de status; revisão, não execução).

## 2026-11-07 — Revisão docker/devops com questionário do dono (este turno)
- Regras fixadas pelo dono: `arquivo Go removido (2026-11)` mantida, só-leitura, nunca fonte; expurgo total do que veio do Go; docs só em `docs/`; skill normativa reescrita.
- Expurgo: `docker/README.md` apagado; `.env.example` sem bloco E2E/OPENWYD (eram convenção Go copiada); compose sem `launcher`, sem `depends_on` do portal no db, `WEB_API_ADDR` default `gateway:7600` (nome reservado p/ FASE-07-1); Dockerfiles com headers e COPY fail-fast; `api/web/v1/README.md` com paths corretos.
- Criado `docs/rust/05-stack-docker.md`: crates → deployáveis (containers só tmserver/gateway/portal/db; libs e launcher fora), o que sobe hoje, alvo por issue dona, integração de observabilidade (06-1 → 12-1, sem código: deps bloqueadas em ADR+vendor).
- Skill `wyd-server-ops` reescrita para o stack Rust (sinais de saúde, 502 esperado sem gateway, proibições incluindo copiar da legada).
- Exceção consciente: `api/web/v1/web.proto` (2046 linhas, origem Go) mantido — é dependência viva do portal até a FASE-07-4; remover quebraria vitest. Dono pode overrule.
- Prova: compose config verde (base e prod). Resíduo Go fora da legada só em 3 skills antigas (client-connect, database, testing-bots) — próxima rodada.

## 2026-11-07 — Remoção do arquivo Go + aderência total Rust (este turno)
- Arquivo Go/C++ removido do repo; 65 arquivos normalizados (roadmap-rust/issues, docs, skills, tools/protocol) para fonte canônica Rust (docs/arquitetura/protocolo.md, docs/cliente/, docs/rust/, crates/, fixtures/).
- Normativos reescritos: AGENTS.md 100% Rust, README sem .deprecated-go, .gitignore sem arquivo, .env.example sem stack Go, gen-protocol.mjs congelado com falha explícita, headers das crates apontando para docs/arquitetura/protocolo.md.
- Skills: wyd-rust-toolchain sem Go, wyd-server-ops sem copiar legado, wyd-rust-core sem docs/migration; grep SKILLS-LIMPAS nas Rust vigentes.
- Portal: guias.test.ts cita payloads Rust (character.rs etc.) + guias.json remapeado para crates/docs/arquitetura; grpc.ts lazy (proto ausente só falha no uso real) — vitest 118/118, typecheck 0.
- Issues de menor esforço executadas: 09-1 (auditoria das 7 skills Go com destino + plano 09-2) e 14-1 (notas datadas em 01-dependencias.md sem diff no workspace) — ambas ✅ no index.
- Prova: cargo fmt --check 0, clippy -D warnings 0, cargo test --workspace ok, compose config base+prod 0, gen-protocol congelado 0.

## 2026-11-07 — FASE-09-2 concluída: skills sucessoras Rust (este turno)
- Criada `wyd-rust-testing` (harness/goldens/carga/bots); atualizadas para Rust: `wyd-database`, `wyd-client-connect`, `wyd-game-admin`, `wyd-economy`, `wyd-web-admin-ui`; `wyd-testing-bots` virou alias de transição; `wyd-server-dev` aposentada com stub e sucessoras por domínio.
- Corrigidas refs obsoletas: `wyd-rust-db`/`wyd-rust-obs` → `docs/arquitetura/modelo-banco.md`; `wyd-client-reverso` sem `Source/`; `keytable.mjs` → `codec/generated/keyword_table.rs`; `captura.mjs` → `wire/generated/opcode_table.rs` (goldens 5/5 de volta).
- Mapas: `AGENTS.md` §0/§6 com `wyd-rust-testing` e ex-Go resolvidas; `docs/manual-técnico/skills.md` com destinos 09-2 ✅; issue `FASE-09-2-criar-skills.md` ✅ com evidência; `roadmap-rust/index.md` 09-2 ✅.
- Prova: cargo fmt 0, clippy -D warnings 0, cargo test 20 suites ok, vitest 118/118, tsc 0, compose config base+prod 0, gen-protocol congelado, captura 5/5, grep SKILLS-LIMPAS.
- Próxima de menor esforço: FASE-11-2 (guia C4 de pacotes — só documentação, sem código).

## 2026-11-07 — FASE-11-2 concluída: C4 + specs de pacotes (este turno)
- Criado `docs/arquitetura/c4-pacotes.md`: N1 contexto (cliente 7662 → servidor Rust → Postgres/portal), N2 containers ancorados nos members do `Cargo.toml` e nos services `db`/`portal`, N3 só componentes existentes (tick/ECS/spatial/ports, codec/wire/payloads, limits, model/repo/pool/transfer, api/auth/audit/service).
- Transporte futuro (framing/sessão 8281) marcado como FASE-07, sem caixa inventada; specs de pacotes ancoradas em `protocolo.md` §4–§6 (login 0x020D, CNF 0x010A–0x011D, ESCENE_FIELD, combines 0x02C3–0x02C4, grants via delivery_queue).
- `visao-pastas.md` corrigido: `docs/arquitetura/` listava só 2 de 5 docs.
- Prova: cargo fmt 0, clippy -D warnings 0, cargo test 20 suites ok; opcodes conferidos no `opcode_table.rs`.

## 2026-11-07 — FASE-11-1 concluída: guia de contribuição + manuais (este turno)
- `docs/manual-técnico/guia-contribuicao.md`: pré-requisitos, 5 comandos do primeiro build, `make check`, portal/ferramentas, fluxo de branch e DoD de 02-execucao, regras que reprovam contribuição, mapa de contexto.
- `docs/manual-usuario/README.md` (pasta estava vazia): jogar no servidor local, CLI real do launcher (`check`/`launch`/`update` + `--dry-run`), telas reais do portal, limites (delivery_queue) e FAQ.
- `README.md`: árvore com `docs/` duplicado e skills "Go (legado)" corrigidas; `wyd-rust-testing` incluída; tabela de docs com C4, guia e manual.
- Prova: cargo fmt 0, clippy 0, cargo test ok, vitest 118/118, tsc 0, links do README todos resolvendo.

## 2026-11-07 — ADR-0008: transporte do Cold Path decidido (este turno)
- Decisão registrada por pedido do dono: HTTP/1.1 + JSON em loopback (`127.0.0.1:7600`), `POST /admin/v1/{servico}/{metodo}`, envelope `AdminResult` (`result` 1..5) + `trace_id`. gRPC/proto ficam **fora da árvore**.
- Fundamento no repo: zero deps externas (`01-dependencias.md`), ADR-0006 difere `tokio`; nenhum `*.proto` versionado e `grpc.ts` aponta para proto inexistente; 16 serviços unários, 1 consumidor; `AdminResult`/`audit`/deadline já são nossos.
- Contrato completo na ADR: rota→params→resposta, regras de auth/auditoria (bind loopback, `StaffSession` do token, audita leitura e escrita, `401` anônimo não audita), limites (32 MiB, 5 s no servidor, 10 s no cliente) e status de transporte (400/401/404/405/413/500) vs recusa de negócio (200 + `result`).
- Árvore §13 do `wyd-admin` ganhou `src/transport/{mod,http,json}.rs` e `tests/contract.rs`; `.proto` proibido.
- Issues atualizadas: `FASE-07.md` (contexto, passos, arquivos, pendências) e `FASE-07-1-gateway.md` (transporte HTTP/JSON, passos, aceite, rollback). Docs alinhados: `estrutura-e-padroes.md` §13, `visao-pastas.md`, `00-visao-estrutura.md`, `05-stack-docker.md`, skill `wyd-web-admin-ui`.

## 2026-11-07 — FASE-07-1: transporte HTTP/JSON do wyd-admin implementado (ADR-0008)
- `crates/wyd-admin/src/transport/json.rs`: JSON do contrato sem dependência (escapes, `\uXXXX` com surrogate, `MAX_DEPTH` 32, rejeita `01`/lixo/aninhamento profundo; nenhum `panic!` em entrada).
- `crates/wyd-admin/src/transport/http.rs`: HTTP/1.1 mínimo em std — POST obrigatório, `Content-Length` obrigatório, sem chunked, head ≤ 16 KiB, corpo ≤ 32 MiB, `cache-control: no-store`, `x-trace-id`, `connection: close`, `bind_loopback` e `serve_conn` só em loopback.
- `crates/wyd-admin/src/transport/mod.rs`: `POST /admin/v1/{servico}/{metodo}` → `accounts/find`, `accounts/create`, `grants/request`; `Authenticator` (porta) + `StaticTokens`; 401 sem token (não audita), papel insuficiente = 200 + `result: 2` com `auth.denied` auditado, 404/400/405/413 de transporte, envelope `result` 1..5 + `trace_id`.
- `crates/wyd-admin/tests/contract.rs`: 9 casos incl. grant idempotente (replay com mesmo `grant_id`) e round-trip real em socket loopback.
- Prova: `cargo fmt --all -- --check` 0, `cargo clippy --workspace --all-targets -- -D warnings` 0, `cargo test --workspace` 22 suites ok.
- Pendente na issue: `admin-client.ts` no portal (remover `@grpc/*`) e os 13 serviços restantes sobre o mesmo roteador.

## 2026-11-08 — FASE-07-1: portal fiado no HTTP/JSON (fim do gRPC no BFF)
- `wyd-portal/src/server/admin-client.ts`: `fetch` para `WEB_API_ADDR` (default `http://127.0.0.1:7600`), `POST /admin/v1/{servico}/{metodo}`, `authorization: Bearer $WEB_API_TOKEN`, `x-trace-id`, `AbortSignal.timeout(10 s)`, `cache: no-store`; mantém as 16 exportações e todos os tipos do antigo `grpc.ts` (serviços via `svc<T>()`), removendo `moderator_id`/`moderatorId` do corpo antes de enviar.
- `promisify.ts` vira fachada `(req, traceId) => Promise<Res>` (deadline no cliente); `trace.ts` expõe `traceHeaders()` puro; `errors.ts` ganha `AdminUnavailableError` (conexão recusada/timeout/5xx/resposta inválida → 502) e `isInfraFailure` deixa de checar código gRPC.
- 25 módulos do BFF reimportados (`@/server/grpc` → `@/server/admin-client`), `grpc.ts` removido; `@grpc/grpc-js`/`@grpc/proto-loader` fora do `package.json` e do lock; `docker/Dockerfile.portal` não copia `.proto`; `.env.example` documenta `WEB_API_ADDR` (URL) + `WEB_API_TOKEN`; README do portal reescrito para HTTP/JSON.
- Prova: `npm run typecheck` 0, `npm run test -- --run` 118/118 (16 arquivos), `grep grpc` em `src/` = 0.
- Pendência honesta: `StaticTokens` não valida HMAC (token de dev até a fiação cripto) e os 13 serviços ainda respondem 404 de transporte enquanto as rotas não existem.

## 2026-11-08 — FASE-03-4: inventário dos dados estáticos (itens/skills/NPCs)
- `docs/arquitetura/modelo-banco.md` §6 (novo): procedência do conteúdo de jogo, com números conferidos no git — `origin/main` guarda `internal/migrations/seed_*.up.sql` (`item_catalog` 3.220, `skill_catalog` 151, `npc_template` 1.991, `npc_definition` 6.100, `npc_shop_item` 6.708, `world_item_seed` 49, `guild_zone` 5) e `0008_platform_content.up.sql` (`guard_spawn`, `level_item` por ordinal, `mob_merc`, `game_setting`, 52 retângulos de `Regions`, com `content_source_file` de proveniência).
- Regra registrada: seed Rust sai das linhas de catálogo (texto + `sha256`), **não** dos `.bin` do cliente — `ItemList.bin` (910.004 B), `SkillData.bin` (23.812 B), `Mixlist.bin` (272.000 B) abrem com bytes repetidos (XOR) e não têm leitor em `tools/client/`; `docs/cliente/inventario.md` já os marcava como CONHECIDO parcial/UNVERIFIED.
- Lacunas declaradas: nenhuma migração Rust cria tabela de conteúdo; `docs/sistemas/*` (13), `docs/lore/*` (3) e `docs/guias/*` (4) existem só em `origin/main` (a árvore perdeu os originais e mantém `guias.json` com 19 slugs); não há documento de mix/refino.
- Issue `FASE-03-4` passou a 🚧 (passo 1 concluído) no índice e no próprio arquivo; falta decidir formato destino e implementar o validador de boot.
- Verificação: contagens lidas de `git show origin/main:<seed>` (cabeçalho de cada arquivo) e `git cat-file -s` para os TXT; nenhum código alterado, logo sem gate Rust/TS neste passo.
