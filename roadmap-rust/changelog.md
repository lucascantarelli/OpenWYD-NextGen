# Changelog de issues concluídas

> Issues que cumpriram os critérios de aceite foram removidas de `roadmap-rust/issues/` e registradas aqui. O histórico completo continua no git (arquivos removidos) e o painel vivo é `docs/rust/03-progresso.md`.

## GAME-19 — Equações de combate: ordem, exemplo e testes

- **ID:** `GAME-19`
- **Concluída em:** 2026-11-12
- **Resumo:** os 9 sistemas de `crates/wyd-sim/src/formula/` e a ordem do golpe/tick em `scenario/combat.rs` já estavam codificados; faltava o exemplo numérico exigido pelo aceite. O documento ganhou a conta completa (`base 100 + 20 + 10`, skill 150%, buff 110%, crítico 200%, defesa 100/perfuração 30, resistência 25%, absorção 19 → **250**) e o teste `worked_example_matches_documented_chain` reproduz o mesmo resultado. As hipóteses D-02..D-04/D-06 continuam explicitamente marcadas; a ordem do legado será confirmada pelo trace da `CLIENT-06`, fora desta issue.
- **Evidência:** `docs/arquitetura/matematica-sim.md` §3.2.1; `crates/wyd-sim/src/formula/damage.rs`; `cargo test -p wyd-sim damage` → 13 passed; [`AUDIT-ISSUES-2026-11-12.md`](../docs/rust/evidencias/AUDIT-ISSUES-2026-11-12.md).

## WEB-01 — Gateway Next.js ↔ `wyd-admin`: transporte e contrato

- **ID:** `WEB-01`
- **Concluída em:** 2026-11-12
- **Resumo:** transporte HTTP/JSON, auth de staff, auditoria persistida, 44 métodos do portal com handler, `admin.pending=0` e binário `wyd-gateway` no compose. O único resíduo era a rota legada `daily_reward_admin/set_reward_item_enabled`, que o portal não declara mais e cuja tabela não tem coluna de liga/desliga; ela saiu de `SERVICOS` e agora responde 404, em vez de 501 permanente. Nenhuma chamada vigente do portal quebrou.
- **Evidência:** `crates/wyd-admin/src/transport/mod.rs`; `crates/wyd-admin/tests/contrato/operacao.rs`; `crates/wyd-admin/tests/contrato/transporte.rs`; `cargo test -p wyd-admin --test contrato_operacao` → 12 passed e `--test contrato_transporte` → 14 passed; [`AUDIT-ISSUES-2026-11-12.md`](../docs/rust/evidencias/AUDIT-ISSUES-2026-11-12.md).

## Auditoria de fechamento das issues — 2026-11-12

- **29 issues ativas conferidas** contra código/teste/doc; **2 fechadas** (`GAME-19`, `WEB-01`) e **27 mantidas** com o critério que falta nomeado na matriz.
- **Status corrigidos por evidência:** `PROC-01` ⬜→🚧 (inventário MkDocs/dedup existe; faltam fusões e build atual), `PROC-24` ⬜→🚧 (fila/grafo publicados; falta onda 1 visual e DoD por onda).
- **Dependências reconciliadas:** `BUG-01` e `BUG-02` apontam para `CLIENT-06`/`GAME-17` 🚧; `GAME-05` para GAME-02/03/04 ✅; `GAME-20/21` para `GAME-19` ✅; `GAME-23` para `QA-11` 🟡; `QA-09` para `QA-02` ✅; `WEB-07` para `WEB-01` ✅.
- **Fila:** `#01..#27`, com as duas posições fechadas removidas e as demais renumeradas; categorias agora **Jogo 10 / Portal 5**.
- **Evidência:** [`AUDIT-ISSUES-2026-11-12.md`](../docs/rust/evidencias/AUDIT-ISSUES-2026-11-12.md); `check-issue-status`, `docs-check`, `skills-check` e `make check` verdes na mesma execução.

## Auditoria de skills — segunda passada (2026-11-12)

- **Resumo:** `.agents/skills/` passou de **23 para 21** `SKILL.md`; `github-actions-docs` foi removida (skill genérica, só usada na FASE-05) e `solid-design-patterns` foi fundida em `software-architecture`. As 21 restantes mantêm frontmatter válido e corpo abaixo de 500 linhas; nenhuma skill nova foi criada.
- **Números:** `SKILL.md` 1.301 → 1.190 linhas e 82.932 → 77.252 bytes; arquivos totais 44 → 40; bytes totais 297.500 → 281.832. `AGENTS.md` ficou em 117 linhas / 6.919 bytes (só o comentário do gate mudou). Também saiu o arquivo órfão `writing-plans/plan-document-reviewer-prompt.md` (1.713 bytes).
- **Referências:** issues FASE-03, FASE-05, FASE-09, FASE-10, GAME-19 e PROC-01 corrigidas; roteador, `docs/manual-técnico/skills.md`, `docs/rust/00-visao-estrutura.md`, `docs/rust/03-progresso.md`, `docs/rust/04-revisao-roadmap.md` e `docs/arquitetura/estrutura-e-padroes.md` alinhados. Nenhuma issue cita skill inexistente.
- **Gate:** `tools/dev/check-skills.mjs` e alvo `skills-check`, incluídos no `make check`; valida frontmatter, nome/diretório, descrição, limite de 500 linhas, recursos citados e caminhos de skill em docs/issues.
- **Dedup de conteúdo:** 18 skills de domínio/processo reescritas para “Norma → decisão específica → prova”; DAG, árvores, patterns/SOLID, ER, taxonomia de testes, ordem de fórmulas, stack/portas e fluxo do launcher deixaram de ser copiados e passaram a linkar os docs normativos. `SKILL.md` 1.190 → 977 linhas e 77.252 → 62.985 bytes.
- **Genéricas (3ª passada):** três skills genéricas foram removidas do projeto porque não tinham regra do WYD e continuam disponíveis como skills de conta/globais. Substitutos: `wyd-web-admin-ui` + `wyd-portal/README.md`, `docs/index.md` + `docs/rust/02-execucao.md` §5, e `docs/manual-técnico/skills.md` + `tools/dev/check-skills.mjs`. Resultado: 21→18 skills; 977→501 linhas e 62.985→26.505 bytes nos `SKILL.md`; 40→19 arquivos; diretório 267.565→29.246 bytes.
- **Evidência:** [`docs/rust/evidencias/AUDIT-SKILLS-2026-11-12.md`](../docs/rust/evidencias/AUDIT-SKILLS-2026-11-12.md); `make skills-check` 0 e `make check` 0.
- **Sem commit/push** nesta auditoria.

