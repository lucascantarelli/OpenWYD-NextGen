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
| `issues/FASE-02-6-consistencia-docs.md` | 2 | Gate de consistência índice ↔ painel ↔ código (`make docs-check`) | ✅ |
| `issues/FASE-03-1-tick.md` | 3 | Tick determinístico 20–50 Hz | ✅ |
| `issues/FASE-03-2-ecs.md` | 3 | ECS: componentes contíguos + gema no inventário | ✅ |
| `issues/FASE-03-3-spatial-grid.md` | 3 | Spatial grid e interest management O(1) | ✅ |
| `issues/FASE-03-4-dados-estaticos.md` | 3 | Fim dos TXT/CSV; validação de schema no boot | ✅ |
| `issues/FASE-03-5-autoridade-servidor.md` | 3 | Autoridade server-side (física/dano/skills/inventário) | ✅ |
| `issues/FASE-03-6-ia-npc.md` | 3 | IA de NPC no Hot Path (FSM + utilidade determinística) | ✅ |
| `issues/FASE-03-7-motor-no-canal.md` | 3 | Motor no canal: tick real dentro do `wyd-server` | ✅ |
| `issues/FASE-03-8-entrada-no-mundo.md` | 3 | Entrada no mundo: spawn do personagem + save do estado vivo | ✅ |
| `issues/FASE-03-9-cena-autoritativa.md` | 3 | Cena autoritativa (`ESCENE_FIELD` 30000) | ✅ |
| `issues/FASE-03-10-cidadania-canal-verde.md` | 3 | Cidadania aplicada + papel do canal verde | ✅ |
| `issues/FASE-03-16-piedade-updateitem.md` | 3 | Piedade de refino persistida + `UpdateItem` de resposta | ✅ |
| `issues/FASE-03-17-entrada-no-mundo-fiel.md` | 3 | Entrada no mundo fiel ao legado (`ClientID`, `UpdateScore`, `CreateType`) | ✅ |
| `issues/FASE-03-18-campo-de-treinamento.md` | 3 | Campo de Treinamento (quest, chaves, buff, boss) | 🟡 |
| `issues/FASE-04-1-modelagem.md` | 4 | Modelo relacional com FKs/CHECKs/índices | ✅ |
| `issues/FASE-04-2-migrations.md` | 4 | Migrations versionadas + seeds | ✅ |
| `issues/FASE-04-3-persistencia-async.md` | 4 | Pool async fora do tick | ✅ |
| `issues/FASE-05-1-pipeline-ci.md` | 5 | CI (fmt/clippy/test/bench/audit/deny) | ✅ |
| `issues/FASE-05-2-release-docker.md` | 5 | Release LTO + imagem mínima | ✅ |
| `issues/FASE-06-1-telemetria.md` | 6 | tracing + Prometheus no motor | ✅ |
| `issues/FASE-06-2-hardening.md` | 6 | DoS, rate limiting, sanitização | ✅ |
| `issues/FASE-06-3-lotacao-manutencao.md` | 6 | Lotação e manutenção do canal (recusa tipada) | ✅ |
| `issues/FASE-06-4-catalogo-ameacas.md` | 6 | Catálogo de ameaças B1–B17 + §6 do `wyd2.md` | ⬜ |
| `issues/FASE-07-1-gateway.md` | 7 | Transporte + contrato + binário `wyd-gateway` em Docker (ADR-0008; [ADR-0022](../docs/arquitetura/adr/0022-binarios-de-servico.md)) | 🚧 |
| `issues/FASE-07-2-painel-usuario.md` | 7 | Painel do jogador | ✅ |
| `issues/FASE-07-3-painel-admin.md` | 7 | Painel de staff auditável | ✅ |
| `issues/FASE-07-4-e2e-playwright.md` | 7 | E2E (auth, painel, entrega idempotente) | ✅ |
| `issues/FASE-07-5-handlers-conteudo.md` | 7 | Handlers de leitura de conteúdo (npc/mob/item/efeito/mapa/teleporte) | ✅ |
| `issues/FASE-07-6-handlers-operacao.md` | 7 | Handlers de operação (rates/daily/donate/evento/conta) | ✅ |
| `issues/FASE-08-1-tabela-protocolo.md` | 8 | Tabela do protocolo com UNVERIFIED | ✅ |
| `issues/FASE-08-2-parsers.md` | 8 | `wyd-protocol` zero-copy + golden | ✅ |
| `issues/FASE-08-3-launcher.md` | 8 | Distribuição do launcher/patch (updater substituto) | ✅ |
| `issues/FASE-08-4-cliente-dossie.md` | 8 | Dossiê, inventário e baseline do cliente | ✅ |
| `issues/FASE-08-5-cliente-reverso-estatico.md` | 8 | RE estático profundo (`WYD.exe` 7662) | ✅ |
| `issues/FASE-08-6-cliente-reverso-dinamico.md` | 8 | Trace dinâmico e correlação de pacotes | 🚧 |
| `issues/FASE-08-7-cliente-customizacao.md` | 8 | Customização sustentável + manifesto do launcher | ✅ |
| `issues/FASE-08-8-launcher-gui-tauri.md` | 8 | GUI do launcher em Tauri v2 (notícias + Jogar) | 🚧 |
| `issues/FASE-08-9-logout-explicito.md` | 8 | Logout explícito (sair sem esperar timeout) | ✅ |
| `issues/FASE-08-10-trade-atomica.md` | 8 | Trade atômica por par (bloqueada em layout) | ⛔ |
| `issues/FASE-09-1-auditoria-skills.md` | 9 | Auditoria das skills Go restantes | ✅ |
| `issues/FASE-09-2-criar-skills.md` | 9 | Skills sucessoras Rust | ✅ |
| `issues/FASE-10-1-equacoes.md` | 10 | Equações exatas de combate | 🚧 |
| `issues/FASE-10-2-velocidade.md` | 10 | Causa raiz da anomalia de velocidade | ⬜ |
| `issues/FASE-10-3-laboratorio.md` | 10 | Laboratório `wyd-sim` em lote | ⬜ |
| `issues/FASE-10-4-exp-por-flags.md` | 10 | Árvore de EXP por flags (default inalterado) | ⬜ |
| `issues/FASE-11-1-guia.md` | 11 | Guia de contribuição + manuais | ✅ |
| `issues/FASE-11-2-c4-pacotes.md` | 11 | C4 + specs de pacotes | ✅ |
| `issues/FASE-12-1-dashboards.md` | 12 | Dashboards + alertas + runbooks | ✅ |
| `issues/FASE-13-1-rito-adr.md` | 13 | Rito ADR + template | ✅ |
| `issues/FASE-13-2-primeiras-adrs.md` | 13 | ADRs (monolito, ECS, binário, async) | ✅ |
| `issues/FASE-14-1-pesquisa.md` | 14 | Notas de pesquisa + PoC | ✅ |
| `issues/FASE-15-1-suite-concorrencia.md` | 15 | Suite completa + concorrência | ⬜ |
| `issues/FASE-15-3-bots-carga.md` | 15 | Bots jogáveis e carga (`crates/wyd-bots`) | ⬜ |
| `issues/FASE-15-4-bots-do-campo-de-treinamento.md` | 15 | Bots autônomos do Campo de Treinamento (4 raças) | 🟡 |
| `issues/FASE-15-2-carga-banco-release.md` | 15 | Carga + banco + release + relatório | ⬜ |

