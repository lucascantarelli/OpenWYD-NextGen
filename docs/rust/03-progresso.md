# Progresso da execução — roadmap-rust

> Documento vivo. Atualizado a **cada issue** executada (regra em
> `docs/rust/02-execucao.md` §5). Última atualização: 2026-11-08 (banco remodelado e catalogado 0001–0024,
> `wyd-sim` organizado por sistema com unidades e determinismo provado — ADR-0011 —, e gema no
> inventário do core).

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
| FASE-03-2 ECS | 🚧 | Dados prontos (`components.rs` + `ComponentStore` SoA). **Gema ligada ao inventário real:** `ComponentStore::apply_gem` é o caminho único — consome uma unidade da mochila, grava o efeito e publica exatamente um `DomainEvent::GemEffectChanged` por aplicação (zero nas recusas tipadas: `EmptySlot`, `UnknownGem`, `GemRefineTooLow`, `GemRefineTooHigh`, `GemNotInBag`, `EventQueueFull`, `EntityNotAlive`), sem mutação parcial; `tests/gem.rs` prova Anct estritamente melhor que comum (+9 → 40 de perfuração contra 0) e `tests/gema_seed.rs` prova a **paridade com o seed real** (`0022_seed_refino.sql`, via `include_str!`): as 4 gemas, todo o refino 10..=15, Anct no mínimo−1 contra comum recusado e a troca de efeito com `from`/`to` do catálogo. Restam `systems/`, ADR de ECS e bench. |
| FASE-03-3 spatial grid | ✅ | `spatial/grid.rs`; guerra 5000: 8,6 ms inserts, 79 µs consulta (debug). |
| FASE-03-4 dados estáticos | 🚧 | Esquema+seed completos em **24 migrations** (DDL 0001–0016, seed 0017–0024), **59 tabelas / 457 colunas**, aplicadas num Postgres 17 real com ciclo up/down/up verde (`make migrations-apply`: 59 tabelas → 0 → 59, contagens idênticas) e schema conferido contra o catálogo (`make banco-verify`); catálogo, mundo, refino, economia, rotas, combate, quests, eventos, arena e progressão modelados. Falta carga no boot (FASE-04-3). |
| FASE-03-5 autoridade do servidor | ✅ | `systems/{movement,combat,skill}.rs` (ADR-0014): a API recebe intenção, nunca resultado — `step` valida orçamento/limites do mundo, `attack` valida alcance/cooldown/alvo e pega o **dano** de `DamageModel` (FASE-10), `cast` valida skill aprendida/MP/cooldown/alcance. **Eixo inventário fechado** (`systems/inventory.rs`, ADR-0015): `equip`/`equip_into` recebem só o slot da mochila e derivam o slot de `item_kind`, conferem `req_level` do `item_catalog`, aplicam política de troca explícita e devolvem o item anterior à mochila; `unequip` usa o primeiro slot livre (mochila cheia recusa) e `move_item` empilha até `max_stack` sem troca silenciosa. `next_attack_ms`/`next_skill_ms` são componentes contíguos; recusa não altera posição, HP, MP, cooldown, item, equipamento nem fila (evento só no aceite). `tests/autoridade.rs` (10) e `tests/inventario.rs` (9) provam pacote forjado recusado sem mutação. Mapa das ações em `docs/arquitetura/autoridade-servidor.md`. |
| FASE-03-6 IA de NPC | ✅ | `npc/` (`state`, `behavior`, `brain`, `route`, `budget`) + `systems/npc.rs` (ADR-0016): a IA **emite intenção**, nunca efeito — `MoveTo`/`Attack`/`Flee`/`Return`/`Idle` passam por `movement::step`/`combat::attack`, então uma intenção forjada é recusada com posição e fila intactas. FSM de 7 estados com precedência explícita (fuga → leash → ataque → perseguição), cadência 5 Hz (`tick % 4 == entity % 4`) com `DecisionBudget` por tick, RNG `xorshift64` semeado por `(world_seed, entity)` e tabela de ameaça com teto 8 e desempate pelo menor `entity` (cópia presa por golden da matemática de `wyd-sim`, ADR-0011). Perfil (`NpcProfile`) vem do servidor e é validado no boot — nenhuma constante de balanceamento na engine; `mob_template_stat` vazia deixou de ser bloqueio. `tests/npc.rs` (12) prova cadência, aggro inclusivo, fuga, leash, recusa atômica (`NotACreature`, `ThreatTableFull`, `EntityNotAlive`) e golden FNV-1a determinístico. **Custo medido** (`cargo bench -p wyd-core --bench tick`, release, 5 execuções): 500 NPCs + 50 jogadores → **88,6–204,5 µs/tick** (p99 119,4–367,8 µs), **0,18%–0,41% do orçamento** de 50 ms a 20 Hz (pior p99 0,74%), 0,71–1,64 µs por decisão; sem alvos, 12,5–31,8 µs/tick. Pior caso ainda com folga de ~13×, então cadência 5 Hz e teto de 133 decisões/tick ficam como estão. |

## Fase 4 — persistência (`wyd-db`)

| Issue | Status | Evidência / observação |
|---|---|---|
| FASE-04-1 modelagem | ✅ | `docs/arquitetura/modelo-banco.md` (ER, decisões) + `docs/arquitetura/banco-catalogo.md` gerado: **59 tabelas**, colunas/tipos/nulidade, PK/FK/CHECK/índices, fonte por tabela e 8 decisões provisórias `D-B01`..`D-B08`; mapa tabela legada → nova com pendências nomeadas. |
| FASE-04-2 migrations | ✅ | 24 migrations reversíveis (0001–0024, uma por domínio, DDL 0001–0016 antes do seed 0017–0024, **sem `ALTER TABLE` no `up`**) + `seeds/dev.sql`; prova em três camadas: estrutural (`check-migrations.mjs`), contrato doc↔SQL (`tests/catalogo.rs`) e banco real — `verify-schema.mjs` (banco vazio → up → `information_schema` == catálogo) e `migrate-cycle.mjs` (up/down/up com contagens idênticas). Números em `crates/wyd-db/migrations/README.md`. |
| FASE-04-3 persistência async | 🚧 | Base pronta (traits, `PoolConfig`, `DeliveryQueue`); resta impl `pg` + pool (rede + ADR). |

## Fases 5–15

