# 01 — Inventário do projeto

> Medições de 2026-11-13 na árvore `docs/arquiva-devops-10` @ `9cfcaa2`. Tudo nesta página foi
> obtido por `ls`/`find`/`wc`/`grep` ou lido de documento citado; nada foi estimado.

## 1. Identidade e estado do repositório

| Item | Valor | Fonte |
|---|---|---|
| HEAD | `9cfcaa2` — “Arquiva a DEVOPS-10 com a v0.1.0 publicada” | `git log` |
| Branch atual | `docs/arquiva-devops-10` | `git branch` |
| Branches presentes | `develop`, `main`, `sync/main-para-develop`, `docs/arquiva-devops-10` | idem |
| Remotos | `origin/main`, `origin/develop`, `origin/docs/arquiva-devops-10` | idem |
| Árvore de trabalho | limpa, exceto o arquivo do prompt (não versionado) | `git status --porcelain` |
| Versão | `v0.1.0` tagueada em `main` (DEVOPS-10) | `roadmap-rust/index.md` |
| Toolchain | Rust pinado em **1.100.0** | `roadmap-rust/index.md` (DEVOPS-05), `rust-toolchain.toml` |
| PRs / contribuidores | **não disponível** nesta máquina (metadados Git ausentes no host) | — |

> **Atenção:** o commit analisado é de uma branch de arquivamento, não de `develop`. O conteúdo
> dos crates é o mesmo que `develop` publica no índice do roadmap; ainda assim, qualquer
> conclusão desta auditoria vale para **este** commit, não para o futuro de `develop`.

## 2. Mapa da raiz

| Caminho | Papel | Estado |
|---|---|---|
| `crates/` | 9 crates do servidor/engine/rede/db/admin/sim/launcher/bots | ativo |
| `apps/launcher-ui/` | janela Tauri do launcher | ativo (membro do workspace) |
| `wyd-portal/` | BFF Next.js sobre `wyd-admin` | ativo |
| `docs/` | 130+ arquivos de norma, cliente, rust, mundo do jogo | ativo |
| `roadmap-rust/` | índice, 27 issues ativas, 15 descritores de fase, changelog | ativo |
| `tools/` | `client/`, `content/`, `dev/`, `launcher/`, `protocol/` (Node) | ativo |
| `docker/` | Dockerfiles, compose base/prod/reaper, monitoramento | ativo |
| `client/` | cliente 7662 (**fora do git**) + assets | artefato local |
| `dist/`, `site/`, `target/`, `tmp/` | saída de build/site/temporários | gerado |
| `.agents/`, `.freebuff/` | skills e configuração de agente | projeto |
| `docs/manual-técnico/`, `docs/wyd/` | conhecimento de jogo e processo | ativo |

## 3. Crates — medido

| Crate | Arq. `src` | LOC `src` | Testes | Arq. em `tests/` | Papel declarado (`AGENTS.md` §1) |
|---|---:|---:|---:|---:|---|
| `wyd-protocol` | 23 | 3.692 | 58 | 3 | wire CPSock, opcodes, payloads, goldens |
| `wyd-core` | 30 | 7.743 | 165 | 9 | engine Hot Path: tick, ECS, combate, visibilidade |
| `wyd-net` | 7 | 833 | 21 | 2 | transporte, sessão, rate-limit, hardening |
| `wyd-db` | 24 | 7.081 | 33 | 6 | persistência, migrations, `delivery_queue` |
| `wyd-admin` | 33 | 10.001 | 112 | 8 | control plane Cold Path, binário `wyd-gateway` |
| `wyd-server` | 34 | 13.225 | 151 | 28 | composição do monólito; 1 processo por canal |
| `wyd-sim` | 22 | 3.799 | 120 | 3 | fórmulas e simulações determinísticas |
| `wyd-launcher` | 18 | 8.104 | 140 | 4 | distribuição/patch, `serverlist`, branding |
| `wyd-bots` | 11 | 1.197 | 10 | 1 | bots de QA/carga sem GUI |
| **Total** | **202** | **55.675** | **810** | **64** | + `apps/launcher-ui` |

Distribuição da massa: `wyd-server` + `wyd-admin` + `wyd-launcher` + `wyd-core` = **39.073 LOC
(70%)**. `wyd-net` (833) é o menor crate apesar de ser o que o cliente toca primeiro — sinal de
que o transporte mora em `wyd-server::cpsock`, não no crate de rede (ver `14-rust-architecture-audit.md`).

## 4. Banco e conteúdo