## PROC-02 — Limpeza do ambiente Windows (Go → Rust)

- **ID:** `PROC-02`
- **Concluída em:** 2026-11-12
- **Resumo:** O host ficou sem toolchain Go: `go1.27.3` foi desinstalado pelo winget, `D:\Programas\Go` removido, `GOMODCACHE`/`GOCACHE`/`GOENV` limpos e as ferramentas de usuário (`gopls`, `dlv`, `golangci-lint`, `goplay`, `impl`, `protoc-gen-go*`) apagadas. `GOPATH` e `GOROOT` ficaram vazios em User/Machine, o `PATH` persistido não tem entrada Go e o VS Code não tem extensão Go. `cargo 1.99.1` e `rustc 1.99.1` respondem.
- **Evidência:** `where go` → `GO-REMOVIDO`; `go version` → `GO-COMANDO-AUSENTE`; `winget list --id GoLang.Go --exact` vazio; `GO-REMOVIDO` documentado em [`docs/rust/evidencias/PROC-02-limpeza-ambiente-2026-11-12.md`](../docs/rust/evidencias/PROC-02-limpeza-ambiente-2026-11-12.md).

## GAME-12 — Aglutinação item a item e teto de pilha no catálogo

- **ID:** `GAME-12`
- **Concluída em:** 2026-11-12
- **Resumo:** O catálogo passou a ter `stackable` decidido para os 3.220 itens: **423 true / 2.797 false / 0 nulo**. A migração `0050_seed_correcao_catalogo_game12_15.sql` corrige os seis itens divergentes e usa a prova da view `item_stack_cap` para os nulos (52 packs com `EF_AMOUNT` + 416/419/420 curados). O gerador `tools/content/gen-seed.mjs` recebeu a mesma curadoria para a próxima regeneração; o `0018` versionado ficou intacto porque o ledger recusa reescrever migration aplicada.
- **Invariantes provados:** nenhum `stackable=true` com teto 1; nenhum equipamento com teto >1; `EF_AMOUNT` continua conteúdo de pack, não teto. `kind` nulo em 1.529 linhas é resíduo declarado — nenhuma categoria foi inventada.
- **Evidência:** `crates/wyd-db/tests/sistema/stack_caps.rs` 1/1 com `WYD_TEST_DSN`; `cargo test -p wyd-core stack_caps` 6/6; `contrato_catalogo` 3/3; `check-migrations` 50/86; banco real `item_stack_cap` 423/2.797.

## GAME-15 — Baldes do catálogo, teto por item e os seis divergentes

- **ID:** `GAME-15`
- **Concluída em:** 2026-11-12
- **Resumo:** Os três baldes continuam **3.102 real / 96 ruído / 22 placeholder**. Correções: `406` era mana (`EF_MP 200`, preço 400) e virou `Poção_de_Mana_M.`; `416 Antídoto` virou `consumable` (teto 120, conteúdo 5); `419`/`420 Resto_*` viraram `material` (teto 120, sem efeito inventado — a fonte não traz `EF_*`). `3385 not_used`, `2433`–`2435 N/A` e `4045 NONAME` foram mantidos como placeholder e teto 1; preço/`EF_AMOUNT` não promovem nome de slot vazio.
- **Evidência:** migração `0050`; teste de banco com nome/kind/stackable/balde dos nove índices; docs `itens.md` §5.2–§5.3, `aglutinacao.md` §6/§8 e `drops.md` §8 atualizados.

## PROC-23 — Notas de pesquisa + PoCs medidas

- **ID:** `PROC-23`
- **Concluída em:** 2026-11-12
- **Resumo:** As PoCs pendentes foram executadas fora de `crates/` e os números publicados. **ECS:** `ComponentStore` SoA fez 10k entidades em **1.979,50 µs** (spawn + varredura + despawn) contra **4.321,40 µs** do `hecs` 0.11.2 (2,18× mais lento); na varredura mutável, **1,90 µs** contra **16,30 µs** (8,58× mais lento). **Serialização:** HEADER de 12 B em **2,53 ns** manual contra **1,42 ns** com `zerocopy` 0.8.63 (1,79×; ~1,1 ns/pacote, irrelevante no orçamento). **io_uring:** 10k leituras de 4 KiB em cache ficaram em **84,24–214,75 µs/op** contra **2,49–11,69 µs/op** do `read_at` síncrono (17–43× mais lento) no WSL2/Docker; o container padrão ainda bloqueia `io_uring_setup` com `EPERM`. **wasmtime:** módulo trivial com compilação 1,62–14,21 ms, instanciação 0,011–0,131 ms, chamada 27,5–74,7 ns e RSS ~10,4 MB, sem caso de uso.
- **Decisão:** manter SoA manual (ADR-0004), parse manual (ADR-0005) e `tokio`; não adotar `hecs`, `zerocopy`, `tokio-uring` nem `wasmtime` sem caso e bench que justifiquem. Nenhuma dependência entrou no workspace; `crates/`, `Cargo.toml` e `Cargo.lock` ficaram intactos.
- **Evidência:** [`docs/rust/evidencias/PROC-23-pocs-2026-11-12.md`](../docs/rust/evidencias/PROC-23-pocs-2026-11-12.md) — método, números, faixas e limitações de plataforma; notas atualizadas em `docs/rust/01-dependencias.md`.