| Issue | Status | Observação |
|---|---|---|
| FASE-05-1 pipeline CI | ✅ | `.github/workflows/rust.yml` com job `rust` (`fmt --all --check`, `clippy --all-targets -D warnings`, `build`, `test --all-targets`), job `bench` (compila todos e roda os dois medidos) e job `supply-chain` (`cargo-audit`/`cargo-deny` via `taiki-e/install-action`). `deny.toml` v2 (advisories, licenças fechadas, `wildcards = deny`, fontes só crates.io, `private.ignore`); `Makefile` com `test-all`, `bench-build`, `deny` e `ci` espelhando o workflow; `publish = false` nas 7 crates e versão nas deps internas. **Medido:** bench da IA 205,86 µs/tick = 0,412% do tick (p99 0,840%), piso 29,14 µs; balde 8,1 ns/op e porte 234,3 ns/handshake (wyd-net). `cargo audit` 0 (7 crates, 1422 advisories) e `cargo deny check` 0 (advisories/bans/licenses/sources ok). **Prova de falha:** lockfile descartável com `time 0.1.44` → `cargo audit` exit 1 com RUSTSEC-2020-0071; diretório removido depois. |
| FASE-05-2 release/docker | ✅ | `[profile.release]` com `lto = "thin"`, `codegen-units = 1`, `strip = "symbols"` e `panic = "abort"` (ADR-0017): `wyd-launcher` **323.072 → 296.448 bytes** (−8,2%), build release do workspace exit 0 e binário executando (`check` de manifesto real exit 0). Imagem mínima multi-stage com runtime **distroless `cc-debian12:nonroot`** (sem shell, sem toolchain), labels OCI de versão/commit, `docker build` exit 0 e **37.847.971 bytes** (36,1 MiB); smoke no container: `check /jogo /jogo/manifest.toml` exit 0 (`1 íntegros`), `--help` imprime uso (exit 2 por contrato), `docker export` sem `/bin/sh` nem `rustc`/`cargo`/`gcc`. `.dockerignore` criado; `Makefile` com `release`/`docker`/`docker-smoke` e workflow com job `release`. Bench sem regressão: IA 85,6–189,7 µs/tick (baseline 205,86) e balde 2,7–2,8 ns / porte 100,1–100,8 ns (baseline 8,1 / 234,3). |
| FASE-06-1 telemetria | ✅ | `tracing` 0.1.44 + `tracing-subscriber` 0.3.23 (ADR-0018) e exporter Prometheus próprio (`TcpListener` + texto 0.0.4) em `crates/wyd-core/src/metrics/`; contadores de rede em `crates/wyd-net/src/metrics.rs` atrás da feature `metrics` (default off — o `fetch_add` por pacote custa ~24,5 ns contra 2,7 ns sem ela). Nível por `WYD_LOG` (fallback `RUST_LOG`, default `info`). **Prova em execução:** `WYD_LOG=debug cargo test -p wyd-core --test telemetry -- --nocapture` loga `DEBUG tick concluido tick=0 micros=2365`… e `INFO resumo de telemetria ticks=40 p50_us=5000 p95_us=5000 p99_us=5000 rss_bytes=0`, e o `GET /metrics` real devolve `wyd_ticks_total 40`, `wyd_tick_duration_microseconds_count 40`, buckets cumulativos com `le="+Inf"} 40` e `wyd_entities_active 104`; com `WYD_LOG=info` nenhuma linha DEBUG sai (0) e valor inválido cai em `info`. `Cargo.lock` 7→24 pacotes; `wyd-launcher` release segue 296.448 bytes e a imagem distroless remede **37.847.971 bytes** (igual à 05-2, launcher não linka o core), smoke no container exit 0. Benches: tick 81,27–82,22 µs (p99 88–116) contra baseline 85,6–189,7; balde 2,7 ns e porte 98,1 ns contra 2,7–2,8 / 100,1–100,8. |
| FASE-06-2 hardening | ✅ | `limits/rate.rs` (balde + porte de handshake) + `tests/flood.rs`; flood 100/10000, fuzz 5000 sem panic; legit preservado. |
| FASE-07-1 gateway | ✅ | Backend do contrato (`AdminError`→HTTP, staff/auditoria/grants idempotentes); 11 lib + 4 integração; fiação do portal aguarda transporte. |
| FASE-07-2 painel usuário | ✅ | Rotas `panel/{self,characters,rankings}` + `accounts/recovery` no roteador do ADR-0008, com `account_id` injetado pelo BFF a partir do cookie assinado (ADR-0019); página `/conta` com conta, personagens, ranking e recuperação. **Prova:** 6 testes de contrato (`tests/painel.rs`: 401, conta, personagens escopados, ranking ordenado sem `account_id`, recuperação idêntica para login existente/inexistente, parâmetros inválidos → `result: 3`, rota desconhecida → 404) + `typecheck` 0 e vitest 137/137 (19 arquivos). |
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
| FASE-10-1 equações | 🚧 | Composição dos 9 sistemas codificada em `crates/wyd-sim/src/formula/` (atributos, progressão, dano, acerto/crítico, efeitos, resistência, drop, regen, aggro) + ordem do tick em `scenario/combat.rs`; a **ordem exata** de mitigação/crítico é hipótese D-02..D-04 (`docs/arquitetura/matematica-sim.md` §4) até o trace da FASE-08-6. Determinismo provado em `tests/determinism.rs`. |
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
| 2026-11-08 | FASE-07-2 | Painel do jogador: `panel/*` + recuperação auditada, `/conta` no portal; SancRate do legado no banco (28 células) e refino `+10 → +11` corrigido para 3% com Lac | `crates/wyd-admin/{src/api/panel.rs,tests/painel.rs,src/transport/mod.rs}`, `crates/wyd-db/migrations/{0009_refino,0022_seed_refino,0025_seed_sanc_rate}.sql`, `wyd-portal/src/{server/panel.ts,app/(public)/conta,app/api/panel/recovery}` |
| 2026-11-08 | FASE-06-1 | Telemetria: `tracing` + exporter Prometheus próprio; `/metrics` real com 40 ticks (`wyd_ticks_total 40`, histograma, `wyd_entities_active 104`), log estruturado com nível por `WYD_LOG`; contadores de rede atrás de feature (custo medido); CRC: lock 7→24, launcher/imagem inalterados | `crates/wyd-core/src/{metrics/,telemetry.rs}`, `crates/wyd-net/src/metrics.rs`, `crates/wyd-core/tests/telemetry.rs`, `crates/wyd-net/tests/metrics.rs`, `docs/arquitetura/adr/0018-telemetria-tracing.md` |

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

## 2026-11-08 — ADR-0009 reescrita: conteúdo vem do PostgreSQL, não de arquivo
- Correção de rumo por veto do dono: a primeira versão da ADR-0009 propunha `content/*.csv` versionado e foi **descartada** — arquivo plano é prática do servidor Go e o projeto está eliminando isso (`pre-release-rust.md` FASE 3 §3.2, FASE 4 §4.1/§4.2).
- Decisão vigente: **PostgreSQL é a única fonte canônica** do conteúdo (`item_catalog`, `skill_catalog`, `npc_*`, `level_item`, `guard_spawn`, `mob_merc`, `game_setting`, `region`, `content_version`) com `PRIMARY KEY`/`FOREIGN KEY`/`CHECK`, populado por migrações versionadas de seed (idempotentes, com `down`); `wyd-db` carrega no boot e `wyd-core` valida antes do primeiro tick (`ports/content.rs`), abortando com `{tabela, pk, coluna, motivo}`.
- Nenhum `.csv`/`.txt`/`.bin` é lido em runtime: os arquivos do legado são **insumo** de `tools/content/gen-seed.mjs` (que emite SQL) e base de conhecimento. `content_source_file` guarda só a proveniência do import.
- Propagado: `docs/arquitetura/adr/0009-conteudo-estatico.md` (reescrito), `modelo-banco.md` §6, `estrutura-e-padroes.md` §10/§12/§15, `visao-pastas.md`, issue `FASE-03-4` (fonte decidida; dependência passa a ser **FASE-04-3**, o `sqlx`).
- Consequência declarada: o servidor passa a **não subir sem Postgres**; até a FASE-04-3 existir, o conteúdo fica semeado mas sem consumidor no core.

