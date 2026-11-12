# Changelog de issues concluídas

> Issues que cumpriram os critérios de aceite foram removidas de `roadmap-rust/issues/` e registradas aqui. O histórico completo continua no git (arquivos removidos) e o painel vivo é `docs/rust/03-progresso.md`.

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
- **Resumo:** O conjunto de instruções foi reduzido de 26 para 23 skills; `AGENTS.md` caiu de 270 para 117 linhas (~4,5 mil → ~1,7 mil tokens) e o total carregável caiu de 2.262 para 1.611 linhas (~33,4 mil → ~25,0 mil tokens); o roteador passou a ser a única lista de skills e as referências às skills removidas/fundidas foram corrigidas no roadmap e nos docs.
- **Evidência:** [`docs/rust/evidencias/AUDIT-IA-2026-11-12.md`](../docs/rust/evidencias/AUDIT-IA-2026-11-12.md) — inventário medido (26→23 `SKILL.md`, 1.992→1.494 linhas, 115.806→93.316 bytes; `AGENTS.md` 17.853→6.796 bytes); removidas `ui-ux-practices`, `finishing-a-development-branch` e `wyd-database` (fundida em `wyd-rust-db`); `writing-plans` e `wyd-server-ops` reescritas; `make check` exit 0 com 40/40 issues e 7/7 tools; varredura de links 229/0; zero caminhos de skill removida.

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
