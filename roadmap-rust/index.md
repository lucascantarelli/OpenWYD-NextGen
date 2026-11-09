# Roadmap Rust — WYD (planejamento e execução)

> Fonte: `pre-release-rust.md` (§16 ordem estratégica + FASES 1–15) e `start-rust.md`
> (planejamento/limpeza/organização; código só após documentação).
> Planejamento Go histórico removido com o arquivo em 2026-11; auditoria preservada em
> `docs/manual-técnico/auditoria-roadmap.md`. Este diretório é o planejamento vigente.
> Esqueleto compilável já existente: workspace raiz + `crates/wyd-{protocol,core,net,db,admin,sim,launcher}`.
>
> **Execução em curso.** Painel vivo: `docs/rust/03-progresso.md`. Ciclo e gates:
> `docs/rust/02-execucao.md`. Status: ✅ concluída · 🚧 em curso · ⬜ pendente · ⛔ bloqueada.

## Ordem estratégica (§16)

1. Auditoria e reorganização do roadmap → `issues/FASE-01.md`
2. Contrato de protocolo/tipos (`wyd-protocol`) → `issues/FASE-08.md` (mapeamento) + `issues/FASE-03.md`
3. Engine core ECS + spatial grid (`wyd-core`) → `issues/FASE-03.md`
4. Persistência assíncrona (`wyd-db`/PostgreSQL) → `issues/FASE-04.md`
5. CI/CD Rust + observabilidade → `issues/FASE-05.md` + `issues/FASE-06.md`
6. Portal Next.js + launcher → `issues/FASE-07.md` + `issues/FASE-08.md`
7. Simulação matemática (`wyd-sim`) → `issues/FASE-10.md`
8. Validação final/carga/release → `issues/FASE-15.md`

Caminho crítico: `00 → 01 → 08-1 → 08-2 → 03 → 04 → 07` e `03 → 10 → 15`.

## Fases

| Issue | Fase | Título |
|---|---|---|
| `issues/FASE-01.md` | 1 | Auditoria do roadmap e organização das issues |
| `issues/FASE-02.md` | 2 | Execução controlada e evolução sequencial |
| `issues/FASE-03.md` | 3 | Engine `wyd-core` (tick, ECS, spatial grid) |
| `issues/FASE-04.md` | 4 | Banco de dados e persistência (`wyd-db`) |
| `issues/FASE-05.md` | 5 | DevOps e CI/CD (stack Rust) |
| `issues/FASE-06.md` | 6 | Observabilidade, monitoramento e segurança |
| `issues/FASE-07.md` | 7 | Portal web e ecossistema digital |
| `issues/FASE-08.md` | 8 | Engenharia reversa e cliente/launcher |
| `issues/FASE-09.md` | 9 | IA, skills e ferramentas auxiliares |
| `issues/FASE-10.md` | 10 | Matemática, bug hunting e balanceamento |
| `issues/FASE-11.md` | 11 | Documentação e padronização |
| `issues/FASE-12.md` | 12 | Dashboard e visualização |
| `issues/FASE-13.md` | 13 | ADRs (decisões de arquitetura) |
| `issues/FASE-14.md` | 14 | Pesquisa técnica contínua |
| `issues/FASE-15.md` | 15 | Auditoria final e release |

## Execução (issues específicas, uma por entrega)