## 2026-11-08 — FASE-03-4: catálogo de conteúdo no banco + base de conhecimento `docs/wyd/`
- **Seed decidido:** migração versionada gerada (não importador manual) — `tools/content/gen-seed.mjs` lê `origin/main` via `git show` e emite SQL idempotente em lotes de 200 linhas; registrado na ADR-0009 (com alternativa do importador descartada e mitigação de tamanho).
- **Esquema (0007):** `content_version`, `game_system`, `game_class`, `skill_lineage`, `item_kind`, `item_effect`, `item_catalog` (+`kind`, `primary_system`, `event_only`, `tradeable`, `stackable`, `class_mask`, `effect_names[]` com GIN, `classification_source`, `confidence`) e `skill_catalog` (+`class_code`, `lineage_code`, `primary_system`, `skill_role`); PK/FK/CHECK em tudo que é provado, `class_mask` documentada como UNVERIFIED.
- **Masssa real (0009/0010, gerados):** 3.220 itens com preço/grade/requisitos/efeitos e 151 skills; 55 símbolos `EF_*` no vocabulário; 1.688 itens (52%) classificados por regra de nome, **1.532 ficam `NULL`** (nome não prova), 99 `event_only`, 156 não negociáveis; 12 skills mapeadas por nome confirmado, 139 `unmapped`.
- **Base de conhecimento:** `docs/wyd/{README,lore,classes,skills,sistemas,itens}.md` — Kersef, Yetzirah/Tzfah, Hekalotia/Akeronia, Kefra, as 4 classes + Arch, linhagens, 10ª/12ª skill, refino/repleção/celestial, drop, reinos; o que é UNVERIFIED está marcado e não vira `CHECK`.
- **Prova:** bloco `up` de 0007–0010 aplicado com `ON_ERROR_STOP=1` no Postgres do compose em banco descartável `wyd_seedcheck` → 3.220 itens, 151 skills, 55 efeitos, 20 sistemas, 1.688 classificados, `content_version = 3`; banco de teste removido e container parado depois.
- **Pendente:** carga tipada + validação no boot (`wyd-db` → `ports/content.rs`) e teste de conteúdo inconsistente — bloqueado pela FASE-04-3 (`sqlx`, ADR-0006).

## 2026-11-08 — FASE-03-4: mundo no banco (NPCs, lojas, prêmios, spawns, regiões) + sanitização

- **Auditoria antes do seed** (`modelo-banco.md` §6.6): 12 insumos em UTF-8 válido e NFC;
  `LevelItem.txt` com 4 bytes truncados, todos em linha `#`; 5 TXT em CRLF; 0 BOM, 0
  invisível, 0 controle, 0 NBSP; 20 nomes de `npc_template` em CP949 lido como latin1
  (chave preservada; 12 recuperados para coreano em `name_recovered`).
- **Duplicidades explicadas** (§6.6.1): 489 grupos de nome de item cobrindo 1.370 itens —
  tiers `(Anct)` g5..g8, `Cupom_da_Sorte` ×78, placeholders `NONAME`/`not_used`, itens de
  classe. **0 duplicata acidental** (nome+grade+preço+efeitos idênticos) → nada deduplicado.
  `npc_template` tem 345 grupos de variantes `@Nome`/`Nome_` — são chaves, mantidas.
- **Esquema (0011):** `content_source`, `npc_template`, `npc_definition`, `npc_shop_item`,
  `level_item`, `guard_spawn(_reward)`, `mob_merc`, `game_setting`/`game_matrix`, `region`,
  `world_item_seed`, `guild_zone` — FK real onde não há órfão e ausência de FK documentada
  onde o catálogo tem lacuna (22 itens de loja, 1 de `level_item`).
- **Busca sem acento:** `content_fold()` IMMUTABLE + `name_fold` gerado em `item_catalog` e
  `region` (sem `unaccent`/`citext`; motivo em `modelo-banco.md` §7).
- **Prova:** 0007–0012 no Postgres do compose em `wyd_seedcheck` → 1.991/6.100/6.706/161/
  27/25/20/52/49/5, `content_version = 4`, `sha256(blob)` 1.991/1.991, `down` inverso →
  0 tabelas.

## 2026-11-08 — `wyd-sim` organizado por sistema, com unidades e determinismo provado

- **Estrutura (ADR-0011):** `crates/wyd-sim/src/` passou de 3 arquivos para 20 — `units.rs`
  (`Bps`/`Tenths`/`mul_bps`), `error.rs` (`SimError`), `roll/` (`Die` + `XorShift64`),
  `formula/{attributes,progression,damage,hit,effect,resist,drop,regen,threat,refine}.rs`,
  `scenario/combat.rs` (ordem do golpe/tick), `report/table.rs` (saída comparável).
  `lib.rs` virou fachada pura (só declara e reexporta).
- **Documento normativo:** `docs/arquitetura/matematica-sim.md` — por sistema, entradas,
  saídas, unidade, ordem no tick, arredondamento e 11 decisões provisórias (D-01..D-11) com o
  trace que confirma cada uma. `docs/rust/06-matematica.md` virou inventário curto + dívidas;
  `estrutura-e-padroes.md` §14 substituído pela árvore nova.
- **Decisões de fonte:** pesos de atributo, curva de EXP e multiplicador de crítico são
  **entrada de conteúdo**, não constante (a fonte não publica). O que é confirmado
  (`docs/wyd/`) está com a citação no comentário: 6 pontos de status/nível, aprendizado
  2/3/1, Arch +8/nível, tetos 200/255/320, roll de drop 1..1.000.000, 64 s de efeito,
  refino e gemas.
- **Prova:** `crates/wyd-sim/tests/determinism.rs` (8 testes: mesma seed → mesma sequência e
  mesmo tick, seed diferente → resultado diferente, rolls gravados → replay idêntico, estado
  de efeitos/regen/ameaça reprodutível). `tests/goldens.rs` **não foi tocado** — mesmas
  asserções de jogo.