## Fila de entregas rápidas (2026-11-09)

Ordem por **tamanho** e por **desbloqueio**: cada linha entrega valor sozinha e não deixa a
seguinte esperando. A coluna “depende” só cita o que já está pronto ou o que vem antes na
mesma fila — nenhuma issue abaixo espera captura de cliente, decisão do dono ou serviço externo.

| # | Issue | Tamanho | Depende de | Entrega |
|---|---|---|---|---|
| 1 | `FASE-02-6` consistência de docs | **S** | — | gate que impede status divergente voltar |
| 2 | `FASE-06-3` lotação/manutenção | **S** | 06-2 ✅ | canal cheio e manutenção com recusa observável |
| 3 | `FASE-08-9` logout explícito | **S** | 08-2 ✅ | sessão liberada na hora, relogin imediato |
| 4 | `FASE-03-7` motor no canal | **M** | 03-6 ✅, 04-3 ✅ | tick real + métricas de motor verdadeiras |
| 5 | `FASE-07-5` handlers de conteúdo ✅ | **M** | 07-1 🚧 | leituras de conteúdo com dado real; mapa de atributos e carry seguem `501` (sem tabela) |
| 6 | `FASE-07-6` handlers de operação ✅ | **M** | 07-1 🚧, 07-3 ✅ | daily/donate/evento/contas/`grant_item` auditados; `501` só sem tabela |
| 7 | `FASE-03-8` entrada no mundo | **M** | 03-7 (fila) | personagem vira entidade e é salvo |
| 8 | `FASE-03-10` cidadania/canal verde ✅ | **M** | 03-7 ✅ | cidadania deixa de ser campo e vira regra |
| 9 | `FASE-03-9` cena autoritativa | **L** | 03-8 (fila) | primeiro mundo visível no cliente |

