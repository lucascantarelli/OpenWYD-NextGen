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

## Fase 3 — engine (`wyd-core`)

| Issue | Status | Evidência / observação |
|---|---|---|
| FASE-03-1 tick | ⬜ | Base: `crates/wyd-core/src/lib.rs` (`TICK_HZ=20`). |
| FASE-03-2 ECS | ⬜ | Depende de ADR de ECS (FASE-13-2) + 03-1. |
| FASE-03-3 spatial grid | ⬜ | Depende de 03-1. |
| FASE-03-4 dados estáticos | ⬜ | Depende de FASE-04 (fonte de dados). |
| FASE-03-5 autoridade do servidor | ⬜ | Depende de 03-2, 03-3, 03-4. |

## Fase 4 — persistência (`wyd-db`)

| Issue | Status | Evidência / observação |
|---|---|---|
| FASE-04-1 modelagem | ⬜ | Referência: `.deprecated-go/docs/analysis/banco-de-dados.md`. |
| FASE-04-2 migrations | ⬜ | Depende de 04-1 + ADR de ferramenta. |
| FASE-04-3 persistência async | ⬜ | Depende de 04-2 + ADR de runtime async. |

## Fases 5–15

| Issue | Status | Observação |
|---|---|---|
| FASE-05-1 pipeline CI | ⬜ | Base: `.github/workflows/rust.yml` + `Makefile` (`check`, `ci`, `audit`, `bench`). |
| FASE-05-2 release/docker | ⬜ | Base: `docker/Dockerfile.rust`. |
| FASE-06-1 telemetria | ⬜ | Depende de FASE-03. |
| FASE-06-2 hardening | ⬜ | Depende de FASE-08. |
| FASE-07-1 gateway | ⬜ | Depende de 04-3. |
| FASE-07-2 painel usuário | ⬜ | Depende de 07-1. |
| FASE-07-3 painel admin | ⬜ | Depende de 07-1. |
| FASE-07-4 E2E Playwright | ⬜ | Depende de 07-2, 07-3. |
| FASE-08-1 tabela do protocolo | ⬜ | Fonte: `.deprecated-go/docs/migration/`. |
| FASE-08-2 parsers | ⬜ | Depende de 08-1 + ADR de serialização. |
| FASE-08-3 launcher | ⬜ | Base: `crates/wyd-launcher/src/main.rs`. |
| FASE-09-1 auditoria de skills | ⬜ | Insumo: FASE-01-3. |
| FASE-09-2 skills sucessoras | ⬜ | Depende de 09-1. |
| FASE-10-1 equações | ⬜ | Depende de 03-5. |
| FASE-10-2 velocidade | ⬜ | Depende de 10-1. |
| FASE-10-3 laboratório | ⬜ | Depende de 10-1, 10-2. |
| FASE-11-1 guia + manuais | ⬜ | Pastas `docs/manual-*/` criadas. |
| FASE-11-2 C4 + pacotes | ⬜ | Depende de 08-1. |
| FASE-12-1 dashboards | ⬜ | Depende de 06-1. |
| FASE-13-1 rito ADR | ✅ | `docs/arquitetura/adr/0000-template.md` criado (estado, contexto, decisão, consequências, alternativas, rastreabilidade); `0001` promovida a **aceita** no formato do template. |
| FASE-13-2 primeiras ADRs | ⬜ | Depende de 13-1. |
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

## Pendências conhecidas

- Desinstalar Go e remover `GOPATH`/`GOROOT` (ação elevada do dono).
- Verificar evidência de implementação das 54 issues em `done/` (Go).
- Classificar as 16 ocorrências de débito no C++ legado.