- **Bots/IA:** a matemática da camada A de ADR-0010 (`formula/threat.rs`) existe: tabela de
  ameaça com teto fixo, alvo = maior ameaça com desempate pelo menor `entity`, expiração por
  TTL e enum `NpcMode`. A FSM e a cadência de 5 Hz continuam planejadas em
  `crates/wyd-core/src/npc/` (FASE-03-6) — nada de IA generativa no tick.
- **Pendente:** `formula/speed.rs` (FASE-10-2, ⛔), `bot/{persona,policy}.rs` (FASE-15-3),
  *harness* de milhões de iterações (FASE-10-3) e o trace que fecha D-02..D-04/D-06.

## 2026-11-08 — Gema ligada ao inventário real do core (consumo, recusas tipadas, evento único)

- **Regra:** `ComponentStore::apply_gem` virou o caminho único. `Inventory::plan_gem` valida
  sem escrever (slot, item, gema, refino), `consume_gem` retira uma unidade da mochila e libera
  a pilha zerada, `commit_gem` grava o efeito — nessa ordem: evento publicado antes da escrita,
  para o estado só mudar se a transação for aceita.
- **Recusas tipadas (nenhuma com mutação parcial):** `SlotOutOfRange`, `EmptySlot`,
  `UnknownGem`, `GemRefineTooLow`, `GemRefineTooHigh`, `GemNotInBag` (novo), `EventQueueFull`,
  `EntityNotAlive`. Fora do crate não há como aplicar gema numa cópia solta: os métodos de
  `Inventory` são `pub(crate)`.
- **Fonte de cada regra:** gema aplicada sobre o item, válida para Anct e +10+, efeito trocável
  e escala 40/nível acima de +9, drop 8%, EXP 2% — `docs/wyd/sistemas/refino.md` §3; tabela
  presa por golden em `crates/wyd-sim` (`gem_scale_matches_kersef`).
- **Decisões provisórias:** D-12 (a gema é consumida; uma unidade por aplicação) e D-13
  (reaplicar a mesma gema troca o efeito em vez de ser recusado, porque a fonte diz que o
  efeito não é definitivo) em `docs/arquitetura/matematica-sim.md` §4; ADR-0012 registra.
- **Prova:** `crates/wyd-core/tests/gem.rs` (6 testes de API pública, incl. Anct +9 com 40 de
  perfuração contra comum recusado com 0) e `gem_tests` (11) em `components.rs`; fixtures dos
  testes antigos ganharam estoque de gema, **sem alterar nenhuma expectativa de jogo**.

## 2026-11-08 — Banco remodelado: catálogo gerado + série única verificada do zero

- **Série consolidada em 24 migrations** (DDL 0001–0016, seed 0017–0024): `0023_inventario_gema`
  e `0024_catalogo_skill_nota` foram **absorvidos** por `0003_inventario` e `0007_conteudo_catalogo`
  — refino/Anct/gema e `skill_catalog.notes` nascem na criação da tabela. `ALTER TABLE` no `up`
  agora é erro de build (`tests/catalogo.rs`).
- **Tabelas que faltavam, com fonte:** `0016_progressao` (`character_status`: atributos, pontos de
  status/aprendizado/skill, fama; `character_lineage`: maestria; `character_skills`: skills
  aprendidas sem FK de catálogo) — `docs/wyd/progressao.md` §2–§3 e `docs/wyd/skills.md` §1. Em
  `characters` entraram `tier` (6 camadas) e `arch_of_id` (Mortal ↔ Arch), ambos CONFIRMADOS em
  `progressao.md` §1. Total: **59 tabelas / 457 colunas / 31 índices**.
- **Catálogo gerado:** `docs/arquitetura/banco-catalogo.md` (59 seções com colunas/tipos/nulidade,
  PK/FK/CHECK/índices, arquivo, fonte e estado) + `D-B01`..`D-B08` (decisões provisórias no padrão
  `D-xx`) + mapa tabela legada → nova com pendências nomeadas.
- **Prova em três camadas, com exit code real:** `check-migrations.mjs` 0 (24 arquivos, 59
  tabelas, 140 `INSERT`s); `cargo test -p wyd-db` 0 (catálogo == migrations, `SCHEMA_VERSION` ==
  0024, seed depois do DDL); `verify-schema.mjs` 0 (**banco vazio → 24 `up` → `information_schema`
  == catálogo**: 59 tabelas, 457 colunas, 59 PKs, zero divergência); `migrate-cycle.mjs` 0
  (up/down/up em ~55 s, 59 → 0 → 59, contagens idênticas, `content_version` 7).
- **Pendências reais:** `quest`/`character_quest`, `mob_template_stat`/`equip` e as três tabelas de
  progressão estão criadas e vazias (dependem da cadeia de quest, da extração do `STRUCT_MOB` e da
  FASE-10); `refine_rate` +11 sem fonte; `city.map_id` nulo (o 7662 não publica); e `item_effect_override`,
  `castle_quest_state`, `mob_carry_override` ficam fora do schema por D-B05/D-B06/D-B07.

## 2026-11-08 — IA de NPC e bots: resposta com evidência + modelo de decisão verificável

- **Pergunta ("os bots com IA estão planejados?") respondida por varredura, não memória:** sim,
  planejados e documentados desde ADR-0010 + `docs/rust/07-ia-e-bots.md` (issues `FASE-03-6` e
  `FASE-15-3` abertas, árvores reservadas em `estrutura-e-padroes.md:16/199/206/358`); **nada
  implementado** — `crates/wyd-core` não tem `npc/`, não existe `crates/wyd-bots`, não há LLM
  nem tabela de comportamento. A "FASE 9 — IA" do plano mestre é IA **de desenvolvimento**
  (`pre-release-rust.md:173–178`), e o `pre-release-bots.md` citado em `start-rust.md:8` não
  está na árvore.
- **Lacuna fechada:** `docs/arquitetura/ia-bots.md` (novo) fixa modelo de decisão em três
  estágios com unidades (tiles/ticks/u32, teto de 8 entradas de ameaça, empate pelo menor
  `entity`), cadência de 5 Hz = 1 decisão a cada 4 ticks de 20 Hz com fase `entity_id % 4`,
  orçamento ≤ 10% do tick (UNVERIFIED até bench), contrato de determinismo verificável (golden
  em `fixtures/npc/decision_golden.txt` + hash FNV-1a), estado/observação por camada, mapa de
  pastas e `D-IA01`..`D-IA06`. Corrigida a ADR-0010 (tick é 20 Hz, não "20–50 Hz"; fase `% 4`).
- **Prova da matemática existente:** `crates/wyd-sim/src/formula/threat.rs` — `ThreatTable`
  (teto 8), `threat_from_damage` (1:1, provisório), `select_target` com desempate determinístico,
  `prune` por TTL e `NpcMode` (7 estados + `should_aggro` inclusivo).
- **Schema intocado:** 59 tabelas / 24 migrations; `npc_behavior`, `npc_dialogue` e `llm_job`
  seguem como pendência declarada (nenhum código as consulta — criar antes seria tabela órfã).