| Issue | Fase | Entrega | Status |
|---|---|---|---|
| `issues/FASE-00-1-limpeza-ambiente.md` | 0 | Limpeza do ambiente Windows (Go → Rust) | 🚧 |
| `issues/FASE-00-2-vscode-rust.md` | 0 | VS Code reconfigurado para Rust | ✅ |
| `issues/FASE-00-3-branch-develop.md` | 0 | `develop` como branch principal | ✅ |
| `issues/FASE-01-1-auditoria-estado.md` | 1 | Inventário auditado das issues (Go + Rust) | 🚧 |
| `issues/FASE-01-2-varredura-dividas.md` | 1 | TODO/FIXME/HACK, código morto e lacunas | ✅ |
| `issues/FASE-01-3-inventario-skills.md` | 1 | Tabela de sucessão Go→Rust das skills | ✅ |
| `issues/FASE-02-1-ciclo-execucao.md` | 2 | `docs/rust/02-execucao.md` (ciclo obrigatório) | ✅ |
| `issues/FASE-02-2-revisao-consolidada.md` | 2 | Checklist de revisão a cada 10 issues | ✅ |
| `issues/FASE-02-3-ordem-priorizacao.md` | 2 | Grafo de dependências e caminho crítico | ✅ |
| `issues/FASE-02-4-estrutura-e-padroes.md` | 2 | Estrutura de módulos, nomenclatura e padrões (gate de código) | ✅ |
| `issues/FASE-02-5-visao-pastas.md` | 2 | Visão de pastas + mapa do cliente (gate de arquitetura) | ✅ |
| `issues/FASE-03-1-tick.md` | 3 | Tick determinístico 20–50 Hz | ✅ |
| `issues/FASE-03-2-ecs.md` | 3 | ECS: componentes contíguos + gema no inventário | ✅ |
| `issues/FASE-03-3-spatial-grid.md` | 3 | Spatial grid e interest management O(1) | ✅ |
| `issues/FASE-03-4-dados-estaticos.md` | 3 | Fim dos TXT/CSV; validação de schema no boot | ✅ |
| `issues/FASE-03-5-autoridade-servidor.md` | 3 | Autoridade server-side (física/dano/skills/inventário) | ✅ |
| `issues/FASE-03-6-ia-npc.md` | 3 | IA de NPC no Hot Path (FSM + utilidade determinística) | ✅ |
| `issues/FASE-04-1-modelagem.md` | 4 | Modelo relacional com FKs/CHECKs/índices | ✅ |
| `issues/FASE-04-2-migrations.md` | 4 | Migrations versionadas + seeds | ✅ |
| `issues/FASE-04-3-persistencia-async.md` | 4 | Pool async fora do tick | ✅ |
| `issues/FASE-05-1-pipeline-ci.md` | 5 | CI (fmt/clippy/test/bench/audit/deny) | ✅ |
| `issues/FASE-05-2-release-docker.md` | 5 | Release LTO + imagem mínima | ✅ |
| `issues/FASE-06-1-telemetria.md` | 6 | tracing + Prometheus no motor | ✅ |
| `issues/FASE-06-2-hardening.md` | 6 | DoS, rate limiting, sanitização | ✅ |
| `issues/FASE-07-1-gateway.md` | 7 | Transporte + contrato + binário `wyd-gateway` em Docker (ADR-0008; [ADR-0022](../docs/arquitetura/adr/0022-binarios-de-servico.md)) | 🚧 |
| `issues/FASE-07-2-painel-usuario.md` | 7 | Painel do jogador | ✅ |
| `issues/FASE-07-3-painel-admin.md` | 7 | Painel de staff auditável | ⬜ |
| `issues/FASE-07-4-e2e-playwright.md` | 7 | E2E (auth, loja, painel) | ⬜ |
| `issues/FASE-08-1-tabela-protocolo.md` | 8 | Tabela do protocolo com UNVERIFIED | ✅ |
| `issues/FASE-08-2-parsers.md` | 8 | `wyd-protocol` zero-copy + golden | ✅ |
| `issues/FASE-08-3-launcher.md` | 8 | Distribuição do launcher/patch (updater substituto) | ✅ |
| `issues/FASE-08-4-cliente-dossie.md` | 8 | Dossiê, inventário e baseline do cliente | ✅ |
| `issues/FASE-08-5-cliente-reverso-estatico.md` | 8 | RE estático profundo (`WYD.exe` 7662) | ✅ |
| `issues/FASE-08-6-cliente-reverso-dinamico.md` | 8 | Trace dinâmico e correlação de pacotes | 🚧 |
| `issues/FASE-08-7-cliente-customizacao.md` | 8 | Customização sustentável + manifesto do launcher | ✅ |
| `issues/FASE-08-8-launcher-gui-tauri.md` | 8 | GUI do launcher em Tauri v2 (notícias + Jogar) | 🚧 |
| `issues/FASE-09-1-auditoria-skills.md` | 9 | Auditoria das skills Go restantes | ✅ |
| `issues/FASE-09-2-criar-skills.md` | 9 | Skills sucessoras Rust | ✅ |
| `issues/FASE-10-1-equacoes.md` | 10 | Equações exatas de combate | 🚧 |
| `issues/FASE-10-2-velocidade.md` | 10 | Causa raiz da anomalia de velocidade | ⬜ |
| `issues/FASE-10-3-laboratorio.md` | 10 | Laboratório `wyd-sim` em lote | ⬜ |
| `issues/FASE-11-1-guia.md` | 11 | Guia de contribuição + manuais | ✅ |
| `issues/FASE-11-2-c4-pacotes.md` | 11 | C4 + specs de pacotes | ✅ |
| `issues/FASE-12-1-dashboards.md` | 12 | Dashboards + alertas + runbooks | ✅ |
| `issues/FASE-13-1-rito-adr.md` | 13 | Rito ADR + template | ✅ |
| `issues/FASE-13-2-primeiras-adrs.md` | 13 | ADRs (monolito, ECS, binário, async) | ✅ |
| `issues/FASE-14-1-pesquisa.md` | 14 | Notas de pesquisa + PoC | ✅ |
| `issues/FASE-15-1-suite-concorrencia.md` | 15 | Suite completa + concorrência | ⬜ |
| `issues/FASE-15-3-bots-carga.md` | 15 | Bots jogáveis e carga (`crates/wyd-bots`) | ⬜ |
| `issues/FASE-15-2-carga-banco-release.md` | 15 | Carga + banco + release + relatório | ⬜ |

**Mapa de funcionalidades (checklist viva do jogo):** o que cada sistema, item e skill precisa ter,
com estado e evidência por linha, fica em [`docs/rust/08-checklist-funcionalidades.md`](../docs/rust/08-checklist-funcionalidades.md).
O roadmap abaixo organiza o trabalho; o mapa diz o que já existe de verdade.

Critério global de conclusão (pre-release-rust §16): workspace estável, sem warnings de
clippy, sem data races, fórmulas críticas cobertas por testes e alta performance determinística.