## Auditoria de roadmap — 2026-11-12 (sem novas conclusões)

- **33 issues ativas auditadas** contra código, testes e docs; nenhuma foi removida porque todas mantêm ao menos um critério de aceite ou subtask bloqueada por evidência externa.
- **4 status corrigidos por evidência:** `GAME-11` ⬜→🚧 (depósito no inventário + `GroundLoot` prontos; falta handler da caixa), `GAME-12` ⬜→🚧 (`StackCaps` de produção lê `item_stack_cap`), `QA-11` ⬜→🟡 (`crates/wyd-bots` joga pelo fio; falta carga/relatório) e `WEB-07` ⬜→🚧 (export de logs/páginas prontos; falta SSE/alertas).
- **Candidatos mantidos abertos por resíduo real:** `WEB-01` (decisão da rota legada `set_reward_item_enabled`), `WEB-11` (rollback/histórico e inventário setting×constante), `GAME-15` (6 itens de catálogo), `GAME-18`/`GAME-17`/`BUG-01`/`BUG-02` (prova visual no 7662) e `CLIENT-10` (captura do `MSG_Trade`).
- **Organização:** fila nova `#01..#33` no `index.md`, com IDs canônicos preservados para não reutilizar IDs já fechados nem quebrar referências; quick wins sem bloqueio primeiro (`PROC-23`, `PROC-02`, `GAME-15`, `GAME-12`, `QA-11`, `WEB-07`).
- **Skills:** 23 skills mantidas (nenhuma duplicata nova); as 23 têm `name`/`description` válidos e ficam abaixo de 500 linhas, conforme o padrão de progressive disclosure. Referências mortas a `ui-ux-practices` removidas das issues `WEB-07/08/10`, contagem do doc histórico corrigida para 23 e o roteador ganhou orçamento explícito (1 principal + até 2 transversais). Na revisão final, uma skill genérica de manutenção (485→360 linhas) e `rust-test-quality` (91→45) foram encurtadas; total de 1.324 linhas / 82.932 bytes, sem remoção ou criação de skill.
- **Checklists:** removidas 6 linhas de template (`Codificar/Testar/Revisar/Documentar/Versionar/Limpar branch`) de 16 issues ativas; as subtasks específicas foram preservadas e seções vazias removidas.
- **Evidência:** `node tools/content/check-issue-status.mjs`, `node tools/dev/check-docs-consistency.mjs`, `cargo fmt --all -- --check` e `git diff --check` verdes na mesma execução final.

### QA-07 — Catálogo de ameaças B1–B17 + §6 do `wyd2.md`

- **ID:** `QA-07`
- **Concluída em:** 2026-11-12
- **Resumo:** O catálogo próprio passou a existir em `docs/wyd/sistemas/ameacas.md`: as 17 ameaças históricas B1–B17 foram confrontadas uma a uma com o Rust (7 PASSAM, 3 PARCIAIS, 5 ABERTOS, 1 OBSERVAR, 1 DIVERGENTE) e as 29 ameaças do §6 têm os dez campos exigidos, com teste/verificação ou motivo explícito de abertura. B1 não foi reproduzido, B6 virou decisão de produto (posição exata × spawn por cidade), B13–B17 continuam abertos por falta de affects/skills no fio e nenhum item visual foi declarado concluído. O checklist ganhou a seção 12 e o índice do manual aponta a nova página.
- **Evidência:** [`docs/wyd/sistemas/ameacas.md`](../docs/wyd/sistemas/ameacas.md); [`PROC-05-auditoria-2026-11-12.md`](../docs/rust/evidencias/PROC-05-auditoria-2026-11-12.md) registra a cobertura e os abertos; `docs/rust/08-checklist-funcionalidades.md` §12.

### PROC-05 — Auditoria do estado real das 36 issues do baseline

- **ID:** `PROC-05`
- **Concluída em:** 2026-11-12
- **Resumo:** As 36 issues do baseline foram classificadas com status declarado × auditado, evidência real ou hipótese explícita e próximo passo. Nenhuma issue dependente do 7662 foi promovida. O histórico Go foi contado (138 arquivos: 62 abertas, 54 em `done/`, 13 em `test/`) e o que não foi reexecutado ficou marcado como hipótese, não como prova.
- **Contagem por status:** **21 abertas, 8 em andamento, 2 parciais, 1 pronta para validar, 1 bloqueada e 3 concluídas = 36** (33 ativas + QA-07, PROC-05 e PROC-11).
- **Evidência:** [`docs/rust/evidencias/PROC-05-auditoria-2026-11-12.md`](../docs/rust/evidencias/PROC-05-auditoria-2026-11-12.md), com a tabela das 36 linhas e a reconciliação §6; `make check` exit 0 com 36/36 issues antes do fechamento e 33/33 depois.

### PROC-11 — Estrutura de módulos, nomenclatura e padrões