## 2026-11-08 — Curadoria de skills por linhagem ampliada (12ª skill) e gerador consertado

- **Defeito encontrado ao retomar:** `tools/content/gen-skill-lineage.mjs` ainda lia
  `0018_seed_skill_catalog.sql` e escrevia `0020_seed_skill_lineage.sql` — o renumbering da
  entrega do banco não acompanhou o gerador, então rodá-lo criaria um arquivo em cima do seed
  do mundo. Corrigido para `0019` (leitura) e `0021` (escrita).
- **Curadoria ampliada com fonte por linha** (`docs/wyd/classes.md` §2, tabela "9ª/10ª/11ª/12ª —
  CONFIRMADO"): 11 linhas novas — a 9ª de Espada Mágica (`Espelho_Mágico`) e a 12ª de 10 das 12
  linhagens (`Clod_Attack`, `WindStorm`, `Seal_Master`, `Holy_Power`, `Thunder_Lising`,
  `Another_Change`, `Manaburn_Master`, `Ice_Armor`, `Freezing`, `Reinforce_Soul`). A 11ª não
  entra (nenhuma fonte publica a lista por linhagem); Natureza e Captura seguem "não confirmada".
- **Honestidade de dado:** `Freezing` e `Reinforce_Soul` entram com `skill_role = NULL` —
  linhagem confirmada, efeito não descrito; `role: null` passou a ser aceito pelo gerador em vez
  de forçar um palpite.
- **Conflitos declarados, não escondidos:** `Clod_Attack` está no seed como `magic_sword` e a
  fonte oficial o coloca em Faith (a curadoria vence porque roda depois); `Another_Change` vinha
  `passive` e entra como `buff`. A limitação do `down` de `0021` registrada nesta entrega foi
  **corrigida** em 2026-11-08 (ver entrada "`down` da 0021 passou a ser seletivo").
- **Medido no banco:** 63 linhas curadas; `skill_catalog` 154 linhas com **72** com
  classe/linhagem (69 da curadoria + 3 da Kersef), contra 63 antes; 82 continuam `unmapped`.
- **Gates:** `cargo test --workspace` 0 · `clippy -D warnings` 0 · `fmt --check` 0 ·
  `check-migrations` 0 · `migrate-cycle` 0 (59→0→59, 24 migrations, 55,1 s, 72 nas duas
  direções) · `gen-catalogo-banco --check` 0 · `verify-schema` 0.

## 2026-11-08 — `apply_gem` provado contra o seed real do banco (paridade core × catálogo)

- **Lacuna encontrada ao retomar:** `crates/wyd-core/tests/gem.rs` provava a transação (consumo
  da mochila, evento único, recusas tipadas) com índices e valores escritos no próprio teste.
  Provava a política, não que ela é a do banco: um `per_level_value` ou `effect_code` errado em
  `0022_seed_refino.sql` passaria despercebido.
- **Teste novo:** `crates/wyd-core/tests/gema_seed.rs` (4 testes) lê **a própria migração**
  `crates/wyd-db/migrations/0022_seed_refino.sql` por `include_str!` e exige do core: as 4 gemas
  (3386–3389) com o efeito que `gem_effect_code` reconhece; `gem_value` igual a
  `flat_value`/`per_level_value × (refino − 9)` em **todos** os refinos 10..=15; `None` abaixo do
  mínimo (sem faixa inventada); Anct em +9 aplicando o valor do seed enquanto o comum no mesmo
  refino recebe `GemRefineTooLow`; troca de efeito (drop → absorção em +12) gravando o novo efeito
  e publicando `GemEffectChanged` com `from`/`to` do catálogo, um evento por troca; EXP fixa
  (`flat 2`, `per_level 0`) para pegar troca indevida de escala.
- **Fronteira preservada:** nenhuma dependência nova — a leitura do seed é de compilação, dentro de
  `tests/`; `wyd-core` continua sem conhecer `wyd-db` e sem I/O no Hot Path (invariante 1).
- **Modelagem:** nada de schema novo. `refine_gem_effect` (migração 0009) + seed (0022) já
  publicam efeito, valor fixo, valor por nível, faixa e Anct; as colunas de gema em
  `character_items` já existiam. Criar tabela nova aqui seria órfã.
- **Docs:** ADR-0012 ganhou a seção "Prova contra o dado real", com a tabela do que cada caso
  garante e a explicação de por que a leitura fica em `tests/`.
- **Gates:** `cargo test --workspace` 0 (27 alvos, era 26) · `clippy -D warnings` 0 ·
  `fmt --check` 0 · `check-migrations` 0 (24 migrations, 59 tabelas, 140 INSERTs) ·
  `gen-catalogo-banco --check` 0 · `migrate-cycle` 0 (up/down/up, 24 migrations, 95,1 s,
  `content_version_max 7`) · `verify-schema` 0 (59 tabelas, 457 colunas, 59 PKs).

## 2026-11-08 — `down` da 0021 passou a ser seletivo (fim da limitação declarada)

- **Causa da perda de linhas:** a `0021` classificava 63 skills e o `down` fazia
  `UPDATE ... SET class_code = NULL, classification_source = 'unmapped'` em todas as 63. Dessas,
  **6 já vinham classificadas pela `0019`** (`Bênção_Divina`, `Chama_Resistente`, `Carga`,
  `Assalto`, `Clod_Attack`, `Another_Change`), incluindo dois conflitos de curadoria — reverter só
  a `0021` apagava dado de outra migração. O `migrate-cycle` não detectava porque desce **todas**
  as migrações (a `0019` também desce e a tabela esvazia por outro motivo), então a falha vivia
  registrada como limitação em vez de corrigida.
- **Correção:** o gerador (`tools/content/gen-skill-lineage.mjs`) passou a ler o estado anterior
  das próprias tuplas da `0019` e emite um `down` em duas partes — `UPDATE ... FROM (VALUES ...)`
  restaurando as 6 herdadas ao valor do seed e zerando para `'unmapped'` apenas as 57 que a `0021`
  classificou pela primeira vez. Nada é apagado; `primary_system` não é tocada. Decisão registrada
  em `docs/arquitetura/adr/0013-down-seletivo-de-seed.md` e regra adicionada ao README das
  migrations.
- **Prova no Postgres real (container efêmero):** após o `down` da `0021`, as 6 herdadas voltaram
  exatamente ao seed (`11 transknight/trans/buff`, `201 transknight/faith/passive`,
  `203 transknight/magic_sword/debuff`, `223 foema/special_magic/passive`,
  `225 beastmaster/elemental/summon`, `8 transknight/trans/damage`, todas
  `rule:name_confirmed`/80); as 57 novas ficaram `unmapped`/`NULL`; **nenhuma linha fora do
  conjunto da `0021` mudou**; classificadas 72 → 15 após o down (12 do seed + 3 da Kersef) → 72
  após novo `up`, com snapshot idêntico ao inicial.
- **Docs:** limitação removida do cabeçalho gerado e de `docs/wyd/classes.md`; números corrigidos
  em `docs/wyd/decisoes-servidor.md` (82 de 154 `unmapped`, 57 novas, cobertura 69 de 151).
- **Gates:** `cargo test --workspace` 0 (27 alvos) · `clippy -D warnings` 0 · `fmt --check` 0 ·
  `check-migrations` 0 (24 migrations, 59 tabelas, 140 INSERTs) · `gen-catalogo-banco --check` 0 ·
  `migrate-cycle` 0 (up→down→up, 24 migrations, 43,7 s, `skills_com_linhagem 72` nas duas
  direções) · `verify-schema` 0 (59 tabelas, 457 colunas, 59 PKs).

## 2026-11-08 — FASE-03-5 entregue: autoridade do servidor em `systems/` (movimento, combate, skill)

- **O que faltava:** o core validava inventário (gema, ADR-0012) mas não tinha **nenhum** ponto de
  validação de movimento, combate ou skill, nem cooldown: um pacote forjado não tinha onde ser
  recusado, e o cliente seria autor do resultado por omissão.
- **Decisão (ADR-0014):** a API dos sistemas recebe **intenção**, nunca resultado — não existe
  parâmetro de dano, de posição final nem de efeito. `systems/movement.rs::step` valida orçamento
  (`tiles_per_second × elapsed_ms`) e limites do mundo; `systems/combat.rs::attack` valida alvo
  vivo, alvo ≠ autor, alcance e cooldown e obtém o dano de `DamageModel` (trait implementado pela
  fórmula da FASE-10 — o core não inventa equação); `systems/skill.rs::cast` valida skill no
  catálogo, skill **aprendida** (`known`, vinda de `character_skills`), MP, cooldown e alcance.
  `MoveRules`/`AttackRules`/`SkillBook` são configuração explícita do servidor: nenhum número de
  balanceamento escondido na engine.
- **Estado no SoA:** `next_attack_ms` e `next_skill_ms` são componentes contíguos (`Vec<u64>`),
  zerados no spawn e na reciclagem de índice — sem `HashMap`, sem mutex, sem alocação no quente.
- **Recusa atômica:** ordem `viva → pré-condições → fila → evento → gravação` nos três caminhos;
  depois de recusa, posição/HP/MP/cooldown/fila ficam idênticos. Eventos novos `Moved`,
  `DamageDealt` e `SkillCast` carregam o que o servidor aceitou (dano e MP já aplicados), não a
  tentativa.
- **Erros tipados novos:** `InvalidMoveRules`, `NoMoveBudget`, `MoveTooFar`, `SelfAttack`,
  `TargetOutOfRange`, `AttackCooldown`, `TargetAlreadyDead`, `InvalidAttackRules`,
  `UnknownSkill`, `SkillNotLearned`, `NotEnoughMana`, `SkillCooldown`, `InvalidSkillRules`.
- **Docs:** `docs/arquitetura/autoridade-servidor.md` (mapa ação → onde é validada → o que é
  UNVERIFIED), ADR-0014 (+ adendo do `skill.rs`), `estrutura-e-padroes.md` §10 marcando
  `movement.rs`/`combat.rs`/`skill.rs` como executados. Movimento, dano, skill e aplicação de
  gema (ADR-0012) têm validação no core; o **efeito** de skill e a fórmula de dano seguem na
  FASE-10, e a troca de item/equipamento continua pendente (declarada na tabela da doc, com o
  dono claro: inventário de rede).
- **Gates:** `cargo test --workspace` 0 (**28 alvos**, era 27; core 71 testes: 48 lib + 10
  autoridade + 6 gem + 4 gema_seed + 3 determinismo) · `clippy -D warnings` 0 · `fmt --check` 0 ·
  `check-migrations` 0 (24 migrations, 59 tabelas, 140 INSERTs) · `gen-catalogo-banco --check` 0 ·
  `migrate-cycle` 0 (up→down→up, 24 migrations, 41,0 s, 72 nas duas direções) · `verify-schema` 0
  (59 tabelas, 457 colunas, 59 PKs).

## 2026-11-08 — Eixo inventário fechado: equipar/desequipar/mover com autoridade do servidor

- **O que estava aberto:** depois da entrega anterior, movimento/combate/skill e aplicação de gema
  tinham validação, mas **equipar, desequipar e mover item** não tinham sistema: um pacote de troca
  de item não tinha onde ser recusado e o cliente podia declarar o slot final.
- **Decisão (ADR-0015):** `systems/inventory.rs` com intenção mínima e uma porta
  `ItemRules { equip_slot, req_level, max_stack }` implementada pelo servidor a partir de
  `item_catalog` + `item_kind` (`0017`; `docs/wyd/itens.md` §5) — nenhuma tabela nova, nenhuma
  constante de balanceamento no core.
  - `equip` recebe **só o slot da mochila** e deriva o slot de equipamento do catálogo;
    `equip_into` existe para quando o protocolo trouxer slot declarado e o **confere**
    (`WrongEquipSlot`), nunca obedece.
  - Requisito de nível vem de `item_catalog.req_level` contra o `Stats.level` autoritativo.
  - `EquipmentRules { allow_swap }` é política explícita: com troca, o item anterior volta para o
    slot de onde o novo saiu; sem troca, slot ocupado é `EquipSlotOccupied` (nada é sobrescrito).
  - `unequip` devolve ao **primeiro** slot livre; mochila cheia é `BagFull` (sem destruir nem
    sobrescrever). `move_item` move para slot vazio, empilha mesmo item até `max_stack` (excedente
    fica na origem) e recusa item diferente (`BagSlotOccupied`) — sem troca silenciosa entre slots.
- **Recusa atômica:** mesma ordem `viva → pré-condições → fila → evento → mutação`; os testes
  comparam o `Inventory` inteiro antes/depois de cada recusa e exigem fila vazia. Eventos novos
  `ItemEquipped`, `ItemUnequipped`, `ItemMoved`. Erros novos: `NotEquippable`, `WrongEquipSlot`,
  `LevelTooLow`, `EquipSlotOccupied`, `BagFull`, `BagSlotOccupied`, `StackFull`, `SameSlot`.
- **Anti-dupe/perda:** `tests/inventario.rs` fecha o ciclo equipar → trocar → desequipar →
  reequipar e conta as unidades por `item_index` em equipamento+mochila (2 antes, 2 depois).
- **Pendente declarado, não mascarado:** `req_str/int/dex/con` do catálogo não são validados (o
  `Stats` não carrega os quatro — FASE-10), `max_stack` numérico segue `UNVERIFIED` (o catálogo só
  tem `stackable`), layout de rede do inventário é `UNVERIFIED` até a FASE-08-6 e o baú
  (`vault_items`) continua fora do core.
- **Docs:** ADR-0015 (novo), `autoridade-servidor.md` (três linhas de inventário + fonte de cada
  número + UNVERIFIED atualizados), `estrutura-e-padroes.md` §10 (`inventory.rs` executado),
  issue da FASE-03-5 com eixos fechados.
- **Gates:** `cargo test --workspace` 0 (**29 alvos**, era 28; core 88 testes: 56 lib + 10
  autoridade + 10 inventário + 6 gem + 4 gema_seed + 3 determinismo, com 1 alvo documental de 0) ·
  `clippy -D warnings` 0 · `fmt --check` 0 · `check-migrations` 0 (24 migrations, 59 tabelas, 140
  INSERTs) · `gen-catalogo-banco --check` 0 · `migrate-cycle` 0 (up→down→up, 24 migrations, 46,6 s,
  72 nas duas direções, `content_version_max 7`) · `verify-schema` 0 (59 tabelas, 457 colunas,
  59 PKs).

## 2026-11-08 — FASE-03-6 entregue: IA de NPC determinística no Hot Path

- **Código:** `crates/wyd-core/src/npc/{mod,state,behavior,brain,route,budget}.rs` e
  `crates/wyd-core/src/systems/npc.rs`; fachada em `src/npc/mod.rs`, sistema em
  `systems/mod.rs` e `lib.rs` (`pub mod npc`). `ComponentStore::kind_of` entrou como leitura.
- **Autoridade:** a IA **emite intenção** (`Idle`/`MoveTo`/`Attack`/`Flee`/`Return`) e não
  escreve posição/HP; `tests/npc.rs` prova que destino forjado é recusado por `step`
  (`NoMoveBudget`/`MoveTooFar`) com posição e fila intactas, e que decidir não publica evento.
- **Determinismo:** decisão é função pura (`brain::decide`), cadência `tick % 4 == entity % 4`
  com `DecisionBudget`, RNG `xorshift64` por `(world_seed, entity)`, ameaça com teto 8 e
  desempate pelo menor `entity`; golden FNV-1a de 24 ticks em `tests/npc.rs`.
- **Regras fora da engine:** `NpcProfile` (aggro/leash/ataque/fuga/patrão) é parâmetro
  validado no boot (`InvalidNpcProfile`); `mob_template_stat` vazia deixou de bloquear a fase.
- **Banco:** nenhuma migration nova — o perfil não é lido do schema ainda (UNVERIFIED).
- **Docs:** ADR-0016 (nova), `ia-bots.md` (camada A confirmada; lacunas reescritas),
  `07-ia-e-bots.md` (estado e cadência `% 4` corrigida), `estrutura-e-padroes.md` §10
  (`npc/` e `systems/npc.rs` como executados), índice e issue da FASE-03-6 fechados.
- **Medição:** `cargo test --workspace` 0 (30 alvos; `wyd-core` 76 lib + 12 `npc` + 10
  `autoridade` + 9 `inventario` + 6 `gem` + 4 `gema_seed` + 3 determinismo); `clippy` 0;
  `fmt --check` 0; `check-migrations` 0 (24/59/140); `gen-catalogo-banco --check` 0;
  `verify-schema` 0 (59/457/59); `migrate-cycle` 0 (48,1 s).
- **Pendente declarado:** bench de custo da IA (≤10% do tick com 500 NPCs), `route_type` do
  generator, `npc_behavior` no banco — todos UNVERIFIED na ADR-0016.

## 2026-11-08 — FASE-03-6: custo da IA medido (`benches/tick.rs`)

- **Bench executável:** `crates/wyd-core/benches/tick.rs` (`harness = false`, release, sem
  dependência nova — `criterion` não está no lock e o build é offline). 500 NPCs + 50 jogadores,
  400 ticks medidos, roda a cadeia real `systems::npc::decide` → `brain::decide` → `budget`.
- **Números (5 execuções):** média **88,6–204,5 µs/tick**, p99 **119,4–367,8 µs**,
  **0,18%–0,41%** do orçamento de 50 ms (pior p99 0,74%), **0,71–1,64 µs por decisão**,
  50.000 decisões (125/tick = 5 Hz). Sem jogador nenhum (piso de percepção): 12,5–31,8 µs/tick
  (0,025%–0,064%). A variação é ruído do host Windows compartilhado, não do subsistema. O bench
  falha (exit 1) acima de 10%. Decisão: **não** afrouxar `DecisionBudget` (133/tick) nem a
  cadência — a folga mínima de ~13× é margem para mundo maior, não convite a decidir mais.
- **Ledger:** o próximo ⬜ na ordem é a `FASE-05-1` (pipeline CI); verificado na árvore que não é
  acionável sem ferramenta/decisão do dono — `cargo audit` e `cargo deny` não estão instalados
  (`no such command`) e o passo 4 exige advisory vulnerável de propósito. Registrado no próprio
  issue, sem mascarar.
- **Docs:** ADR-0016 (adendo com a tabela medida), `ia-bots.md` §3, `07-ia-e-bots.md` §10/§11,
  `estrutura-e-padroes.md` §10 (`benches/tick.rs` executado), issue `FASE-03-6` (aceite de custo
  fechado) e correção da contagem de `tests/inventario.rs` (9, não 10).

## 2026-11-08 — FASE-05-1 entregue: CI com bench, audit e deny

- **Ferramentas:** instaladas no host para verificação real — `cargo-audit 0.22.2` e
  `cargo-deny 0.20.2` (o CI instala as mesmas por `taiki-e/install-action`). O bloqueio
  registrado na passada anterior (ferramenta ausente) deixou de valer.
- **Pipeline:** `.github/workflows/rust.yml` ganhou `--all-targets` no clippy/teste, job de
  bench (roda o bench da IA, que falha acima de 10% do tick, e o de limites) e job
  `supply-chain` (`cargo audit` + `cargo deny check`). `deny.toml` criado; `Makefile` com
  `test-all`, `bench-build`, `deny` e `ci` = `check test-all bench-build audit deny`.
- **Benches por crate crítica:** `crates/wyd-core/benches/tick.rs` (IA, 500 NPCs + 50
  jogadores) e `crates/wyd-net/benches/limits.rs` (balde e porte), ambos `harness = false`,
  sem dependência nova. `cargo bench --workspace --no-run` verde.
- **Medição:** IA 205,86 µs/tick (p99 420,10) = 0,412% do orçamento de 50 ms (p99 0,840%);
  sem jogador 29,14 µs (0,058%). Transporte: `TokenBucket` 8,1 ns/op e
  `ConnectionGate` 234,3 ns por handshake (execuções anteriores: 1,4 ns e 595 ns — a faixa
  está no baseline de `02-execucao.md` §4).
- **Supply-chain:** `cargo audit` exit 0 (7 crate dependencies, 1422 advisories);
  `cargo deny check` exit 0 (advisories ok, bans ok, licenses ok, sources ok; só warnings
  `license-not-encountered`). Prova de falha: lockfile descartável com `time 0.1.44` →
  `cargo audit` exit 1 com `RUSTSEC-2020-0071`, diretório removido em seguida.
- **Higiene de manifesto:** `publish = false` nas 7 crates (código do jogo não é publicado) e
  `version = "0.1.0"` nas dependências internas por path, o que tirou os erros de wildcard.
- **Gates desta passada:** `cargo test --workspace` 0 (30 alvos), `--all-targets` 0 (23),
  `clippy --all-targets` 0, `fmt --check` 0, `check-migrations` 0 (24/59/140),
  `gen-catalogo-banco --check` 0, `verify-schema` 0 (59/457/59), `migrate-cycle` 0 (50,9 s),
  `cargo audit` 0, `cargo deny check` 0.
- **Aberto:** paridade de runner (o workflow roda em `ubuntu-24.04`, a verificação foi no host
  Windows), baseline de bench sem histórico anterior para comparar tendência, e
  `multiple-versions = warn` que a primeira dependência externa pode transformar em ADR.

## 2026-11-08 — FASE-05-2 entregue: release LTO + imagem mínima (distroless)

- **Perfil de release:** `[profile.release]` no `Cargo.toml` do workspace com `lto = "thin"`,
  `codegen-units = 1`, `strip = "symbols"` e `panic = "abort"` (ADR-0017). Medido no mesmo
  binário: **323.072 → 296.448 bytes** (−26.624, −8,2%); `cargo build --release --workspace`
  exit 0 e os bins executam (`check` de manifesto real sai 0). O bloco `[profile.bench]` foi
  removido porque o Cargo ignora `panic` nesse perfil e só gerava warning de manifesto.
- **Imagem:** `docker/Dockerfile.launcher` reescrito em multi-stage real — builder
  `rust:1-bookworm` com cache BuildKit e runtime `gcr.io/distroless/cc-debian12:nonroot`
  (glibc para binário gnu, **sem shell, sem apt, sem toolchain**), usuário `nonroot`,
  `ENTRYPOINT` do launcher e labels OCI de versão/commit. `docker build` exit 0;
  `docker image inspect` → **37.847.971 bytes (36,1 MiB)**, 20 camadas, `User=nonroot:nonroot`.
- **Smoke no container:** `docker create` + `docker cp target/smoke/. :/jogo/` +
  `docker start -a` → `OK jogo.txt` / `manifesto build 1: 1 íntegros, 0 com problema`,
  **exit 0**; `--help` imprime o uso (exit 2, contrato do binário). `docker export` da imagem
  não contém `/bin/sh`, `bash`, `rustc`, `cargo`, `gcc` nem `apt`/`dpkg`.
- **Higiene de build:** `.dockerignore` criado (fora `target/`, `client/`, `.git`,
  `node_modules`, `.next`), então o contexto deixa de arrastar artefatos locais.
- **Ferramental:** `Makefile` ganhou `release`, `docker` e `docker-smoke`; o workflow ganhou o
  job `release` (`cargo build --release --workspace` + `docker build` + `docker image inspect`
  + `docker build --check` do portal) e `docker/**` entrou no gatilho. `deny.toml` não mudou:
  a imagem não adiciona dependência de crate.
- **Bench (sem regressão contra a baseline da 05-1):** IA de NPC **189,74 µs/tick** (p99 397,50)
  contra 205,86 (p99 420,10) — melhora de ~8%; lote de transporte `TokenBucket` **2,8 ns/op** e
  `ConnectionGate` **100,8 ns/handshake** contra 8,1 e 234,3.
- **Gates desta passada (árvore final):** `cargo test --workspace` 0 (30 alvos),
  `clippy --all-targets -D warnings` 0, `fmt --all --check` 0, `check-migrations` 0
  (24/59/140), `gen-catalogo-banco --check` 0, `verify-schema` 0 (59/457/59),
  `migrate-cycle` 0 (49,3 s), `cargo audit` 0, `cargo deny check` 0.
- **Aberto:** build da imagem verificado no host (o job de CI roda em `ubuntu-24.04`);
  imagem do portal só teve sintaxe validada (`docker build --check`), não build completo
  (`npm ci` é do artefato do portal, fora desta issue); `WYD_VERSION`/`WYD_COMMIT` ainda são
  build-args locais, sem tag de release automatizada; sem assinatura/attestation de imagem.

## 2026-11-08 — Refino +0..+15 auditado contra público e contra o legado

- **Documento:** `docs/wyd/sistemas/refino-auditoria.md` — escada passo a passo, divergências
  D-R1..D-R7, backlog de port com fonte (`_MSG_UseItem.cpp`, `_MSG_CombineItem*.cpp`,
  `tmserver/internal/{refine,combine,content}`) e o que segue UNVERIFIED.
- **Erro encontrado e corrigido:** `sanc_rate` guardava só as 28 linhas do `SancRate.txt`, mas
  o loader do legado aplica o arquivo **sobre o default compilado** (`g_pSancRate`) e lê em
  `nível + 1`. Faltavam 8 células (PO 6–11, PL 10–11 = 0): sem elas, +5→+6 (Ori) e +9→+10
  (Lac) ficariam sem taxa. Migração `0026_seed_sanc_rate_defaults.sql` → **36 células**,
  `SCHEMA_VERSION` **26**, emenda na ADR-0019.
- **Correção de leitura:** as células Âmago 10/11 (5%) não são a taxa do +10→+11 —
  `BASE_GetSuccessRate` curto-circuita para 15% (100% no Âmago).
- **`wyd-sim`:** `formula/refine.rs` passou de 9 passos para os **15** (com família
  `dust`/`compose`), e ganhou o caminho de pó do legado: `Powder`, `RefinePolicy` (6/9/10/11),
  piedade (peso `5,5,5,5,4,4,3,3,2,1,0`, 3 de 4 falhas, teto 20, Ori só até +5), rolagem
  inclusiva `rand()%100`, falha que preserva o item (destruição só no item selado +7..+9),
  bênção (1/1/2/3/4/5/6/7/8/10%), Anct nas **duas** réguas (legado 1% + 2/4/10; publicado 5% +
  2/4/10/12) e +12..+15 (4/3/2/1% + 1%/pedra + bênçãos, falha destrói ou rebaixa).
- **Paridade com o dado:** `crates/wyd-sim/tests/refino_seed.rs` lê `0022`/`0025`/`0026` por
  `include_str!` e confere 15 passos, as 36 células, a leitura em `nível+1` e os passos
  especiais fora da tabela.
- **Gates:** `cargo test -p wyd-sim` 0, `cargo test --workspace` 0,
  `clippy --workspace --all-targets -D warnings` 0, `fmt --all --check` 0,
  `check-migrations` 0 (26 migrations), `gen-catalogo-banco --check` 0, `verify-schema` 0.
- **Aberto (por desenho):** fiação de jogo do refino (comando do cliente, consumo, evento) e
  o port dos NPCs de composição (Ailyn, Anct, Odin, Tiny, Agatha, Shany, Alquimia, Lindy,
  Ehre, Extracao) — inventário em `refino-auditoria.md` §4.
