# Programa pré-release: as 15 fases e a meta de bots com dono

Mapa dos dois documentos de missão (`pre-release.md` e `pre-release-bots.md`) para issues concretas.
Toda fase tem **onde mora** e **quem executa**; fase sem issue é fase sem dono, e foi exatamente esse o
buraco que a auditoria de 06/11 encontrou em matemática, segurança e bots.

Leia junto com [`AUDITORIA-PRE-RELEASE-2026-11-06.md`](AUDITORIA-PRE-RELEASE-2026-11-06.md) (o que foi
medido) e [`index.md`](index.md) (a fila viva).

## Ordem vigente

A ordem estratégica do documento de missão (§16) é preservada, com uma correção: a fila de evidência
humana (10 issues em `test/`) anda em paralelo, porque ela já está paga e só espera o cliente Windows.

| # | Onda | Conteúdo | Estado em 06/11 |
|---:|---|---|---|
| 0 | Auditoria | mapear, medir, reorganizar | ✅ esta entrega |
| 1 | Evidência | fechar o que só falta cliente real | 🔁 build/vet/suíte medidos em 06/11 (§11 da auditoria: 0 falha; 47 skips justificados após a onda 2); issue-08/09 regularizadas em `test/`; o restante depende de sessão humana |
| 2 | Plataforma | CI, segurança, banco, conteúdo | 🔁 segurança medida ponta a ponta (gitleaks + govulncheck + npm audit + middleware) com `check-secrets` ampliado, e issue 59 com fatia entregue (6 tabelas, parsers, importador, golden); escrita no banco vivo em `test/issue-59.md` |
| 3 | Produto | portal, bots com tráfego real, cliente | ⏳ |
| 4 | Conhecimento | matemática, dashboards, release | ⏳ |

## FASE 1 — Auditoria e organização do roadmap

| Entrega | Onde | Estado |
|---|---|---|
| Diagnóstico, inventário, duplicidades, lacunas, prioridades, plano | [`AUDITORIA-PRE-RELEASE-2026-11-06.md`](AUDITORIA-PRE-RELEASE-2026-11-06.md) | ✅ |
| Issues de revisão contínua | REVIEW-12 … REVIEW-19 | ✅ criadas |
| Skills atualizadas | `wyd-testing-bots` aponta para o pacote real de bots | ✅ parcial (2 pendências em REVIEW-19) |

## FASE 2 — Execução do roadmap

Método já existente e mantido: uma issue por vez, evidência antes de fechar, revisão consolidada a cada
10 fechamentos (contador em **2 de 10**, ver `REVIEW-11-2026-11-06.md`).

Fila de evidência humana em 06/11: **12 issues** em `roadmap/test/`, incluindo
[issue-08](test/issue-08.md) (criação nas 4 classes provada; falta a conferência visual) e
[issue-09](test/issue-09.md) (inventário/gold/última cidade já documentados no `SESSION-PRIMER`;
falta anexar SQL + restart + cargo/affects). Nenhuma delas
depende de código novo.

## FASE 3 — Servidor

| Frente | Issue | Estado |
|---|---|---|
| Wiring dos `main.go` (`newApp`) | REFACTOR-01 | ✅ 06/11: `config` + `newApp` + `run` em `app.go`; mains de 33–56 linhas; testes por bufconn sem porta/DSN |
| Comentários/docstrings em português | REFACTOR-03 | 🔁 13 diretórios migrados + gate; placar 06/11: 12.754 violações restantes (handler 6.095) |
| Código legado e caminhos duplicados | REVIEW-12 | 🔁 contínua |
| Conteúdo em arquivo × banco | 50, 59, 60 | ⏳ (62 e 57 fechadas) |
| Arquitetura e concorrência | REVIEW-12 | 🔁 contínua |

## FASE 4 — Banco de dados