- **ID:** `PROC-11`
- **Concluída em:** 2026-11-12
- **Resumo:** A revisão fria mediu a árvore real e corrigiu o documento normativo: o DAG agora reflete `wyd-launcher → wyd-bots`, `wyd-db → wyd-core` (ADR-0009), `wyd-sim` independente e `wyd-server` com `wyd-sim`; foram acrescentados `payload/quest.rs`, os módulos reais do launcher, `repo/config.rs`/`sessions.rs`, `api/daily`/`donate`/`event`/`metrics`/`password`/`pin`, `formula/chaos.rs` e `apps/tools`; a §18 publica a árvore executada e o texto stale que negava o call site de loot foi removido. Nenhum código de jogo foi alterado.
- **Evidência:** [`docs/rust/evidencias/PROC-11-revisao-fria-2026-11-12.md`](../docs/rust/evidencias/PROC-11-revisao-fria-2026-11-12.md); `docs/arquitetura/estrutura-e-padroes.md` §1, §3, §9, §12–§16 e §18; `make check` exit 0.

### QA-08 — Dashboards Grafana + alertas (Rust, banco, produção)

- **ID:** `QA-08`
- **Concluída em:** 2026-11-12
- **Resumo:** O monitoramento já tinha dashboards, 10 regras e smoke; o que faltava era a série real do motor. O container Prometheus em execução ainda montava o diretório antigo `monitoring/prometheus`, removido quando o monitoramento passou para `docker/monitoring/prometheus`. O `file_sd` não encontrava `targets/wyd-server.json` e o job `wyd-server` não existia. Recriar só o Prometheus com o compose atual trouxe os dois canais para `up` e as séries de tick/entidades; nenhum código foi alterado.
- **Evidência:** [`docs/rust/evidencias/QA-08-prova-viva-2026-11-12.md`](../docs/rust/evidencias/QA-08-prova-viva-2026-11-12.md) — canais 1/2 `up`; `wyd_ticks_total` 927210/927319; `wyd_entities_active` 6110/6110; `node tools/dev/monitoring-smoke.mjs` exit 0 com 10 regras e 12 painéis. O alerta de falha de migration segue sem métrica na QA-04 (lacuna declarada).

### WEB-04 — E2E Playwright (auth, loja, painel)

- **ID:** `WEB-04`
- **Concluída em:** 2026-11-12
- **Resumo:** A suíte E2E passou a cobrir seis fluxos no portal real: login/logout, área do jogador, painel administrativo com busca de conta, entrega de ouro, idempotência de entrega e financeiro (vitrine + crédito com replay). A execução expôs e corrigiu um bug em `creditDonateAction`: `redirect()` dentro do `try` era capturado pelo `catch` depois de o crédito já ter sido gravado.
- **Evidência:** [`docs/rust/evidencias/WEB-04-e2e-2026-11-12.md`](../docs/rust/evidencias/WEB-04-e2e-2026-11-12.md) — `npm run test:e2e` **6 passed** em três execuções consecutivas (38.2 s · 37.8 s · 38.5 s); `make portal-test` 154/154 + `tsc --noEmit` 0; `make check` 0. A compra feita pelo jogador não foi simulada: sem contrato/tela, ela pertence à `WEB-09`.

### PROC-10 — Ordem de priorização por dependências

- **ID:** `PROC-10`
- **Concluída em:** 2026-11-12
- **Resumo:** O índice passou a publicar o grafo das 15 fases (21 arestas declaradas), a ordem topológica e os caminhos críticos de release e de produto/portal. A checagem de fase órfã ficou explícita: `FASE-01` é raiz, `FASE-15` é sorvedouro e `FASE-14` é contínua isolada por desenho.
- **Evidência:** [`roadmap-rust/index.md`](index.md) — seção “Grafo de dependências das fases”; `make check` verde com `38/38` issues ativas, 9 crates, 10 membros, 48 migrations e links consistentes.

### AUDIT-IA — Auditoria do conjunto de instruções de IA

- **ID:** `AUDIT-IA` (auditoria avulsa; não é issue do roadmap)
- **Concluída em:** 2026-11-12
- **Resumo:** O conjunto de instruções foi reduzido de 26 para 23 skills; `AGENTS.md` caiu de 270 para 117 linhas (~4,5 mil → ~1,7 mil tokens) e o total carregável caiu de 2.262 para 1.441 linhas (~33,4 mil → ~22,5 mil tokens); o roteador passou a ser a única lista de skills e as referências às skills removidas/fundidas foram corrigidas no roadmap e nos docs. Na revisão final, uma skill genérica de manutenção e `rust-test-quality` perderam conteúdo de outro host e duplicação com os docs, sem mudar o inventário de 23.
- **Evidência:** [`docs/rust/evidencias/AUDIT-IA-2026-11-12.md`](../docs/rust/evidencias/AUDIT-IA-2026-11-12.md) — inventário medido (26→23 `SKILL.md`; `SKILL.md` 1.992→1.494→1.324 linhas e 115.806→93.316→82.932 bytes; `AGENTS.md` 17.853→6.903 bytes); removidas `ui-ux-practices`, `finishing-a-development-branch` e `wyd-database` (fundida em `wyd-rust-db`); `writing-plans` e `wyd-server-ops` reescritas; uma skill genérica de manutenção e `rust-test-quality` encurtadas; `make check` exit 0 com 40/40 issues e 7/7 tools; varredura de links 229/0; zero caminhos de skill removida.

### PROC-13 — Gate de consistência entre roadmap, progresso, checklist e código