Depois da fila, o que resta aberto é: `FASE-08-8` (GUI Tauri, M), `FASE-10-1` (fechar a ordem
matemática, M), `FASE-10-2` (velocidade, M), `FASE-10-3` (laboratório em lote, M), `FASE-10-4`
(EXP por flags, M), `FASE-06-4` (catálogo de ameaças, S),
`FASE-15-3` (bots, M), `FASE-15-1` (suite/concorrência, L), `FASE-15-2` (carga/release, L),
`FASE-00-1` (desinstalar Go — **ação do dono**, exige elevação) e `FASE-01-1` (auditoria do
planejamento Go, valor histórico). `FASE-08-6` continua ⛔ por depender de VM + cliente em
execução e `FASE-08-10` (trade) herda o mesmo bloqueio; são os únicos bloqueios externos da árvore.

## Lotes paralelos

Duas frentes podem rodar ao mesmo tempo sem tocar os mesmos arquivos:

- **Motor** — 03-7 → 03-8 → 03-9 (+ 03-10 em paralelo com 03-8, no `wyd-sim`/`wyd-core`).
- **Cold Path** — 07-5 e 07-6 (gateway/DB) em paralelo com o motor; `02-6`, `06-3` e `08-9`
  são curtas e podem entrar entre qualquer uma das duas.

**Mapa de funcionalidades (checklist viva do jogo):** o que cada sistema, item e skill precisa ter,
com estado e evidência por linha, fica em [`docs/rust/08-checklist-funcionalidades.md`](../docs/rust/08-checklist-funcionalidades.md).
O roadmap abaixo organiza o trabalho; o mapa diz o que já existe de verdade.

Critério global de conclusão (pre-release-rust §16): workspace estável, sem warnings de
clippy, sem data races, fórmulas críticas cobertas por testes e alta performance determinística.

## Ondas 1–4 — curva de crescimento (2026-11-11)

Issues novas fora do padrão `FASE-NN-N` (não entram no gate `check-issue-status`):
`DOC-01`, `ROAD-01`, `PORTAL-01..05`, `LAUNCHER-01`, `BUG-01`, `BOTS-01` em `issues/`.

| Onda | Objetivo | Issues | Prova |
|---|---|---|---|
| 1 — base sólida | login, PIN, seleção, entrada no mundo, cena `ESCENE_FIELD`, logout explícito | `BUG-01` (reproduzir senha-sem-mensagem + char-no-mundo), FASE-08-2 ✅, 03-7/03-8/03-9/03-17 ✅, 06-2/06-3 ✅, 08-9 ✅ | conta admin: login → mundo → logout → relogin sem timeout (`login`, `selchar`, `cena_mobs`) |
| 2 — mundo jogável | caçar, upar, refinar com goldens | drops no inventário ✅, `SendItem` ✅, refino/EXP no fio ✅ | `campo_treinamento`, `config_gameplay` |
| 3 — operação | portal auditável que diagnostica o resto | `PORTAL-01` (monitor/logs/DB/alertas), `PORTAL-02` (contas/2FA/auditoria), `PORTAL-03` (premium/cash), `PORTAL-04` (catálogo/economia), `PORTAL-05` (config viva), 07-5/07-6 ✅ | `portal-test` + `/monitoramento` com fonte fora degradando sem 500 |
| 4 — escala | bots, carga, balanceamento, release | `BOTS-01`, `LAUNCHER-01`, 10-1 🚧, 10-2/10-3/10-4 ⬜, 15-1/15-2/15-3/15-4 | relatório carga p50/p99 + run id + seed |

Congeladas com critério explícito: `FASE-08-6` (captura ao vivo, exige VM+cliente) e
`FASE-08-10` (trade, layout `MSG_Trade` UNVERIFIED) — nenhuma onda espera por elas.
Docs: `DOC-01` (MkDocs `mkdocs.yml` + `docs/index.md`, `mkdocs build --strict` verde).