| Frente | Issue | Estado |
|---|---|---|
| Roles, restore, índices | DB-01, DB-02 | ✅ fechadas |
| Baseline e versão de schema | issue-58 | ✅ fechada |
| Conteúdo restante | 50, 59 | 🟡 59 entregue até a fronteira do banco (dry-run 5/318/1/26/64/52) e `worldatlas` já lê `region`; 50 com proveniência por hash; falta `-content-source=db` e a escrita no Postgres vivo |
| Mapas: banco ou asset | 60 | ⏳ |
| Revisão permanente | REVIEW-13 | 🔁 contínua |
| Enriquecimento (item/skill/NPC/quest) | 33, 35, 46 | ⏳ / 🟡 |

## FASE 5 — DevOps e CI/CD

| Frente | Issue | Estado |
|---|---|---|
| Workflows reutilizáveis + `make ci` | CI-01 | ✅ |
| Banco efêmero no pipeline | CI-02 | ✅ |
| Boot completo + login e2e no CI | CI-03 | ⏳ (mapas fora do git) |
| CI completo (race, cobertura, artefatos) | issue-55 | ⏳ |
| Núcleo de bots no CI | BOT-08 | ⏳ |
| Revisão permanente | REVIEW-14 | 🔁 contínua |

## FASE 6 — Observabilidade e segurança

| Frente | Issue | Estado |
|---|---|---|
| `/metrics`, `trace_id`, alertas | OBS-01..04 | ✅ fechadas |
| Logs por requisição (`logpipe`/`/logs`) | issue-40 | ✅ |
| Varredura de segurança do domínio | REVIEW-15 | 🔁 criada |
| `db-roles` × migrations | DB-01 + REVIEW-13 | ✅ / 🔁 |
| Dashboard do projeto (roadmap, CI, cobertura) | REVIEW-14 | ⏳ |

## FASE 7 — Portal

| Frente | Issue | Estado |
|---|---|---|
| Admin (contas, conteúdo, drops, eventos, donate, logs, atlas) | 22–32, 34, 36, 40–42, 45, 47 | ✅ maioria |
| Fila de evidência in-game | `test/` 24, 25, 34, 36, 43, 44, 45, 46, 48 | 🟡 |
| Público × administrativo | REVIEW-08, PORTAL-01 | 🔁 / ✅ |
| e2e com Playwright | TEST-03 | ⏳ |
| Feedback do jogador | issue-53 | ⏳ |
| Identidade visual e assets | REVIEW-16 | 🔁 contínua |

## FASE 8 — Cliente e engenharia reversa

| Frente | Issue | Estado |
|---|---|---|
| Auditoria do legado C++ | REVIEW-07 | ✅ |
| `ClientPatch_v7662` | CANAL-03, `docs/analysis/legado-clientpatch.md` | ⏳ |
| Offsets e assets do build 12000 | issue-52 | ⏳ |
| Launcher | issue-51 | ⏳ |
| Revisão permanente | REVIEW-17 | 🔁 criada |

## FASE 9 — IA, Freebuff, skills e agentes

| Frente | Issue | Estado |
|---|---|---|
| `AGENTS.md` + skills por domínio | issue-14 (`done`) | ✅ |
| Limpeza de arquivos de IA | issue-56, REVIEW-10 | ✅ / 🔁 |
| Skills que ensinam comando errado | REVIEW-19 | 🔁 contínua |
| LLM opcional para bots | BOT-07 | ⏳ (desligado por padrão) |

## FASE 10 — Matemática, bug hunting e balanceamento

| Frente | Issue | Estado |
|---|---|---|
| Catálogo de atributos e fórmulas com nível de confiança | MATH-01 | ⏳ |
| Velocidade, hits e DPS; simulador | MATH-02 | ⏳ |
| Curva de EXP e rebalanceamento | issue-35 | ⏳ |
| Bug hunter em jogo pelos bots | BOT-03, BOT-05 | ⏳ |
| Revisão permanente | REVIEW-18 | 🔁 criada |

## FASE 11 — Documentação e padronização