- **ID:** `PROC-13`
- **Concluída em:** 2026-11-12
- **Resumo:** O `make check` passou a validar links relativos de `README.md`, `AGENTS.md`, `docs/` e `roadmap-rust/` (182 arquivos/259 links) e a contagem/nomes das crates contra o workspace, além dos gates já existentes de migrations e status. A contagem documentada foi corrigida para 9 crates (10 membros com `apps/launcher-ui`).
- **Evidência:** [`docs/rust/evidencias/PROC-13-gate-docs.md`](../docs/rust/evidencias/PROC-13-gate-docs.md) — `docs-check` exit 0 com `39 issues, 9 crates, 10 membros, 48 migrations, 182 docs, 259 links`; prova negativa alterando o README para 8 crates (exit 1) e restaurada (exit 0); `make check` verde.

### WEB-02 — Painel do usuário (conta, recuperação, rankings, personagens)

- **ID:** `WEB-02`
- **Concluída em:** 2026-11-12
- **Resumo:** Os quatro fluxos do painel funcionaram no portal e no gateway vivos, com dados reais, sessão assinada, recuperação auditada e respostas pessoais sem cache.
- **Evidência:** [`docs/rust/evidencias/WEB-02-painel-vivo.md`](../docs/rust/evidencias/WEB-02-painel-vivo.md) — portal: login 200, `/conta` 200 com `Cantarelli`/slot 0, `/ranking` 200 com top 1 real, recuperação 200 com `no-store`; gateway: `panel/self`, `panel/characters`, `panel/rankings` e `accounts/recovery` com `result=1`; auditoria `accounts.recovery` em `audit_log`.

### PROC-03 — Reconfiguração do VS Code para Rust

- **ID:** `PROC-03`
- **Concluída em:** 2026-11-12
- **Resumo:** O VS Code está configurado para rust-analyzer, formatação e clippy, sem extensão Go; a análise real do workspace terminou sem erros.
- **Evidência:** [`docs/rust/evidencias/PROC-03-rust-analyzer.md`](../docs/rust/evidencias/PROC-03-rust-analyzer.md) — `rust-analyzer 0.3.3113-standalone`; `analysis-stats . --parallel` exit 0 com 77 crates, 74.384 linhas, 5.587 declarações, 5.419 corpos, 0 panics e 0 layouts falhos; `diagnostics .` exit 0 com `diagnostic scan complete`, 0 erros e 410 WeakWarnings (409 `inactive_code` e 1 aviso de estilo em `portal_client.rs:465`). `.vscode/settings.json` usa clippy com `-D warnings` e format on save; `cargo clippy --workspace --all-targets -- -D warnings` verde no `make check`.

### PROC-07 — Inventário Go→Rust das skills e instruções de agentes

- **ID:** `PROC-07`
- **Concluída em:** 2026-11-12
- **Resumo:** O mapa de skills Go→Rust está publicado, com destino, motivo e conflito de cada skill de domínio, além das skills Rust, de engenharia e genéricas.
- **Evidência:** `docs/manual-técnico/skills.md` §1–§2; tabela de sucessão com destino/motivo/conflito; 2 stubs removidos; 26 `SKILL.md` vigentes e roteador em `.agents/skills/README.md`.

### PROC-04 — Branch `develop` como principal de desenvolvimento

- **ID:** `PROC-04`
- **Concluída em:** 2026-11-12
- **Resumo:** `develop` é a branch atual e o fluxo issue→branch→merge em `develop` está normatizado, com force-push proibido.
- **Evidência:** `git branch --show-current` = `develop`; `docs/rust/02-execucao.md` §2 define branch por issue, merge `--no-ff` e proibição de force-push em `develop`/`main`; `git reflog show develop` não contém reset/rebase/forced update.

### PROC-06 — Varredura de dívidas: TODO/FIXME/HACK e código morto

- **ID:** `PROC-06`
- **Concluída em:** 2026-11-12
- **Resumo:** As ocorrências de dívida foram medidas e classificadas: 0 marcadores no Go e 16 no C++ legado, cada uma com destino e issue dona.
- **Evidência:** `docs/manual-técnico/auditoria-roadmap.md` §3 e §3.1: 0 marcadores Go, 16 C++ com destino `portar`/`descartar`/`investigar`; §6 registra as limitações da leitura amostral.

### PROC-08 — Documento do ciclo de execução unitária

- **ID:** `PROC-08`
- **Concluída em:** 2026-11-12
- **Resumo:** O ciclo obrigatório por issue, gates, fluxo de branch e Definition of Done estão publicados e são normativos.
- **Evidência:** `docs/rust/02-execucao.md` §1–§3: análise → design/ADR → código → teste/gate → docs → fechamento; branch e DoD definidos.

### PROC-09 — Processo de revisão consolidada a cada 10 issues

- **ID:** `PROC-09`
- **Concluída em:** 2026-11-12
- **Resumo:** A revisão consolidada tem checklist fixo, gatilho a cada 10 issues e local de registro definido.
- **Evidência:** `docs/rust/02-execucao.md` §4: checklist de gates, benchmark contra baseline, integração, ADRs, docs vivos e débito; registro em `docs/rust/03-progresso.md`.

### GAME-04 — Fim dos TXT/CSV: validação de schema no boot

- **ID:** `GAME-04`
- **Concluída em:** 2026-11-12
- **Resumo:** O conteúdo vem do banco, é validado no boot e um dado corrompido aborta a carga; não há leitura de TXT/CSV de jogo em runtime.
- **Evidência:** `crates/wyd-core/src/ports/content.rs` (`ContentSource`) + `crates/wyd-db/src/repo/content.rs`; `crates/wyd-db/tests/sistema/pg.rs::conteudo_estatico_carrega_e_aborta_com_dado_corrompido`; schema/seed em migrations 0007–0024.

### CLIENT-05 — Engenharia reversa estática do cliente