| Item | Valor | Fonte |
|---|---:|---|
| Migrations | 51 arquivos (`0001`…`0050` + `README.md`) | `ls crates/wyd-db/migrations` |
| Última migration | `0050_seed_correcao_catalogo_game12_15.sql` | idem |
| Domínios de migration | personagens, inventário, guildas, auditoria, fila de entrega, refino, quests, campo, canal, donate, chaos | nomes dos arquivos |
| Tabelas | 61 (doc) / 56 (legado) | `docs/arquitetura/contrato-persistencia.md` |
| Itens no catálogo | 3.220 (doc) | `docs/wyd/itens.md` via `fontes.md` |
| Sistemas classificados | 20 (`refino`, `replecao`, `mix`, `craft`, `drop`, `loja`, `donate`, `teleporte`, `armazem`, `combate`, `guilda`, `guerra_guilda`, `castelo`, `reino`, `quest`, `evento`, `mundo`, `progressao`, `skill`, `montaria`) | `docs/wyd/sistemas.md` |
| Slots de item no cliente | 6.500 índices em `ItemList.bin` | aritmética do tamanho (`04-client-analysis.md` §3) |

## 5. Portal e launcher

| Item | Estado | Evidência |
|---|---|---|
| BFF Next.js (`wyd-portal/`) | ativo; `app/`, `components/`, `lib/`, `server/`, `proxy.ts` | `ls wyd-portal/src` |
| Sessão httpOnly + `adminCall` | ✅ | `docs/rust/08-checklist-funcionalidades.md` 11.1–11.4 |
| E2E Playwright | 6/6 em três execuções (doc) | `docs/rust/evidencias/WEB-04-e2e-2026-11-12.md` |
| Launcher CLI | manifesto, patch atômico, Ed25519, `serverlist`, `branding`, `gui`, `retheme` | `docs/cliente/customizacao.md` §6 |
| GUI Tauri | 🚧 — `LauncherGUI.html` existe, shell Tauri não | checklist 11.8 |

## 6. Ferramentas (`tools/`)

| Ferramenta | Função | Fonte de verdade |
|---|---|---|
| `client/wyt.mjs` | `.wyt` → PNG/BMP, `--scan` | `docs/cliente/formatos.md` §1 |
| `client/serverlist.mjs` | gera/inspeciona `serverlist.bin` + `sn.bin` | `formatos.md` §3–§4 |
| `client/captura.mjs` | valida goldens C→S | `docs/cliente/pacotes.md` |
| `client/captura-live.mjs` | captura ao vivo (exige cliente aberto) | `docs/cliente/depuracao.md` |
| `client/trade-layout.mjs` | verifica layout candidato de `MSG_Trade` | `CLIENT-10` |
| `client/formats.mjs`, `pe.mjs`, `strings.mjs`, `extract-tables.mjs`, `map-skills.mjs`, `evidence.mjs`, `diff-captura.mjs` | inventário estático e evidência | `docs/cliente/inventario-gerado.md` |
| `launcher/sign-package.mjs` | assina pacote Ed25519 | `customizacao.md` §6 |
| `content/migrate-cycle.mjs` | up → down → up em Postgres efêmero | `Makefile` |
| `dev/check-docs-consistency.mjs`, `check-skills.mjs`, `check-versao.mjs`, `check-actions-pin.mjs`, `check-compose-hardening.mjs` | gates | `Makefile` (`make check`) |

## 7. Legado — onde ele realmente está

| Fonte | Local | Volume medido |
|---|---|---|
| C++ original (`TMSrv`, `DBSrv`, `BISrv`, `ClientPatch`, `CPSock`, `Basedef`) | **`origin/main`** (`Source/`) | 142 arquivos; 70.987 linhas |
| Reescrita Go anterior | **`origin/main`** (`tmserver/`, `internal/`, `dbserver/`, `binserver/`) | 484 arquivos; 99.606 linhas (51.582 sem testes) |
| Portal/web anterior | `origin/main` (`webserver/`, `api/`, `portal/`) | 292 arquivos |
| Referência externa | `github.com/Jean1dev/w2pp-OpenWYD` | Go, GPL-2.0, 672 commits, `pushed_at` 2026-10-29 |

**Nada do legado está na árvore de `develop`** — é decisão registrada (`AGENTS.md` §0: stack
Go/C++ removida em 2026-11). Comparar “local × remoto” exige `git show origin/main:`; esta
auditoria não fez checkout nem diff byte a byte (ver `02-legacy-analysis.md` §2).

## 8. O que **não** está versionado (por decisão)

| Item | Regra |
|---|---|
| `client/` (binários, DLLs, assets, `serverlist.bin` com IP) | `.gitignore`; dossiê documenta hash/offset, nunca conteúdo |
| `.env` | presente localmente, ignorado |
| `target/`, `dist/`, `site/`, `tmp/` | artefatos de build |
| dumps, `.pcap`, segredos | proibidos (`AGENTS.md` §7) |

## 9. Duplicidades e descontinuados

| Item | Situação |
|---|---|
| Stack Go/C++ | **removida** da árvore; vive só em `origin/main` |
| `docs/rust/09-duplicidades.md` | registro da dedup da documentação (PROC-01 ainda 🚧) |
| Portal anterior (Go + Next) | substituído por `wyd-portal/` + `wyd-admin` |
| `crates/wyd-launcher` × `WYDLauncher.exe` do cliente | convivem: o nosso é distribuição; o do cliente é updater legado (endpoint morto) |
| `tools/client/serverlist.mjs` × `wyd-launcher serverlist` | paridade de bytes é teste golden — duplicação **intencional e vigiada** |