| Frente | Issue | Estado |
|---|---|---|
| Site mkdocs, páginas de sistema, ADRs, `/guias` | DOC-01 | ✅ |
| ADR do laboratório de bots | [ADR 0007](../docs/adr/0007-laboratorio-de-bots.md) | ✅ |
| Página de matemática | MATH-01 | ⏳ |
| Runbooks pendentes | issue-54 | ⏳ (escopo reduzido) |

## FASE 12 — Dashboard e visualização

| Frente | Issue | Estado |
|---|---|---|
| Infra (Prometheus/Grafana, 7 alvos, 4 alertas) | OBS-03 | ✅ |
| Dashboard do laboratório de bots | BOT-08 | ⏳ |
| Dashboard de projeto (roadmap/CI/cobertura) | REVIEW-14 | ⏳ |
| Portal (faturamento, drops, logs, atlas) | 31, 26, 40, 41 | ✅ |

## FASE 13 — Propostas técnicas e ADRs

ADRs existentes: 0001 Go, 0002 goroutine única, 0003 portal BFF, 0004 banco como fonte de conteúdo,
0005 delivery queue, 0006 arquivamento do legado C++, 0007 laboratório de bots.
Propostas de alto risco (protocolo, auth, dados) **não** são executadas sem aprovação; ficam registradas
em issue com alternativas e consequências.

## FASE 14 — Pesquisa contínua

Pesquisa externa só entra como nível **C (hipótese)** até ser medida (regra em REVIEW-18). Fontes
históricas de WYD/WYD2 ainda não foram levantadas de forma sistemática: é parte de MATH-01/MATH-02.

## FASE 15 — Auditoria final e release

| Item | Estado |
|---|---|
| Auditoria técnica por domínio | 🔁 REVIEW-12..19 |
| Validação funcional (login, combate, loja, refino) | parcial: evidência in-game pendente (`test/`) |
| Validação de segurança | ⏳ REVIEW-15 |
| Release engineering (versão, changelog, artefatos, rollback) | ⏳ sem issue única — depende de CI-03 e issue-55 |
| Relatório final | ⏳ ao fim da onda 4 |

## Meta de bots: etapas do `pre-release-bots.md` → issues

| Etapa do documento | Issue | Estado |
|---|---|---|
| 1 — Pesquisa e arquitetura (ADR, limites) | [BOT-01](done/BOT-01.md) | ✅ |
| 2 — Cliente headless mínimo | [BOT-02](done/BOT-02.md) | ✅ |
| 3 — Motor de bots (FSM, perfis, ações, erros) | [BOT-02](done/BOT-02.md) | ✅ (ações `sim_*`) |
| 4 — Navegação e combate | [BOT-03](issues/BOT-03.md), [BOT-04](issues/BOT-04.md) | ⏳ |
| 5 — Orquestração, containers, cenários | [BOT-05](issues/BOT-05.md) | ⏳ |
| 6 — Stress testing e métricas | [BOT-05](issues/BOT-05.md), [BOT-08](issues/BOT-08.md) | ⏳ |
| 7 — Bug hunting e regressão | [BOT-03](issues/BOT-03.md), [BOT-08](issues/BOT-08.md) | ⏳ |
| 8 — Simulação econômica | [BOT-06](issues/BOT-06.md) | ⏳ |
| 9 — Comportamento social e LLM opcional | [BOT-07](issues/BOT-07.md) | ⏳ |
| 10 — Observabilidade e CI/CD | [BOT-08](issues/BOT-08.md) | ⏳ parcial (`bot-test` no Makefile) |
| 11 — Simulação matemática integrada | [MATH-02](issues/MATH-02.md) | ⏳ |

Critérios de aceitação da meta de bots (20 itens do documento) hoje atendidos: **1–3, 10 (parcial), 11,
12, 16, 19**. O restante depende das issues acima — não há atalho: cada ação nova exige o formato de
pacote provado antes.

## Regra de atualização

Ao fechar issue de qualquer fase, atualizar **na mesma entrega**: a tabela desta página (estado), a
linha em [`index.md`](index.md) e, se o comportamento observável mudou, `docs/migration/SESSION-PRIMER.md`.