- **ID:** `CLIENT-05`
- **Concluída em:** 2026-11-12
- **Resumo:** O pipeline estático gera os artefatos, classifica âncoras/patchs com bytes e documenta formatos; T2 foi transferido para CLIENT-06.
- **Evidência:** 6/6 âncoras e 3/3 patches confirmados por bytes/objdump; `docs/cliente/` com inventário e gerados; T2 movido para CLIENT-06 por limitação da ferramenta.

### PROC-17 — Auditoria profunda das skills Go restantes

- **ID:** `PROC-17`
- **Concluída em:** 2026-11-12
- **Resumo:** Todas as skills do inventário têm destino, motivo e trechos obsoletos; a auditoria de 2026-11-12 removeu os stubs aposentados.
- **Evidência:** `docs/manual-técnico/skills.md` §2 (auditoria de 2026-11-12): 9 skills de domínio com destino/motivo/conflito, 2 stubs removidos e 26 `SKILL.md` vigentes; roteador em `.agents/skills/README.md`.

### PROC-21 — Rito ADR + template fixo

- **ID:** `PROC-21`
- **Concluída em:** 2026-11-12
- **Resumo:** O template ADR está publicado, o rito é normativo e a numeração está íntegra de 0000 a 0028.
- **Evidência:** `docs/arquitetura/adr/0000-template.md`; rito em `docs/rust/02-execucao.md`; `ls docs/arquitetura/adr` sem lacunas entre 0000 e 0028.

### PROC-12 — Visão de pastas e mapa do cliente

- **ID:** `PROC-12`
- **Concluída em:** 2026-11-12
- **Resumo:** Fechar o gate de arquitetura exigido pela missão-mãe: inventário real do repositório, tabela-alvo pasta→responsabilidade→padrão→SOLID→nomenclatura, regras anti-sprawl da raiz e o mapa do `client/` como usuário do `wyd-launcher` — tudo antes de qualquer nova implementação.
- **Evidência:** ```bash test -f docs/arquitetura/visao-pastas.md -a -f docs/cliente/mapa.md # nenhum arquivo NOVO de implementação (.rs/.mjs/.ts/.py) nem novo na raiz: git status --porcelain | grep '^??' | grep -E '\.(rs|mjs|ts|py)$' && echo FALHA || echo SEM-IMPLEMENTACAO-NOVA git status --porcelain | grep '^??' | grep -v '/' && echo FALHA-RAIZ || echo SEM-ARQUIVO-NOVO-NA-RAIZ cargo fmt --all -- --check && cargo clippy --workspace -- -D warnings && cargo test --workspace ```

### GAME-01 — Tick em frequência fixa determinística

- **ID:** `GAME-01`
- **Concluída em:** 2026-11-12
- **Resumo:** Implementar o loop de tick 20–50 Hz com intervalo fixo e ordem de sistemas documentada, sem alocação no caminho quente.
- **Evidência:** ```bash cargo test -p wyd-core make check ```

### GAME-02 — ECS: componentes contíguos das entidades

- **ID:** `GAME-02`
- **Concluída em:** 2026-11-12
- **Resumo:** Modelar Players, NPCs, Monsters e Items no chão como dados contíguos (`Vec<Position>`, `Vec<Stats>`, `Vec<Inventory>`), sem hierarquia OOP.
- **Evidência:** ```bash cargo test -p wyd-core cargo bench -p wyd-core 2>&1 | tail -n 5 make check ```

### GAME-03 — Spatial grid e interest management O(1)

- **ID:** `GAME-03`
- **Concluída em:** 2026-11-12
- **Resumo:** Particionar o mapa em grade espacial com visibilidade e gerenciamento de interesse em tempo O(1) por consulta.
- **Evidência:** ```bash cargo test -p wyd-core grid make check ```

### GAME-06 — IA de NPC no Hot Path (FSM + utilidade, determinística)

- **ID:** `GAME-06`
- **Concluída em:** 2026-11-12
- **Resumo:** Dar comportamento a NPC/monstro dentro do tick: aggro, perseguição, ataque, fuga e retorno ao spawn, com decisão determinística, orçamento medido e nenhum I/O.
- **Evidência:** (medidos nesta entrega)

### GAME-07 — Motor no canal: tick real dentro do `wyd-server`

- **ID:** `GAME-07`
- **Concluída em:** 2026-11-12
- **Resumo:** Fazer o processo do canal **rodar o motor**: um `World` de `wyd-core` por processo, tick determinístico a 20 Hz (ou 50 Hz configurável), métricas reais e desligamento limpo — sem I/O dentro do tick.
- **Evidência:** ```bash cargo test -p wyd-server --test tick_processo cargo bench -p wyd-core --bench tick curl -s localhost:9898/metrics | grep -E 'wyd_ticks_total|wyd_entities_active' cargo clippy --workspace --all-targets -- -D warnings ```

### GAME-08 — Entrada no mundo: spawn do personagem e estado vivo persistido

- **ID:** `GAME-08`
- **Concluída em:** 2026-11-12
- **Resumo:** Transformar `_MSG_CharacterLogin` em **entidade viva no mundo**: ler o personagem do Postgres, criar a entidade no `ComponentStore` na posição salva e devolver ao banco posição/HP/MP/EXP no logout, queda de conexão e save periódico.
- **Evidência:** ```bash WYD_TEST_DATABASE_URL=... cargo test -p wyd-server --test entrada_mundo WYD_TEST_DATABASE_URL=... cargo test -p wyd-db --test pg cargo test --workspace          # sem DSN: os testes de banco pulam com aviso ```

### GAME-09 — Cena autoritativa (`ESCENE_FIELD` 30000): o primeiro pixel de mundo

- **ID:** `GAME-09`
- **Concluída em:** 2026-11-12
- **Resumo:** Emitir o pacote autoritativo de cena com `HEADER.ID = ESCENE_FIELD` (30000): entidades visíveis, posição, HP/EXP e remoção — o que faz o cliente desenhar o mundo e atualizar as barras.
- **Evidência:** ```bash cargo test -p wyd-protocol --test scene cargo test -p wyd-core --test visibility cargo test -p wyd-server --test cena cargo test -p wyd-protocol -p wyd-net     # bytes no fio ```

### GAME-10 — Cidadania e canal verde: bônus e papel do canal aplicados

- **ID:** `GAME-10`
- **Concluída em:** 2026-11-12
- **Resumo:** Fazer `citizen` deixar de ser só um número no `STRUCT_SELCHAR` e virar **regra aplicada**: bônus de caça/drop do canal de cidadania, recusa de serviço de NPC de outra cidade e papel verde (sem PvP, loja permitida) como propriedade do processo.
- **Evidência:** ```bash cargo test -p wyd-core --test cidadania cargo test -p wyd-sim cargo test -p wyd-server --test canal_verde cargo test -p wyd-db --test catalogo   # se houver migration nova ```

### GAME-16 — Piedade de refino persistida + `UpdateItem` de resposta

- **ID:** `GAME-16`
- **Concluída em:** 2026-11-12
- **Resumo:** Persistir a piedade (pity) de refino por item e responder o refino com o pacote de atualização visível ao cliente, sem mudar nenhuma taxa vigente.
- **Evidência:** ```bash cargo test -p wyd-server --test refino_piedade cargo test -p wyd-sim ```

### PROC-14 — Modelagem relacional e integridade

- **ID:** `PROC-14`
- **Concluída em:** 2026-11-12
- **Resumo:** Modelar contas, personagens, inventários, baús, guildas e auditoria com FKs, CHECKs e índices para o caminho quente, documentado em `docs/arquitetura/`.
- **Evidência:** ```bash make check && ls docs/arquitetura/ | grep -i -E "model|banco|db|er" ```

### PROC-15 — Migrations versionadas e seeds reproduzíveis

- **ID:** `PROC-15`
- **Concluída em:** 2026-11-12
- **Resumo:** Subir o PostgreSQL do zero via migrations versionadas (up/down testado) com seeds dev/prod reproduzíveis.
- **Evidência:** ```bash cargo test -p wyd-db make check ```

### PROC-16 — Persistência assíncrona fora do tick

- **ID:** `PROC-16`
- **Concluída em:** 2026-11-12
- **Resumo:** Implementar o pool async (`sqlx`) com transações curtas e fila de persistência, sem nenhum I/O bloqueante no tick loop.
- **Evidência:** ```bash cargo test -p wyd-db make check ```

### QA-02 — Pipeline CI (fmt, clippy, test, bench, audit, deny)

- **ID:** `QA-02`
- **Concluída em:** 2026-11-12
- **Resumo:** Pipeline verde e espelhável localmente (`make ci`) com fmt, clippy, testes, benches, `cargo audit` e `cargo deny`.

### QA-03 — Release LTO + imagem mínima + proveniência

- **ID:** `QA-03`
- **Concluída em:** 2026-11-12
- **Resumo:** Release otimizado (LTO) empacotado em imagem mínima multi-stage, com build reproduzível e smoke do binário.

### QA-04 — Telemetria: tracing + Prometheus no motor

- **ID:** `QA-04`
- **Concluída em:** 2026-11-12
- **Resumo:** Instrumentar o motor com `tracing` e expor tick P50/P95/P99, pacotes/s, entidades ativas e memória, sem `println!` como telemetria.

### QA-05 — Hardening: DoS, rate limiting, sanitização do parser

- **ID:** `QA-05`
- **Concluída em:** 2026-11-12
- **Resumo:** Endurecer o handshake (limite por IP, timeout, buffer máximo) com rate limiting e sanitização estrita, tudo testado contra flood básico.
- **Evidência:** ```bash cargo test -p wyd-net make check ```

### QA-06 — Lotação e manutenção do canal: recusa de entrada tipada

- **ID:** `QA-06`
- **Concluída em:** 2026-11-12
- **Resumo:** Fechar o checklist 1.14 (“canal cheio / manutenção / recusa de entrada”): limite de jogadores por canal e modo manutenção, com recusa antes de criar sessão, log e métrica.
- **Evidência:** ```bash cargo test -p wyd-server --test lotacao cargo test -p wyd-server           # suíte do processo cargo clippy --workspace --all-targets -- -D warnings ```

### WEB-03 — Painel administrativo (auditoria, economia, moderação)

- **ID:** `WEB-03`
- **Concluída em:** 2026-11-12
- **Resumo:** Entregar o painel de staff com auditoria de logs, estatísticas de economia e ferramentas de moderação, tudo rastreável.
- **Evidência:** ```bash cargo test -p wyd-admin make check ```

### WEB-05 — Handlers de leitura de conteúdo no gateway (npc/mob/item/efeito/mapa/teleporte)

- **ID:** `WEB-05`
- **Concluída em:** 2026-11-12
- **Resumo:** Tirar do `501` as rotas de **leitura** de conteúdo do portal, servindo o Postgres real por portas do `wyd-db`, sem escrita e sem estado vivo.
- **Evidência:** ```bash WYD_TEST_DATABASE_URL=... cargo test -p wyd-admin --test conteudo cargo test -p wyd-admin            # contrato + paridade WYD_TEST_DATABASE_URL=... cargo test -p wyd-db ```

### WEB-06 — Handlers de operação no gateway (rates/daily/donate/evento/admin de conta)

- **ID:** `WEB-06`
- **Concluída em:** 2026-11-12
- **Resumo:** Tirar do `501` as rotas de **operação** que hoje só existem no contrato: rates de conteúdo, recompensa diária, donate/saldo, evento de mundo, receita e administração de conta — cada escrita com auditoria e sem tocar estado vivo.
- **Evidência:** ```bash WYD_TEST_DATABASE_URL=... cargo test -p wyd-admin --test operacao cargo test -p wyd-admin -p wyd-db node tools/content/check-migrations.mjs      # se houver migration nova ```

### CLIENT-01 — Tabela do protocolo (headers, opcodes, tamanhos, payload)

- **ID:** `CLIENT-01`
- **Concluída em:** 2026-11-12
- **Resumo:** Publicar em `docs/arquitetura/` a tabela exata do protocolo e entregar a crate `wyd-protocol` **organizada na árvore planejada**, com opcodes tipados, payloads documentados e round-trip testado.
- **Evidência:** ```bash cargo test -p wyd-protocol make check ```

### CLIENT-02 — `wyd-protocol` zero-copy + validação de limites + golden

- **ID:** `CLIENT-02`
- **Concluída em:** 2026-11-12
- **Resumo:** Implementar header/opcodes com `#[repr(C)]` e validação de limites antes de qualquer parse, com testes golden byte-exatos.
- **Evidência:** ```bash cargo test -p wyd-protocol -p wyd-net make check ```

### CLIENT-03 — Distribuição do launcher/patch (updater substituto)

- **ID:** `CLIENT-03`
- **Concluída em:** 2026-11-12
- **Resumo:** Entregar o canal de **distribuição**: o updater substituto que busca manifesto remoto e pacotes de patch, verifica assinatura e os entrega ao `patch::apply` da CLIENT-07.
- **Evidência:** ```bash cargo test -p wyd-launcher ./target/debug/wyd-launcher update <dir> <manifesto-remoto> --dry-run ```

### CLIENT-04 — Dossiê, inventário e baseline do cliente

- **ID:** `CLIENT-04`
- **Concluída em:** 2026-11-12
- **Resumo:** Publicar o dossiê do cliente fechado (`client/`) como fonte canônica no repositório: inventário verificado, hashes do build de referência, superfície já conhecida e o plano de engenharia reversa — antes de qualquer trabalho de desassemble.
- **Evidência:** ```bash # os hashes do dossiê batem com o disco cd client && sha256sum WYD.exe WYDLauncher.exe ClientPatch.dll # links do README resolvem for p in $(grep -o '](\([^)]*\.md\))' README.md | sed 's/](//; s/)$//'); do test -e "$p" || echo "QUEBRADO $p"; done # gate do workspace continua verde (nenhuma crate tocada) cargo fmt --all -- --check && cargo clippy --workspace -- -D warnings && cargo test --workspace ```

### CLIENT-07 — Customização sustentável e integração com o launcher

- **ID:** `CLIENT-07`
- **Concluída em:** 2026-11-12
- **Resumo:** Transformar o conhecimento de RE em **camada de customização sustentável** do cliente (hooks/patch por versão, UI, comandos) e no contrato que o `wyd-launcher` usa para validar, atualizar e iniciar o jogo.
- **Evidência:** ```bash cargo fmt --all -- --check && cargo clippy --workspace --all-targets -- -D warnings && cargo test --workspace ./target/debug/wyd-launcher check client/ <manifesto-7662>   # 6/6 OK, exit 0 ./target/debug/wyd-launcher launch client/ <manifesto> --dry-run ```

### CLIENT-09 — Logout explícito: sair do mundo sem esperar timeout

- **ID:** `CLIENT-09`
- **Concluída em:** 2026-11-12
- **Resumo:** Tratar o pedido de saída do jogador (voltar à seleção ou fechar o cliente) de forma explícita: liberar `character_session`, sair do mundo e salvar — em vez de depender só do timeout de 120 s.
- **Evidência:** ```bash WYD_TEST_DATABASE_URL=... cargo test -p wyd-server --test selchar logout_explicito WYD_TEST_DATABASE_URL=... cargo test -p wyd-server --test sessao_unica ``` Saída esperada: `character_session.released_at` preenchido no logout explícito e na queda; relogin imediato aceito. Sem DSN os testes **pulam** com aviso (`SKIP:`), sem derrubar a suíte.

### PROC-18 — Criar/atualizar skills sucessoras Rust ✅ CONCLUÍDA

- **ID:** `PROC-18`
- **Concluída em:** 2026-11-12
- **Resumo:** Publicar as skills sucessoras (padrão frontmatter + regras numeradas) e anotar as Go com sucessão, sem instruções conflitantes.

### PROC-19 — Guia de contribuição + manuais (usuário e técnico) ✅ CONCLUÍDA

- **ID:** `PROC-19`
- **Concluída em:** 2026-11-12
- **Resumo:** Publicar o guia que permite a um novato compilar, testar e contribuir, mais o esqueleto dos manuais de usuário e técnico.

### PROC-20 — C4 + especificações de pacotes de rede ✅ CONCLUÍDA

- **ID:** `PROC-20`
- **Concluída em:** 2026-11-12
- **Resumo:** Publicar diagramas C4 (níveis 1–3) fiéis ao código e as especificações de pacotes vigentes em `docs/arquitetura/`.

### PROC-22 — Primeiras ADRs (monolito, ECS, binário, async)

- **ID:** `PROC-22`
- **Concluída em:** 2026-11-12
- **Resumo:** Promover a `0001` e registrar as ADRs de ECS, formato binário e runtime async, travando as escolhas que as fases de execução consomem.
- **Evidência:** ```bash make check && ls docs/arquitetura/adr/ ```
