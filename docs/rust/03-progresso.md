# Progresso da execução — roadmap-rust

> Documento vivo. Atualizado a **cada issue** executada (regra em
> `docs/rust/02-execucao.md` §5). Última atualização: 2026-11-11 (árvore por domínio em
> `wyd-server`/`wyd-bots`; a entrada anterior era a mega revisão do roadmap:
> 9 issues novas com fila por tamanho, status reconciliado nos três lugares que carregam status,
> `wyd-server` incorporado à árvore normativa, gate de `cargo test --workspace` consertado e
> banco medido em **31 migrations / 69 tabelas + ledger** (2026-11-09; **reaberto em 2026-11-10:
> código = 36 migrations / 74 tabelas**, ver Fase 1 de pesquisa, item FASE-02-6) — ver §6 de
> `docs/rust/04-revisao-roadmap.md`. Em seguida, **FASE-03-7 entregue**: o motor do core roda
> no processo do canal com métricas reais e parada limpa. **FASE-03-8 entregue**: personagem
> persistido vira entidade viva (entrada por fila `EngineHandle`), sai salvando posição/HP/MP/EXP
> e o monitoramento virou índice + subpáginas (`/monitoramento/{servidor,grafana,prometheus,logs}`)
> com painel e alertas de orçamento de tick). **2026-11-10**: seleção/criação de personagem no
> layout real do motor antigo (`STRUCT_SELCHAR` 840 B / `STRUCT_MOB` 816 B, `BaseMob` por classe
> embutido) e a prova viva do gauge de entidades revalidada com esse formato.
> **2026-11-10 (inventário persistido)**: `character_items`/`vault_items` com repositório
> real, kill mark (item 547) e `ShortSkill[16]` persistidos, cofre/`Coin` da conta lidos no
> `CNFAccountLogin` e `STRUCT_MOB`/`STRUCT_SELCHAR` montados do banco (template só na
> criação). Migrations **0033**/`0034` — **34 migrations / 72 tabelas**.
> **2026-11-10 (ganho de nível)**: curva de EXP do legado (`g_pNextLevel`, `Basedef.cpp:625`)
> com a máquina `grant_exp` do `wyd-sim`; level-up MORTAL (+5 atributo, +3 skill — 4 a partir do
> nível interno 200 —, +2 maestria, HP/MP pelos incrementos da classe, vitais cheios) com
> gravação transacional de `level`/`exp` e `send_progress` (`_MSG_UpdateEtc` +
> `_MSG_UpdateScore`). O `Level` do fio passou a ser **banco − 1** (o cliente soma 1; o `BaseMob`
> traz `Level = 0`) e o `_MSG_CNFDeleteCharacter` passou a devolver o `STRUCT_SELCHAR` (844 B) em
> vez de um frame só de cabeçalho. Sem migration nova (nível/EXP já existiam em `characters`).
> **2026-11-10 (configuração viva)**: gravação do portal sobe `config_revision`; a thread
> `wyd-config` (`WYD_CONFIG_POLL_MS`, default 1 s) lê fora do tick e troca um
> `Arc<GameConfig>` atômico (`crates/wyd-server/src/config/service.rs`). Falha de banco, revisão
> inválida ou snapshot incompleto mantêm o último snapshot bom e contam em
> `wyd_config_reload_failures_total`; boot sem snapshot válido é fatal. Nesta entrega o único
> consumidor era o multiplicador de EXP em `SelectionStore::gain_exp`; refino, SancRate,
> composição, acessório, drops, NPC, teleporte e `game_setting` ainda não tinham resolvedor
> (o parágrafo seguinte fecha refino e drops no resolvedor, mas não no fio). Prova:
> `crates/wyd-server/tests/integracao/config_recarga.rs` (4 testes, sem banco externo).
> **2026-11-10 (configuração no gameplay)**: `crates/wyd-server/src/gameplay/service.rs` passou a ser
> o resolvedor de refino e loot; cada chamada pega o `Arc<GameConfig>` vigente, nunca uma
> cópia de boot. `Gameplay::refine` aplica `RefinePolicy::LEGACY` + `powder_rate_bps` e
> devolve taxa/piedade/falha; `Gameplay::loot` faz um sorteio por item, aplica o
> multiplicador de drop e recusa tabela ausente/sorteio faltando. Prova:
> `crates/wyd-server/tests/integracao/config_gameplay.rs` (6 testes: reload muda refino/loot, reverter
> restaura, valor inválido mantém o último snapshot). **Refino no fio (2026-11-10)**: o
> `Opcode::UseItem` já portado deixou de cair no `outro`; `cpsock::dispatch` decodifica
> `_MSG_UseItem` e chama `SelectionStore::use_item`, que resolve
> `Gameplay::use_item` no snapshot vigente, consome o pó (412/413/4141), sobe o refino ou
> destrói o selado e grava o inventário em `character_items` na mesma transação. O teste
> `use_item_e_o_caminho_do_pacote_leem_a_sancrate_viva` (`config_gameplay.rs`, 6º) prova a
> mesma chamada de handler mudando de sucesso para falha com SancRate 10→0, voltando ao
> sucesso ao reverter e mantendo o snapshot bom quando a leitura falha. **Drops ligados ao
> inventário (2026-11-11)**: `Gameplay::loot` roda no abate e o resultado é entregue por
> `gameplay::add_item` (pilha até `MAX_STACK = 120`, slot livre `16..80`) + `SelectionStore::grant_loot`
> (persiste com `replace_items`) + `_MSG_SendItem` por slot na borda. Caixa no chão e pickup
> seguem sem fio (offsets em `docs/cliente/pacotes.md`).
> Inspeção nominal sem resolvedor de morte/caixa/pickup: `wyd_sim::resolve_attack`
> (`AttackOutcome` só hit/crit/dano), `wyd_sim::TICK_PHASES` (`deaths` é rótulo, não função),
> `wyd_sim::formula::drop::{drops, effective_rate, gem_bonus}` (predicado taxa×roll),
> `wyd_sim::formula::accessory::attempt` (refino de acessório),
> `wyd_sim::formula::progression::{grant_exp, exp_to_next}` (EXP), `wyd_core::systems::combat::attack`
> (`Hit { remaining_hp }`, sem evento de morte), `wyd_core::systems::npc::decide` (intenção de
> NPC), `wyd_server::world::{Command, Engine::step}` (`Enter`/`Leave`/`Snapshots`; `step` só
> drena eventos) e `cpsock::dispatch` (só Create/Delete/Login/Logout/UseItem/ApplyBonus; o
> resto cai em `outro` tolerado). O portal marca drops como não consumidos e nomeia esses
> caminhos. `_MSG_UpdateItem` também não tem corpo portado; não se inventa frame vazio, então
> o cliente só redesenha o item no relogin.
>
> **2026-11-10 (EXP no fio, refino com goldens e NPC do snapshot)**: `_MSG_Attack`,
> `_MSG_AttackOne` e `_MSG_AttackTwo` deixaram o `outro` tolerado. O handler decodifica o
> pacote real (`crates/wyd-protocol/src/payload/combat.rs`), ignora o `current_exp` do cliente
> e chama `SelectionStore::attack` → `Engine::attack_at`: o dano sai de
> `wyd_core::systems::combat::attack` com `Stats` autoritativos e a cadeia portada do
> `wyd-sim` (`DamageInput`/`damage_chain`). No abate, `gain_exp` credita a EXP do
> `STRUCT_MOB` do template (`npc_template`, 816 B) já com o multiplicador `exp` do snapshot
> (`Gameplay::exp_reward`) e responde `_MSG_UpdateEtc` + `_MSG_UpdateScore`. Refino: o
> handler passou a ter prova explícita de golden — a taxa devolvida é comparada com
> `powder_success_bps(RefinePolicy::LEGACY, …)` em cada passo que o pó alcança, antes e
> depois da recarga. NPC: `npc_definition` do snapshot passou a carregar `template_name` +
> `blob`, e `NpcRule` expõe nome/EXP/nível/HP/ataque/defesa; `world::npc_specs` materializa o
> mapa e `Engine::sync_npcs` reconcilia a cada revisão (spawn/remoção sem duplicar);
> `_MSG_Action` valida alcance contra a posição autoritativa e responde `_MSG_MessagePanel`
> com o `display_name` — sem tabela de diálogo, não há frase inventada. O log
> `config_reload` passou a publicar `multiplier_exp`, o valor que a edição do portal fez
> valer no processo. Prova: `config_gameplay.rs` (8 testes, incl. `_MSG_AttackOne` real
> montado/ofuscado/lido e `UpdateEtc` com a EXP multiplicada), `config_recarga.rs` (6,
> incl. template inválido recusando a foto) e `world` (7 unitários). E2E com o compose
> local, 2026-11-10: as imagens novas aplicaram `event=schema applied=2 skipped=36
> version=0038` e o servidor materializou o seed real com `event=npc result=spawn map=0
> spawned=6100 kept=0 removed=0 rejected=0`; o login do portal (cookie de staff) leu
> `/configuracoes`, enviou `saveRateBatchAction` com `rate:multiplier:exp` 10000→25000 e a
> resposta foi 303 `/configuracoes?ok=1`, `config_revision` 1→2 e `content_rate_override`
> `exp|25000|portal-e2e`; o processo de jogo, sem reinício, registrou `config_reload
> result=applied from=1 to=2 … multiplier_exp=25000` e `event=npc result=reload revision=2
> spawned=0 kept=6100 removed=0`, provando recarga sem duplicar NPC; a tela já exibia
> 25000, e a reversão para 10000 levou a revisão a 3 com `multiplier_exp=10000` no mesmo
> processo. Sem migration nova (0037/0038 já existiam na árvore).
>
> **2026-11-11 (árvore por domínio em `wyd-server` e `wyd-bots`)**: `crates/wyd-server/src`
> deixou de ter domínios planos na raiz. `character/` foi dividido em `crud.rs`, `snapshot.rs`
> e `session.rs`; `gameplay/`, `config/` e `status/` viraram diretórios com `mod.rs` de
> fachada e `service.rs` com as regras do domínio (subdivisão fina adiada até haver consumidor
> que a justifique). A raiz ficou com `lib.rs`, `main.rs`, `channels.rs`, `metrics.rs` e
> `logs.rs` (ADR-0026). `crates/wyd-bots/src` virou `bot/` (`config`, `cena`, `campanha`,
> `comercio`, `combate`, `movimento`, `recusa` separados de socket/autenticação) + `tela/`,
> preservando `Bot`, `BotConfig`, `Race`, `SceneMob`, `parse_etapa`, `nome_do_campo` e
> `wyd_bots::tela::*` (ADR-0027). Gates: `cargo fmt --all --check` 0;
> `cargo clippy -p wyd-server -p wyd-bots --all-targets -- -D warnings` 0;
> `cargo test -p wyd-bots` verde. Suíte do `wyd-server` com `WYD_TEST_DSN` (banco `wyd_test`,
> `--all-targets -j 1`): **146 passed, 0 failed, 0 skipped**, exit 0 — incluindo
> `sistema_selchar` 9/9. No teste do sussurro, a consulta imediata de sessão virou
> `esperar_ate` (o `release` roda no `Drop` do `SessionLease`, ao encerrar a conexão), sem
> afrouxar a asserção.

## Legenda

✅ concluída · 🚧 em curso · ⬜ pendente · ⛔ bloqueada

## Rotina de fechamento — ambiente local sempre atualizado

O teste manual do dono roda no **portal do compose em http://127.0.0.1:3000**. Antes de
dar qualquer issue como concluída, execute o rebuild completo a partir do código atual:

```bash
docker compose up -d --build --wait   # caminho único (não exige make)
make local-rebuild                   # atalho equivalente + provas de contrato
```

O comando é o próprio compose, sem script paralelo: build das imagens, `up -d --wait`
(`--wait` só retorna 0 com todos os serviços `running`/`healthy`; o gateway e os canais aplicam
o schema no próprio boot, ADR-0024) e as provas de contrato — `/login` 200, `/entregas` 307 para
`/login`, gateway `/health` com `schema:36`, 3100/3110 livres. Issue concluída sem essa
saída verde é issue não concluída.

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

### Imersão `wyd2.md` §15 — Fase 1 (conhecimento do projeto)

Registro da leitura inicial da missão de pesquisa. Sem alteração de código; classificação: CONFIRMADO = lido no arquivo; DIVERGENTE = dois documentos dizem coisas diferentes.

| Item | Classificação | Evidência |
|---|---|---|
| `AGENTS.md` (Rust, `client/` não versionado, docs em português, ciclo por issue) | CONFIRMADO | raiz do repositório |
| Skills vigentes `wyd-rust-*`, `wyd-client-reverso`, boas práticas; legadas Go não são norma | CONFIRMADO | `docs/manual-técnico/skills.md`, FASE-09-1/09-2 |
| Ciclo, gate, branch, DoD e revisão a cada 10 issues | CONFIRMADO | `docs/rust/02-execucao.md` §1–§4 |
| Modelo de issue (Objetivo, Contexto, Passos, Arquivos, Testes, Critérios, Riscos) | CONFIRMADO | `roadmap-rust/issues/FASE-03-10-cidadania-canal-verde.md` |
| Roadmap: 15 fases, ordem estratégica, fila de entregas rápidas | CONFIRMADO | `roadmap-rust/index.md` |
| Checklist funcional (✅🚧⬜⛔), contagem 29/18/36/2 em 2026-11-09 | CONFIRMADO (lido nas linhas 1–213) | `docs/rust/08-checklist-funcionalidades.md` |
| Contagem de migrations/tabelas | RESOLVIDA PELO CÓDIGO (2026-11-10): **36 migrations / 74 tabelas / 147 INSERTs** | Código: 36 arquivos `crates/wyd-db/migrations/0001`–`0036`, `SCHEMA_VERSION = 36`, `cargo test -p wyd-db --test contrato_catalogo` 3/3, `node tools/content/check-migrations.mjs` ok. Os números 31/69 (FASE-03-4, 04-1, 04-2, D-4 e `docs/wyd/README.md`) e 34/72 eram de medições anteriores; histórico não reescrito. `docs/wyd/README.md`, `modelo-banco.md` e `04-revisao-roadmap.md` (D-4) foram alinhados. Falta só `FASE-02-6` (⬜) para o gate impedir nova deriva |
| Estado da auditoria de planejamento (`FASE-01-1`) | CONFIRMADO 🚧 | inventário publicado com 138 arquivos Go; verificação item a item pendente |

Leituras pendentes fechadas nesta rodada:

| Leitura | Resultado |
|---|---|
| `docs/rust/04-revisao-roadmap.md` (completo, §1–§6) | CONFIRMADO: 60 issues; revisão de 2026-11-09 com defeitos D-1..D-10 (D-4 fixa 31 migrations / 69 tabelas; D-5 classifica 72/154 skills; D-10 marca `00-visao-estrutura.md` como snapshot histórico); 9 issues novas (02-6, 03-7, 03-8, 03-9, 03-10, 06-3, 07-5, 07-6, 08-9); perguntas abertas P-1..P-8 em `docs/wyd/referencia-externa-2026-11.md` §9. |
| `docs/manual-técnico/skills.md` (completo) | CONFIRMADO: mapa Go→Rust; `wyd-server-dev` aposentada; `wyd-server-ops`, `wyd-database`, `wyd-client-connect`, `wyd-game-admin`, `wyd-economy`, `wyd-web-admin-ui` marcadas para atualização ou sucessão. |
| `docs/manual-técnico/auditoria-roadmap.md` (completo, 97 linhas) | CONFIRMADO: 138 arquivos Go, 62 abertas, 54 em `done/`, 13 em `test/`; 16 ocorrências C++ com destino (§3.1). §4 deixa a verificação item a item de `done/` aberta (amostra: BOT-01, BOT-02, BUILD-01, CANAL-01, CANAL-03, CHANNEL-01, CHANNEL-02, CI-01); §6 confirma que as 138 fontes não foram lidas integralmente. |
| `docs/rust/03-progresso.md` (linhas 1–240 de 1444) | Lidas: tabela por issue e registro de execução. Linhas 240–1444 não lidas nesta rodada; não afetam os achados de Fase 1. |
| Skills citadas em `AGENTS.md` | CONFIRMADO: todas as 27 pastas citadas existem em `.agents/skills/` com `SKILL.md` (`^name:` verificado). Nenhuma skill citada faltando. Conteúdo dos SKILL.md não lido integralmente (só presença e cabeçalho). |
| Divergência de migrations/tabelas | DIVERGENTE, ainda aberta: 31/69 (`04-revisao-roadmap.md` D-4, `docs/wyd/README.md`) × 36/74 (`05-stack-docker.md`, `banco-catalogo.md`, último `check-migrations`). Fechamento depende de FASE-02-6. |

Fase 1 concluída nas leituras possíveis dentro do repositório, com os limites acima.

### Fase 2 da missão — Conhecimento do WYD (repositório)

Fonte: `wyd2.md` §15 Fase 2 (seis itens). Classificação: CONFIRMADO = lido no arquivo do repositório ou medido aqui; VALIDADO = reproduzido nesta rodada; PROVÁVEL = inferência com evidência parcial; HIPÓTESE = sem evidência ainda; DIVERGENTE = fontes conflitantes; DESCONHECIDO = não investigado.

| Item da Fase 2 | Status | Resultado | Artefato |
|---|---|---|---|
| 1. Estudar o OpenWYD | CONFIRMADO (parcial) | Código Rust em `crates/` (9 crates) e Go legado só como histórico. Protocolo, Hot Path e banco estão documentados em `docs/arquitetura/`. Estudo do código não foi refeito nesta rodada. | `AGENTS.md`, `docs/arquitetura/protocolo.md` |
| 2. Funcionamento original | CONFIRMADO (parcial) | Mecânicas de personagem, skills, itens, canais, cidadania, economia e quests estão em `docs/wyd/` com fonte e confiança por documento. Itens 3.6 (teleportes), 4.2 (fórmulas) e 4.8 (`ESCENE_FIELD`) seguem pendentes no checklist. | `docs/wyd/README.md`, `docs/wyd/sistemas.md`, `docs/rust/08-checklist-funcionalidades.md` |
| 3. Fontes externas | CONFIRMADO | Níveis de confiança A–E e fontes por site em `docs/wyd/fontes.md`. Pesquisa consolidada com URLs e perguntas P-1..P-8 em `docs/wyd/referencia-externa-2026-11.md`. Nenhum número externo vira regra sem teste. | `docs/wyd/fontes.md`, `docs/wyd/referencia-externa-2026-11.md` |
| 4. Cliente do projeto (`client/`) | CONFIRMADO | `client/WYD.exe`: PE32/i386, 2.347.008 B, SHA-256 começa com `04c9d2a1…`, conferido nesta rodada. Dossiê em `docs/cliente/`: inventário, âncoras, formatos decodificados, serverlist decodificado. Trace dinâmico (08-6) segue 🚧. | `docs/cliente/README.md`, `docs/cliente/mapa.md`, `docs/cliente/inventario.md` |
| 5. Comparar com o WYD Global | CONFIRMADO (2026-11-10, leitura somente) | Autorizado pelo usuário. Medido em `C:\Users\Canta\AppData\Local\wyd_launcher\WYD Global`: `WYD.exe` 6.494.000 B (SHA-256 prefixo `a058262d36631fd8`), `ItemList.bin` 2.494.004 B, `SkillData.bin` 25.796 B; mesmos valores da medição de 2026-11-08 em `docs/cliente/referencia-wyd-global.md`. Pasta externa não modificada; não foi lido conteúdo dos binários. | `docs/cliente/referencia-wyd-global.md` §1–§7 |
| 6. Diferenças entre versões | PROVÁVEL / DIVERGENTE | Limite de nível: 400 por camada no 7662 × 800/1500 em servidores privados (`fontes.md` §5). Migrations e tabelas: resolvido pelo código em 36/74 (ver Fase 1). Itens `ItemSet`, `MountData`, `CollectionData`, `Missionitems`, `ArenaVisualData` aparecem no Global e não há evidência de que existam no 7662 como arquivo separado (pergunta, não conclusão). | `docs/cliente/referencia-wyd-global.md` §2, `docs/wyd/referencia-externa-2026-11.md` §1 |

**Comparação `ItemList.bin` e `SkillData.bin` (2026-11-10, leitura somente, XOR `0x5A` após 4 B de cabeçalho):**

- `client/ItemList.bin`: 6.500 registros de 140 B, resto 0. Nomes em texto claro concentram-se no offset 136 do registro (2.685 de 3.705 strings ≥ 4 B). **CONFIRMADO.**
- `WYD Global/ItemList.bin`: **decodificado por hipótese testada**: cabeçalho de 4 B, XOR `0x5A`, registro de **172 B** → 14.500 registros, resto 0. O registro de 140 B (hipótese inicial) deixa resto 40 e não separa nomes (3,0% de nomes terminados em nulo, contra 28,6% a 172 B).

**Hipóteses testadas no Global (medidas em 2026-11-10):**

| Hipótese | Resultado | Veredito |
|---|---|---|
| Cabeçalho 0, 4, 8, 12 ou 16 B, XOR `0x5A`, R=140 | resto 44/40/36/32/28 (nunca zera) | rejeitada |
| Periodicidade do padrão (XOR-independente) | Global: pico em 172 (88,9%); client: pico em 140 (90,0%) | sustenta R=172 |
| Byte dominante após cabeçalho 4 B | Global `0x5A` 60,1% e `0xA4` 23,0% (client: `0x5A` 83,5%) | `0x5A` segue como chave (padding de cabeçalho/registro) |
| Cabeçalho 4 B, R=172 | 14.500 registros, resto 0; nomes ASCII nulo-terminados em 28,6% dos registros (controle R=140: 3,0%) | **CONFIRMADO** (contagem e nomes) |

**Tabela campo a campo, client × Global (`ItemList.bin`):**

| Campo / faixa | Client (140 B) | Global (172 B) | Estado |
|---|---|---|---|
| Cabeçalho | 4 B, XOR `0x5A` | 4 B, XOR `0x5A` | CONFIRMADO |
| Tamanho do registro | 140 B (6.500 reg., resto 0) | 172 B (14.500 reg., resto 0) | DIVERGENTE (valor diferente) |
| Nome (offset 0, texto nulo-terminado) | 37,6% dos registros | 28,6% dos registros | CONFIRMADO no 7662; DIVERGENTE de proporção (o Global tem mais registros com nome vazio/padding) |
| Bloco variável de campos | offsets 0–20 com faixas próprias; padding a partir de 21 | offsets 0–~30 variáveis; padding (`0x00` 93–100%) de 62 a 170 | DIVERGENTE: bloco mais longo no Global, nenhum campo individual decodificado |
| Colunas enum u16 | offsets 22–34, 76–122, 130, 134 (faixas medidas) | nenhuma coluna constante detectada | NÃO DECODIFICADO no Global |
| Campos de nome/texto amplos | strings em 136 (2.685 de 3.705) | sem offset dominante (strings em 0 e 160) | NÃO DECODIFICADO no Global |

Não há documentação existente do formato de registro do Global; a referência lida (`docs/cliente/referencia-wyd-global.md`) traz só tamanhos e hipóteses por nome. Nenhum campo foi inventado. Falta, para decodificar os campos do Global: correlacionar offsets com a tabela de itens do servidor (ids e nomes) e medir inteiros u16/u32 por offset.
- `SkillData.bin`: mesma faixa de periodicidade nos dois lados (período 192), resto 0 no client e 64 no Global. Não decodificado nesta rodada.
- Nenhuma fórmula ou conteúdo de item foi inferido. Nada foi modificado nas pastas.
- Gates (2026-11-10): `node tools/content/check-issue-status.mjs` → `status: 52 issues comparadas | só no índice: 7` / `ok`, exit 0; `node tools/content/check-migrations.mjs` → exit 0.

**Bloqueios desta fase:**

- Campos individuais do `ItemList.bin` do Global seguem NÃO DECODIFICADOS: falta correlacionar offsets com ids/nomes conhecidos. O tamanho de registro (172 B) e o cabeçalho (4 B) já estão confirmados por contagem.
- Captura ao vivo do 7662 (FASE-08-6) exige servidor executável e VM. Fora do alcance desta rodada.
- Fontes externas (sites listados em `fontes.md`) não foram consultadas nesta rodada: a missão pede pesquisa dentro do repositório. Os achados externos já registrados vêm da pesquisa de 2026-11-09.

**Fases restantes:** Fase 2 (item 5 bloqueado; demais itens com parcial); Fase 3 (mapeamento de sistemas por domínio, fórmulas, dependências, evidências, base de conhecimento); Fase 4 (auditoria do Rust contra o mapeamento); Fase 5 (documentação, checklist, issues e matriz de rastreabilidade); Fase 6 (relatório final).

## Fase 2 — processo

| Issue | Status | Evidência / observação |
|---|---|---|
| FASE-02-1 ciclo de execução | ✅ | `docs/rust/02-execucao.md` criado (ciclo, gate, branch, DoD, revisão, doc viva). |
| FASE-02-2 revisão consolidada | ✅ | Checklist e gatilho de 10 issues em `docs/rust/02-execucao.md` §4. |
| FASE-02-3 ordem de priorização | ✅ | Grafo e caminho crítico em `roadmap-rust/index.md` (dependências declaradas por issue). |
| FASE-02-5 visão de pastas + mapa do cliente | ✅ | `docs/arquitetura/visao-pastas.md` (inventário real + tabela-alvo + anti-sprawl) e `docs/cliente/mapa.md` (documentado×lacuna, client↔launcher, plano de desassemble); lacunas registradas (`manual-usuario/` vazia, `tools/dev/` placeholder). |
| FASE-02-4 estrutura e padrões | ✅ | `docs/arquitetura/estrutura-e-padroes.md` + ADR-0002 (árvore, nomenclatura, padrões, SOLID, testes); regra aplicada em `AGENTS.md` §0 e nas skills `software-architecture`, `wyd-rust-net`, `rust-code-quality`. Revisão fria pendente. |
| FASE-02-6 consistência de docs | ✅ | `migrations-check` + `issue-status-check` no `make check`: `DOCUMENTED` 39/80 após `0039_classificacao_e_teto`; 52+ issues comparadas, divergência falha o gate (prova negativa feita). |

## Fase 3 — engine (`wyd-core`)

| Issue | Status | Evidência / observação |
|---|---|---|
| FASE-03-1 tick | ✅ | `tick/scheduler.rs` (acumulador + clamp) + `ports/clock.rs`; fatiamento 7×286 = 50×40. |
| FASE-03-2 ECS | ✅ | Dados prontos (`components.rs` + `ComponentStore` SoA). **Gema ligada ao inventário real:** `ComponentStore::apply_gem` é o caminho único — consome uma unidade da mochila, grava o efeito e publica exatamente um `DomainEvent::GemEffectChanged` por aplicação (zero nas recusas tipadas: `EmptySlot`, `UnknownGem`, `GemRefineTooLow`, `GemRefineTooHigh`, `GemNotInBag`, `EventQueueFull`, `EntityNotAlive`), sem mutação parcial; `tests/integracao/gem.rs` prova Anct estritamente melhor que comum (+9 → 40 de perfuração contra 0) e `tests/golden/gema_seed.rs` prova a **paridade com o seed real** (`0022_seed_refino.sql`, via `include_str!`): as 4 gemas, todo o refino 10..=15, Anct no mínimo−1 contra comum recusado e a troca de efeito com `from`/`to` do catálogo. Fechada em 2026-11-08: `systems/` entregue, ADR-0004 mantém SoA manual e baseline de 10 000 entidades medido (p50 3 381,70 µs). |
| FASE-03-3 spatial grid | ✅ | `spatial/grid.rs`; guerra 5000: 8,6 ms inserts, 79 µs consulta (debug). |
| FASE-03-4 dados estáticos | ✅ | Esquema+seed entregues em **24 migrations** (DDL 0001–0016, seed 0017–0024; hoje **31 / 69 tabelas** após 0025–0031), aplicadas num Postgres 17 real com ciclo up/down/up verde (`make migrations-apply`: 59 tabelas → 0 → 59, contagens idênticas) e schema conferido contra o catálogo (`make banco-verify`); catálogo, mundo, refino, economia, rotas, combate, quests, eventos, arena e progressão modelados. Carga e validação no boot entregues: `ContentSource` em `wyd-db` + `ContentTables::validate` em `wyd-core`, com teste de dado corrompido abortando o boot. O que falta é o **processo** chamar o loader — `FASE-03-7`. |
| FASE-03-5 autoridade do servidor | ✅ | `systems/{movement,combat,skill}.rs` (ADR-0014): a API recebe intenção, nunca resultado — `step` valida orçamento/limites do mundo, `attack` valida alcance/cooldown/alvo e pega o **dano** de `DamageModel` (FASE-10), `cast` valida skill aprendida/MP/cooldown/alcance. **Eixo inventário fechado** (`systems/inventory.rs`, ADR-0015): `equip`/`equip_into` recebem só o slot da mochila e derivam o slot de `item_kind`, conferem `req_level` do `item_catalog`, aplicam política de troca explícita e devolvem o item anterior à mochila; `unequip` usa o primeiro slot livre (mochila cheia recusa) e `move_item` empilha até `max_stack` sem troca silenciosa. `next_attack_ms`/`next_skill_ms` são componentes contíguos; recusa não altera posição, HP, MP, cooldown, item, equipamento nem fila (evento só no aceite). `tests/integracao/autoridade.rs` (10) e `tests/integracao/inventario.rs` (9) provam pacote forjado recusado sem mutação. Mapa das ações em `docs/arquitetura/autoridade-servidor.md`. |
| FASE-03-6 IA de NPC | ✅ | `npc/` (`state`, `behavior`, `brain`, `route`, `budget`) + `systems/npc.rs` (ADR-0016): a IA **emite intenção**, nunca efeito — `MoveTo`/`Attack`/`Flee`/`Return`/`Idle` passam por `movement::step`/`combat::attack`, então uma intenção forjada é recusada com posição e fila intactas. FSM de 7 estados com precedência explícita (fuga → leash → ataque → perseguição), cadência 5 Hz (`tick % 4 == entity % 4`) com `DecisionBudget` por tick, RNG `xorshift64` semeado por `(world_seed, entity)` e tabela de ameaça com teto 8 e desempate pelo menor `entity` (cópia presa por golden da matemática de `wyd-sim`, ADR-0011). Perfil (`NpcProfile`) vem do servidor e é validado no boot — nenhuma constante de balanceamento na engine; `mob_template_stat` vazia deixou de ser bloqueio. `tests/integracao/npc.rs` (12) prova cadência, aggro inclusivo, fuga, leash, recusa atômica (`NotACreature`, `ThreatTableFull`, `EntityNotAlive`) e golden FNV-1a determinístico. **Custo medido** (`cargo bench -p wyd-core --bench tick`, release, 5 execuções): 500 NPCs + 50 jogadores → **88,6–204,5 µs/tick** (p99 119,4–367,8 µs), **0,18%–0,41% do orçamento** de 50 ms a 20 Hz (pior p99 0,74%), 0,71–1,64 µs por decisão; sem alvos, 12,5–31,8 µs/tick. Pior caso ainda com folga de ~13×, então cadência 5 Hz e teto de 133 decisões/tick ficam como estão. |

## Fase 4 — persistência (`wyd-db`)

| Issue | Status | Evidência / observação |
|---|---|---|
| FASE-03-7 motor no canal | ✅ | `world.rs`: um `Engine` do core por processo (`ComponentStore` + grade 4096²/célula 128), `Scheduler` a `WYD_TICK_HZ` (20–50, default 20), spawn/despawn coerentes entre armazém e grade, thread `wyd-tick` com parada limpa (`Drop` → `stop_and_join`) e registro do motor anexado ao `/metrics` **antes** do primeiro scrape. Os três nomes da FASE-06-1 passam a ser medidos (`wyd_ticks_total`, histograma, `wyd_entities_active`), sem I/O no laço. `tests/integracao/tick_processo.rs` (4) prova tick contínuo, métricas no texto do scrape, ausência de tick órfão e recusa de `WYD_TICK_HZ` inválido. **Custo medido** (`cargo bench -p wyd-server --bench tick`, release): 0,38 µs/tick com mundo vazio e 0,73 µs com 1 000 entidades — **0,001% do orçamento** de 50 ms. Personagem como entidade/save: 03-8. |
| FASE-03-8 entrada no mundo | ✅ | `SelectionStore::with_world` + `EngineHandle` (`world.rs`): o `_MSG_CharacterLogin` vira entidade `Player` na posição salva, com stats validados do banco (HP/MP/máx/nível) e recusa tipada para duplicado, estado inválido ou motor fora. Saída (logout, timeout ou queda) tira do mundo e enfileira o snapshot na `SaveQueue`; save periódico de 5 s e dreno de 1 s com flush final no shutdown. Ouro **não** entra no save do mundo (quem escreve é o claim de grant, para não regredir concessão). `tests/sistema/entrada_mundo.rs` prova com banco real: entra na posição 640,480, sai, grava e relê os vitais. |
| FASE-03-10 cidadania/canal verde | ✅ | Papel do canal como **dado**: `channel_role` (migration `0041`, seed `1 = normal`/`2 = verde`) + `WYD_SERVER_CHANNEL_ROLE` (vazio = banco, valor desconhecido ignorado) resolvido no boot por `channels::effective_role`, publicado em `/health` (`role`, `pvp`, `shop`) e logado (`event=channel`). Regra em `channels::ChannelRules`: `pvp_allowed(map)` (verde nunca liga, normal segue o mapa) e `personal_shop_allowed()`. Bônus de cidadania no drop como **parâmetro**: `Channel Settings.citizen_bonus_bps` (`game_setting`, default 0, teto 50%) entra em `Gameplay::loot*` por `citizen` e no peso efetivo via `effective_rate_with`. `_MSG_MessageWhisper` (`/getout`) é tolerado com `event=citizen action=getout` — layout do corpo UNVERIFIED, nenhum pacote inventado. Provas: `tests/integracao/canal_verde.rs` 4/4 (política, env, bônus na mesma seed com roll 600.000 vs peso 500.000, faixa inválida) e `tests/sistema/selchar.rs::sussurro_de_getout_e_tolerado_e_nao_derruba_a_sessao`. **Resíduo declarado:** cidadania ainda derivada do canal (compra na Kibita = diálogo de NPC, 3.5/7.2) e PvP/loja no fio chegam com 4.5/7.2. |
| FASE-03-9 cena autoritativa | ✅ | NPCs + personagens na mesma cena com `MobID` = índice da entidade (espaço único; id de `npc_definition` não vai ao fio): `world::{ActorSkin, SceneMob, SceneSnapshot, plan_scene}`, delta entrar/sair por sessão em `cpsock::refresh_scene` com `HEADER.ID = ESCENE_FIELD` sempre e `RemoveType` 0/1/2 (morte usa o id de cena via `target_scene`), teto `WYD_SCENE_MAX_PER_REFRESH` (default 64, erro fatal no boot; teto atingido não avança a geração). Provas: `world::tests` (ator com nome/equip/guilda, saída, delta, teto, UTF-8) + `cpsock::tests` em fio loopback (`ESCENE_FIELD` no header, `MobID` no corpo, `LOGOUT` no tipo). **Resíduos:** movimento no fio, raio de visão e prova de render no 7662 (captura 08-6). |
| FASE-03-18 portões com chave | 🟡 | **`EF_KEYID` real nos três portões.** `gameplay::{CAMP_GATES, GateBook}`: portões (2075,2016)/(2143,1986)/(2081,1962), chaves 2/3/4 (itens 451/452/453), abertura só com o `EF_KEYID` certo, recusa de segunda chave, fechamento por tempo (30 s, comparado no instante) e consumo da chave gravado pelo `SelectionStore`. `Engine::use_key_at` valida ponto exato + alcance 2 na posição autoritativa; `cpsock::handle_action` tenta portão antes do diálogo. Divergência declarada: o clique chega por `_MSG_Action` (layout provado) e não por `_MSG_UpdateItem`, porque o snapshot não tem item de porta. Prova: `cargo test -p wyd-server --test integracao_campo_treinamento` 9/9 (os dois testes de portão inclusos). |
| FASE-15-4 bots do Campo | 🟡 | **Crate `wyd-bots` criado** (4 raças, protocolo real): handshake completo, `_MSG_Quest` e campanha de quatro etapas validando a ordem pelo `_MSG_MessagePanel`; CLI `--conta/--senha/--raca/--npc`. Leitura de cena (`read_scene` sobre `_MSG_CreateMob`), `attack_at` (`_MSG_AttackOne`), `kill` (ataca até o `_MSG_RemoveMob` do id) e `open_gate` (`_MSG_Action` + painel). Prova: `cargo test -p wyd-bots` 4/4. Resíduo: os NPCs do Campo fora do snapshot e o teste ponta a ponta com fixtures — **pendente** (ver issue FASE-15-4). |
| FASE-03-18 Campo de Treinamento (quest) | 🟡 | **`_MSG_Quest` 0x028B portado** (`payload/quest.rs`, `MSG_STANDARDPARM2` = 20 B) e despachado em `cpsock::handle_quest`, com o `merchant` vindo do mundo (`npc_merchant`) — o corpo do pacote só aponta o NPC. Progresso persistido em `character_quest` (migration `0044`, `SCHEMA_VERSION` 44) com `stage` 0..=4 e `reward_mask` de 1 bit por etapa (reentrega = `AlreadyRewarded`). Motor puro em `gameplay::{quest_engage, apply_quest_reward}` com a **sequência A** (682; `Equip[6]` 50 + ReqLvl; `Equip[1..7]` + sanc 3; sorteio 682/481/652), teto de nível 35 e consumo da prova. `SelectionStore::quest` grava inventário/estado e aplica o affect 44 (200,200) como restauração. Prova: `cargo test -p wyd-server --test integracao_campo_treinamento` **7/7** (fluxo completo, 3 sorteios, nível 36, prova errada, fora de ordem, reentrega, sem arma). Resíduo: NPCs/guardiões/boss no mundo dependem dos templates `STRUCT_MOB` do cliente 7662; portões como objeto (`_MSG_UpdateItem`+`EF_KEYID`) pendentes. |
| FASE-03-17 entrada no mundo fiel | ✅ | **“Falha ao conectar” na entrada tinha causa de contrato.** `world::scene_snapshot` usava o índice do armazém como id de cena; como os 6 100 NPCs nascem primeiro, o primeiro jogador saía com `ClientID > 6100` e o cliente 7662 decide “é personagem” por `id < MAX_USER` (`IsPlayer`, `MaxUser = 1000` no porte Go do mesmo alvo). Agora há duas faixas (`world::{MAX_USER, MAX_MOB}`) com alocação/liberação por entidade, `Engine::enter` devolve `Entered { entity, scene_id }` e a borda usa o id de cena em `ClientID`/`HEADER.ID`/`MobID`. A rajada de entrada ganhou o `UpdateScore` do próprio personagem logo depois do `CNFCharacterLogin` (`sendScoreSelf` do legado) e `CreateType` 2 só para o dono da sessão. Prova: `cargo test -p wyd-server --lib` 42/42 (`ator < MAX_USER`, criatura `>= MAX_USER`, `UpdateScore`, `CreateType`), `tests/sistema/cena_mobs.rs` com as mesmas exigências no fio vivo. **Correção de seleção (2026-11-11):** o cliente 7662 manda `_MSG_CharacterLogin` de **36 B**; o servidor exigia 20 e recusava com `esperado 20, recebido 36` (log do login admin). `CharacterLogin::decode_client` aceita 20/36 lendo `slot`/`force` em 12/16 e descarta os 16 B extras (UNVERIFIED), com teste de protocolo. Resíduo: prova visual no 7662. |
| FASE-03-16 piedade + `SendItem` | ✅ | Piedade em tabela própria `refine_pity` (migration `0043`) lida no `use_item` e gravada na mesma transação (falha acumula, sucesso zera, `replace_items` poda órfãs); resposta é `SendItem` (0x0182, 24 B — `UpdateItem` 0x0374 é porta/door no legado, correção provada por RE + porte); recusa não reenvia nada; B16 só no encoding do par EF_SANC. Provas: `tests/sistema/refino_piedade.rs` 5/5 em banco real + `pg.rs` 8/8 + workspace 69 suítes ok, clippy/fmt/migrations-check verdes (2026-11-10, sem commit). |
| FASE-04-1 modelagem | ✅ | `docs/arquitetura/modelo-banco.md` (ER, decisões) + `docs/arquitetura/banco-catalogo.md` gerado: **69 tabelas**, colunas/tipos/nulidade, PK/FK/CHECK/índices, fonte por tabela e 8 decisões provisórias `D-B01`..`D-B08`; mapa tabela legada → nova com pendências nomeadas. |
| FASE-04-2 migrations | ✅ | 31 migrations reversíveis (0001–0031, uma por domínio, DDL 0001–0016 antes do seed 0017–0031, **sem `ALTER TABLE` no `up`**) + `seeds/dev.sql`; prova em três camadas: estrutural (`check-migrations.mjs`), contrato doc↔SQL (`tests/contrato/catalogo.rs`) e banco real — `verify-schema.mjs` (banco vazio → up → `information_schema` == catálogo) e `migrate-cycle.mjs` (up/down/up com contagens idênticas). Números em `crates/wyd-db/migrations/README.md`. |
| FASE-04-3 persistência async | ✅ | ADR-0021 (substitui parcialmente a ADR-0006): `tokio` + `sqlx` 0.8 no `wyd-db`. Pool com `acquire_timeout` e `SET LOCAL statement_timeout` (`pool/pg.rs`), adapters `Accounts`/`Characters`/`Grants` (`repo/pg.rs`) e fila de save com coalescência + dreno periódico/shutdown (`transfer/save.rs`). **Prova:** `cargo test --workspace` com todos os alvos verdes (0 failed); `cargo clippy --workspace --all-targets -- -D warnings` exit 0; integração real `tests/sistema/pg.rs` **6/6 ok** em Postgres 17 com 28 migrations (66 tabelas) + seed; `enqueue` no Hot Path (N=20 000, dreno real em paralelo) **p50 500 ns · p99 1 600 ns · max 257 µs** — p99 é 0,003% do tick de 50 ms. Runner próprio de migration (`crates/wyd-db/src/migrate.rs`, ledger + checksum + advisory lock) roda no boot desde a ADR-0024; `sqlx::migrate!`/`refinery` seguem descartados por decisão. **Fila durável fechada em 2026-11-09:** `Grants::enqueue` grava na `delivery_queue` com `idempotency_key` UNIQUE e `created_by`, e `Grants::claim_gold` aplica o gold ao personagem e baixa na mesma instrução; o canal drena a cada 1 s. |

## Fases 5–15

| Issue | Status | Observação |
|---|---|---|
| FASE-05-1 pipeline CI | ✅ | `.github/workflows/rust.yml` com job `rust` (`fmt --all --check`, `clippy --all-targets -D warnings`, `build`, `test --all-targets`), job `bench` (compila todos e roda os dois medidos) e job `supply-chain` (`cargo-audit`/`cargo-deny` via `taiki-e/install-action`). `deny.toml` v2 (advisories, licenças fechadas, `wildcards = deny`, fontes só crates.io, `private.ignore`); `Makefile` com `test-all`, `bench-build`, `deny` e `ci` espelhando o workflow; `publish = false` nas 7 crates e versão nas deps internas. **Medido:** bench da IA 205,86 µs/tick = 0,412% do tick (p99 0,840%), piso 29,14 µs; balde 8,1 ns/op e porte 234,3 ns/handshake (wyd-net). `cargo audit` 0 (7 crates, 1422 advisories) e `cargo deny check` 0 (advisories/bans/licenses/sources ok). **Prova de falha:** lockfile descartável com `time 0.1.44` → `cargo audit` exit 1 com RUSTSEC-2020-0071; diretório removido depois. |
| FASE-05-2 release/docker | ✅ | `[profile.release]` com `lto = "thin"`, `codegen-units = 1`, `strip = "symbols"` e `panic = "abort"` (ADR-0017): `wyd-launcher` **323.072 → 296.448 bytes** (−8,2%), build release do workspace exit 0 e binário executando (`check` de manifesto real exit 0). Imagem mínima multi-stage com runtime **distroless `cc-debian12:nonroot`** (sem shell, sem toolchain), labels OCI de versão/commit, `docker build` exit 0 e **37.847.971 bytes** (36,1 MiB); smoke no container: `check /jogo /jogo/manifest.toml` exit 0 (`1 íntegros`), `--help` imprime uso (exit 2 por contrato), `docker export` sem `/bin/sh` nem `rustc`/`cargo`/`gcc`. `.dockerignore` criado; `Makefile` com `release`/`docker`/`docker-smoke` e workflow com job `release`. Bench sem regressão: IA 85,6–189,7 µs/tick (baseline 205,86) e balde 2,7–2,8 ns / porte 100,1–100,8 ns (baseline 8,1 / 234,3). |
| FASE-06-1 telemetria | ✅ | `tracing` 0.1.44 + `tracing-subscriber` 0.3.23 (ADR-0018) e exporter Prometheus próprio (`TcpListener` + texto 0.0.4) em `crates/wyd-core/src/metrics/`; contadores de rede em `crates/wyd-net/src/metrics.rs` atrás da feature `metrics` (default off — o `fetch_add` por pacote custa ~24,5 ns contra 2,7 ns sem ela). Nível por `WYD_LOG` (fallback `RUST_LOG`, default `info`). **Prova em execução:** `WYD_LOG=debug cargo test -p wyd-core --test sistema_telemetry -- --nocapture` loga `DEBUG tick concluido tick=0 micros=2365`… e `INFO resumo de telemetria ticks=40 p50_us=5000 p95_us=5000 p99_us=5000 rss_bytes=0`, e o `GET /metrics` real devolve `wyd_ticks_total 40`, `wyd_tick_duration_microseconds_count 40`, buckets cumulativos com `le="+Inf"} 40` e `wyd_entities_active 104`; com `WYD_LOG=info` nenhuma linha DEBUG sai (0) e valor inválido cai em `info`. `Cargo.lock` 7→24 pacotes; `wyd-launcher` release segue 296.448 bytes e a imagem distroless remede **37.847.971 bytes** (igual à 05-2, launcher não linka o core), smoke no container exit 0. Benches: tick 81,27–82,22 µs (p99 88–116) contra baseline 85,6–189,7; balde 2,7 ns e porte 98,1 ns contra 2,7–2,8 / 100,1–100,8. |
| FASE-06-2 hardening | ✅ | `limits/rate.rs` (balde + porte de handshake) + `tests/integracao/flood.rs`; flood 100/10000, fuzz 5000 sem panic; legit preservado. |
| FASE-06-3 lotação/manutenção | ✅ | `channel_capacity` (migration `0040`, seed 1000 por canal) + `SessionGate::claim` recusa `Capacity`/`Maintenance` antes de abrir sessão; env `WYD_SERVER_MAX_PLAYERS`/`WYD_SERVER_MAINTENANCE` por processo; recusa em `wyd_packets_rejected_total{reason="canal_lotado"\|"manutencao"}` + log; `maintenance` no `/health`. Prova: `tests/sistema/lotacao.rs` 5/5. |
| FASE-06-4 catálogo de ameaças | ⬜ | Issue aberta a partir da pesquisa (`docs/wyd/pesquisa-wyd2-missao-fase3-2026-11-10.md` item 8): matriz B1–B17 + §6 do `wyd2.md`. |
| FASE-07-1 gateway | 🚧 | Backend do contrato (`AdminError`→HTTP, staff/auditoria/grants idempotentes); transporte HTTP/JSON + 72 rotas conhecidas; **binário `wyd-gateway` em Docker** (`:7600`, ADR-0022): `/health` com `SELECT 1`, `/metrics` Prometheus, `item_catalog/list_items` servido da foto real do Postgres (3.220 itens), `/api/health` do portal deixa de ser 502. Imagem distroless com `--healthcheck`. Restam os handlers de domínio, **divididos** em `FASE-07-5` (leitura de conteúdo ✅ em 2026-11-10: `list_npcs`, `list_mob_templates`, `list_effects` com dado real; mapa de atributos e carry de mob seguem `501` por falta de tabela) e `FASE-07-6` (operação auditada ✅ em 2026-11-10: daily/donate/evento/contas/`grant_item`; `501` só onde não há tabela). |
| FASE-07-2 painel usuário | ✅ | Rotas `panel/{self,characters,rankings}` + `accounts/recovery` no roteador do ADR-0008, com `account_id` injetado pelo BFF a partir do cookie assinado (ADR-0019); página `/conta` com conta, personagens, ranking e recuperação. **Prova:** 6 testes de contrato (`tests/contrato/painel.rs`: 401, conta, personagens escopados, ranking ordenado sem `account_id`, recuperação idêntica para login existente/inexistente, parâmetros inválidos → `result: 3`, rota desconhecida → 404) + `typecheck` 0 e vitest 137/137 (19 arquivos). |
| FASE-07-3 painel admin | ✅ | Moderação (papel/bloqueio com motivo e confirmação), trilha `audit_log` persistida pelo gateway e lida em `/auditoria`, economia de fonte real (`city_tax_ledger` + `delivery_queue`) em `/economia`. Migration `0031_moderacao`. **Provado na stack em 2026-11-09** (imagem do gateway reconstruída, rotas por HTTP real com `x-wyd-actor`, travas negadas, `account_moderation`/`audit_log`/economia conferidos no banco, login do portal bloqueado/liberado e `cargo test -p wyd-server --test login` 7/7 contra banco de teste). |
| FASE-07-4 E2E Playwright | ✅ | **Provada na stack (2026-11-09):** suíte `@playwright/test` versionada (`wyd-portal/e2e/`, config + 3 specs) que provisiona banco próprio `wyd_e2e` (31 migrations, role dedicado) e containers efêmeros por `npm run test:e2e`; **3 passed / 0 failed** (login do staff, concessão pela tela `/entregas` com gold 0→250 lido do banco, idempotência com a mesma chave → replay, uma linha e gold 300 sem duplicar). A tela de staff usa o contrato real (`character_admin/search_characters`, `character_admin/list_character_grants`, `grants/request` com token + `x-wyd-actor`). **Aberto por decisão do dono:** compra na loja com moeda premium/saldo (`contrato-persistencia.md` U-P4) e entrega de `item`/`level` (inventário/nível do Hot Path ainda não consome a fila). |
| FASE-08-1 tabela do protocolo | ✅ | Reorganizado na árvore §3 (`wire/`, `codec/`, `payload/`, `error.rs`, `tests/`); `protocolo.md` publicado (HEADER, ofuscação, 198 opcodes gerados, 16 payloads, UNVERIFIED); gate verde (32 lib + 4 integração). |
| FASE-08-2 parsers | ✅ | `repr(C)` + offsets provados; `inspect` (tamanho → opcode) + `require_ported` (gate UNVERIFIED); 9 goldens contra `fixtures/cliente/*.json` (ida-volta byte-idêntica); zero `unsafe`; sem deps novas (zerocopy/nom diferidos p/ 13-2). **Seleção de personagem no canal** (`_MSG_CreateCharacter`/`_MSG_DeleteCharacter`/`_MSG_CharacterLogin`) com autoridade no Postgres e sessão que sobrevive ao handshake (`crates/wyd-server/src/{characters.rs,cpsock.rs}`, `tests/sistema/selecao.rs`). |
| FASE-08-3 distribuição | ✅ | `distribuicao.rs` + `update`/`--dry-run`; 27/27 testes; gate verde; e2e em tmp (aplica+backup+check verde, adulterado recusado exit 1). |
| FASE-08-4 dossiê do cliente | ✅ | `docs/cliente/` publicado (README, inventário, superfície conhecida, plano de RE); baseline de `WYD.exe` 7662 com SHA-256 `04c9d2a1…` e PE32/i386; skill `wyd-client-reverso`. Desassemble é 08-5/08-6. |
| FASE-08-5 RE estático | ✅ | `tools/client/` (7 módulos) + `docs/cliente/gerado/` (PE, strings, evidência, formatos); **6/6 âncoras** e **3/3 patches** confirmados por bytes + objdump; os 3 sites de checksum já estão `EB` no disco; `.wyt`/`.msa`/`sn.bin` com estrutura inicial; `serverlist.bin` opaco. T2 (structs) movido para 08-6. |
| FASE-08-6 RE dinâmico | 🚧 | Roteiro em `docs/cliente/depuracao.md` + `captura.mjs` (`--gen`/`--check`, 5/5 goldens gerados C→S, determinístico). Captura ao vivo bloqueada: exige servidor executável + VM. |
| FASE-08-7 customização do cliente | ✅ | `manifest/patch/bootstrap/sha256` + CLI `check`/`launch`; 19/19 testes; `check` real 6/6 OK. Spawn real pendente de ambiente com jogo (registrado na issue). |
| FASE-08-8 GUI Tauri | 🚧 | `gui.rs` gera `LauncherGUI.html` a partir da identidade; shell Tauri (janela, updater, instalador) ainda não existe — FASE-08-8. |
| FASE-08-7 customização | ✅ | `manifest/patch/bootstrap/sha256.rs` + CLI `check`/`launch`, 19/19 testes, `check` real 6/6 OK; spawn real pendente de ambiente com jogo. |
| FASE-08-9 logout explícito | ✅ | `CharacterLogout` (0x0215) → `CNFCharacterLogout` + `leave_world` + `Drop` (`cpsock.rs:1030-1045`); queda loga `reason=socket`. Prova: `tests/sistema/selchar.rs::logout_explicito_fecha_a_conexao_e_libera_a_sessao` (CNF + EOF + `live_session_of=None` ≤ 5 s). Golden do frame real pendente da captura 08-6. |
| FASE-08-10 trade atômica | ⛔ | Issue aberta bloqueada em layout (`MSG_Trade` UNVERIFIED até a captura 08-6); codar sem captura proibido. |
| FASE-09-1 auditoria de skills | ✅ | Relatório em `roadmap-rust/issues/FASE-09-1-auditoria-skills.md` (7 skills com destino + trechos obsoletos + regras atemporais; plano da 09-2). |
| FASE-09-2 skills sucessoras | ✅ | `.agents/skills/` com 27 pastas: `wyd-rust-*` vigentes, legadas Go aposentadas/aliases (`wyd-server-dev` APOSENTADA, `wyd-testing-bots` alias), mesmas regras atemporais preservadas; evidência na própria issue. |
| FASE-10-1 equações | 🚧 | Composição dos 9 sistemas codificada em `crates/wyd-sim/src/formula/` (atributos, progressão, dano, acerto/crítico, efeitos, resistência, drop, regen, aggro) + ordem do tick em `scenario/combat.rs`; a **ordem exata** de mitigação/crítico é hipótese D-02..D-04 (`docs/arquitetura/matematica-sim.md` §4) até o trace da FASE-08-6. Determinismo provado em `tests/integracao/determinism.rs`. |
| FASE-10-4 EXP por flags | ⬜ | Issue aberta a partir da pesquisa (item 5): árvore L3 por trás de flags em `game_setting`, default = comportamento atual. |
| FASE-10-2 velocidade | ⬜ | Depende de 10-1. |
| FASE-10-3 laboratório | ⬜ | Depende de 10-1, 10-2. |
| FASE-11-1 guia + manuais | ✅ | `docs/manual-técnico/guia-contribuicao.md` + `docs/manual-usuario/README.md` publicados, links do README conferidos. |
| FASE-11-2 C4 + pacotes | ✅ | `docs/arquitetura/c4-pacotes.md` (N1–N3 ancorados no código) + specs de pacotes por referência ao `protocolo.md`. |
| FASE-12-1 dashboards | ✅ | Stack de monitoramento em Docker (`monitoring/`), 8 alertas com runbook, dashboard `wyd-saude` com 9 painéis. **Série real do gateway** (`wyd_gateway_requests_total`, `wyd_gateway_uptime_seconds`) desde a ADR-0022; painéis do motor seguem vazios até existir `tmserver`. |
| FASE-13-1 rito ADR | ✅ | `docs/arquitetura/adr/0000-template.md` criado (estado, contexto, decisão, consequências, alternativas, rastreabilidade); `0001` promovida a **aceita** no formato do template. |
| FASE-13-2 primeiras ADRs | ✅ | ADR-0004 (SoA manual), 0005 (LE+`repr(C)`), 0006 (std até rede/vendor); 7 arquivos 0000–0006 sem lacuna, seções conformes; consumidoras vinculadas. |
| FASE-14-1 pesquisa | ✅ | Notas datadas em `docs/rust/01-dependencias.md` (manter/descartar por candidato + PoC fora de `crates/`; sem diff no workspace). |
| FASE-15-1 suite + concorrência | ⬜ | Depende de 05-1, 10-3. |
| FASE-15-2 carga + banco + release | ⬜ | Depende de 15-1, 12-1. |
| FASE-15-3 bots e carga | ⬜ | `crates/wyd-bots` planejado (ADR-0010); hoje o harness cobre protocolo, não sessão jogável. |
| FASE-15-4 bots do Campo de Treinamento | 🟡 | 4 bots autônomos, um por raça, executando `FASE-03-18` pelo fio; depende de 15-3 e 03-18. Referência de estados em `main:tmserver/internal/bot/client.go`. |

## Fila de entregas rápidas (aberta em 2026-11-09)

Ordem por tamanho e desbloqueio (tabela completa em `roadmap-rust/index.md`):

| # | Issue | Tamanho | Depende de |
|---|---|---|---|
| 1 | FASE-02-6 consistência de docs ✅ | S | — |
| 2 | FASE-06-3 lotação/manutenção ✅ | S | 06-2 ✅ |
| 3 | FASE-08-9 logout explícito ✅ | S | 08-2 ✅ |
| 4 | FASE-03-7 motor no canal ✅ | M | 03-6 ✅, 04-3 ✅ |
| 5 | FASE-07-5 handlers de conteúdo ✅ | M | 07-1 🚧 |
| 6 | FASE-07-6 handlers de operação ✅ | M | 07-1 🚧, 07-3 ✅ |
| 7 | FASE-03-8 entrada no mundo ✅ | M | 03-7 |
| 8 | FASE-03-10 cidadania/canal verde ✅ | M | 03-7 |
| 9 | FASE-03-9 cena autoritativa ✅ | L | 03-8 ✅ |

Nenhuma delas espera captura de cliente, decisão do dono ou serviço externo. O único bloqueio
externo da árvore continua sendo `FASE-08-6` (VM + cliente em execução).

## Registro de execução

| Data | Issue | O que mudou | Evidência |
| 2026-11-11 | Gate de testes do `wyd-server` com banco de teste | **O gate de sistema passou a rodar com Postgres real, não com `SKIP:`.** Banco `wyd_test` isolado no container `wyd-db` (48 migrations, sem tocar no `wyd` de jogo). A primeira execução completa expôs testes de sistema que nunca haviam rodado (só pulavam sem DSN): `sistema_login` esperava recusa silenciosa e cadastro de PIN na primeira entrada; `portal_pin`/`pin_troca` idem; `personagens_crud` esperava 5/3 pontos no nível 1; `selchar` não drenava `UpdateScore`/`UpdateEtc` de entrada e esperava 10/6 no level-up. Todos foram alinhados ao contrato vigente (BUG-02, `NewAccountFail` e pontos por nível), sem mudar comportamento de produção. | `WYD_TEST_DSN=postgres://…/wyd_test cargo test -p wyd-server --all-targets -j 1` → **EXIT 0**, **146 passed / 0 failed / 0 ignored**, 6 `SKIP:` só de `WYD_LIVE_CPSOCK` (servidor vivo); `cargo fmt --all --check` 0 · `cargo clippy -p wyd-server --all-targets -D warnings` 0 · sem commit |
| 2026-11-11 | ADR-0026 etapa 3 (`world/`) | **O `world.rs` de 2 363 linhas deixou de ser um arquivo único.** `world/` agora tem `mod.rs` (Engine/tipos/`EngineHandle`/testes), `tick.rs` (`EngineThread`, `run_loop`, movimento e avanço do relógio), `scene.rs` (spawn/despawn, entrada/saída, cena e ids `MobID`), `npc.rs` (`npc_specs`, `sync_npcs`, diálogo), `combat.rs` (golpe autoritativo e dano por template) e `save.rs` (`SaveSink`/`SaveQueueSink`/snapshots). API pública preservada (`Engine`, `EngineThread`, `SaveSink`, `SaveQueueSink`, `plan_scene`, `npc_specs` e tipos) e nenhuma lógica editada, só movida. | `cargo fmt --all --check` 0 · `cargo clippy -p wyd-server --all-targets -D warnings` 0 · `cargo test -p wyd-server --lib -j 1` 54/54 · `integracao_tick_processo` 4/4 · `sistema_entrada_mundo` 1/1 · `sistema_cena_mobs` 1/1 · `integracao_config_gameplay` 9/9 · sem commit |
| 2026-11-11 | ADR-0026 etapa 2 (`cpsock/`) | **O `cpsock.rs` de 2 862 linhas deixou de ser um arquivo único.** `cpsock/` agora tem `mod.rs` (accept loop, `Stage`/`Scene`, respostas e testes de fio), `handshake.rs` (INITCODE/frame/checksum/versão), `login.rs` (senha/PIN e cerca de sessão), `selection.rs` (seleção e Create/Delete/CharacterLogin), `action.rs` (dispatch de item/ataque/diálogo/chat/cena) e `limits.rs` (limite por IP). API pública preservada (`serve_listener`, `serve_listener_with_limit`, `ConnectionLimit`, `HandshakeOk`, `HandshakeError`, `send_progress`) e nenhuma lógica editada, só movida. | `cargo fmt --all --check` 0 · `cargo clippy -p wyd-server --all-targets -D warnings` 0 · `cargo test -p wyd-server --lib -j 1` 54/54 · `sistema_handshake` 5/5 · `sistema_limite_ip` 1/1 · `sistema_selchar` 9/9 · `sistema_selecao` 4/4 · sem commit |
| 2026-11-11 | ADR-0026 etapa 1 (`boot/`, `auth/`) | **O `main.rs` do canal deixou de ser o dono do boot.** `boot/` passou a concentrar `main()`/`run()` (env → migrations → pool → listeners → loops), `env.rs` (binds/defaults) e `healthcheck.rs`; `auth/` virou domínio com `mod.rs` (fachada `Authenticator`/`AuthError`/`PinOutcome`), `password.rs` (Argon2id) e `pin.rs` (política 4..=6, cadastro/troca, bloqueio persistido). `main.rs` tem 8 linhas e só chama `boot::main()`. API pública preservada: `wyd_server::auth::{Authenticator, AuthError, PinOutcome, MAX_PIN_FAILURES}`. | `cargo fmt --all --check` 0 · `cargo clippy -p wyd-server --all-targets -D warnings` 0 · `cargo test -p wyd-server --lib -j 1` 54/54 · sem commit |
| 2026-11-11 | Qualidade e organização dos testes | **A suíte deixou de ser um bloco único.** 173 arquivos/731 testes inventariados e classificados em `docs/arquitetura/testes.md`: 116 arquivos/465 testes de unidade, 57/266 em `tests/` (16 integração, 10 contrato, 5 golden, 26 sistema). Os 57 arquivos foram movidos para `tests/{integracao,contrato,golden,sistema}/` com `[[test]]` explícito no `Cargo.toml` (o Cargo não descobre subpastas) e o nome do alvo preservado (`--test sistema_login` continua válido); referências em Makefile, CI e docs atualizadas. Gate mudo corrigido em `sistema_selecao` e `contrato_identidade`; `skip:` minúsculo padronizado em `SKIP:`. Skill nova `rust-test-quality` e `wyd-rust-testing` apontando para a taxonomia. Dívida restante priorizada: DSN duplicado, sleeps de sincronização, monólitos (`sistema_selchar`, `sistema_cena_mobs`, `contrato_operacao`) e testes em `main.rs`. | `cargo fmt --all --check` 0 · `cargo clippy -p wyd-server -p wyd-launcher --all-targets -D warnings` 0 · `sistema_personagens_crud` 2/2 e `sistema_selecao` 4/4 (sem infra: `SKIP:` explícito, não provam comportamento) · `contrato_identidade` 4/4 · `contrato_monitoramento` 4/4 |
| 2026-11-10 | FASE-03-17 (prova viva no canal) | **A entrada no mundo passou a ser observável no fio.** Stack reconstruída com o código novo (`docker compose -f docker-compose.yml -f docker-compose.override.yml up -d --build server-canal-1`), conta `qavivo01` provisionada com senha argon2id e personagem no slot 0 (`tests/sistema/conta_de_teste.rs`). O log do canal mostra `entity=6100 client_id=1`: a entidade interna continua no índice alto do armazém, mas o `ClientID` que vai ao cliente está na faixa de jogador — exatamente a divergência que causava “Falha ao conectar”. Rajada observada: `handshake accepted version=12000 channels=2 kw=11` → `login password_ok` → `session open` → `character ok action=select ... client_id=1` → cena com **1281 `CreateMob` de 232 B** (NPCs `>= 1000`, self `CreateType=2`) e `UpdateScore` de entrada; `_MSG_Action` respondeu `npc=1212 Guarda_Carga merchant=2`. | `WYD_LIVE_CPSOCK=127.0.0.1:8281 WYD_LIVE_ACCOUNT=qavivo01 ... cargo test -p wyd-server --test sistema_cena_mobs -- --nocapture` → `1 passed` (`PASS(cena): 1281 NPCs visíveis ... jogador em (2103, 2050)`; `PASS(cena): diálogo respondeu "Guarda_Carga"`); `docker logs wyd-server-canal-1-1` com as linhas citadas |
| 2026-11-10 | FASE-03-17 | **Entrada no mundo fiel ao legado.** Faixa de id de cena por entidade (`1..MAX_USER` jogador, `MAX_USER..MAX_MOB` criatura) com `Entered { entity, scene_id }`; `UpdateScore` do dono entre o `CNFCharacterLogin` e a cena; `CreateType` 2 só para quem entra. Pesquisa do Campo de Treinamento publicada em `docs/wyd/pesquisa-campo-de-treinamento-2026-11-10.md`; issues `FASE-03-18` (quest) e `FASE-15-4` (bots) abertas | `cargo fmt --all --check` 0 · `cargo clippy --workspace --all-targets -D warnings` 0 · `cargo check --workspace --all-targets` 0 · `cargo test -p wyd-server --lib` 42/42 · `cargo test -p wyd-protocol` 0 falhas · prova visual no 7662 pendente |
| 2026-11-10 | FASE-07-6 | Handlers de operação no gateway com Postgres real: `daily_reward_admin` (lista/upsert/delete em `daily_reward_item`, teto 200, cursor), `donate_admin` (loja em `donate_shop_item` + `credit_donate_balance` idempotente em `account_wallet` via `donate_credit` `0042`, sem `ALTER`), `world_event_admin` (foto com horários + liga/desliga), `account_admin` (lista/conta sem segredo) e `character_admin/grant_item` (fila idempotente + dreno `claim_item` → slot livre de `Carry` 16–79, mochila cheia pende com `attempts`); `daily set_reward_item_enabled` e `donate_revenue_admin/*` seguem `501` (sem tabela/coluna; D-B07); escrita audita autor/alvo, leitura não | `cargo test -p wyd-admin --test contrato_operacao` 12/12 (7 contrato + 5 Postgres real em container descartável 15432) · `cargo test --workspace --all-targets` 60 alvos ok · `cargo fmt --check` 0 · `cargo clippy --workspace --all-targets -D warnings` 0 · `check-migrations` 42/83/152 ok · `check-issue-status` ok · `gen-catalogo-banco --check` ok (`SCHEMA_VERSION` = 42) · sem commit/push |
| 2026-11-10 | FASE-07-5 | Leituras de conteúdo do gateway com dado real: `config_npc_page`, `config_mob_templates` (de `npc_template`), `config_effects` (símbolo `EF_*`); `ConfigStore` + dispatch para `list_npcs`/`list_mob_templates`/`list_effects` (teto 200, cursor, `result:3` fora da faixa, sem auditoria de leitura); `get_attribute_map_info`/`list_mob_carry` seguem `501` (sem tabela; D-B07) | `cargo test -p wyd-admin --test contrato_conteudo` 7/7 (contrato + Postgres real 6.100/1.991/55 em container descartável) · `cargo test --workspace --all-targets` 59 alvos ok · `fmt` 0 · `clippy -D warnings` 0 · `check-migrations` 41/82/152 ok · `check-issue-status` 61/0 ok |
| 2026-11-10 | FASE-03-9 | Cena autoritativa: jogadores entram na cena (`ActorSkin` do `STRUCT_MOB` do login), `MobID` = índice da entidade para NPC e jogador (sem colisão), delta entrar/sair com teto `WYD_SCENE_MAX_PER_REFRESH` (default 64, fatal no boot) e `RemoveType` 0/1/2, morte com id de cena; checklist 1.11/4.8 ✅, 10.4 🚧; resíduos: movimento no fio, raio de visão, render no 7662 (08-6) | `cargo test -p wyd-server --lib` 38/38 (ator, delta, teto, UTF-8, fio loopback com `ESCENE_FIELD`) · `cargo test --workspace --all-targets` 58 alvos ok, 0 falhas · `cargo fmt --check` 0 · `cargo clippy --workspace --all-targets -D warnings` 0 · `check-migrations` 41/82/152 ok · `check-issue-status` 60/0 ok · `gen-catalogo-banco --check` ok (sem esquema novo, `SCHEMA_VERSION` = 41) |
| 2026-11-10 | FASE-03-10 | Cidadania aplicada: papel do canal em `channel_role` (`0041`) com env de sobrescrita, `/health` publicando `role`/`pvp`/`shop`, `ChannelRules` decidindo PvP/loja, bônus de cidadania como parâmetro (`Channel Settings.citizen_bonus_bps`, default 0) no drop, `/getout` tolerado sem layout inventado; checklist 2.3 ✅ | `cargo test -p wyd-server --test integracao_canal_verde` 4/4 · `cargo test -p wyd-sim --lib` 95/95 · `cargo test -p wyd-db --test contrato_catalogo` 3/3 · `cargo test -p wyd-server` 23 alvos ok · `cargo fmt --check` 0 · `cargo clippy -D warnings` 0 · `check-migrations` 41/82/152 ok · `gen-catalogo-banco --check` ok |
| 2026-11-10 | FASE-08-9 | Logout explícito: opcode `0x0215` provado na tabela congelada, handler responde `CNF` e encerra (`cpsock.rs:1030-1045`), queda de socket loga `reason=socket`; checklist 1.13 ✅, `pacotes.md` 4b com prova | `cargo test -p wyd-server --test sistema_selchar logout_explicito` (pula com `SKIP:` sem DSN; prova real com banco) · `cargo fmt --check` 0 · `cargo clippy -D warnings` 0 · `check-issue-status` ok |
| 2026-11-10 | FASE-06-3 | Lotação e manutenção do canal: tabela `channel_capacity` + seed 1000, gates em `SessionGate::claim` (`Capacity{limit}`/`Maintenance`), motivos próprios no `cpsock` e no Prometheus, env por processo com aviso de boot para teto < 10, `maintenance` no `/health`, env nos dois composes; checklist 1.14 ✅ | `cargo test -p wyd-server --test sistema_lotacao` 5/5 · `cargo fmt --check` 0 · `cargo clippy -D warnings` 0 · `cargo test --workspace --all-targets` 57 suítes ok · `check-migrations` 40/81/150 ok · `check-issue-status` ok |
| 2026-11-10 | FASE-02-6 | Fechamento do gate: `DOCUMENTED` 39/80 após `0039_classificacao_e_teto`, `banco-catalogo.md` alinhado, 4 issues da pesquisa no índice e no painel, parágrafo pós-fila atualizado | `node tools/content/check-migrations.mjs` exit 0 · `node tools/content/check-issue-status.mjs` exit 0 (só no índice: avisos, sem divergência) |
|---|---|---|---|
| 2026-11-09 | FASE-03-8 | Entrada no mundo: `CharacterLogin` encarna o personagem na posição salva, saída/queda salva o estado vivo pela `SaveQueue`, save periódico de 5 s | `cargo test -p wyd-server --test sistema_entrada_mundo` (banco real) 1/1 · `tests/tick_processo` 4/4 · `clippy -D warnings` 0 |
| 2026-11-09 | FASE-03-7 | Motor no canal: `Engine` do core rodando na thread `wyd-tick`, spawn/despawn coerentes, métricas de motor reais no `/metrics` e parada limpa | `cargo test -p wyd-server --test integracao_tick_processo` 4/4 · `cargo bench -p wyd-server --bench tick` 0,38 µs (vazio) e 0,73 µs (1 000 entidades) · `clippy -D warnings` 0 · `/metrics` de processo real com `wyd_ticks_total` crescente |
| 2026-11-09 | **Revisão** (02-6, 03-7…03-10, 06-3, 07-5/07-6, 08-9) | Mega revisão do roadmap: 9 issues novas com fila por tamanho, status reconciliado nos três lugares, `wyd-server` na árvore normativa, 4 testes de integração deixam de panicar sem DSN, contagens medidas no banco (31 migrations, 69 tabelas + ledger, 72/154 skills) | `cargo fmt --check` **0** · `cargo clippy -D warnings` **0** · `cargo test --workspace --all-targets` **0** (49 alvos ok, 0 falhas) · `psql` em 2026-11-09 · detalhes em `04-revisao-roadmap.md` §6 |
| 2026-11-09 | FASE-07-4 (E2E Playwright + tela de entregas) | **E2E real fechado.** O gateway passou a servir `character_admin/search_characters` (personagens vivos por nome, com `gold` atual) e `character_admin/list_character_grants` (fila do personagem com estado, somente leitura), antes 501. O portal ganhou `/entregas` (menu Operação, guarda de rota no `matcher`): busca personagem, enfileira ouro com chave de idempotência visível e mostra a fila com `pending`/`entregue`. A chave também pode vir de `?key=` para retry sobreviver ao refresh do server action. `@playwright/test` + `scripts/e2e.mjs` provisionam o banco `wyd_e2e` (role próprio, migrations reais) e containers do gateway/canal/portal (imagem do compose em 3110, sem disputar a 3000); o teardown derruba tudo e o exit do Playwright é o exit do script. **Stack reconstruída** (`docker compose build` + `up -d`) com portal em 3000 | **E2E:** `npm run test:e2e` → `3 passed (23.4s)`, 0 failed: `entregas.spec` (gold 0→250 na tela), `idempotencia.spec` (mesma chave → replay, 1 linha, gold 300) e `login.spec` (sem sessão → `/login`; staff entra). **HTTP real:** `/health` → `{"ok":true,"database":"up","schema":31}`; `/entregas` na 3000 responde 307 para `/login` sem sessão (guarda ativa). **Testes:** `cargo test -p wyd-db -p wyd-admin` 107/0; `cargo test -p wyd-db --test sistema_pg` 8/8 com `WYD_TEST_DSN`; `cargo test -p wyd-server` 39/0; `check-migrations` 31/69/147 ok; `tsc --noEmit` 0; vitest 86/86 |
| 2026-11-09 | FASE-07-4 (pré-requisito: fila de grants durável) | **A entrega web→jogo deixou de ser em memória.** `delivery_queue` (0006) ganhou `idempotency_key TEXT UNIQUE` e `created_by BIGINT REFERENCES accounts(id)`; `Grants::enqueue` resolve a conta pelo personagem (nunca pelo corpo), faz `INSERT ... ON CONFLICT (idempotency_key) DO NOTHING RETURNING id` e, no conflito, devolve o `id` existente com `replay=true`; `Grants::claim_gold` credita o personagem e marca `claimed` em uma única instrução (`UPDATE ... FROM alvo RETURNING c.gold`), então não existe gold entregue sem baixa nem baixa sem gold. `GrantService` trocou a fila em memória pela porta `GrantQueue` (teste usa `MemoryGrantQueue` com a mesma semântica); o gateway liga `Blocking` ao pool e o binário do canal drena a cada 1 s (`event=grant delivered`). Item/nível continuam pendentes de propósito: o inventário/nível do Hot Path ainda não consome a fila | **Stack:** imagem `wyd/gateway:dev` 15:52:06 e `wyd/server:dev` 15:51:26; `/health` → `{"ok":true,"database":"up","schema":31}`. HTTP com token + `x-wyd-actor: 1:admin`: chave `prova-07-4-k1` (gold 123) → `{"result":1,"grant_id":2,"replay":false}`; mesma chave → `grant_id:2,"replay":true`; chave `k2` (gold 7) → `grant_id:4,"replay":false`. Banco: `id=2 account_id=49 character_id=2 gold=123 status=claimed idempotency_key=prova-07-4-k1 created_by=1`, `id=4 … gold=7`, e `count(*)=1` para k1. Consumo: canal loga `delivered kind=gold queue=1 character=2 gold=100`, `queue=2 … gold=223`, `queue=4 … gold=230`; `characters.gold` do personagem 2 passou de 0 → 230 (100 antigo + 123 + 7) e as três linhas ficaram `claimed`. Replay após entrega manteve `grant_id:2` e uma única linha. Testes: `cargo test -p wyd-db --test sistema_pg` 8/8 com `WYD_TEST_DSN` (novo `enqueue_durável_e_idempotente_e_gold_entregue_uma_vez`), `cargo test -p wyd-db -p wyd-admin` 107/0, `cargo test -p wyd-server` 39/0 com DSN |
| 2026-11-09 | FASE-07-3 (prova em execução na stack) | **A fase deixou de existir só no código.** Gateway reconstruído do fonte atual e subido sem tocar nos canais de jogo (16 sessões vivas, último save `16:14:29Z` e `characters.updated_at` `14:58Z`, mais de 1 h antes do restart; a migration `0031` já aplicada não foi reexecutada nem apagada). As cinco rotas responderam o contrato com ator de staff vindo do BFF: `accounts/get_moderation` (`result:1`, conta 6 `test5`), `accounts/set_role` (`test4` `player→moderator→player`, `result:1`), `accounts/set_blocked` (moderador 3 bloqueia `test5` com motivo, `result:1`, liberação depois), `audit_admin/recent` e `economy_admin/summary`. **Travas provadas por HTTP:** jogador → `result:2` "papel de staff necessário"; moderador bloqueando staff (`test1`) → `result:2` "só admin bloqueia staff"; bloqueio sem motivo → `result:3`; admin trocando o próprio papel → `result:5`; moderador trocando papel → `result:2`; admin bloqueando a própria conta → `result:5`; moderador lendo a trilha → `result:2`; `x-wyd-actor` malformado → HTTP `400`. **Efeito no banco:** `account_moderation` = `6\|true\|prova 07-3 rodada 2\|3` (blocked, motivo, autor, `blocked_at`) e volta a `blocked=false` na liberação; `audit_log` guarda as ações com o autor real (`1:admin accounts.set_role`, `3:moderator accounts.block/accounts.unblock`) e a negação `3:moderator auth.denied:accounts.set_role`; economia veio das tabelas-fonte (`city_tax_ledger`: `armia` 700 + `noatun` 1500; `delivery_queue`: `gold` 1 × 100). **Login:** conta-sonda criada por `--seed-staff` provou `verify_credentials` `ok:true` → bloqueio → `ok:false, blocked:true` → senha errada na conta bloqueada `blocked:false` (não vaza) → desbloqueio → `ok:true` (conta removida ao fim); no jogo, `cargo test -p wyd-server --test login` rodou **7/7** contra `wyd_test_login` (31 migrations), incluindo `conta_bloqueada_e_recusada_e_volta_a_logar_apos_desbloqueio`. **Defeitos encontrados na execução e corrigidos:** (a) `set_blocked` devolvia o estado **anterior** — no PostgreSQL as sub-instruções de um `WITH` compartilham o snapshot, então reler `account_moderation` não via o upsert; agora `blocked/reason/blocked_at` vêm do `RETURNING`; (b) `log_denied` gravava `id:player` fixo e perdia o papel real do ator (moderador negado virava `3:player`); agora recebe o papel da sessão; (c) o gate de login entrava em panic sem DSN e colidia `login` em testes paralelos (relógio do Windows) — `skip:` explícito + sequência atômica; `personagens_crud` também passou a pular explicitamente sem DSN; (d) `trace.test.ts` foi atualizado para o terceiro argumento `actor` do `promisify` | Imagem `wyd/gateway:dev` criada `2026-11-09T18:20:02Z`; container `StartedAt 18:20:52Z`, `/health` → `{"ok":true,"database":"up","schema":31}`. HTTP (token do portal + `x-wyd-actor`): `200 {\"result\":1,\"moderation\":{\"account\":{\"id\":6,\"login\":\"test5\",\"role\":\"player\"},\"blocked\":true,\"reason\":\"prova 07-3 rodada 2\",\"blocked_at_ms\":1794248468519}}`; `economy` → `{\"tax_by_city\":[{\"city_code\":\"armia\",\"total\":700},{\"city_code\":\"noatun\",\"total\":1500}],\"grants\":[{\"kind\":\"gold\",\"count\":1,\"gold\":100}]}`. SQL: `select account_id,blocked,reason,blocked_by,blocked_at from account_moderation` → `6 \| t \| prova 07-3 rodada 2 \| 3 \| 2026-11-09 18:09:21.988049+00`; `select actor,action,target from audit_log` → `3:moderator \| auth.denied:accounts.set_role`, `1:admin \| accounts.set_role \| account:5`. Login: `{\"ok\":false,\"account_id\":0,\"blocked\":true,\"role\":\"\"}` com senha certa e `blocked:false` com senha errada. Testes: `cargo test -p wyd-server --test login` → 7 passed em 29 s contra `postgres://wyd_test:***@127.0.0.1:5432/wyd_test_login`; `cargo test -p wyd-server` (com DSN) → 12+5+1+7+1+3+1+1+1+4+1+2 passed, 0 failed; sem DSN → `skip:` e verde nos arquivos ajustados. Gates: `cargo fmt --check` 0; `cargo clippy -p wyd-db -p wyd-admin -p wyd-server --all-targets -D warnings` 0; `cargo test -p wyd-db -p wyd-admin` 106 passed/0 failed; `check-migrations` 31 migrations/69 tabelas/147 INSERTs ok; `tsc --noEmit` 0; `vitest` 86/86 (13 arquivos) |
| 2026-11-09 | FASE-07-3 (painel de staff auditável) | **Moderação, auditoria e economia com fonte real no painel.** Nova tabela `account_moderation` (0031) guarda bloqueio + motivo + autor, separada da identidade; `CredentialRecord.blocked` faz o login do portal **e** do jogo recusar antes de comparar senha/PIN (`crates/wyd-server/src/auth.rs` → `AuthError::Blocked`). Portas novas no `wyd-db` (`AuditTrail` grava/lê `audit_log`; `Economy` soma `city_tax_ledger` e conta `delivery_queue`), com ADR-0023. O gateway troca o `StdoutSink` por `PersistentSink` (grava `audit_log` + stdout), expõe `accounts/get_moderation|set_role|set_blocked`, `audit_admin/recent` e `economy_admin/summary`; travas: só admin troca papel/lê trilha, ninguém altera o próprio papel nem bloqueia a própria conta, moderador não bloqueia staff, bloqueio exige motivo. Portal: `/contas` ganhou painel de moderação com diálogo de confirmação, `/auditoria` (admin) e `/economia` (staff) nasceram com link no menu, guarda no `matcher` e README | `cargo fmt --check` 0; `cargo clippy -p wyd-db -p wyd-admin --all-targets -D warnings` 0; `cargo test -p wyd-db -p wyd-admin` 0 (inclui `contract` 11/11 e os 5 testes de moderação); `node tools/content/check-migrations.mjs` → 31 migrations ok; `node tools/content/gen-catalogo-banco.mjs` → 69 tabelas; `npx tsc --noEmit` 0; `npx vitest run` 85/85 |
| 2026-11-09 | FASE-12-1 (durabilidade do log de login) | **Prova ponta a ponta de que o log dos canais sobrevive à recriação do container.** Cadeia: `container → Alloy (socket RO) → Loki (volume wyd-loki-data, 14 d) → BFF do portal`. Linhas reais de `event=login` geradas pelo `_MSG_AccountLogin` do cliente 7662: **00:45:03Z** `account=test1 channel=1 session=5` e **00:45:04Z** `account=test2 channel=2 session=6`. Recriação só dos canais (`docker compose up -d --force-recreate server server-canal-2`) na janela **00:45:44Z–00:46:02Z**, com os containers iniciados em 00:45:51Z/00:45:52Z. Depois do ciclo o `json-file` tem **0** linhas `event=login` e o Loki devolve **as mesmas duas**, idênticas; o endpoint de log do portal (`/api/logs/<container>` → `src/server/loki.ts`, sessão de staff) devolve exatamente as mesmas linhas com o timestamp ISO do Loki — `diff` antes×depois vazio nos dois canais, conta/canal/sessão/timestamp intactos. **Defeito encontrado e corrigido no caminho:** as métricas do Loki eram agrupadas por `service` (nome do serviço no compose, `server`, `portal`…) e comparadas com a whitelist de `container` (`wyd-canal-1`…), então `/api/logs/status` respondia "sem linhas (5m)" para todos; agora agrupa por `container` e exibe nome curto (`logLabel`), com `ok:true` e 9/10 serviços `registrando`. Neste ciclo ainda não havia linha de login do dono no durável — ela aparece 3 h depois, em **2026-11-08T23:08:51Z** (linha seguinte), e a investigação inicial de 1 h não a via por causa da janela padrão do `query_range` | `docker logs wyd-canal-1 \| grep -c event=login` = **0** após recriar; Loki `{job="docker", container="wyd-canal-1"} \|= "event=login"` → `account=test1 group=1 channel=1 version=12000 session=5`; SSE `/api/logs/wyd-canal-2` → `data: 2026-11-09T00:45:04.339Z … account=test2 … session=6`; `diff` dos arquivos antes×depois → `IDENTICO_1`/`IDENTICO_2`; `/api/logs/status` `ok:true`; `test1..test5` http=200 (admin, moderator×2, player×2) e `admin` default http=200 `role:admin`; `/monitoramento` renderiza `presença · banco` com `canal 1: 0 · canal 2: 0` (sessões de prova já liberadas) e `loki · ingerindo`; `cargo fmt --check` 0, `clippy -D warnings` 0, `cargo test --workspace` **477 ok / 0 falhas**, `npx tsc --noEmit` 0, `npx vitest run` 0, `docker compose config` dev e prod 0, Grafana `/api/health` `database:ok` (12.2.0), Prometheus `/-/ready` 200 |
| 2026-11-09 | FASE-08-1 (login com a conta do dono) | **Existe linha de `event=login` com a conta do dono no durável; a atribuição ao `client/` está refutada (ver fim da célula).** `account=s79331058442700` no **canal 1**, `session=8`, em **2026-11-08T23:08:51Z** — 3 h antes da consulta, dentro da janela de 12 h e **fora da janela padrão de 1 h** do `query_range` (motivo de o vigia de 1 h ter concluído "nenhum handshake": o login já existia). Linhas cruas: `event=session result=open account=s79331058442700 group=1 channel=1 id=8`, `event=login result=accepted account=s79331058442700 group=1 channel=1 version=12000 channels=2 session=8` e, no canal 2, `event=login result=conflict account=s79331058442700 group=1 channel=2 origin_channel=1 version=12000 reason=conta já tem sessão viva no canal 1` (o cliente tentou os dois canais; a cerca de sessão recusou o segundo com a sessão viva no primeiro — comportamento correto). O handshake é aceito **sem senha**: `handle_connection`/`handshake` validam INITCODE, frame, checksum, opcode `AccountLogin`, gate de versão e a cerca de sessão, e não consultam `accounts` — a autenticação de conta é FASE-08-2, então a conta do dono (ausente de `accounts`, que só tem `admin` e `test1..test5`) entra como qualquer outro nome. Nada de `character_session` para essa conta: a instância do canal que atendeu em 23:08 é anterior à recriação de 00:45 e não gravava a sessão no banco | `docker run --rm --network wyd alpine:latest sh -c "wget -qO- 'http://loki:3100/loki/api/v1/query_range?query=%7Bjob%3D%22docker%22%7D%20%7C%3D%20%22event%3Dlogin%22&limit=200&direction=backward&since=12h'"` → 3 linhas de `s79331058442700` + 4 de `test1..test4`; ns convertido com `date -u -d @<segundos>` → `2026-11-08T23:08:51Z`; `docker exec wyd-db psql -U wyd -d wyd -tAc "select id, login, role from accounts order by id"` → `admin`, `test1..test5` (conta do dono ausente); `select account_login from character_session` → só `test1..test4`; `client/serverlist.txt` confirma os dois canais (`192.168.100.3:8281`, `192.168.100.4:8281`) e `docker port` publica ambos, aceitando TCP do host. **Proveniência (conclusão honesta):** essa linha **não** veio deste `client/` — o `WYD.log` da pasta não tinha nenhuma execução entre 19:33 local (22:33Z) e 22:45 local (01:45Z de 11-09), enquanto o login é de 20:08 local (23:08Z); o handshake CPSock não registra `peer`, e o único `peer` do host (`172.18.0.1` no `/serv00.htm`) não distingue launcher de cliente. Ou seja: **refutada** como prova do `client/` e **não confirmada** quanto à origem (outra cópia/máquina na LAN ficou indeterminado) |
| 2026-11-09 | FASE-08-6 (handshake pelo `client/WYD.exe`) | **Primeiro `event=login` gerado por uma execução datada do `client/`.** `account=admin group=1 channel=1 version=12000 channels=2 session=9` em **2026-11-09T01:53:03Z** (= 22:53:03 local, UTC-3), com `event=session result=open account=admin group=1 channel=1 id=9` e `event=handshake result=accepted version=12000 channels=2 kw=194 session=9` na mesma janela. A execução é a do `WYD.log` iniciada em **22:45:56 local = 01:45:56Z** (PID 20084, `Start-Process` com `-WorkingDirectory client`), cuja tela de login (formulário com a conta `admin` já preenchida) foi submetida ~7 min depois. **`Error in Init Render Target Texture` é não fatal:** a janela renderiza o *server select* (`NextGen` na coluna Servidor) e o cliente **concluiu o handshake** — medido por captura da janela e pelo par sessão+handshake no Loki, não por inferência; o script de automação por mouse/teclado foi bloqueado pelo antivírus (AMSI) e o clique em Conectar não abriu TCP, então o caminho provado é o login pelo formulário. | `docker run --rm --network wyd alpine:latest sh -c "wget -qO- 'http://loki:3100/loki/api/v1/query_range?query=%7Bjob%3D%22docker%22%7D%20%7C%3D%20%22event%3Dlogin%22&limit=50&direction=backward&since=12h'"` → ns `1794189183373118311` → `date -u -d @1794189183` = **2026-11-09T01:53:03Z**; `client/WYD.log` → `11/08 22:45:56 - VertexShader HW Accel Enabled` + `22:45:57 - Error in Init Render Target Texture` (execução que gerou a sessão); `node tools/client/serverlist.mjs --check client` → 1 servidor `NextGen`, canal 1 `192.168.100.3`, canal 2 `192.168.100.4` |
| 2026-11-09 | FASE-12-1 (ruído do healthcheck do Postgres) | **Fim do `FATAL: database "admin" does not exist` a cada 10 s.** Causa no compose: o healthcheck chamava `pg_isready -U ${POSTGRES_USER:-wyd}` sem `-d`, e com `POSTGRES_USER=admin` o `pg_isready` tenta o banco de mesmo nome do usuário — que não existe (o banco é `wyd`). Correção: `-d ${POSTGRES_DB:-wyd}`, validada em `docker compose config` e aplicada ao recriar o `db`. Efeito colateral tratado: com o `db` recriado, os canais ficaram com o pool apontando para o container antigo (listener aceitava TCP mas nenhum `event=status` saía) e o gateway ficou fora até `docker compose up -d` — ambos reerguidos e verificados | `docker inspect wyd-db --format '{{json .Config.Healthcheck.Test}}'` → `["CMD-SHELL","pg_isready -U admin -d wyd"]`; `docker logs --since 2m wyd-db \| grep -c 'database "admin" does not exist'` → **0**; `docker ps` → `wyd-db`, `wyd-gateway`, `wyd-portal` e os dois canais healthy; `curl -o /dev/null -w %{http_code} http://127.0.0.1:7600/health` → 200 |
| 2026-11-09 | FASE-08-6 (canais em loopback depois da troca de IP) | **Stack novamente acessível ao cliente.** A máquina passou a ter `192.168.100.7` no lugar de `192.168.100.3/.4`, então as publicações do override (`${WYD_CANAL1_IP:-192.168.100.3}` e `${WYD_CANAL2_IP:-192.168.100.4}`) ficaram órfãs: `docker port` vazio, nenhum listener no host, `serv00.htm` e `:8281` em timeout — o cliente não tinha para onde discar. Correção na origem: os dois IPs voltaram como alias **/32 no adaptador de loopback** (`netsh interface ipv4 add address "Loopback Pseudo-Interface 1" 192.168.100.3 255.255.255.255`, idem `.4`) — /32 de propósito, para não criar rota `192.168.100.0/24` que quebraria a LAN (o Wi-Fi está em `192.168.100.7/24`); o override passou a publicar **os dois conjuntos** (`${WYD_CANAL1_IP:-192.168.100.3}` + `127.0.0.1`, `\${WYD_CANAL2_IP:-192.168.100.4}` + `127.0.0.2`), o `.env` voltou a `.3/.4` e o `client/serverlist.bin` foi **regenerado pelo launcher** a partir do registro (sem editar o `.bin`). Assim o cliente **já aberto** (que leu o `.bin` antigo) e um cliente novo encontram o canal | `Get-NetIPAddress` → `192.168.100.3 Loopback Pseudo-Interface 1 /32` e `192.168.100.4 … /32`; `docker port wyd-canal-1` → `8080/tcp -> 127.0.0.1:80`, `8080/tcp -> 192.168.100.3:80`, `8281/tcp+udp` em `127.0.0.1` e `192.168.100.3`; `docker port wyd-canal-2` → `8281/tcp+udp` em `127.0.0.2` e `192.168.100.4`; `curl http://192.168.100.3/serv00.htm` → `200` com `0\n0\n` (4 B), idem `127.0.0.1`; eco UDP `1,2,3,4` e TCP aberto em `.3`, `.4`, `127.0.0.1`, `127.0.0.2`; `--print client/serverlist.bin` → `1 1 192.168.100.3` / `1 2 192.168.100.4`; `--check client` → `ok` |
| 2026-11-09 | FASE-08-1 (handshake vivo nos DOIS endereços do `.bin`) | **Prova de que o cliente em execução não precisa ser reiniciado**: o handshake real foi aceito em `192.168.100.3:8281` (canal 1) e `192.168.100.4:8281` (canal 2), exatamente os hosts que o `serverlist.bin` carregado pelo cliente usa | `WYD_LIVE_CPSOCK=192.168.100.3:8281 WYD_LIVE_ACCOUNT=test1 cargo test -p wyd-server --test sistema_handshake handshake_contra_servidor_vivo` → `1 passed`; `docker logs wyd-canal-1` → `event=session result=open account=test1 group=1 channel=1 id=15`, `event=login result=accepted account=test1 group=1 channel=1 version=12000 channels=2 session=15`, `event=handshake result=accepted version=12000 channels=2 kw=11 session=15`; canal 2 (`.4`) → mesmas linhas com `channel=2 … id=16`/`session=16`; `character_session` 15 e 16 com `released_at` preenchido; `select count(*) … where released_at is null` → `0` |
| 2026-11-09 | FASE-08-1 (qual cliente rodou: o nosso não foi executado) | Terceira checagem depois do aviso de login, agora pelo lado do binário. **Nenhum arquivo novo** em `client/`: os mais recentes são `serverlist.txt`/`serverlist.bin` de `08:13:01` local (escritos pelo launcher nesta sessão) e `client/WYD.log` com último run às `07:44:44` local (`10:44Z`), terminando em `Error in Init Render Target Texture`. Zero processos com nome casando `WYD*`/`*launcher*` (`Get-CimInstance Win32_Process`), zero conexões com porta remota `8280–8290` (`Get-NetTCPConnection`) e, no Loki, apenas as duas recusas de TCP vazio que eu mesmo gerei (`11:33:54.352Z` no canal 1, `.354Z` no canal 2). O launcher global da referência autorizada (`C:\Users\Canta\AppData\Local\wyd_launcher`) tem **outro** executável (`With Your Destiny.exe`, 438 792 B) que aponta para o servidor oficial/privado — esse tráfego nunca passa pelos nossos binds. Conclusão: não falta servidor nem config do nosso cliente (`client/serverlist.txt` = `.3/.4:8281` casa com os binds publicados); falta **executar o cliente deste repositório** (`D:\dolorestec\wyd2\OpenWYD\client\WYD.exe`), selecionar `NextGen` e clicar em Conectar | `Get-ChildItem client -File \| Where LastWriteTime -gt (Get-Date).AddMinutes(-60)` → só `serverlist.bin`/`serverlist.txt` `08:13:01` e `WYD.log` `07:44:44`; `Get-CimInstance Win32_Process` filtrado por `WYD`/`launcher` → vazio; `Get-NetTCPConnection` com `RemotePort 8280..8290` → vazio; Loki 25 min → `2026-11-09T11:33:54.352Z wyd-canal-1` e `…354Z wyd-canal-2` `event=handshake result=rejected reason=io: failed to fill whole buffer`; `docker logs --since 10m` dos dois canais → só essas recusas; `character_session` abertas = `0` |
| 2026-11-09 | FASE-08-1 (login real do cliente `WYD.exe` capturado: `account=admin`, sessão 17) | **O cliente deste repositório conectou.** Janela `2026-11-09T11:24:08Z → 11:39:08Z` devolveu só duas linhas `event=(login\|session)`, e elas são o login novo: `event=session result=open account=admin group=1 channel=1 id=17` às `11:38:11.315Z` e `event=login result=accepted account=admin group=1 channel=1 version=12000 channels=2 session=17` às `11:38:11.316Z`, seguidas de `event=handshake result=accepted version=12000 channels=2 kw=241 session=17`. A conta veio como **`admin`** (não `s79331058442700`) — registrado como chegou, sem normalizar: é o login digitado no cliente. `kw=241` (contra `kw=11` das minhas sondas) é a assinatura de que o frame é do cliente real, não da sonda. Correlação no mesmo instante: `character_session` id **17**, `account_login=admin`, `group_id=1`, `channel=1`, `client_version=12000`, `opened_at=2026-11-09 11:38:11.304682+00`, `released_at=2026-11-09 11:38:11.727829+00` (a sessão fecha ~423 ms depois, porque a seleção de personagem ainda não existe — FASE-08-2), e o stdout de `wyd-canal-1` traz as mesmas linhas da conta/sessão. Veio um **segundo** login idêntico pelo canal 2: `character_session` id **18**, `admin`, `channel=2`, `opened_at=2026-11-09 11:39:17.576041+00`, `released_at=2026-11-09 11:39:18.100069+00`, com `2026-11-09T11:39:17.612Z \| wyd-canal-2 \| level=info event=login result=accepted account=admin group=1 channel=2 version=12000 channels=2 session=18` — o cliente tentou os dois canais, como faz ao listar o servidor. O processo foi provado do lado do cliente: `WYD.exe` **PID 23392**, criado `09/11/2026 08:37:40` local (`11:37:40Z`), `client/WYD.log` reescrito no mesmo segundo, e socket `192.168.100.4:64389 → 192.168.100.4:8281` em `CloseWait` pertencente a esse PID — o cliente discou para os nossos binds, não para o servidor oficial do launcher global | Consulta Loki exata: `{job="docker"} \|~ "event=(login\|session)"`, `limit=300`, `direction=backward`, `start=2026-11-09T11:24:08Z`, `end=2026-11-09T11:39:08Z` → 2 linhas (as duas citadas). `docker logs --since 15m wyd-canal-1` → `level=info event=session result=open account=admin group=1 channel=1 id=17`; `level=info event=login result=accepted account=admin group=1 channel=1 version=12000 channels=2 session=17`; `level=info event=handshake result=accepted version=12000 channels=2 kw=241 session=17`. `select … from character_session order by id desc limit 5` → `17\|admin\|1\|1\|12000\|2026-11-09 11:38:11.304682+00\|2026-11-09 11:38:11.727829+00`; agregado `min_id=5 max_id=17 total=10 abertas=0` (após o segundo login: `total=11 abertas=0`); `select … where id >= 17` → `17\|admin\|1\|1\|12000\|2026-11-09 11:38:11.304682+00\|2026-11-09 11:38:11.727829+00` e `18\|admin\|1\|2\|12000\|2026-11-09 11:39:17.576041+00\|2026-11-09 11:39:18.100069+00`. `Get-CimInstance Win32_Process` → `23392 WYD.exe` criado `09/11/2026 08:37:40`; `Get-NetTCPConnection` → `192.168.100.4:64389 → 192.168.100.4:8281 CloseWait 23392`; `docker ps` → `wyd-db`, `wyd-gateway`, `wyd-portal`, `wyd-canal-1`, `wyd-canal-2` todos `healthy` |
| 2026-11-09 | FASE-08-2 (seleção de personagem no canal) | **A sessão não fecha mais no handshake.** Causa medida no código: o laço antigo lia só o `HEADER` de 12 B e **retornava no primeiro pacote** — o cliente era desconectado e a conta liberada antes de escolher personagem; opcode desconhecido também fechava. Agora `serve_selection` consome o frame inteiro (`Size` → corpo, sem dessincronizar o próximo), renova a cerca a cada pacote e atende `_MSG_CreateCharacter`/`_MSG_DeleteCharacter`/`_MSG_CharacterLogin` por `characters.rs`: a conta vem do `_MSG_AccountLogin` e slot/nome só escolhem **entre** os personagens dela (autoridade no Postgres, sem criação sem dono e sem seleção alheia). Nome/classe/slot são validados antes do banco (1..=16 ASCII imprimível, classe 0–3, slot 0–3) e o `DeleteCharacter` exige o nome que veio no pacote; a senha de 12 B **não** é conferida (`password_checked=false` no log) porque pertence à autenticação de conta, que ainda não existe. Pacote ainda não portado (`_MSG_Ping`, guerra, chat) é **tolerado** (`result=tolerated`) e a conexão continua; só `Size` inválido e falha de I/O encerram. As confirmações continuam **só de cabeçalho** (`Size=12`) com o opcode de sucesso ou o `*Fail` do domínio, porque `STRUCT_SELCHAR`/`STRUCT_MOB` seguem UNVERIFIED (invariante 5) — por isso a tela do 7662 ainda não desenha a lista completa: o blob é do corpo do `CNFAccountLogin`, que só fecha com a captura da 08-6. A conta do dono ganhou linha em `accounts` (id 9, `NO_CREDENTIAL`, sem senha) porque criar personagem exige dono; autenticação segue pendente. | `cargo test -p wyd-server --test sistema_selecao` 4/4 (sessão viva depois de pacote tolerado e de duas recusas sem banco; `Size=8193` fecha; `KeyWord` ecoado na resposta); `WYD_LIVE_CPSOCK=127.0.0.1:8281 WYD_LIVE_ACCOUNT=s79331058442700 cargo test -p wyd-server --test sistema_selecao selecao_contra_servidor_vivo` → `1 passed`; canal 1 (imagem `sha256:899c630c…`): `event=session result=open account=s79331058442700 group=1 channel=1 id=19`, `event=login result=accepted account=s79331058442700 group=1 channel=1 version=12000 channels=2 session=19`, `event=character result=ok action=create … slot=3 name=QA846587000 class=0 id=1`, `action=select … level=1 force=1`, `stage=in_game`, `action=delete … password_checked=false`; `character_session` 19 aberta `11:58:19.831Z` e liberada `11:58:19.880Z` (nenhuma viva sobra); `characters` id 1 com `deleted_at` preenchido (soft-delete, nenhum resíduo vivo); Loki `{job="docker"} \|= "event=login"` devolve a mesma linha de `s79331058442700 channel=1 session=19`. |
| 2026-11-10 | FASE-08-2 (`_MSG_ApplyBonus` tipo 1 — maestria) | **O último ramo recusado do pacote fechou.** Migration **0036** `character_special` (`special_0..3` 0–255 + `special_bonus`, CHECK coerente; `BaseMob` das quatro classes traz tudo zero, então linha ausente = template) — série em **36 migrations / 74 tabelas**, catálogo regenerado. `Progression` ganhou `special[4]` + `special_bonus`, lidos/gravados com `characters` + `character_scores` + `character_learned_skill` numa transação (`store_progression`) e aplicados ao `STRUCT_MOB` pelo `apply_progression` (login e relogin). **Regra portada de `_MSG_ApplyBonus.cpp:85-123`:** exige `SpecialBonus > 0`, `Detail` 0–3, teto `3 × (nível + 1) ÷ 2` e teto 200 — 255 com a skill de linha correspondente (`Detail` 1/2/3 → bits 7/15/23 do `LearnedSkill`); debita 1 ponto por clique. **Contrato de fio:** `_MSG_UpdateScore.Special[4]` é **sentinela `0xCC`** no `SendScore` do legado (`SendFunc.cpp:1274-1277`) — a maestria real vai no `STRUCT_MOB` e o que muda no clique é o `SpecialBonus` do `_MSG_UpdateEtc`. **Prova (protocolo real, Postgres real):** `maestria_do_bonus_tipo_1_vai_ao_banco_e_volta_no_relogin` — 10 pontos concedidos no banco, 3 em `Special[0]` (teto do nível 1 = 3), `SpecialBonus` 10→9→8→7 no `UpdateEtc`, `Special[4]` do `UpdateScore` = `CC CC CC CC`; 4º ponto no mesmo atributo **recusado sem gravar** (o pedido seguinte em `Special[2]` entra e o banco fica `[3,0,1,0]` com 6 pontos); relogin devolve `BaseScore.Special@84 = [3,0,1,0]`, `CurrentScore.Special@132` igual e `SpecialBonus@790 = 6`. **Gates:** fmt 0, clippy 0, `cargo test --workspace --all-targets` 0, `check-migrations` `36/74`. **Fora:** ganho de pontos por nível (`CMob.cpp:1128/1142`) e bônus celestial (`+3 × 400`), que exige `ClassMaster`. |
| 2026-11-10 | FASE-08-2 (`_MSG_ApplyBonus` + cofre/moeda reais no login) | **Os 5 pontos de atributo e os 3 de skill deixaram de ser número decorativo do `STRUCT_MOB`.** Migration **0035** (`character_learned_skill`: máscara `LearnedSkill` de 24 bits por personagem, CHECK 0–16777215) — série em **35 migrations / 73 tabelas**, catálogo regenerado. Portas novas: `Characters::{progression, store_progression}` (transação `characters` + `character_scores` + `character_learned_skill`), `Accounts::{store_vault_items, set_account_coin}` e `SkillRules::skill_rule` (`skill_catalog.skill_point` + `item_catalog.req_*` da skill `5000 + N`). **Login:** `Selected` carrega a progressão e o `STRUCT_MOB` do `CNFCharacterLogin` aplica atributos/pontos/`LearnedSkill` do banco; o `CNFAccountLogin` leva `Cargo[128]` de `vault_items` e `Coin` de `account_wallet`. **`_MSG_ApplyBonus` (0x0277), porte de `_MSG_ApplyBonus.cpp`:** tipo 0 gasta ponto (`ScoreBonus >= 300` → 100 de uma vez) e soma 2 de `MaxHp`/`MaxMp` por `Con`/`Int`; tipo 2 aprende skill (classe, custo, `Detail` 5000–5095, tier 7/15/23 com as 7 anteriores + 50 000 000 de ouro, nível/maestria do catálogo); tipo 1 (maestria) é recusado com log até `Special` ser persistido — nada de estado pela metade. Resposta: `_MSG_UpdateEtc` (0x0337, corpo 36 B: EXP, `Learn`, pontos, `Coin`) + `_MSG_UpdateScore` (0x0336, corpo 140 B: `CurrentScore` inteiro, `CurrHp/CurrMp`, `Special[4]=0xCC`), `HEADER.ID` = índice do mob. **Provas (Postgres real, frames reais):** `cofre_e_moeda_da_conta_chegam_no_login` (`Cargo[3]=1234`, `Cargo[127]=547`, `Coin@1884=7777` do banco, `Cargo[0]=0`) e `bonus_de_atributo_e_skill_chegam_ao_cliente_e_ao_banco` (`Con 6→7`, `MaxHp 83→85`, `ScoreBonus 5→4` no `UpdateScore`; skill 0 do TK com `SkillBonus 100→76` e `Learn=0x1`; relogin devolve `LearnedSkill@780=1`, `Con=7`, `MaxHp=85`). **Gates:** fmt 0, clippy 0, `cargo test --workspace --all-targets` 0, `check-migrations` `35/73`. |
| 2026-11-10 | FASE-08-2 (inventário persistido: `character_items` é a fonte da verdade) | **O equipamento deixou de ser derivado do template.** Migrations **0033** (`character_short_skill`: `ShortSkill[16]` por personagem, CHECK de 16 B) e **0034** (`account_wallet`: o `Coin` da conta, teto `i32::MAX` porque o campo do fio é `int`) — série em **34 migrations / 72 tabelas**, catálogo regenerado. `crates/wyd-db`: `model/character.rs` ganhou `ItemRow` (+ `EQUIP_SLOTS`/`CARRY_SLOTS`/`CARRY_FIRST_SLOT`/`SHORT_SKILL_LEN`), a porta `Characters` ganhou `items`/`insert_items`/`short_skill`/`set_short_skill` e `Accounts` ganhou `vault_items`/`account_coin` (defaults para os fakes do Cold Path), todas com implementação `pg` transacional (INSERT por slot com `ON CONFLICT DO NOTHING`). **Criação** (`characters.rs::create`): o `BaseMob` da classe vira linhas (`Equip[i]` → slot `i`, `Carry[i]` → `16 + i`, `EF_AMOUNT` do template vira `qty` — 401/406 = 120) mais o **kill mark** (item 547 em `Carry[KILL_MARK=63]`, os três `EF_CURKILL`/`EF_LTOTKILL`/`EF_HTOTKILL` do `ProcessDBMessage.cpp`) e `ShortSkill = 0xFF×16`; nada mais copia o template. **Leitura:** `Mob::from_persisted(&character, &items)` monta `Equip`/`Carry` **só** das linhas (rosto = aparência padrão da classe, porque a customização ainda não existe), `Selchar::from_roster` desenha a seleção com o inventário persistido e o `CNFAccountLogin` leva `Cargo[128]` de `vault_items` + `Coin` de `account_wallet` (sem linha: slots vazios e zero). Personagem anterior à entrega materializa o inventário uma vez, na primeira leitura. `_MSG_ApplyBonus` continua fora (registrado em `docs/wyd/classes.md` §1.2). **Prova ponta a ponta** (`cargo test -p wyd-server --test sistema_selchar inventario_persistido_vira_struct_mob_no_relogin`, Postgres real, frames reais): `_MSG_CreateCharacter` → **12 linhas** em `character_items` (slots 0–6 do `Equip`, 16–19 da `Carry`, 79 = 547 em `qty=1`, potions `qty=120`) e `character_short_skill` = `FF×16`; item **1234 inserido só no banco** no slot 7 apareceu no `STRUCT_MOB` do segundo `_MSG_CharacterLogin` (`Equip[7] = d2 04 …` = 1234), `Carry[63] = 23 02 4b 00 4c 00 4d 00` (547 + efeitos 75/76/77), `ShortSkill@1034 = FF×16` e o corpo de **816 B** comparado **byte a byte** com o esperado montado das linhas lidas do Postgres. **Gates:** `cargo fmt --check` 0, `cargo clippy --workspace --all-targets -- -D warnings` 0, `cargo test --workspace --all-targets` 0, `check-migrations` `34 migrations / 72 tabelas` e `cargo test -p wyd-db --test contrato_catalogo` 3/3. |
| 2026-11-10 | FASE-08-2 (seleção/criação no layout **real** do motor antigo) | **Fim do `SELCHAR`/`STRUCT_MOB` provisório (`rust-v1`, 144/256 B).** `crates/wyd-server/src/character/snapshot.rs` agora fala o layout do `Basedef.h`, conferido também contra o porte Go de `main:tmserver/internal/protocol/{mob,selchar}.go`: `STRUCT_SCORE` 48 B, `STRUCT_MOB` **816 B** (`Coin`@28 int, `Exp`@32 int64 — daí `BaseScore`@44 e `CurrentScore`@92 —, `Equip[16]`@140, `Carry[64]`@268, `SaveMana`@795, `SkillBar`@796, `Resist`@806) e `STRUCT_SELCHAR` **840 B** (`Name[4][16]`@16, `Score[4]`@80, `Equip[4][16]`@272, `Coin[4]`@792, `Exp[4]`@808). Os três corpos passaram a ser completos: `CNFAccountLogin` **2008** (`sel`@20, `Cargo[128]`@860, `Coin`@1884, `AccountName`@1888, `Unknow_28=1`), `CNFNewCharacter` **856** (`sel`@4) e `CNFCharacterLogin` **1832** (`PosX/Y`@0/2, mob@4, `Slot`@1028, `ClientID`@1030, `Weather`@1032, `ShortSkill[16]=0xFF`@1034, `AccountName` em `Unk2[448]`@1500), com `HEADER.ID` 30002/30001/30000 (`ESCENE_FIELD`). **Valores iniciais reais:** os quatro `BaseMob` de `main:tmserver/content/DBsrv/run/BaseMob/{TK,FM,BM,HT}` (816 B, SHA-256 em `docs/wyd/classes.md` §1.2) foram copiados para `crates/wyd-server/assets/basemob/` e entram por `include_bytes!`; a criação copia o template como o `CFileDB.cpp` (`memcpy(mob, &g_pBaseSet[cls])`) e grava **ouro 5.000.000**, **spawn 2096/2096** e a aparência/equipamento da classe (rosto `Equip[0]` = 1/11/21/31, que é como o cliente descobre a classe — `STRUCT_SELCHAR` não tem campo `Class`), sobrepondo nome, nível 1, HP/MP (`BaseSIDCHM` + incremento: 83/46, 61/68, 71/57, 77/61) e pontos 5/3. O `STRUCT_MOB` enviado ao cliente usa `CurrentScore` do banco e mantém `Ac=4`, `Damage=5`, `Merchant`, `AttackRun=82` do template (o `0xCC` de `Direction`/`ChaosRate` é lixo do dump do MSVC, preservado como no porte Go); `Direction` real chega no primeiro `_MSG_Action`. **Prova viva reexecutada** nos dois canais reconstruídos com a imagem nova: `LiveStruct1` (hold 60 s) e `LiveStruct2` (hold 180 s) criados por `_MSG_CreateCharacter` no slot 1 do canal 1 e encarnados por `_MSG_CharacterLogin`; Prometheus `wyd_entities_active{canal="1"}` = **1** durante a sessão e `{canal="2"}` = 0, série de 15 min mostrando 0 → 1 → 0, e **0** depois do FIN; orçamento de tick P99 **0,990 %** em ambos os canais na mesma janela; painel 11 ("Entidades por canal") com canal 1 = 1 e painel 10 (orçamento) em 0,990 % (capturas em `tmp/grafana-entidades-canal-vivo.png` e `tmp/grafana-orcamento-tick-vivo.png`). Banco dos dois: `class 0, level 1, hp 83/83, mp 46/46, gold 5.000.000, pos 2096/2096`, `character_scores` 8/4/7/6 e 5/3 pontos, `deleted_at` preenchido após a deleção pelo protocolo. **Gates:** `cargo fmt --check` 0, `cargo clippy --workspace --all-targets -D warnings` 0, `cargo test --workspace --all-targets` **0** (50 binários, nenhuma falha), `check-migrations` `32 migrations / 70 tabelas`, `monitoramento_contrato` 4/4. |
| 2026-11-10 | FASE-08-6 (cena viva: `MSG_CreateMob`/`MSG_RemoveMob` no fio) | **Os 6100 NPCs de `npc_definition` passaram a existir no cliente, não só no motor.** `crates/wyd-protocol/src/payload/scene.rs` (ADR-0025) porta o layout do `Basedef.h` + porte Go do mesmo alvo 7662: `CreateMob` **232 B** (`PosX/PosY`+12, `MobID`+16, `MobName[16]`+18, `Equip[16]`+34, `Affect[32]`+66, `Guild`+130, `GuildMemberType`+132, `Unknow[3]`+133, `STRUCT_SCORE`+136, `CreateType`+184, `AnctCode[16]`+186, `Tab[26]`+202, `Hold`+228) e `RemoveMob` **16 B** (`RemoveType`+12; 0 fora de vista, 1 morte, 2 logout), com `HEADER.ID = ESCENE_FIELD` (30000) no `CreateMob` e o id da criatura no `RemoveMob`. Aparência vem do template (`NpcRule`/`NpcSpec` ganharam `equip[16]`, `attack_run`, `direction`); `Affect`/`AnctCode`/`Tab`/`Hold` vão zerados porque o snapshot não os tem — nada inventado. A borda (`cpsock.rs::Stage::InGame { scene }`) emite a cena **depois** do `CNFCharacterLogin` e faz delta por **geração do motor** (`npcs_generation` sobe em `sync_npcs`) a cada `SCENE_POLL=1s`, com `RENEW_EVERY=30s`: NPC que sai da revisão vira `RemoveMob`, o que entra vira `CreateMob`, o que não mudou não é reenviado. **Prova viva** contra os canais reconstruídos (`wyd/server:dev` `cd62cc9e41cd`): `crates/wyd-server/tests/sistema/cena_mobs.rs` entra pelo caminho do cliente (`INITCODE` → `_MSG_AccountLogin` com senha Argon2id → `_MSG_AccountSecure` → `_MSG_CharacterLogin`) e decodifica o stream — `PASS 6100 NPCs visíveis no slot 1 (232 bytes cada), jogador em (1690,1840)`, alvo `mob_id=1 nome="Bardes" em (1689,1840)` (id 1 de `npc_definition`, `display_name = Bardes`) e `diálogo respondeu "Bardes"` pelo `_MSG_Action`→`_MSG_MessagePanel`. **Delta de revisão na mesma sessão** (revisão 3→4→5, NPC 1 desligado e religado): `event=npc result=reload revision=4 spawned=0 kept=6099 removed=1` + `event=scene result=applied generation=2 mobs=6099 created=0 removed=1`, depois `revision=5 spawned=1 kept=6099 removed=0` + `generation=3 mobs=6100 created=1 removed=0`; o teste fechou em `creates=6101 removes=1 ativos=6100` — nenhuma duplicata, nenhum mob órfão. **Gates:** `cargo fmt --all --check` 0, `cargo clippy --workspace --all-targets -- -D warnings` 0, `cargo test --workspace` 0. |
| 2026-11-10 | FASE-08-6 (direção e pack de caixa/pickup lidos do binário) | **`_MSG_CreateItem` (0x026E) e `_MSG_GetItem` (0x0270) deixaram de depender de captura para direção e layout.** Leitura estática de `client/WYD.exe` com `objdump -d -M intel` sobre `.text`, instrução por VA: o roteador de recebimento `0x495d..` (cadeia `cmp <Type>,imm` + `call`) tem `0x26e → 0x486c25`, o que prova **S→C** e refuta o “C→J” do `Basedef.h`; no mesmo roteador aparecem `0x16f → 0x487959`, `0x171 → 0x487580` e `0x175 → 0x4872ba` (todos S→C), enquanto `0x270` **não** está ali: o cliente o monta em `0x4fee1b` (`memset(buf,0,0x1c)`, `Type=0x270` em +4, `ID` em +6, `DestType=1` em +12, `DestPos` em +16, `ItemID` em +20, `GridX` em +22, `GridY` em +24) e chama `SendPacket` (`0x54da23`) com **28 B** — 14 B de corpo + **2 B de padding**, o que um decoder de 26 B rejeitaria. No handler do `CreateItem`, todos os campos do `Basedef.h` conferem byte a byte: `GridX` +0x0C, `GridY` +0x0E, `ItemID` +0x10, `STRUCT_ITEM` +0x12 (lido em dois dwords), `Rotate` +0x1A, `State` +0x1B, `Height` +0x1C (**`movsx`**, com sinal), `Create` +0x1D comparado com 1 → **packed, 30 B**. Caminho auditável da captura estático registrado em `docs/cliente/pacotes.md` §“Caixa de item no chão e pickup”, com os três pontos que seguem UNVERIFIED (`DestType` só visto como 1; padding de 30×32 no lado do servidor; faixa de ids de caixa). **Não houve captura ao vivo:** exige o cliente aberto na GUI e um servidor que emita os pacotes — o canal Rust não emite nenhum dos dois, e o cliente é GUI com anti-debug. **Gates:** `cargo fmt --all --check` 0, `cargo clippy --workspace --all-targets -- -D warnings` 0, `cargo test --workspace` 0 (nenhuma linha de Rust mudou nesta entrega). |
| 2026-11-10 | FASE-08-2/10 (primeira morte de monstro ligada ao loot) | **O abate deixou de ser só dano: agora a criatura morre, sai da cena e o loot é sorteado no instante da morte.** `world::Engine::attack_at` no golpe fatal faz `despawn` da entidade (nada de cadáver com HP 0 no mapa) e incrementa `npcs_generation`; `NpcSpec` ganhou `template` (o `npc_template.template_name`, chave da tabela de drop) e `AttackOutcome`/`AttackReport` carregam `template`. `Gameplay::loot_at_death(template, die)` lê o snapshot **uma vez** por abate, tira **um roll por item da tabela** (`1..=1.000.000`, como `docs/wyd/sistemas/drops.md` §1 — nunca ponderado) do `XorShift64` do chamador e devolve `LootResolution { template, rolls, drops }`, então o resultado é reproduzível pela seed. `SelectionStore::attack` recebe o `die` da sessão e resolve o loot no abate; a borda (`cpsock::handle_attack`) loga `event=death ... template=... seed=...`, `event=loot result=resolved ... rolls=[...] drops=[...]`, tira o id da vista local e envia `_MSG_RemoveMob` com `RemoveType = 1` (morte) — o delta de cena não repete a remoção. **Provas:** `world::tests::abate_usa_a_exp_do_template_e_npc_sem_combate_nao_e_alvo` passa a exigir segundo golpe sem alvo (`Ok(None)`), `npc_actors()==1` (só o NPC de loja) e `generation == 2`; `tests/integracao/config_gameplay.rs::morte_sorteia_um_roll_por_item_da_tabela_do_snapshot` prova um roll por linha, `caiu ⇔ roll ≤ peso`, mesma seed → mesmo resultado, recarga do portal com peso 1 mudando a decisão e `NoTable` para template sem tabela. **Caixa/pickup (levantamento):** `Basedef.h` dá `_MSG_CreateItem` 0x026E (GridX/Y+12, ItemID+16, `STRUCT_ITEM`+18, Rotate/State/Height/Create+26..29), `_MSG_DecayItem` 0x016F, `_MSG_GetItem` 0x0270 (DestType+12, DestPos+16, ItemID+20, Grid+22/24) e `_MSG_CNFGetItem` 0x0171; o flag do `CreateItem` está declarado C→J e o pack da região não foi conferido byte a byte, então **nada é emitido**: o menor caminho auditável (morte → caixa → `GetItem` → `CNFGetItem`+`DecayItem`) está proposto em `docs/cliente/pacotes.md` §“Caixa de item no chão e pickup”. **Por que não há prova viva de kill:** os templates reais têm HP de verdade (Bardes = 29.000, Lanceiro_Zakum = 8.000, lidos do `npc_template.blob`), enquanto o dano do personagem ainda é o do template de nível 1 — um abate no fio depende do balanceamento da FASE-10, não do handler. **Gates:** `cargo fmt --all --check` 0, `cargo clippy --workspace --all-targets -- -D warnings` 0, `cargo test --workspace` 0. |
| 2026-11-09 | FASE-08-1 (varredura total calibrada: login da conta real existe; `character_session` foi apagada pelo reseed) | Calibração primeiro, porque era o furo: o relógio do host e o do Loki **não divergem** — a mesma linha (`session=16`, canal 2) aparece em `docker logs --timestamps` como `2026-11-09T11:14:36.692153810Z` e o Loki guardou `1794222876697578885` ns = `2026-11-09T11:14:36.697Z` (≈5 ms), então a janela curta anterior não cortava nada por fuso. Varredura **sem corte** (`2026-11-01T00:00:00Z` → `11:31:52Z`, limit 5000): 12 linhas de `event=login`, contas `s79331058442700` (3), `test1` (5), `test2`, `test3`, `test4`, `admin` (1 cada); intervalo real `2026-11-08T23:08:51.100Z` → `2026-11-09T11:14:36.692Z`. O login do capitão **está registrado**: sessão 8 no canal 1 e 10 no canal 2, com a recusa de conflito entre eles. O que falta é a row de `character_session`: o banco foi **re-semeado às 00:07Z** (`accounts` id 1 em `00:07:28Z`; `test1..test5` em `00:11:29Z`), então hoje `character_session` id 8 é `test4`/canal 2 (`00:47:13Z`) e o stdout dos canais rotacionou (`json-file` 10m×5). O handshake **não autentica** (`cpsock::handshake` cerca por `account_login`, TEXT CHECK 1..16 — 15 caracteres cabem), logo a conta apagada não bloqueia o próximo login; quando FASE-08-2 autenticar, a row em `accounts` volta a ser obrigatória | Loki `{job="docker"} \|= "s79331058442700"` (08/11 22:00Z→09/11 01:00Z, forward): `2026-11-08T23:08:51.100Z \| wyd-canal-1 \| level=info event=session result=open account=s79331058442700 group=1 channel=1 id=8`; `2026-11-08T23:08:51.100Z \| wyd-canal-1 \| level=info event=login result=accepted account=s79331058442700 group=1 channel=1 version=12000 channels=2 session=8`; `2026-11-08T23:08:51.132Z \| wyd-canal-2 \| level=warn event=login result=conflict account=s79331058442700 group=1 channel=2 origin_channel=1 version=12000 reason=conta já tem sessão viva no canal 1`; `2026-11-08T23:08:51.561Z \| wyd-canal-2 \| level=info event=login result=accepted account=s79331058442700 group=1 channel=2 version=12000 channels=2 session=10`; `select id, account_login, channel, opened_at from character_session where id=8` → `8\|test4\|2\|2026-11-09 00:47:13.263392+00`; `docker logs wyd-canal-1 \| grep 'session=8'` → vazio (rotação) |
| 2026-11-09 | FASE-08-6 (endereço do cliente × binds do servidor: nada divergente, cliente não discou) | Comparação feita dos dois lados. **Cliente:** `client/serverlist.txt` = `http://192.168.100.3/serv00.htm`, `192.168.100.3:8281`, `192.168.100.4:8281`, e `--print client/serverlist.bin` devolve as mesmas células. **Servidor:** publica `8281/tcp+udp` e `:80` em `192.168.100.3` (canal 1) e `192.168.100.4` (canal 2), além de `127.0.0.1`/`127.0.0.2`. **Rota:** `/32` de `.3` e `.4` na `Loopback Pseudo-Interface 1`, com `192.168.100.0/24` na Wi-Fi — host e cliente escolhem o alias. Não há lado divergente; a divergência que existia (cliente em `.3/.4`, binds em loopback) foi corrigida e reverificada (handshake aceito nos dois IPs). A ausência de conexão é **medida**, não falta de log: um TCP vazio em `.3:8281`/`.4:8281` gera `event=handshake result=rejected reason=io: failed to fill whole buffer` nos dois canais, e na janela de 20 min não há nenhuma linha dessas nem `event=login` de conta ≠ `test1` (só 2 de sonda). O cliente que rodou hoje (`client/WYD.log`, mtime `07:44` local = `10:44Z`) parou em `Error in Init Render Target Texture`, antes de falar com a rede | `Get-NetRoute` → `192.168.100.3/32 Loopback Pseudo-Interface 1 0.0.0.0`, `192.168.100.4/32 Loopback Pseudo-Interface 1 0.0.0.0`, `192.168.100.0/24 Wi-Fi`; `TcpClient` em `.3` e `.4` → `conectou=True` e ambos os canais logam `level=warn event=handshake result=rejected reason=io: failed to fill whole buffer`; Loki 20 min → `2 account=test1`; `tail -30 client/WYD.log` → `11/09 07:44:44 - Error in Init Render Target Texture` |
| 2026-11-09 | FASE-08-1 (tentativa de login real: nenhum pacote chegou ao servidor) | Capitão informou o login por volta de `11:29Z`, mas a evidência mostra que **nada alcançou o canal**: nenhum processo do cliente (`Get-Process` filtrado por `wyd\|launcher\|game` → vazio) e `netstat` só com os listeners do Docker — zero conexões para `8281`. Os dois canais, nas últimas `0h30`, só registram `event=tcp service=status peer=127.0.0.1` (healthcheck interno do container), sem nenhum `event=tcp` de host, sem `handshake` e sem `session`. Conclusão medida: o cliente não chegou a discar para o servidor; a linha `event=login` da conta real continua sem existir no Loki | `netstat -ano \| grep ':8281'` → `192.168.100.3:8281 LISTENING`, `127.0.0.1:8281`/`127.0.0.2:8281` e UDP nos quatro binds, **nenhum `ESTABLISHED`**; `docker logs --since 30m wyd-canal-1 \| tail -25` → só `event=tcp service=status`; Loki `{job="docker"} \|= "event=login"` em `2026-11-09T11:04:28Z → 11:29:28Z` (limit 100) → 3 linhas, todas `account=test1` (sessões 14, 15, 16); `released_at is null` → `0` |
| 2026-11-09 | FASE-08-1 (login do capitão: ainda ausente, cliente fechado) | Canal alcançável e handshake aceito nos dois endereços, mas **nenhuma linha de conta real entrou**: a janela limitada (40 min) só tem `account=test1` (as quatro linhas são da sonda) e `Get-Process WYD` → `0`, ou seja, o cliente não estava em execução durante a observação. Falta **só** a ação do capitão: abrir `client/WYD.exe` (o `.bin` em disco e os publishes agora apontam para `192.168.100.3/.4`, então não há nada a regenerar nem reiniciar antes), selecionar `NextGen` na coluna Servidor e clicar em Conectar. Nada pendente do lado do servidor | Loki `{job="docker"} \|= "event=login"`, janela `2026-11-09T10:35:00Z → 11:15:00Z` (limit 60, backward) → `4 account=test1`, zero outras contas; `character_session … order by id desc` → id 16/15/14 (`test1`), todas liberadas; `released_at is null` → `0`; `docker ps` → `wyd-db`, `wyd-gateway`, `wyd-portal`, `wyd-canal-1`, `wyd-canal-2` todos `healthy` |
| 2026-11-09 | FASE-08-1 (janela ampla: nenhuma sessão do capitão) | **Varredura de 2026-11-08T00:00:00Z a 2026-11-09T10:54:12Z** por `event=login`, `event=session` e `event=handshake` (`limit=200`, `direction=forward`). Todos os `event=login` da janela: `23:08:51Z s79331058442700 c1 s8` + `23:08:51Z s79331058442700 c2 conflict orig=1` + `23:08:51Z s79331058442700 c2 s10`; `00:45:03Z test1 c1 s5`; `00:45:04Z test2 c2 s6`; `00:47:12Z test3 c1 s7`; `00:47:13Z test4 c2 s8`; `01:53:03Z admin c1 s9`; `10:50:49Z test1 c1 s13`. Depois de `2026-11-09T01:53Z` **não existe login de conta diferente de `test1`**: a única linha posterior é a sonda de protocolo. Distinção exigida: **provado por medição** = a sonda `test1` (grupo 1, canal 1, sessão 13, `version=12000`, `channels=2`) com linha no Loki, no stdout do canal e em `character_session`; **provado por código** = o caminho INITCODE → `_MSG_AccountLogin` → checksum `Strict` → gate de versão → cerca de sessão → `_MSG_CNFAccountLogin` (`crates/wyd-server/src/cpsock.rs::handshake`); **depende só do capitão** = a sessão dele pelo cliente gráfico, que não ocorreu nesta janela (a última é de `2026-11-08T23:08:51Z`, sessão 8, e a atribuição ao `client/` segue refutada). | Query (janela explícita): `curl -sG 'http://loki:3100/loki/api/v1/query_range' --data-urlencode 'query={job="docker"} \|= "event=login"' --data-urlencode "start=${S}000000000" --data-urlencode "end=${E}000000000" --data 'limit=200&direction=forward'`; labels do stream: `{"container":"wyd-canal-1","detected_level":"info","job":"docker","service":"server","service_name":"server"}`; `docker logs --since 3m wyd-canal-1` → `event=session result=open account=test1 ... id=13`, `event=login result=accepted account=test1 group=1 channel=1 version=12000 channels=2 session=13`, `event=handshake result=accepted version=12000 channels=2 kw=11 session=13`; `select ... from character_session where account_login='test1'` → `13,test1,1,1,12000,10:50:49.648Z,10:50:49.712Z` |
| 2026-11-09 | FASE-08-6 (origem da coluna Canal) | **O canal do server select vem do dado real do servidor — nenhum rótulo foi reescrito no `.bin`.** Cadeia medida: Postgres `server_channel` = `(1,0,http://192.168.100.3/serv00.htm)`, `(1,1,192.168.100.3)`, `(1,2,192.168.100.4)` → `client/serverlist.bin` com as células 1..2 iguais a esses **hosts** → `client/sn.bin` com `contadores[0]=3` (= 2 canais + cabeçalho, logo o cliente desenha **2 linhas** de canal: `tools/client/serverlist.mjs:184-196`) e slot 1 `NextGen` contador 2 = grupo 1 → `/serv00.htm` com **um número por canal de jogo na ordem do banco** (`crates/wyd-server/src/channels.rs::render_status_body` + `crates/wyd-server/src/status/service.rs::online_por_canal`, filtrado por grupo **e** canal). No binário: GET da célula 0 e `sscanf` de 10 destinos em `0x46d9d6` (formato `"%d\n…"` em `0x6143b8`); célula de host do grupo lida em `0x46dae2` (`g_pServerList` = grupo × `0x2c0` + canal × `0x40`); rótulo = nome do `sn.bin` + índice (`0x46d6ce`–`0x46dc76`, `docs/cliente/formatos.md` §3–§4). Conclusão: **provado por código** que o canal vem dos dados reais e casa com `group=1`/`channel=1` do handshake; **provado por medição** que o endpoint e o `.bin`/`sn.bin` publicam exatamente 2 canais; **depende só do capitão** ver a coluna preenchida, porque a lista de canais só é desenhada com o servidor selecionado na janela (`0x46daf8` compara o grupo da linha com o grupo exibido) e a entrada sintética está bloqueada (UIPI/AMSI) | `docker exec wyd-db psql -U wyd -d wyd -tAc "select group_id, channel, host, status_url, enabled from server_channel order by group_id, channel"` → `1\|0\|192.168.100.3\|http://192.168.100.3/serv00.htm\|t`, `1\|1\|192.168.100.3\|\|t`, `1\|2\|192.168.100.4\|\|t`; `node tools/client/serverlist.mjs --print client/serverlist.bin` → `1 0 http://192.168.100.3/serv00.htm` / `1 1 192.168.100.3` / `1 2 192.168.100.4`; `node tools/client/serverlist.mjs --sn client/sn.bin` → `slot 0: contador=3`, `slot 1: nome="NextGen" contador=2 → grupo 1 · 1=192.168.100.3 2=192.168.100.4`; `curl -s http://192.168.100.3/serv00.htm` → `0\n0\n` |
| 2026-11-09 | FASE-08-1 (handshake vivo com conta real) | **Login com conta real de `accounts`, sem automação de GUI.** O teste vivo que já existia (`crates/wyd-server/tests/sistema/handshake.rs::handshake_contra_servidor_vivo`, gated por `WYD_LIVE_CPSOCK`) ganhou `WYD_LIVE_ACCOUNT` para usar conta real do banco em vez de nome sintético. Contra o canal 1: `event=session result=open account=test1 group=1 channel=1 id=13` → `event=login result=accepted account=test1 group=1 channel=1 version=12000 channels=2 session=13` → `event=handshake result=accepted version=12000 channels=2 kw=11 session=13`, e a **mesma linha no Loki em 2026-11-09T10:50:49Z**. `character_session` id 13 (`test1`, grupo 1, canal 1, versão 12000) aberta em `10:50:49.648Z` e liberada em `10:50:49.712Z` ao fechar o socket — **nenhuma sessão viva sobra**. Isso prova o caminho de servidor que o botão Conectar dispara (INITCODE + `_MSG_AccountLogin` ofuscado + checksum `Strict` + gate de versão + cerca de sessão + `_MSG_CNFAccountLogin`); o que **não** está provado é o clique no server select do próprio cliente: a entrada sintética continua bloqueada (UIPI/AMSI) e a coluna Canal do cliente só desenha com o servidor selecionado na janela | `WYD_LIVE_CPSOCK=192.168.100.3:8281 WYD_LIVE_ACCOUNT=test1 cargo test -p wyd-server --test sistema_handshake handshake_contra_servidor_vivo -- --nocapture` → `1 passed`; `docker logs --since 3m wyd-canal-1` → as três linhas acima; Loki `{job="docker"} \|= "event=login"` com janela explícita (`start` = 1 h atrás, `end` = agora) → `2026-11-09T10:50:49Z ... account=test1 group=1 channel=1 version=12000 channels=2 session=13`; SQL `select id, account_login, group_id, channel, client_version, opened_at, released_at from character_session where account_login='test1' order by id desc` → `13,test1,1,1,12000,10:50:49.648Z,10:50:49.712Z` |
| 2026-11-09 | FASE-12-1 (embed do Grafana) | **O dashboard deixou de parecer “outra página dentro”.** Antes eram até 4 iframes de 320 px, cada um com borda/raio e o chrome do Grafana (menu lateral, breadcrumb, barra de tempo/variáveis); desde o Grafana 11.3 o `kiosk` sozinho não esconde mais esses controles. Agora: **um** embed de altura útil (`clamp(560px, calc(100vh - 240px), 1200px)`), sem moldura dupla, fundo `#111217` (o do tema dark, sem flash branco) e seleção do dashboard por chips (`?dash=<uid>`, server-side, sem JS). URL de embed: `kiosk=true` + `_dash.hideTimePicker` + `_dash.hideVariables` + `_dash.hideLinks` + `orgId=1` + `theme=dark` + `refresh=30s`. **Medido no Grafana 12.2.0 desta stack: `kiosk=tv` não esconde a navegação** (menu, breadcrumb e busca continuam no iframe); `kiosk=true` esconde menu, navbar e breadcrumb | Captura de `http://127.0.0.1:3001/d/wyd-saude?...&kiosk=tv` → menu Home/Dashboards/Alerting + breadcrumb (`tmp/grafana-kiosk.png`); a mesma URL com `kiosk=true` → `nav`/`header` ausentes no DOM (`preview_evaluate`) e só os painéis (`tmp/grafana-kiosk2.png`); `npx tsc --noEmit` 0; `npx vitest run` 137/137; `docker compose up -d --build portal` 0 e container **healthy**; `grep ui-embed` no `.next` confirma o markup novo no bundle |
| 2026-11-09 | FASE-08-6 (server select: coluna Canal) | **Hipótese do contador refutada por medida.** Injetando uma sessão viva de sonda (`character_session` grupo 1 / canal 1, `client_version` 12000) o `/serv00.htm` passou a responder `1\n0\n` e o cliente **fez o GET** (`online=1/0` no Loki), mas a coluna Canal continuou **vazia** — logo o cliente não esconde a linha por contador `0`. Contrato do endpoint está correto (`HTTP 200`, `content-length: 4`, corpo `0\n0\n`) e o eco UDP de 4 B responde. Desmontagem (`objdump`, build 7662): `0x46d9d6` faz `sscanf` com formato `"%d\n%d\n…%d\n"` (10 destinos, VA `0x6143b8`) e a rotina itera as 10 células de host do grupo (`0x46dae2`), comparando o grupo da linha com o **grupo selecionado** (`0x46daf8`) — a coluna depende da seleção na janela. Bloqueio honesto: entrada sintética não chega ao cliente (UIPI/proteção — `taskkill` responde “Acesso negado”) e o antivírus (AMSI) bloqueia os scripts de automação, então **o clique em Conectar não foi exercitado** | Sonda: `insert into character_session (account_login, group_id, channel, client_version, opened_at, released_at) values ('probe', 1, 1, 12000, now(), null)` → `curl -s http://192.168.100.3/serv00.htm` = `1\n0\n`; Loki `{container="wyd-canal-1"} |= "event=status"` → `online=1/0`; captura `tmp/canal-probe.jpg` (coluna Servidor com `NextGen`, coluna Canal vazia); sonda removida (`DELETE 1`, `/serv00.htm` volta a `0\n0\n`, `select count(*) from character_session where released_at is null` = 0) |
| 2026-11-09 | FASE-12-1 (fim da tela `/logs`) | **A tela antiga de log dos containers sai do portal.** Removidos: rota `(admin)/logs` (`page.tsx` + `logs-view.tsx`), SSE `GET /api/logs/<container>`, `GET /api/logs/status`, o módulo `src/server/docker.ts` (só fazia o mascaramento de ANSI/DSN dessa tela), o item de menu, os links em `/painel`, `/monitoramento` e no formulário de NPC e a linha `/logs/:path*` do matcher do `proxy.ts`. `src/server/loki.ts` ficou só com o agregado que o monitoramento consome (`getLogStats` + `logLabel`), agrupando por `container`; sem endpoint, nenhum texto do browser chega a LogQL. A leitura linha a linha passa a ser o Grafana (datasource Loki provisionado, dashboard `wyd-logs`) e `docker compose logs` para o stdout vivo | `grep -rn 'api/logs\|href="/logs"\|Logs dos containers' wyd-portal/src` → só o comentário histórico em `loki.ts`; `npx tsc --noEmit` 0 (após limpar `.next/types` obsoleto das rotas apagadas); `npx vitest run` 137/137; `docker compose up -d --build portal` 0 e `next build` sem a rota (`/logs` e `/api/logs/status` → 404 com e sem cookie); `/painel` e `/monitoramento` 200, com o bloco `Pipeline de logs (Loki)` e sem link para `/logs` no menu |
| 2026-11-07 | FASE-00-1 | Caches Go limpos (`go clean` exit 0) | `go clean -cache -modcache -testcache` |
| 2026-11-07 | FASE-00-2 | VS Code verificado: rust-analyzer + clippy; zero extensão Go | `code --list-extensions` |
| 2026-11-07 | FASE-00-3 | `develop` confirmada como branch atual | `git branch --show-current` |
| 2026-11-07 | FASE-01-1 | Planejamento Go restaurado do histórico (`roadmap/` estava vazio no arquivo) e inventariado | `git archive HEAD roadmap \| tar -x -C arquivo-Go-removido` → 138 arquivos |
| 2026-11-07 | FASE-02-1/2/3 | Ciclo, checklist de revisão e grafo de dependências documentados | `docs/rust/02-execucao.md`, `roadmap-rust/index.md` |
| 2026-11-08 | análise legado | Três contratos de porte escritos a partir de `origin/main`: [transporte CPSock](../arquitetura/contrato-transporte-cpsock.md), [dispatch por opcode](../arquitetura/contrato-dispatch-opcode.md) e [persistência](../arquitetura/contrato-persistencia.md) | 63 `case` × 61 rotas Go × 198 opcodes Rust; 56 tabelas legadas × 61 Rust; zero link quebrado |
| 2026-11-08 | análise legado (conformidade) | Contratos alinhados às normas do projeto: cada um cita `AGENTS.md` e a skill aplicável (`.agents/skills/wyd-rust-net` / `wyd-rust-db` / `documentation-writer`), módulos renomeados para a árvore ADR-0002 (§11/§12) e política `Strict` de checksum restaurada | `grep` de skill por caminho existente em cada contrato; links 0 quebrados; módulos conferidos contra `estrutura-e-padroes.md` |
| 2026-11-08 | D-T1 (transporte) | `MAX_PACKET_BYTES` deixa de duplicar 4096: passa a `wyd_protocol::MAX_MESSAGE_SIZE` (8192), ganha piso `MIN_PACKET_BYTES` (12) e `accepts_len` valida a faixa do CPSock | `cargo test -p wyd-net` (inclui `packet_range_matches_the_cpsock_contract`: 12 e 8192 aceitos, 11 e 8193 recusados) |
| 2026-11-08 | D-T1 (fechamento de protocolo) | Auditoria da fonte única: só `wyd-protocol/src/wire/header.rs` define o teto (**8192**); `wyd-net` referencia `MAX_MESSAGE_SIZE`/`MIN_MESSAGE_SIZE` e não tem literal, buffer nem código de framing dimensionado por 4096. Teste de fronteira reexecutado nas duas pontas (8192/8193 e 12/11) e §10 do contrato de transporte passou de "defeito aberto" para "resolvido" | `cargo test -p wyd-net packet_range_matches_the_cpsock_contract` (ok) + `cargo test -p wyd-protocol frame` (8192 ok / 8193 erro) |
| 2026-11-08 | U-P7 + ADR-0020 | Sete tabelas sem espelho separadas em jogo × portal (decisão em `modelo-banco.md` §8) e versão de cliente decidida por ADR | `docs/arquitetura/modelo-banco.md` §8; `docs/arquitetura/adr/0020-versao-de-cliente-no-login.md` |
| 2026-11-08 | D-D1 (fechamento do ADR de versão) | ADR-0020 consolidado como norma: `APP_VERSION = 7640` é **valor histórico** de um cliente específico (não conjunto aceito); o gate compara **igualdade** com `PROTOCOL_VERSION` (12 000, medido em `client/`). ADR ganhou a tabela "onde 7640 aparece" com arquivo:linha (definição em `wire/header.rs:31`, reexports, 3 testes, nada em launcher/portal) e a seção "como o servidor compara" (igualdade em `handshake.rs:72`, sem faixa/lista; `Tolerant` só em captura). Contradições de contrato corrigidas (`APP_VERSION` do legado × gate nosso e D-T2 marcado resolvido) | `cargo test -p wyd-net session::` 4/4 ok (aceita 12 000, recusa 7640 com `ClientVersionUnsupported`); `cargo test --workspace` 0 failed |
| 2026-11-08 | U-P7 (fechamento de banco) | `modelo-banco.md` §8 reorganizada como norma: tabela de cinco colunas (nome, classificação, evidência por arquivo/consulta, implicação), fontes de evidência declaradas (inclui por que o cliente `client/` não é prova aqui e por que `blogal` nunca é fonte de requisito) e **regra** que define "sem espelho" como estado do schema — entrar, ficar fora, UNVERIFIED e sair. Conferido que as 5 de jogo são exatamente as de `0028` e que as 2 de portal não existem em `crates/wyd-db/migrations/` | `cargo test -p wyd-db` 6 suítes / 0 failed; `node tools/content/check-migrations.mjs` 28/66/146 ok; `banco-catalogo.md` lista as 5 em `0028` |
| 2026-11-08 | FASE-08-6 (serverlist) | `serverlist.bin` **decodificado**: grade `MAX_SERVERGROUP(10) × MAX_SERVERNUMBER(11) × 64 B` com soma de tabela própria (`Basedef.cpp:6874/6927`), célula 0 = URL de status (≤ 32 caracteres), células 1..10 = host do canal (8281 fixa, um IP por canal), linhas decididas pelos contadores do `sn.bin`. Gerador/inspetor próprio em `tools/client/serverlist.mjs` (porte do `client-tools/serverlist` da `main`, sem Go) e dados do servidor Rust em `tools/client/serverlist-servers.txt`; `client/serverlist.bin` regenerado com backup e o cliente passou a listar **um** servidor (`NextGen`) | `node tools/client/serverlist.mjs --print client/serverlist.bin` → `1 0 http://192.168.100.3/serv00.htm` / `1 1 192.168.100.3`; round-trip dados→bin→decode idêntico; screenshot `target/janela-blt.png` (uma linha, sem `NextGen2`) |
| 2026-11-08 | FASE-08-6 (entrega serverlist) | Geração e instalação do `serverlist.bin` passam a ser do **launcher Rust**: `crates/wyd-launcher/src/serverlist.rs` (grade 10×11×64, `OBFUSCATION_KEY`, parse do registro `<grupo> <canal> <endereço>` com erro nomeado para URL > 32, porta no canal e IP repetido; instalação atômica `.tmp`→`rename` com `.bak`) + subcomandos `serverlist` e `launch <dir> <manifesto> <registro>` fail-closed (exit 2 sem manifesto). Registro passa a ter **2 canais**: 192.168.100.3 (normal) e 192.168.100.4 (verde). Captura estática do `serv00.htm`: WinInet com **uma** leitura de `0x400` B e corpo CSV consumido por `strtok`/`sscanf` (`0x46d7xx`/`0x4b013b`) | `cargo test -p wyd-launcher` 39 testes ok (paridade Rust×Node byte-a-byte, URL de 36 recusada nos dois lados, instalação atômica/backup/no-op); `cargo run -p wyd-launcher -- serverlist tools/client/serverlist-servers.txt client/` instalou 7040 B com backup; `--print` devolve as 3 células; `cargo fmt --check` e `cargo clippy -D warnings` = 0 |
| 2026-11-08 | stack única + monitoramento no portal + login de staff | Monitoramento deixou de ser perfil separado: `prometheus`/`grafana`/`node-exporter` subiram para o `docker-compose.yml` (o `docker-compose.monitoring.yml` foi removido) e entram no `up` normal. O portal ganhou `/monitoramento` (server-side, `PROMETHEUS_URL=http://prometheus:9090`) com canais, handshakes, sessões, memória, host e estado dos demais alvos. A causa de "não consigo acessar o portal" era de contrato: `account_web/verify_credentials` caía em `pending()` (501) e não havia hash de senha em nenhum crate — agora tem handler com **argon2id** (`crates/wyd-admin/src/auth/password.rs`, dep `argon2` 0.5) e o seed de boot do gateway (`WYD_SEED_STAFF=1`; `--seed-staff` para uso manual) cria/atualiza a conta `PORTAL_ADMIN_USER`/`PORTAL_ADMIN_PASSWORD` do `.env` (idempotente, papel `admin`), sem serviço one-shot que fica parado no `docker compose up`. Canais: `sn.bin` decodificado de verdade (nomes 9 B desde o offset 0, contadores `int32` no 99, `contador - 1` = grupo do slot) — era isso que invertia a lista e deixava a primeira linha apontando para grupo vazio; o gerador passou a escrever `--sn-out` e o inspetor `--sn` mostra slot → grupo → canais. | `cargo test --workspace` 0 falhas (inclui `crates/wyd-admin/tests/contrato/login.rs`: senha certa, senha errada = conta inexistente, `NO_CREDENTIAL` nunca autentica, hash nunca volta), `clippy -D warnings` 0, `fmt --check` 0, `npx tsc --noEmit` 0, `docker compose config` (dev e prod) 0, `promtool` 0 |
| 2026-11-08 | FASE-12-1 (observabilidade) | Stack de monitoramento versionada e subindo em Docker: override `docker-compose.monitoring.yml` com perfil `monitoring` (Prometheus v3.5.0, Grafana 12.2.0, node-exporter v1.9.1 — tags fixas, portas só em loopback), `monitoring/prometheus/{prometheus.yml,alerts.yml,alerts.test.yml,targets/wyd-server.json}`, provisioning do Grafana e dashboard `wyd-saude.json` (tick P50/P95/P99, ticks/s, entidades, pacotes, recusas por motivo, sessões, RSS). Seis alertas com runbook em `docs/rust/06-observabilidade-runbooks.md`; alvo do motor descoberto por arquivo e **vazio** porque não há binário de servidor (só `wyd-launcher` tem `[[bin]]`) — casa nenhum alerta por ausência estrutural. `make monitoring-up/rules/check` e `stack-config` validam o override | `docker compose ... --profile monitoring config --quiet` = 0; `promtool check config`/`check rules`/`test rules` = 0 (disparo **e** resolução de `WydServidorForaDoAr`, `WydTickP99Alto`, `WydPacotesRecusados`); `curl 127.0.0.1:9090/-/ready` = 200 e `/api/v1/targets` com prometheus+node `up`; `curl 127.0.0.1:3001/api/health` = ok e `/api/dashboards/uid/wyd-saude` = 200 |
| 2026-11-08 | FASE-04 (parcial) | `0028_estado_de_jogo` cria as 5 tabelas de estado: `character_affects`, `kingdom_cape_price`, `daily_reward_claim`, `guild_relations`, `guild_tower`; `SCHEMA_VERSION` 27→28; catálogo regenerado (66 tabelas) | `check-migrations` 28/66 ok; `cargo test -p wyd-db --test contrato_catalogo` 3 ok; `migrations-apply` up/down/up |
| 2026-11-08 | FASE-08 (parcial) | Gate de versão do login em `wyd-net::session::handshake` (`VersionGate`), erro tipado `NetError::ClientVersionUnsupported` e contador `wyd_client_version_total{version}` | `cargo test -p wyd-net --features metrics` (12 testes) |
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
| 2026-11-07 | FASE-08-2 | Parsers: `repr(C)` + `inspect` + `require_ported`; 9 goldens byte-idênticos; zero `unsafe`; sem deps novas | `crates/wyd-protocol/src/{wire,lib.rs,payload/}`, `crates/wyd-protocol/tests/golden/pacotes.rs` |
| 2026-11-07 | FASE-03 | Engine: `CoreError`, SoA, scheduler 20–50 Hz, grade O(1), relógio injetável; 12 lib + 3 determinismo; guerra 8,6 ms/79 µs | `crates/wyd-core/src/{error,ecs,tick,spatial,ports}/`, `crates/wyd-core/tests/integracao/determinism.rs` |
| 2026-11-07 | FASE-04 | Banco: 6 migrations reversíveis + modelo + traits + fila; prova ao vivo PG17 (FK/CHECK falham, revert limpa, seed OK); 8 lib + 2 constraints | `crates/wyd-db/{migrations,seeds,src/{error,model,repo,pool,transfer}}`, `docs/arquitetura/modelo-banco.md` |
| 2026-11-07 | FASE-07 (núcleo) | Control plane: `AdminError`→HTTP, staff/auditoria, grants idempotentes; 11 lib + 4 integração (401/403/replay) | `crates/wyd-admin/src/{error,auth,audit,api,service}/`, `crates/wyd-admin/tests/contrato/autorizacao.rs` |
| 2026-11-07 | FASE-06-2 | Hardening: balde de tokens + porte de handshake + pipeline estrito; flood contido, fuzz sem panic; 4 lib + 6 flood | `crates/wyd-net/src/limits/`, `crates/wyd-net/tests/integracao/flood.rs` |
| 2026-11-07 | FASE-13-2 | ADRs 0004/0005/0006 aceitas (ECS/serialização/async diferidos com gate de revisão); 7/7 conformes ao template | `docs/arquitetura/adr/000{4,5,6}-*.md` + links nas consumidoras |
| 2026-11-08 | FASE-07-2 | Painel do jogador: `panel/*` + recuperação auditada, `/conta` no portal; SancRate do legado no banco (28 células) e refino `+10 → +11` corrigido para 3% com Lac | `crates/wyd-admin/{src/api/panel.rs,tests/contrato/painel.rs,src/transport/mod.rs}`, `crates/wyd-db/migrations/{0009_refino,0022_seed_refino,0025_seed_sanc_rate}.sql`, `wyd-portal/src/{server/panel.ts,app/(public)/conta,app/api/panel/recovery}` |
| 2026-11-08 | FASE-06-1 | Telemetria: `tracing` + exporter Prometheus próprio; `/metrics` real com 40 ticks (`wyd_ticks_total 40`, histograma, `wyd_entities_active 104`), log estruturado com nível por `WYD_LOG`; contadores de rede atrás de feature (custo medido); CRC: lock 7→24, launcher/imagem inalterados | `crates/wyd-core/src/{metrics/,telemetry.rs}`, `crates/wyd-net/src/metrics.rs`, `crates/wyd-core/tests/sistema/telemetry.rs`, `crates/wyd-net/tests/integracao/metrics.rs`, `docs/arquitetura/adr/0018-telemetria-tracing.md` |
| 2026-11-08 | Análise (legado→Rust) | Portabilidade medida: legado em `origin/main` = 116 arquivos C++/70.987 linhas + 484 Go/99.606 linhas; workspace Rust = 112 arquivos/18.242 linhas. De-para por subsistema, lacunas L1–L6 (transporte CPSock ausente, `sqlx`, dispatch por opcode, comércio/party/guilda/guerra/quests/eventos/refino) e prioridade de resgate do conhecimento de `docs/migration`/`docs/sistemas`/`docs/lore`/`docs/analysis` | `docs/manual-técnico/analise-portabilidade-legado.md`; `docs/cliente/referencia-wyd-global.md` |

## Pendências conhecidas

- Desinstalar Go e remover `GOPATH`/`GOROOT` (ação elevada do dono).
- Verificar evidência de implementação das 54 issues em `done/` (Go).
- Revisão fria dos goldens 08-2 contra captura real quando a FASE-08-6 liberar capturas (fixtures atuais são sintéticos).
- Revisão fria do `estrutura-e-padroes.md` por outro agente/humano (subtarefa da FASE-02-4).
- Captura ao vivo (08-6): structs S→C, offsets de campo do header e causa do freeze — exige
  `wyd-net`/`wyd-core` executáveis + VM com o jogo.
- Validar spawn real em VM com o jogo (complemento da 08-7).
- **ADR de RNG:** `wyd-sim`/`wyd-core` usam `XorShift64` semeado (ADR-0011), mas o legado exige a
  ordem do LCG do MSVC para paridade de drop/crítico/refino (`docs/migration/game-rules.md`).
  Declarar se perseguimos paridade byte-a-byte do original ou se o RNG novo é a verdade do
  servidor — hoje a decisão está implícita (`analise-portabilidade-legado.md` §7.4).
- ~~**Defeito de transporte D-T1**~~ — **fechado em 2026-11-08:** `MAX_PACKET_BYTES` passou a
  `wyd_protocol::MAX_MESSAGE_SIZE` (8192), com `MIN_PACKET_BYTES` (12) e `accepts_len` validando a
  faixa inteira ([contrato](../arquitetura/contrato-transporte-cpsock.md) §10).
- **Versão de cliente (U3/D-D1) — resolvida:** [`ADR-0020`](../arquitetura/adr/0020-versao-de-cliente-no-login.md)
  decide `APP_VERSION` 7640 como constante **histórica** (nunca comparada) e `PROTOCOL_VERSION`
  = 12.000 (`wyd-protocol/src/lib.rs:40`, medido no cliente oficial `client/`) como valor de gate,
  configurável no boot, com erro tipado + métrica por versão recebida.
- **Opcode sem handler no C++ (U-C):** `_MSG_SplitItem` (0x02E5) tem constante, arquivo
  `_MSG_SplitItem.cpp` e rota no Go, mas não está entre os 63 `case` de
  `ProcessClientMessage.cpp` — decidir se é código morto ou caminho aninhado.
- ~~**Persistência sem espelho (U-P7)**~~ — **resolvida e entregue** (migration `0028`); decisão em
  [`modelo-banco.md`](../arquitetura/modelo-banco.md) §8 — cinco tabelas entram no `wyd-db`
  (`character_affects`, `kingdom_cape_price`, `daily_reward_claim`, `guild_relations`,
  `guild_tower`) e duas são do Cold Path (`account_play_code`, `donate_topup_order`). O DDL fica
  para a issue da FASE-04 (`0028_estado_de_jogo.sql`); nenhuma migration nasceu do contrato.
- **Checksum: divergência consciente (D-T3).** O legado tolera divergência (`CPSock.cpp:457-464`);
  a stack nova usa `Strict` na entrada de rede (`AGENTS.md` §3 invariante 2, `protocolo.md` §3,
  `codec/checksum.rs:13-19`). `Tolerant` só em captura/diagnóstico.
- Resgatar de `origin/main` o conhecimento P0/P1 (game-rules, parity-tests, data-formats,
  handlers/58, protocol-spec) antes de qualquer limpeza de histórico.

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
- Árvore §13 do `wyd-admin` ganhou `src/transport/{mod,http,json}.rs` e `tests/contrato/transporte.rs`; `.proto` proibido.
- Issues atualizadas: `FASE-07.md` (contexto, passos, arquivos, pendências) e `FASE-07-1-gateway.md` (transporte HTTP/JSON, passos, aceite, rollback). Docs alinhados: `estrutura-e-padroes.md` §13, `visao-pastas.md`, `00-visao-estrutura.md`, `05-stack-docker.md`, skill `wyd-web-admin-ui`.

## 2026-11-07 — FASE-07-1: transporte HTTP/JSON do wyd-admin implementado (ADR-0008)
- `crates/wyd-admin/src/transport/json.rs`: JSON do contrato sem dependência (escapes, `\uXXXX` com surrogate, `MAX_DEPTH` 32, rejeita `01`/lixo/aninhamento profundo; nenhum `panic!` em entrada).
- `crates/wyd-admin/src/transport/http.rs`: HTTP/1.1 mínimo em std — POST obrigatório, `Content-Length` obrigatório, sem chunked, head ≤ 16 KiB, corpo ≤ 32 MiB, `cache-control: no-store`, `x-trace-id`, `connection: close`, `bind_loopback` e `serve_conn` só em loopback.
- `crates/wyd-admin/src/transport/mod.rs`: `POST /admin/v1/{servico}/{metodo}` → `accounts/find`, `accounts/create`, `grants/request`; `Authenticator` (porta) + `StaticTokens`; 401 sem token (não audita), papel insuficiente = 200 + `result: 2` com `auth.denied` auditado, 404/400/405/413 de transporte, envelope `result` 1..5 + `trace_id`.
- `crates/wyd-admin/tests/contrato/transporte.rs`: 9 casos incl. grant idempotente (replay com mesmo `grant_id`) e round-trip real em socket loopback.
- Prova: `cargo fmt --all -- --check` 0, `cargo clippy --workspace --all-targets -- -D warnings` 0, `cargo test --workspace` 22 suites ok.
- Pendente na issue: `admin-client.ts` no portal (remover `@grpc/*`) e os 13 serviços restantes sobre o mesmo roteador.

## 2026-11-08 — FASE-07-1: portal fiado no HTTP/JSON (fim do gRPC no BFF)
- `wyd-portal/src/server/admin-client.ts`: `fetch` para `WEB_API_ADDR` (default `http://127.0.0.1:7600`), `POST /admin/v1/{servico}/{metodo}`, `authorization: Bearer $WEB_API_TOKEN`, `x-trace-id`, `AbortSignal.timeout(10 s)`, `cache: no-store`; mantém as 16 exportações e todos os tipos do antigo `grpc.ts` (serviços via `svc<T>()`), removendo `moderator_id`/`moderatorId` do corpo antes de enviar.
- `promisify.ts` vira fachada `(req, traceId) => Promise<Res>` (deadline no cliente); `trace.ts` expõe `traceHeaders()` puro; `errors.ts` ganha `AdminUnavailableError` (conexão recusada/timeout/5xx/resposta inválida → 502) e `isInfraFailure` deixa de checar código gRPC.
- 25 módulos do BFF reimportados (`@/server/grpc` → `@/server/admin-client`), `grpc.ts` removido; `@grpc/grpc-js`/`@grpc/proto-loader` fora do `package.json` e do lock; `docker/Dockerfile.portal` não copia `.proto`; `.env.example` documenta `WEB_API_ADDR` (URL) + `WEB_API_TOKEN`; README do portal reescrito para HTTP/JSON.
- Prova: `npm run typecheck` 0, `npm run test -- --run` 118/118 (16 arquivos), `grep grpc` em `src/` = 0.
- Pendência honesta: `StaticTokens` não valida HMAC (token de dev até a fiação cripto) e os 13 serviços ainda respondem 404 de transporte enquanto as rotas não existem.

## 2026-11-08 — FASE-04-3: pool `sqlx`, adapters `pg` e save fora do tick

- **Rede medida:** `index.crates.io/config.json` → 200 e download de `.crate` → 200. Isso derrubou a
  premissa da ADR-0006 ("async bloqueado até rede/vendor") para o `wyd-db`; escrita a ADR-0021, que
  substitui a ADR-0006 **só** nessa parte.
- **Código:** `pool/pg.rs` (pool dimensionado, `acquire_timeout`, `SET LOCAL statement_timeout`,
  mapeamento `sqlx::Error` → `DbError` com 23505/23503/23514/23502), `repo/pg.rs` (as três portas),
  `transfer/save.rs` (`SavePolicy`, `SaveQueue`, `SaveDrain`, `FlushReport`).
- **Hot Path intocado:** `SaveQueue::enqueue` é `try_send` síncrono — não há `await` possível no
  tick, e o invariante 1 do `AGENTS.md` fica garantido por construção, não por disciplina.
- **Ruptura pequena de API:** `Accounts::create`/`Characters::create` viraram `create_account` e
  `create_character`. `PgBackend` implementa as duas portas, então dois `create` no mesmo tipo
  obrigavam todo chamador a qualificar a trait (o compilador acusou E0034 em 15 sítios).
  Fakes de `wyd-admin` (`src/api/{accounts,panel}.rs`, `tests/{contract,painel,unauthorized}.rs`)
  acompanharam; `DbError` ganhou `NotFoundKey`, `SaveQueueFull` e `SaveTimeout` e o mapeamento
  HTTP do portal cobre as três.
- **Prova real:** Postgres 17 descartável (`postgres:17-alpine`, porta 55432), ups de 28 migrations
  aplicados por `psql` (28 ok, 66 tabelas), `seeds/dev.sql` ok, `WYD_TEST_DSN=... cargo test -p wyd-db
  --test sistema_pg -- --test-threads=1` → **6/6 ok**: contas (roundtrip + UNIQUE tipado), personagens
  (CRUD, soft delete, ranking), grants (3 tipos + baixa idempotente), save (coalescência grava o
  último, shutdown 2/2 sem perda, apagado conta como `missing`), constraints tipadas e custo do
  `enqueue`.
- **Custo medido no quente:** `enqueue` N=20 000 com dreno real em paralelo → p50 **500 ns**, p99
  **1 600 ns**, max 257 µs (preempção do SO durante o flush concorrente). Orçamento: 50 ms/tick.
- **Decisão que ficou aberta de propósito:** runner embutido de migration (`sqlx::migrate!` ou
  `refinery`) **não** entrou — a aplicação continua por `tools/content/migrate-cycle.mjs`/`psql -f`,
  que já prova up/down/up em banco real. Trocar exige issue própria.
- **Gate:** `cargo fmt --all -- --check` exit 0, `cargo clippy --workspace --all-targets --
  -D warnings` exit 0, `cargo build --workspace` exit 0, `cargo test --workspace` 36 suítes / 0
  failed, `node tools/content/check-migrations.mjs` 28/66/146 ok.
- **Desbloqueia:** FASE-03-4 (carga de conteúdo no boot), FASE-07-1 (BFF do portal) e FASE-15-2
  (carga de banco).

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
- **Prova:** `crates/wyd-sim/tests/integracao/determinism.rs` (8 testes: mesma seed → mesma sequência e
  mesmo tick, seed diferente → resultado diferente, rolls gravados → replay idêntico, estado
  de efeitos/regen/ameaça reprodutível). `tests/golden/formulas.rs` **não foi tocado** — mesmas
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
- **Prova:** `crates/wyd-core/tests/integracao/gem.rs` (6 testes de API pública, incl. Anct +9 com 40 de
  perfuração contra comum recusado com 0) e `gem_tests` (11) em `components.rs`; fixtures dos
  testes antigos ganharam estoque de gema, **sem alterar nenhuma expectativa de jogo**.

## 2026-11-08 — Banco remodelado: catálogo gerado + série única verificada do zero

- **Série consolidada em 24 migrations** (DDL 0001–0016, seed 0017–0024): `0023_inventario_gema`
  e `0024_catalogo_skill_nota` foram **absorvidos** por `0003_inventario` e `0007_conteudo_catalogo`
  — refino/Anct/gema e `skill_catalog.notes` nascem na criação da tabela. `ALTER TABLE` no `up`
  agora é erro de build (`tests/contrato/catalogo.rs`).
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

- **Lacuna encontrada ao retomar:** `crates/wyd-core/tests/integracao/gem.rs` provava a transação (consumo
  da mochila, evento único, recusas tipadas) com índices e valores escritos no próprio teste.
  Provava a política, não que ela é a do banco: um `per_level_value` ou `effect_code` errado em
  `0022_seed_refino.sql` passaria despercebido.
- **Teste novo:** `crates/wyd-core/tests/golden/gema_seed.rs` (4 testes) lê **a própria migração**
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
- **Anti-dupe/perda:** `tests/integracao/inventario.rs` fecha o ciclo equipar → trocar → desequipar →
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
  escreve posição/HP; `tests/integracao/npc.rs` prova que destino forjado é recusado por `step`
  (`NoMoveBudget`/`MoveTooFar`) com posição e fila intactas, e que decidir não publica evento.
- **Determinismo:** decisão é função pura (`brain::decide`), cadência `tick % 4 == entity % 4`
  com `DecisionBudget`, RNG `xorshift64` por `(world_seed, entity)`, ameaça com teto 8 e
  desempate pelo menor `entity`; golden FNV-1a de 24 ticks em `tests/integracao/npc.rs`.
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
  fechado) e correção da contagem de `tests/integracao/inventario.rs` (9, não 10).

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
- **Paridade com o dado:** `crates/wyd-sim/tests/golden/refino_seed.rs` lê `0022`/`0025`/`0026` por
  `include_str!` e confere 15 passos, as 36 células, a leitura em `nível+1` e os passos
  especiais fora da tabela.
- **Gates:** `cargo test -p wyd-sim` 0, `cargo test --workspace` 0,
  `clippy --workspace --all-targets -D warnings` 0, `fmt --all --check` 0,
  `check-migrations` 0 (26 migrations), `gen-catalogo-banco --check` 0, `verify-schema` 0.
- **Aberto (por desenho):** fiação de jogo do refino (comando do cliente, consumo, evento) e
  o port dos NPCs de composição (Ailyn, Anct, Odin, Tiny, Agatha, Shany, Alquimia, Lindy,
  Ehre, Extracao) — inventário em `refino-auditoria.md` §4.

## 2026-11-08 — FASE-03-4 entregue: conteúdo estático validado no boot (porta + loader)

- **Regra vigente:** o conteúdo estático vem do PostgreSQL (ADR-0009) e o boot **falha
  fechado**. `ContentTables::validate()` roda antes de abrir socket e devolve
  `ContentError` com até 20 `{tabela, pk, coluna, motivo}`; `ContentCounts` vai para o log
  junto da versão. Nenhum `.csv`/`.txt`/`.bin` é lido em runtime.
- **O que o SQL não expressa e o Rust passa a cobrar:** `content_version.version >= 1`;
  catálogo de itens e de skills não vazio; `item_index`/`skill_index` repetido; nome de
  item vazio ou com caractere de controle; preço negativo; `game_setting` com seção/chave
  vazia ou repetida; e a checagem **cruzada** de `skill_catalog.lineage_code` contra
  `skill_lineage` (a FK só cobre o caso de seed íntegro — um truncamento parcial passaria).
- **Fronteiras (DIP):** porta em `crates/wyd-core/src/ports/content.rs` (`ContentSource`
  + tipos + validação pura, sem I/O); adapter em `crates/wyd-db/src/repo/content.rs` faz
  **uma** transação de leitura (`content_version`, `item_catalog`, `skill_catalog`,
  `game_setting`, `skill_lineage`), valida e só então devolve. `wyd-db` passou a depender
  de `wyd-core` — sem ciclo (`wyd-core` só conhece `wyd-protocol`). Falha de leitura vira
  issue com `tabela = "*"`: infraestrutura não é confundida com dado inconsistente.
- **Evidência com o seed real** (`postgres:17-alpine`, 28 migrations aplicadas do zero):
  `conteúdo v9 "Composição: 36 valores de Aylin/Compositor/Odin/pó em combine_value":
  3220 itens, 154 skills, 25 settings, 13 linhagens`.
- **Testes:** `crates/wyd-core/tests/integracao/content.rs` **7 passed, 0 failed** (sem banco:
  índice repetido, nome em branco/controle, linhagem inexistente, catálogo vazio + versão
  inválida, chave de setting repetida/vazia, teto de 20 com `truncated`);
  `crates/wyd-db/tests/sistema/pg.rs` **7 passed, 0 failed** com banco, incluindo
  `conteudo_estatico_carrega_e_aborta_com_dado_corrompido` (corrompe
  `skill_catalog.name` com `U+0001`, exige o relatório apontando `skill_catalog`/`name`,
  restaura a linha **antes** dos asserts e reconfirma o caminho válido).
- **Gates desta entrega:** `cargo fmt --all -- --check` **0**;
  `cargo clippy --workspace --all-targets -D warnings` **0**;
  `cargo test --workspace` **0** (37 alvos com `test result: ok`, 0 failed).
- **Aberto (por desenho):** o loader **não** está fiado no binário do servidor porque o
  binário de mundo só nasce com o gateway (FASE-07-1 + FASE-06); NPC/loja/região entram
  no mesmo relatório quando o tick consumir essas tabelas.

## 2026-11-08 — Fechamento dos três entregáveis da missão (prova re-executada)

- **`MAX_PACKET_BYTES` = 8192 (fonte única):** `crates/wyd-net/src/lib.rs:18` é
  `wyd_protocol::MAX_MESSAGE_SIZE` e `crates/wyd-protocol/src/wire/header.rs:22` vale `8192`;
  `MIN_PACKET_BYTES` é `MIN_MESSAGE_SIZE` (= `HEADER_SIZE` = 12). `accepts_len` aceita
  exatamente `[12, 8192]`. **Prova:** `cargo test -p wyd-net --lib
  tests::packet_range_matches_the_cpsock_contract -- --exact` → **1 passed, 0 failed** (afirma
  `accepts_len(8192)`, rejeita 8193 e 11, aceita 12, e iguala as constantes a 8192/12); mais
  `tests::rejects_empty_and_oversized_packets` → 1 passed (0, 11, 12, 8192, 8193). `wyd-net`
  completo: **10 + 6 = 16 testes**, 0 falhas. **Nenhum outro teto no workspace:** `grep` por
  `65_536|65536|64 KiB|65535|8191` em `crates/`, `docs/`, `tools/`, `fixtures/` e
  `wyd-portal/src` só encontra capacidade de fila (`queue_capacity`) e domínio `u16` do portal.
- **Sete tabelas sem espelho:** `docs/arquitetura/modelo-banco.md` §8, linhas 469-475 — cinco
  **requisito do jogo** (`affect`→`character_affects`, `sapphire_balance`→`kingdom_cape_price`,
  `daily_reward_claim`, `guild_relation`→`guild_relations`, `guild_tower_state`→`guild_tower`,
  as cinco em `0028_estado_de_jogo.sql`) e duas **exclusivas do portal legado**
  (`account_play_code`, `donate_topup_order`, ausentes do `wyd-db`), cada linha com evidência
  arquivo:linha e consequência.
- **ADR-0020 + índice:** `docs/arquitetura/adr/0020-versao-de-cliente-no-login.md` (aceita,
  implementada) decide 7640 como **valor histórico**, não conjunto aceito; o gate é igualdade
  com `PROTOCOL_VERSION` = 12 000 medido em `client/`; o launcher do WYD Global é citado como
  referência de terceiro. Índice criado em `docs/arquitetura/adr/README.md` (tabela 0001–0021,
  linha 33 aponta a 0020) e `README.md` da raiz passa a linkar o índice.
- **Gates:** `cargo fmt --all -- --check` **0**; `cargo clippy --workspace --all-targets --
  -D warnings` **0**; `cargo test --workspace` **0** — 37 alvos, **422 testes passados**,
  0 falhas.

## 2026-11-08 — FASE-03-2 fechada: spawn/despawn em lote tudo-ou-nada + baseline SoA

- **O que faltava** eram dois pontos do critério de aceite: operação em lote sobre os
  componentes e o bench de iteração contígua. Spawn/despawn unitários já existiam desde a
  FASE-03, e a escolha do ECS já estava na ADR-0004 (SoA manual até o bench mandar o contrário);
  agora a decisão tem número.
- **Lote na fronteira fria:** `ComponentStore::spawn_batch(&[SpawnSpec])` valida **todos** os
  `Stats` antes da primeira escrita — um `Stats` inválido no meio do lote devolve
  `CoreError::InvalidStats` e o armazém fica como estava (nem entidade, nem índice novo):
  o mundo não sobe meio populado. `despawn_batch(&[Entity])` devolve quantos ids eram válidos e é
  idempotente como `despawn` (id morto/geração antiga não conta). `SpawnSpec { kind, position,
  stats }` é dado de entrada; inventário nasce vazio porque item inicial é entrega de conteúdo.
- **Baseline medido** — `cargo bench -p wyd-core --bench tick` (release, 10 000 entidades por
  rodada, 40 rodadas de spawn + uma passada contígua em `positions()` + despawn):
  **p50 3 381,70 µs · p99 11 427,80 µs · 383,867 ns por entidade**, p99 = **22,856%** do
  orçamento de um tick de 50 ms (teto do cenário: 50%). Estado final: 0 vivas e 10 000 índices no
  armazém — índice é reciclado, não vaza. O cenário **falha** (exit 1) se passar de 50% do tick.
- **Testes novos:** `spawn_em_lote_e_tudo_ou_nada` (lote recusado não cria entidade nem índice e
  não deixa buraco entre as fatias) e `despawn_em_lote_conta_apenas_ids_vivos` (60 de 100, repetir
  o lote morto devolve 0, o restante devolve 40), em `ecs/components.rs`. `cargo test -p wyd-core`:
  todos os alvos `ok` (91 na lib + 63 nos testes de integração = 154 no crate).
- **Baseline da IA (contexto, mesma execução):** 500 NPCs + 50 jogadores → média 196,59 µs/tick,
  p99 362,60 µs (**0,725%** do tick); sem jogadores → média 31,78 µs, p99 53,20 µs (**0,106%**).
- **Gates:** `fmt --all --check` 0, `clippy --workspace --all-targets -D warnings` 0,
  `cargo test --workspace` 0 (ver saída do fechamento abaixo).

## 2026-11-08 — Item de rede da missão: teto 8192 com fonte única e pre-check delegado

- **O valor já estava certo** desde D-T1: `MAX_PACKET_BYTES` (`crates/wyd-net/src/lib.rs:18`) é
  alias de `wyd_protocol::MAX_MESSAGE_SIZE` (`crates/wyd-protocol/src/wire/header.rs:22` = `8192`),
  e `MIN_PACKET_BYTES` é `MIN_MESSAGE_SIZE` (= `HEADER_SIZE` = 12). **Não existe constante
  duplicada** no workspace: `MAX_PACKET_BYTES`/`MIN_PACKET_BYTES` só aparecem no `wyd-net`, e
  `portal`, `launcher`, `tools/` e `fixtures/` não têm nenhuma ocorrência de `8192`.
- **O que faltava era a lógica duplicada:** `accepts_len` repetia `(12..=8192).contains(&len)` em
  paralelo a `validate_size` (`wyd-protocol/src/wire/frame.rs:59`), que é a checagem usada pelo
  **caminho real de leitura** (`split_frame` → `validate_size`). Agora `accepts_len` **delega** para
  `validate_size` (`u16::try_from(len).is_ok_and(...)`, recusando o que nem cabe no campo `Size`),
  então pre-check e enquadramento não podem divergir. Semântica confirmada na doc de planejamento:
  `Size` é o tamanho **total do pacote** — os 12 B do `HEADER` estão **dentro** dele — válido em
  `[12, 8192]` (`docs/arquitetura/protocolo.md` §1, `CPSock.h:38`, `CPSock.cpp:397-406`).
- **Testes (todos passando, mensagem nomeia a borda):** `limites_nomeiam_qual_borda_falhou`
  — aceita `8192` (“limite superior … maior pacote válido”), rejeita `8193` (“excede
  MAX_PACKET_BYTES = 8192”), aceita `12` (“HEADER puro”), rejeita `11` (“não cabe o HEADER de
  12 B”) e rejeita `u16::MAX + 1`; `pre_check_concorda_com_o_enquadramento_real` — compara
  `accepts_len` com `validate_size` em `0, 11, 12, 13, 8191, 8192, 8193, 70 000`, monta um frame
  **real** de 8192 B com `Header::encode` e prova `split_frame` → `8192`, e com `Size = 8193`
  prova a recusa dos dois lados. Nenhum teste antigo foi afrouxado (`rejects_empty_and_oversized_packets`
  e `packet_range_matches_the_cpsock_contract` seguem intactos).
- **Docs reconciliadas:** `docs/rust/00-visao-estrutura.md` passou a dizer que
  `MAX_PACKET_BYTES = 8192` é **alias** de `MAX_MESSAGE_SIZE` (fonte única);
  `docs/arquitetura/contrato-transporte-cpsock.md` §10 (D-T1) já registrava a fonte e a prova.
- **Prova de execução:** `cargo test -p wyd-net --lib` → **12 passed, 0 failed** (eram 10);
  `cargo fmt --all -- --check` **0**; `cargo clippy --workspace --all-targets -D warnings` **0**;
  `cargo test --workspace` **0** — 37 alvos, **426 testes passados**, 0 falhas.
- **Os outros dois itens da missão já estavam fechados** e foram reverificados nesta rodada:
  sete tabelas em `docs/arquitetura/modelo-banco.md` §8 (linhas 469-475: 5 jogo, 2 portal legado,
  com evidência arquivo:linha e consequência) e ADR-0020 em
  `docs/arquitetura/adr/0020-versao-de-cliente-no-login.md` (7640 = histórico; gate por igualdade
  com `PROTOCOL_VERSION` = 12 000 medido em `client/`), indexada em `docs/arquitetura/adr/README.md`.

## 2026-11-08 — FASE-07-1 (paridade de rotas), FASE-08-6 (golden do handshake) e régua de bench no CI

### FASE-07-1 — transporte fechado: nenhuma chamada do portal pode virar 404

- `crates/wyd-admin/src/transport/mod.rs`: nova tabela `SERVICOS` com **16 serviços / 72
  métodos** declarados por `wyd-portal/src/server/admin-client.ts`, já em `snake_case`
  (mesmo `toSnake` do cliente) + `conhecido()`. Rota declarada sem handler → **`501`**
  nomeando a rota e auditoria `admin.pending` com autor da sessão; `404` fica reservado a
  contrato violado. Não criei valor novo em `AdminResult` (o portal mapeia desconhecido
  para 500); `501` é falha de transporte, que o BFF já traduz em `502`.
- `crates/wyd-admin/tests/contrato/transporte.rs`: `cada_metodo_do_portal_tem_rota_no_control_plane`
  extrai a superfície do TypeScript com `include_str!` (build quebra se o caminho mudar) e
  falha se faltar rota, sobrar rota ou a contagem divergir (16/72);
  `rota_do_portal_sem_handler_e_501_auditado_e_nunca_404` despacha as 72 rotas com staff e
  exige `501`/`200` (nunca `404`), mensagem nomeando a rota e 72 pendências auditadas.
- Pendência honesta: os 13 serviços ganharam **rota determinística**, não regra de negócio —
  cada handler de domínio exige `api/<dominio>.rs` + porta de repositório. Nenhum escreve
  estado vivo (AGENTS §3.7).

### FASE-08-6 — o handshake está provado no servidor; a captura ao vivo segue bloqueada

- **Nenhuma captura foi executada** (não houve VM nem servidor fechando login → campo); não
  existe `.pcap` novo e nada foi inventado como "captura".
- `crates/wyd-protocol/tests/golden/pacotes.rs`: novo
  `golden_handshake_offsets_confirmam_a_adr_0020` congela, contra os bytes do golden, o que a
  ADR-0020 usa: `Size` 116, `ClientVersion` i32 no **offset 92** = **12 000**, senha 12,
  conta 24, zero 40, `db_need_save` 96, `adapter_name` 100 — e confere que o frame claro
  bate com o `payloadHex` fora o byte de `CheckSum` (que só existe após a selagem).
- Os 5 goldens C→S seguem com round-trip byte-exato (corte → descortinado → checksum
  `Strict` → opcode → payload → reselagem idêntica ao `wireHex`); o gate do login
  (`wyd-net/src/session/handshake.rs`) aceita 12 000 e recusa 7640.

### Bench no CI — falha em regressão, por cenário

- `crates/wyd-core/benches/tick.rs` passou a imprimir uma linha de contrato por cenário:
  `BENCH_PONTO {"cenario":"ia_500_npcs_com_alvos", …}`,
  `ia_500_npcs_sem_alvos`, `ecs_lote_10000` (p50/p99 em µs, `ns_por_entidade` no ECS).
- `tools/dev/bench-gate.mjs` roda `cargo bench -p wyd-core --bench tick`, lê essas linhas e
  **falha** (exit 1) quando um cenário estoura o limite declarado — e também quando um
  cenário deixa de reportar (bench que some não passa verde). Limites: IA com alvos
  p50 ≤ 300 µs, IA sem alvos p50 ≤ 60 µs, ECS p50 ≤ 9 000 µs e ≤ 700 ns/entidade.
- A régua usa **p50 e custo por entidade**, não p99: medindo 4 execuções seguidas na mesma
  máquina, o p99 variou de 88,00 a 739,00 µs (IA) e de 7 480 a 14 408 µs (ECS) — régua em
  p99 seria teste instável. O teto de p99 em % do orçamento continua cobrado pelo próprio
  bench (IA ≤ 10% do tick pela ADR-0016, ECS ≤ 50% pela FASE-03-2), que sai com status 1.
- Ligado ao CI (`.github/workflows/rust.yml`, job `bench`) e ao `make bench-gate`; o filtro
  de `paths` do workflow passou a incluir `tools/dev/**`.
- Medições desta rodada (release, Windows): IA com alvos p50 181,90–243,70 µs / p99
  604,80–739,00 µs; IA sem alvos p50 24,40–41,10 µs; ECS p50 3 081,50–3 516,60 µs,
  371,98–422,87 ns/entidade. Gate: **3/3 cenários dentro dos limites, exit 0**.

### Gate final desta rodada

`cargo fmt --all -- --check` **0**; `cargo clippy --workspace --all-targets -D warnings`
**0**; `cargo test --workspace` **0** — 37 alvos, **429 testes passados**, 0 falhas;
`cargo test -p wyd-protocol --test golden_pacotes` 10/10; `node tools/dev/bench-gate.mjs` **0**.

## 2026-11-08 — ADR-0022: gateway `:7600` em Docker (primeiro serviço de jogo real)

O workspace tinha um único `[[bin]]` (launcher): nada escutava `:7600`, o `wyd-portal` ficava
`unhealthy`/502 e o job `wyd-server` do Prometheus não tinha alvo. Entregue:

- `docs/arquitetura/adr/0022-binarios-de-servico.md` — decisão dos dois binários de serviço
  (`wyd-gateway` agora, `tmserver` com contrato fixado), imagem `wyd/<svc>:${TAG}`, container
  `wyd-<svc>`, healthcheck pelo próprio binário (`--healthcheck`) e `/metrics` com nomes
  declarados (`wyd_gateway_requests_total{result}`, `wyd_gateway_uptime_seconds`).
- `crates/wyd-admin/src/main.rs` (`wyd-gateway`): `DATABASE_URL` + `WEB_API_TOKEN` obrigatórios
  (fail-closed), `SELECT 1` no boot, foto do catálogo via `PgBackend::catalog_snapshot`
  (novo, `wyd-db/repo/content.rs`), `GET /health` (503 sem banco), `GET /metrics`,
  `POST /admin/v1/...` pelo roteador existente, auditoria em stdout estruturado.
- Rota `item_catalog/list_items` servida da foto real do banco (`api/catalog.rs`) — é a rota que
  `/api/health` do portal consome; sem ela o portal continuaria 502 mesmo com o binário no ar.
- `docker/Dockerfile.gateway` (multi-stage, distroless non-root, `--healthcheck`), serviço
  `gateway` no compose (depends_on `db` saudável, token do `.env`, sem porta publicada na base),
  `portal` agora `depends_on: gateway (service_healthy)` + `WEB_API_TOKEN`.
- `make db-migrate`: aplica as 28 migrations no `wyd-db` da stack (hoje pelo runner `wyd_db::migrate` no boot, ADR-0024; o script Node foi removido)
  (66 tabelas, `item_catalog` 3.220, `content_version` 1) — sem isso o gateway não tem catálogo.
- Observabilidade: job `wyd-gateway` em `prometheus.yml`, alertas `WydGatewayForaDoAr` e
  `WydGatewayErroDeTransporte` (com runbook), caso de teste em `alerts.test.yml`, dois painéis
  novos no `wyd-saude.json`, smoke cobrando o alvo do gateway.

## 2026-11-08 — canais: dois processos, um banco, status real no `:80`

Canal no WYD é **processo**, não flag: `crates/wyd-server` (binário `tmserver`) passou a existir e
o compose roda **um container por canal**, mesma imagem `wyd/server:${TAG}` e mesmo Postgres:

- `wyd-canal-1` → `192.168.100.3:8281` (normal) e `wyd-canal-2` → `192.168.100.4:8281` (verde);
- canal é dado, não build: migration `0029` criou `server_channel` (grupo, canal, host, URL de
  status, `enabled`) com 3 linhas (0 = status, 1 = normal, 2 = verde) — **67 tabelas, 497 colunas**,
  `banco-catalogo.md` e `banco-fontes.mjs` regenerados;
- `GET :80/serv00.htm` (canal 1) devolve `192.168.100.3,192.168.100.4` — corpo de 27 bytes, uma
  leitura de `0x400` do cliente; `/health` faz `SELECT 1` e reporta `channels=2`;
- CPSock `:8281`: `INITCODE` → desofuscação → checksum estrito → `inspect` → `AccountLogin` →
  gate de versão (ADR-0020) → `_MSG_CNFAccountLogin` (só cabeçalho; corpo é UNVERIFIED);
  probe real contra os dois containers: aceito em 12 000, recusado em 7 640;
- `/metrics` `:9898` com o fragmento da FASE-06-1 + `wyd_server_*`; dois alvos no Prometheus
  (`server:9898`, `server-canal-2:9898`) ambos `up=1`;
- launcher: `serverlist` regenerado a partir do registro (`.bin` + `serverlist.txt` com os dois canais).

**Prova no cliente:** o server select passou a desenhar `NextGen-1` e `NextGen-2` (antes eram duas
linhas `Inexistente`) — a coluna Canal é alimentada pelo `serv00.htm` real. **A conexão pelo cliente
gráfico ainda não foi provada:** o clique em Conectar não gerou nenhum `accept` nos containers
(log sem `handshake`), então o cliente está discando outro destino/porta; o diagnóstico seguinte é
captura de socket do próprio `WYD.exe`, não ajuste especulativo no servidor.

## 2026-11-08 — canais: o `Inexistente` era o separador do `serv00.htm`

A linha extra `Inexistente` não vinha do `serverlist.bin` (que já estava certo: grupo 1, célula 0
com a URL de status e células 1–2 com `192.168.100.3`/`.4`) e sim do **separador** do corpo
publicado. Entregue:

- `crates/wyd-server/src/channels.rs` (`render_status_body`): um número **por linha** (`0\n0\n`) em
  vez de lista com vírgula. O `sscanf("%d")` do cliente pula espaço/quebra de linha entre as
  conversões e **para na vírgula**, então `0,0` alimentava só o primeiro destino e o segundo ficava
  `-1` → `Inexistente`;
- `docs/cliente/formatos.md` §4/§4.1: corpo numérico com quebra de linha, a rotina `strtok` por
  vírgula marcada como **não** usada pela tela de seleção, e as referências externas conferidas;
- leitura dos binários de referência: `w2pp-OpenWYD` 7040 B (grupo 0, células 0 e 1 com IP) +
  `serverlist.txt` `<grupo> <canal> <ip>` + `serv00.htm` de 20 B (dez linhas `1`);
  `open-wyd/ows-clientdata` 7040 B (3 grupos × 3 células) e `sn.bin` em texto; cliente real
  WYD Global/Raidhut 7044 B (**4 bytes de cabeçalho** + 7040, outra cifra — nenhum deslocamento da
  nossa tabela decodifica) com `sn.bin` de 143 B igual ao nosso, `Global` no slot 1 e contador 2.

**Provas:** `curl http://192.168.100.3/serv00.htm` devolve `0\n0\n` (4 B); o *server select* do
`client/WYD.exe` desenha **`NextGen`** na coluna Servidor e **`NextGen-1` + `NextGen-2`** na coluna
Canal (`target/prova-linhas-canal1-3-canal-selecionado.jpg`,
`target/prova-v3-canal1-2-canal-selecionado.jpg`); `WYD_LIVE_CPSOCK` contra `.3:8281` e `.4:8281`
→ `event=handshake result=accepted version=12000 channels=2` nos **dois** containers (5/5 testes de
integração por canal); `cargo fmt --check`, `cargo clippy -p wyd-server --all-targets -D warnings`
e `cargo test -p wyd-server` (12 testes) em 0.

**Prova no cliente gráfico (login manual, 2026-11-08):** o teste humano entrou nos **dois** canais e
cada container registrou o par login+handshake:

| Container | Conta | Log |
|---|---|---|
| `wyd-canal-1` | `test` | `22:25:53 event=login account=test version=12000` + `event=handshake result=accepted … kw=11` |
| `wyd-canal-2` | `test` | `22:25:54 event=login account=test version=12000` + `event=handshake result=accepted … kw=11` |
| `wyd-canal-1` | `admin` | `22:35:25 event=login account=admin …` + `accepted … kw=165` |
| `wyd-canal-2` | `admin` | `22:36:14 event=login account=admin …` + `accepted … kw=79` |

Isto fecha o que faltava: o `Conectar` do cliente **dialoga com os dois canais**. A automação de
mouse continua sem alcançar os botões do `WYD.exe` (o botão `Fechar`, cujo efeito é encerrar o
processo, não fecha por `mouse_event`, `PostMessage` nem `SendMessage`, embora a seleção na lista
mude com o cursor sobre a linha) — hipótese: filtro de entrada de
`JProtectDll.dll`/`ClientPatch.dll`. Por isso a prova é o login manual acima mais os testes vivos
(`WYD_LIVE_CPSOCK`), não clique sintético.

## 2026-11-08 — sessão única por conta entre canais (migration 0030)

Cada canal é processo e mundo separado, mas o personagem é um só no Postgres do grupo: dava para
entrar no canal 1 e no canal 2 ao mesmo tempo. Entregue:

- `crates/wyd-db/migrations/0030_sessao_unica.sql`: `character_session` (conta, personagem
  opcional, grupo/canal, versão, `opened_at`/`renewed_at`/`released_at`) com os índices parciais
  `character_session_uma_por_conta` e `character_session_uma_por_personagem`; `SCHEMA_VERSION` 30
  e catálogo regenerado (68 tabelas, 506 colunas, 30 migrations);
- `crates/wyd-db/src/repo/sessions.rs` + adapter `pg`: `claim_session` (libera a órfã pelo TTL e
  insere; conflito devolve o canal ocupado), `renew_session`, `release_session` idempotente e
  `live_sessions`;
- `crates/wyd-server/src/character/session.rs`: `SessionGate`/`SessionLease` (claim, renovação e liberação
  no `Drop`), TTL mínimo de 30 s;
- `crates/wyd-server/src/cpsock.rs`: a cerca entra **depois** do gate de versão;
  `HandshakeError::Session` recusa sem `CNF` e loga `event=session result=open|conflict`; a conexão
  passou a ser segurada (`MAX_IDLE_ROUNDS` × `WYD_SERVER_SESSION_IDLE_SECS`) para a sessão viver
  enquanto o cliente está do outro lado;
- `main.rs`: `WYD_SERVER_GROUP_ID` (default 1), `WYD_SERVER_SESSION_IDLE_SECS` (120) e
  `WYD_SERVER_SESSION_STALE_SECS` (180).

**Provas:** `cargo fmt --check` 0; `cargo clippy --workspace --all-targets -D warnings` 0;
`cargo test --workspace` 0 (nenhuma falha); `cargo test -p wyd-db --test contrato_catalogo` 3/3; teste vivo
novo `crates/wyd-server/tests/sistema/sessao_unica.rs` contra `.3:8281` e `.4:8281` → **ok**: canal 1 abre
(`event=session result=open account=… channel=1`), canal 2 recusa
(`result=conflict … channel=2 reason=conta já tem sessão viva no canal 1`) e, depois que o socket
do canal 1 cai, o canal 2 aceita a mesma conta; ambos os containers `healthy` com
`event=session group=1 channel=N idle=120s stale=180s`.

**Comportamento de jogo (pesquisa):** canal é mundo separado e o jogador **transita** entre eles
mantendo os dados (guia do WYD Global/Raidhut: escolher canal no login; sistema de guildas fala em
"migração de servidor" entre canais). No legado a troca existia como `_MSG_DBServerChange`
(`0x0814`) / `_MSG_DBCNFServerChange` (`0x052A`) porque **cada canal tinha o próprio DB**; aqui os
canais dividem um Postgres, então trocar de canal é relogar — nada é copiado, nada se perde.
A pasta `.deprecated-go` citada nas instruções **não existe** na árvore nem no histórico do git
(removida em 2026-11, `AGENTS.md` §0); a referência usada foi o legado C++ do `w2pp-OpenWYD`
(`Source/Code/DBSrv`) e a tabela de opcodes já congelada em
`crates/wyd-protocol/src/wire/generated/opcode_table.rs`.

**Ainda não provado no cliente gráfico:** "jogador do canal 1 não vê o do canal 2" só vira
observação visual quando existir pacote de cena (`ESCENE_FIELD`): hoje o processo não tem tick nem
mundo, só handshake/sessão. O que está provado é o isolamento estrutural (um `World` por processo,
sem IPC) e a cerca por conta; o efeito visível no cliente hoje é a segunda entrada ser recusada.
(2026-11-08: a presença passou a ser publicada e conferida por canal — ver a última entrada.)

**Isolamento dos canais (2026-11-08):** canal é **processo**, não flag — `wyd-canal-1` e
`wyd-canal-2` rodam a mesma imagem e o mesmo Postgres, mas cada um monta o próprio `World` em
memória (`crates/wyd-server/src/channels.rs`) e não existe IPC nem registro compartilhado; a única
tabela comum é `character_session` (cerca de conta), sempre consultada **escopada por grupo e
canal** (ver a entrada de 2026-11-08 sobre presença por canal). Dois clientes em canais diferentes **não se veem**: são instâncias de mundo
separadas, como manda `docs/wyd/canais-e-cidadania.md` §5 ("um processo por canal, mesmo banco").
Hoje o isolamento é total por construção porque o mundo ainda não existe (só handshake +
`_MSG_CNFAccountLogin`); quando o tick/scene entrar, continua separado pelo mesmo motivo. O que
exige cuidado **não** é a visibilidade e sim o banco comum: sem arbitragem de sessão (FASE-08-2) o
**mesmo personagem** pode ficar logado nos dois canais ao mesmo tempo — o `cpsock` hoje nem
autentica a conta, apenas registra o nome que veio no pacote.

## 2026-11-08 — login manual rastreável e presença por canal (isolamento entre canais)

Pedido: confirmar nos logs dos **dois** containers o login feito manualmente pelo cliente (não as
conexões de teste), explicar eventual recusa da cerca e provar que canal 1 e canal 2 são mundos
separados. Resultado dos dois itens:

**1) Login manual — onde está e por que não está no `docker logs` de agora.** As linhas existem e
estão registradas na entrada anterior deste documento, capturadas na hora: `22:25:53` e `22:25:54`
para a conta `test`, `22:35:25` e `22:36:14` para `admin`, cada container com `event=login` +
`event=handshake result=accepted` (este é o pacote de confirmação que o cliente recebe). O que
aconteceu depois: os dois containers foram **recriados às 22:54:13Z** (rebuild da imagem que trouxe
a cerca de sessão) e o driver `json-file` guarda log **por container** — recriar abre um stream novo.
Não foi rotação nem nível de log: o stream antigo deixou de existir. O log atual só contém as
conexões dos testes vivos (`s78609152345400`, `l969163000`), nunca o login humano.

Para que um login manual seja rastreável sem depender do nome do container:

- `crates/wyd-server/src/cpsock.rs`: `event=login` passou a ser auto-descritivo —
  `result=accepted|conflict account=<conta> group=<grupo> channel=<canal> version=<versão>
  channels=<n> session=<id>`; a recusa da cerca emite **a mesma linha** com `result=conflict` e
  `origin_channel=<canal onde a sessão viva está>` (antes só saía `event=session result=conflict`,
  sem conta, e o `channel=` não distinguia origem de destino);
- `docker-compose.yml` + `docker-compose.override.yml`: `logging` json-file com `max-size=10m` e
  `max-file=5` nos dois serviços (rotação não come a janela recente). Recriar container continua
  abrindo stream novo — auditoria de longo prazo exige exportar o log para fora da máquina.

Formato novo comprovado no ar (teste vivo `tests/sistema/sessao_unica.rs`, 23:08:51Z, conta sintética):

| Container | Linha de log |
|---|---|
| `wyd-canal-1` | `23:08:51.100713 level=info event=login result=accepted account=s79331058442700 group=1 channel=1 version=12000 channels=2 session=8` |
| `wyd-canal-2` | `23:08:51.132607 level=warn event=login result=conflict account=s79331058442700 group=1 channel=2 origin_channel=1 version=12000 reason=conta já tem sessão viva no canal 1` |
| `wyd-canal-2` | `23:08:51.561703 level=info event=login result=accepted account=s79331058442700 group=1 channel=2 version=12000 channels=2 session=10` |

O próximo login manual sai exatamente nesse formato: `account=` do jogador, `group`/`channel` do
processo que aceitou e `session=` da linha em `character_session`.

**2) Isolamento de mundo — provado no que existe hoje, sem vazamento encontrado.** Canal é
processo, não flag: `World` é montado por processo (`channels.rs`), `cpsock` só escreve no próprio
socket, métricas (`wyd_sessions_open`) são por processo e não há IPC, fila ou broadcast entre
containers. A única leitura cruzada possível é o banco comum, e ela passou a ser **por canal**:

- `crates/wyd-server/src/status/service.rs`: novo `online_por_canal` consulta `live_sessions(group, channel)`
  para **cada** canal; o `serv00.htm` deixa de publicar `ONLINE_PER_CHANNEL = 0` fixo e passa a
  publicar a contagem real daquele canal (`render_status_body(world, &online)`), e o `/health`
  devolve `group`, `channel` e `online` **do processo**;
- nenhuma consulta de presença era global antes (não havia `count(*)` do grupo), então o defeito
  encontrado foi de **contrato** — o `serv00.htm` publicava `0` fixo e não provava nada; o risco de
  virar agregado do grupo ficou fixado por teste;
- `crates/wyd-db/tests/sistema/sessoes.rs` (novo, gated por `WYD_TEST_DSN`): sessão no canal 1 aparece em
  `live_sessions(1,1)` e **não** em `live_sessions(1,2)` nem em `live_sessions(2,1)`; conta repetida
  no canal 2 devolve `Conflict { channel: 1 }`; liberar devolve a presença do canal à linha de base.
  Uma consulta sem `channel` (o vazamento plausível) quebra esse teste.

Evidência viva (23:08Z, sessão viva **só** no canal 2 — linha sintética `prova_online_c2` inserida e
removida depois, para não depender de cliente):

| Prova | Resultado |
|---|---|
| `curl http://192.168.100.3/serv00.htm` (publicado pelo canal 1) | `0\n1\n` — linha 1 conta o canal 1 (0), linha 2 conta o canal 2 (1) |
| `curl http://192.168.100.3/health` (canal 1) | `{"ok":true,"database":"up","channels":2,"group":1,"channel":1,"online":0,"schema":30}` |
| `/health` do canal 2 (rede interna) | `{"ok":true,"database":"up","channels":2,"group":1,"channel":2,"online":1,"schema":30}` |
| `event=status` do canal 1 | `path=/serv00.htm bytes=4 online=0/1`, e `online=0/0` depois da limpeza |

Ou seja: o canal 1 mede **0** enquanto o canal 2 tem jogador, e vice-versa — cada processo só lê a
própria fatia de `character_session`. A visibilidade de **entidade** (um personagem aparecendo na
tela do outro) ainda não é observável porque não existe `ESCENE_FIELD`/tick; o que se prova hoje é o
isolamento estrutural (processos separados, sem IPC) mais a presença escopada por canal.

**Comportamento esperado no trânsito (sessão única):** entrar no canal 1 abre `character_session`
(grupo 1, canal 1); tentar a mesma conta no canal 2 é recusado com `result=conflict` e
`origin_channel=1`, sem `CNF` ao cliente; sair (socket cai) libera a sessão no `Drop` do
`SessionLease` e a mesma conta entra no canal 2 — nada é copiado, os dados vêm do banco (relog, não
teleporte). Ociosidade é renovada por `renewed_at`; processo morto libera pelo TTL
(`WYD_SERVER_SESSION_STALE_SECS`, 180 s).

**Provas/gates:** `cargo fmt --check` 0; `cargo clippy --workspace --all-targets -D warnings` 0;
`cargo test --workspace` 0 falhas; `WYD_TEST_DSN=… cargo test -p wyd-db --test sistema_sessoes` 1/1;
teste vivo `sessao_unica` 1/1 contra `.3:8281`/`.4:8281`; os dois containers `healthy` com a imagem
nova.

## 2026-11-09 — observabilidade padrão de mercado, presença auditável e banco do zero

Quatro entregas na mesma rodada. Contexto: o login manual do cliente continua sem linha nova nos
logs (só as conexões sintéticas dos testes aparecem) — as quatro tentativas humanas de 22:25/22:35Z
seguem registradas na entrada anterior, perdidas quando os containers foram recriados às 22:54Z.
Com o banco recriado abaixo, o próximo login manual cai nas contas de teste e a linha sai no formato
`event=login result=accepted account=… group=… channel=… session=…`; conferir com
`docker logs -t wyd-canal-1 | grep event=login` (idem canal 2).

**1) Logs duráveis, sem serviço caseiro.** O `logpipe` (sidecar próprio com o socket do Docker) e o
modo `docker compose logs -f` foram **removidos**. Novo caminho, padrão da stack Grafana:
`containers → Alloy (socket read-only) → Loki (volume wyd-loki-data, retenção 14 d) → BFF do portal`.
O motivo é o defeito que o pedido apontou: stdout do Docker é por container e morre na recriação —
exatamente no deploy em que o log importa. Arquivos: `monitoring/loki/loki.yml`,
`monitoring/alloy/config.alloy`, serviços `loki`/`alloy` no compose, volume `wyd-loki-data`.
Prova no ar: `/loki/api/v1/labels` devolve `container`, `job`, `service`; `query_range` traz linhas
dos dois canais e do Postgres; `up{job="loki"}` = 1 no Prometheus (job novo `loki`).
O portal consulta `LOKI_URL` server-side (`wyd-portal/src/server/loki.ts`) com whitelist de
containers, janela `\d+[mhd]`, `tail` 1..1000 e escape de string LogQL — nada cru do browser
(essa tela e os endpoints foram **removidos em 2026-11-09**, ver registro de execução).

**2) Monitoramento profissional na área de admin.** `/monitoramento` passou a ser um único painel
com: saúde da stack (badges de canal/gateway/loki/grafana/presença/alertas), KPIs (presença no banco,
canais, sessões, status, CPU/memória/load, gateway, volume de log 15 min), **presença por canal**,
**alertas** (firing/pending de `alerts.yml`), **pipeline de logs** (erros/avisos por serviço) e
**dashboards do Grafana embutidos** (embed único em kiosk na loopback, acesso anônimo `Viewer`; em 2026-11-09 virou `kiosk=true` + `_dash.hide*` e iframe de altura útil, sem moldura dupla). Dashboards
provisionados: `wyd-saude` e o novo `wyd-logs` (`monitoring/grafana/dashboards/wyd-logs.json`), com
datasource Loki ao lado do Prometheus (`monitoring/grafana/provisioning/datasources/loki.yml`).
Cada fonte degrada isolada: Prometheus fora não apaga presença/logs.
**Causa do 502 do portal encontrada e corrigida:** o compose repassava `WEB_API_ADDR` do `.env`
(`http://127.0.0.1:7600`, valor para rodar no host) para dentro do container, onde loopback é ele
mesmo — `/api/health` respondia 502 e o login web falhava. Agora o compose fixa
`WEB_API_ADDR: http://gateway:7600`; `/api/health` = 200 com `items: 3220`.

**3) Presença por canal auditável.** Nova rota de contrato
`server_admin/list_channel_presence` (`crates/wyd-admin/src/api/presence.rs` + `PresenceCheck` no
transporte + fiação no `wyd-gateway`): staff obrigatório, leitura **por canal** de
`character_session` (nunca agregado do grupo) e registro em auditoria (`server.presence`). O portal
consome via `adminCall` (`presenceAdmin` no `admin-client.ts`), então o `moderator_id` vem da sessão.
Nenhuma escrita: o portal só enxerga a foto. Prova: com uma sessão viva só no canal 2, a rota
devolveu `channel:1 online:0 / channel:2 online:1`; removida a linha, volta a 0/0. Testes unitários:
presença de um canal não aparece no outro e `player` não lê presença.

**4) Banco do zero + 5 contas de teste.** `DROP SCHEMA public CASCADE` seguido das **30 migrations**
(68 tabelas, `server_channel` 3 linhas, `content_version` 1). Conta default `admin` recriada pelo
seed do gateway (`WYD_SEED_STAFF=1`). Contas novas: `test1` (admin), `test2`/`test3` (moderator),
`test4`/`test5` (player) — ids 2..6, `password_hash` argon2id individual (97 bytes, `$argon2id$v=19$…`).
A senha pedida (`test123`, 7 caracteres) não passa do `MIN_PASSWORD_LEN = 8` do seed, então o hash
foi gerado com os **mesmos parâmetros** (`Argon2::default()`) por um utilitário descartável em
`tmp/hashgen` (pasta ignorada) e gravado por SQL; a política de força continua valendo para contas
reais. Prova: `account_web/verify_credentials` devolve `ok:true` com o papel certo para as cinco e
`ok:false` com senha errada. Login web pelo portal: `test1` → 200 `role:admin`; `test4` → 200
(entra no painel do jogador) mas `/monitoramento` = 307.

**Provas/gates:** `cargo fmt --check` 0; `cargo clippy --workspace --all-targets -D warnings` 0;
`cargo test --workspace` 0 falhas (contrato do portal ajustado para 17 serviços/73 métodos e rota de
presença montada no fake); `npx tsc --noEmit` 0; `npx vitest run` 137/137; `docker compose config`
(dev e prod) 0; containers `wyd-loki`/`wyd-alloy`/`wyd-grafana`/`wyd-gateway`/`wyd-portal` saudáveis.

---

### Identidade única (portal + launcher + cliente) — 2026-11-09

**1) Uma definição só.** `docs/arquitetura/identidade.tokens` (13 cores, `espaco.1..7`, `raio`,
`raio.pequeno`, `tipografia.*`, `servidor.1 = WYD`) é a fonte; o parser estrito vive em
`crates/wyd-launcher/src/identidade.rs` (`chave = valor`, nome de servidor ≤ 8 caracteres) e o
binário o embute com `include_str!` — não há literal de cor nem nome de servidor em Rust. O portal
tem os mesmos valores em `globals.css`; a cor do embed do Grafana virou `cor.embed`/`--embed`
(era `#111217` solto na linha 650). Guardas: `crates/wyd-launcher/tests/contrato/identidade.rs`
(tokens × CSS + proibição de qualquer hex de 6 dígitos fora da identidade) e
`wyd-portal/src/identidade.test.ts` (5 testes).

**2) Skin e nomes gerados da identidade.** `crates/wyd-launcher/src/branding.rs` gera
`Launcher.bmp` 598×369 24 bpp (gradiente `cor.surface`→`cor.bg`, faixa `cor.accent`, wordmark) e
`Launcher.hat` com `bgcolor = cor.bg`; `snbin.rs` escreve os 143 B medidos com o nome do slot
vindo de `servidor.N`. Comandos novos: `wyd-launcher branding <dir>` e o `sn.bin` passou a sair do
`serverlist`. O gerador Node (`tools/client/serverlist.mjs`) lê `servidor.N` do mesmo arquivo —
`NextGen` fixo foi removido; slot com grupo e sem nome agora é erro.

**3) Aplicado e lido de volta no `client/`.** `client/Launcher.bmp` = 662.778 B, `Launcher.hat` =
268 B (`bgcolor="#0f1115"`, `img Launcher.bmp`), ambos com `.bak`; `client/sn.bin` =
`d6 65 73 65 72 76 65 64 00 | 57 59 44` e contadores `[3, 2, 0…]` (slot 0 = canais+1, slot 1 =
grupo 1). Strings servidas conferidas: `curl http://127.0.0.1/serv00.htm` → `30 0a 30 0a` (`0\n0\n`)
e `server_channel` no banco = grupo 1 com canais 0/1/2 em `192.168.100.3` (status + canal 1) e
`192.168.100.4` (canal 2) — a ordem que o `sn.bin`/`serverlist.bin` desenham.

**4) Bloqueio declarado.** A moldura da tela de entrada dentro do `WYD.exe`
(`NUI/ServerList.wyt` 321.470 B, `main.wyt` 739.230 B, `mainchat/mainparts/ccmode/popup/iconon/
iconoff.wyt`, `UI/*.wyt`) tem magic `WT10` e nenhum decoder nosso confirma nada além do magic
(`tools/client/formats.mjs` → `recognised: false`). `strings client/wyd.exe` mostra `sn.bin`,
`UI\minimap.wyt`, `%s%02d%02d.wyt`, `mesh\*.wyt`, mas não a string `NUI\ServerList.wyt`; trocar
esses arquivos seria às cegas e não foi feito. A segunda aba de atributos/cidadania continua
dependente da captura de `STRUCT_SELCHAR`/`STRUCT_MOB` (FASE-08-6).

**Provas/gates desta passada:** `cargo fmt --all --check` 0; `cargo clippy --workspace
--all-targets -D warnings` 0; `cargo test --workspace` 0 falhas (launcher: 45 unitários +
4 de identidade + paridade Rust×Node); `npx vitest run src/identidade.test.ts` 5/5.
Nada commitado.

---

### `.wyt` decodificado, checklist da 2ª aba e GUI/login na identidade — 2026-11-09

**1) O `.wyt` não era opaco: é raster puro.** `tools/client/wyt.mjs` (novo) lê o magic `WT10` e
prova o leiaute `22 B de cabeçalho + largura*altura*(bits/8) + 8 B finais` em **256/259** arquivos de
`client/NUI` + `client/UI` (`NUI` 8/8); as 3 exceções (`NewAmul` 8 bpp, `hanui`/`joyui` 16 bpp) são
recusadas com erro, não chutadas. Dimensões reais medidas: `NUI/ServerList.wyt` **287×280 32 bpp**
(80.360 pixels, 12.273 cores, alfa 100%), `NUI/main.wyt` 616×300, `UI/PCK_Background.wyt` 800×600.
Prova visual: `node tools/client/wyt.mjs client/NUI/ServerList.wyt --png tmp/wyt/ServerList.png`
reproduziu a moldura e as duas colunas de slots da tela de entrada (texto não está no arquivo — é
desenhado pelo `.exe`). Relatório: `docs/cliente/gerado/wyt.json`. `formats.mjs` passou a delegar ao
decoder (`analyse.mjs` → `wyt 4/4`). Docs: `formatos.md` §1, `superficie-conhecida.md` §5, `mapa.md`.

**2) FASE-08-6 com checklist fechado.** A issue ganhou a seção de captura: 14 campos de
`STRUCT_SELCHAR` (nome, classe, nível, EXP, HP/MP, tier, cidadania, guilda, posição, PvP/PK, equip,
atributos, buffs, datas) e 6 de `STRUCT_MOB` (tamanho do frame, `STRUCT_SCORE`, inventário, gold,
guilda/cidadania, posição) — cada um com a verdade que já temos (`0002_personagens.sql`,
`canais-e-cidadania.md`) e o que o diff de bytes precisa provar. Método: variar **um** campo no
banco, capturar `0x010A`/`0x0114` nos dois estados e correlacionar; nada vira tipo antes disso.

**3) Login e GUI na mesma identidade.** `crates/wyd-launcher/src/gui.rs` (novo) gera
`LauncherGUI.html` (4.948 B) — login, canais da identidade e botão Jogar — com **só** `cor.*`,
`espaco.1..7`, `raio*`, `tipografia.*` e `servidor.N`; teste conta os hex do HTML e reprova qualquer
cor fora das 13 da identidade (e qualquer `http(s)://`). CLI: `wyd-launcher gui <dir> [--dry-run]`.
No portal, a tela de login trocou estilo inline por classes (`.login-*`) declaradas em
`globals.css` com `--space-*`, `--title-size` e `--legend-size`; `identidade.tokens` ganhou
`tipografia.titulo = 22` e `tipografia.legenda = 13`. Teste vitest novo: login sem hex nem
`style={{`, classes presentes no CSS.

**Provas/gates desta passada:** `cargo fmt --all` 0; `cargo clippy --workspace --all-targets -D
warnings` 0; `cargo test --workspace` 0 falhas (49 suítes `ok`; launcher 47 unitários + 4 de
identidade + paridade Rust×Node); `npx tsc --noEmit` 0; `npx vitest run src/identidade.test.ts` 6/6;
`wyt.mjs --scan` NUI 8/8 e UI 248/251. Nada commitado.

---

## Passada 2026-11-09 — entrada do jogo: mecânica de carga do NUI, senha numérica e mapa de funcionalidades

**1) A tela de seleção agora está na identidade, com prova visual em arquivo.** O caminho de carga
foi fechado por evidência: `client/UI/UITextureList.bin` (tabela de 779 entradas, 23 com "NUI")
lista `NUI\ServerList.wyt`, e o `.exe` monta o caminho com `%s\%s` (string em VA `0x613378`,
referenciada em `0x438340`) — não existe literal "NUI" nem "UITextureList" no binário. O re-tema é
feito em Rust (`crates/wyd-launcher/src/wyt.rs`, CLI `wyd-launcher retheme <origem> <destino>`), com
rampa de luminância montada só de `cor.bg`/`cor.surface-2`/`cor.border-strong`/`cor.accent`, alfa e
trailer preservados e instalação atômica com `.bak`. Capturas reais da janela (título "WYD CDK",
1030×788): `tmp/cliente/antes.png` (original) e `tmp/cliente/depois.png` (tema) — 72% dos pixels do
frame diferem (o número inclui a animação do cenário, que muda entre sessões); o que prova a troca é
o painel: moldura de madeira/dourado → navy com filetes dourados, no mesmo retângulo. Nada commitado.

**2) Densidade de pixel medida (respondendo "por que parece ampliado").** `ServerList.wyt` é
287×280 com duas colunas de 140 px; na tela as mesmas colunas aparecem com 181 px — **ampliação
≈1,29×**. O retângulo de desenho é do `.exe`, não do cabeçalho: logo o `.wyt` aceita arte mais densa
(2×, 4×) e o cliente reduz, sem mudar layout. Registro em `docs/cliente/customizacao.md` §9 e
`formatos.md` §6.

**3) Senha numérica pesquisada e mapeada.** É a segunda credencial (só dígitos) cadastrada na
primeira entrada, na tela de personagem, e validada a cada login; troca no painel da conta e
recuperação por e-mail (FAQ WYD Global). No fio é `_MSG_AccountSecure` `0x0FDE` / `_MSG_AccountSecureFail`
`0x0FDF`, corpo de 32 B já tipado em `payload/auth.rs`; o `wyd2encdec` publica pacotes de exemplo dos
dois opcodes. Doc novo: `docs/wyd/senha-numerica.md`.

**4) Mapa de funcionalidades criado.** `docs/rust/08-checklist-funcionalidades.md` com 13 áreas
(entrada/conta, personagem, mundo, combate, itens, skills, economia, guildas, quests/eventos,
social, admin/cliente), estado ✅/🚧/⬜/⛔ e evidência por linha. Achado importante do mapa: o login
**não verifica a senha** (`account_password` não é comparado com `accounts.password_hash`) e a senha
numérica **não tem handler** — os dois primeiros itens da fila.

---

## Passada 2026-11-09 — portal reformulado sobre o contrato real do `wyd-admin`

**1) Funcionalidade legada removida, não escondida.** O portal carregava telas e
serviços do servidor antigo (`npc_admin`, `drop_admin`, `donate_admin`,
`daily_reward_admin`, `world_state_admin`, `character_admin`…) que o gateway não
expõe. Foram removidos: 22 páginas administrativas (atlas, castelo, config,
diária, drops, eventos, faturamento, guildas, loja, mapa, mobs, NPCs, preços,
teleportes, personagens, detalhe de conta e detalhe de item), os componentes e
bibliotecas correspondentes, as rotas `/api/atlas`, `/api/mapa`, `/api/items` e
`/api/search`, o cliente de serviços legados e o cadastro de contas por lista.
Restaram apenas telas com fonte no contrato: `/painel`, `/contas`, `/itens`,
`/monitoramento`, `/conta` (jogador), `/guias` e `/enciclopedia`.

**2) Identidade e shell refeitos.** `globals.css` foi reescrito como sistema de
componentes sobre `docs/arquitetura/identidade.tokens` (obsidiana + ouro velho):
rail de navegação, topbar com blur, cabeçalho de página, hero com números,
cards em camadas, KPIs com ícone, tabelas densas, chips de status, diálogo e
formulários. Ícones passaram a `lucide-react`. O menu virou fonte única sem item
“em breve”; o teste de paridade da identidade continua verde. A tela de login
virou painel dividido e a área do jogador ganhou barra própria.

**3) Mentira de implementação saiu da UI.** `/monitoramento` não mostra mais
“Coletado em …”, não cita `issue`, `0x0FDE`, `prometheus:9090` nem caminho de
arquivo; o rodapé técnico e o link morto para `/logs` foram removidos. Os textos
descrevem estado e consequência, não a implementação.

**4) Senha numérica explicada corretamente.** O texto agora diz que a segunda
senha é por conta, pedida no cliente depois do login, **criada na primeira
entrada na tela de personagem** e trocada pelo fluxo de duas etapas do `0x0FDE`
tanto no cliente quanto no portal; o portal só troca uma senha existente e não
finge criar a primeira. Mesmas regras: 4–6 dígitos, 5 falhas → 10 minutos,
hash só no servidor, nada de PIN/senha em log ou browser.

**5) Contrato do BFF reduzido e alinhado.** `admin-client.ts` declara só
`accounts/find`, `accounts/create`, `item_catalog/list_items`,
`account_web/verify_credentials` e `server_admin/list_channel_presence`.
`/contas` virou busca exata + criação com papel (a listagem paginada não existe
no gateway); `/itens` mostra só os campos do payload real (índice, nome,
exibição); `/api/health` separa portal vivo de gateway fora. README reescrito
com a tabela de telas e as referências de desenho consultadas.

**Provas/gates desta passada:** `npx tsc --noEmit` 0; `npx vitest run` 79/79;
`npm run build` 0 (22 rotas). Nada commitado.

## 2026-11-10 — criação de personagem completa e entidade no gauge

**Criação com a ficha do legado.** `characters.rs` deixou de gravar `hp=1/mp=0/max=0`
(o que fazia `Stats::validate` recusar a entrada) e passou a derivar a ficha de
`origin/main:Source/Code/Basedef.cpp`: `BaseSIDCHM[4][6]`, `g_pIncrementHp/Mp`,
`BASE_GetHpMp`, 5 pontos de atributo e 3 de skill por nível no MORTAL. A tabela 1:1
`character_scores` (migration 0032) guarda FOR/INT/DES/CON e os pontos livres; o roster e o
`STRUCT_MOB` provisórios também carregam esses campos, e `CNFNewCharacter` devolve o roster
real (como o `_MSG_DBCNFNewCharacter` antigo), não só o cabeçalho.

**Prova viva do painel.** Harness `tests/sistema/selecao.rs::sessao_viva_publica_entidade_no_gauge`
com `admin` + PIN: criação de `LiveEnt3` no slot 1 pelo protocolo, entrada no canal 1 e
sessão mantida aberta. Durante a sessão o Prometheus devolveu
`wyd_entities_active{canal="1"} 1`, `wyd_tick_budget_microseconds{canal="1"} 50000` e p99
em 0,99% do orçamento; o painel 11 do Grafana renderizou canal 1 em 1 e canal 2 em 0.
Depois do FIN, o gauge voltou a 0, o personagem foi removido pelo `DeleteCharacter` e
`character_session` ficou sem linha aberta. Banco no intervalo: HP 83/83, MP 46/46,
FOR/INT/DES/CON 8/4/7/6, pontos 5/3 para TransKnight nível 1.

### 2026-11-11 — teste manual do dono: movimento, NPCs e mensagens

O log do canal mostrou a causa raiz dos "NPCs que não abrem": **todo `_MSG_Action` de
movimento era recusado** (`combate recusado: passo de 22 tiles acima do permitido (4)`) e o
`talk` exigia o personagem a 5 tiles do clique — a loja nunca chegava a abrir e só o NPC
colado funcionava. Correções: clique é **destino de caminhada** (`Engine::walk_pending`, um
orçamento por tick), clique no sprite resolve o NPC a distância (`CLICK_RANGE_TILES = 2`) e o
raio da cena subiu de 24 para **40** tiles (o anel de mercadores de Armia em `y >= 2121`
ficava fora). Testes: `clique_longe_e_destino_de_caminhada_e_nao_recusa`,
`clique_no_sprite_do_npc_longe_devolve_alvo`; gates `fmt`/`clippy -D warnings`/`test` verdes e
canais 1 e 2 reconstruídos healthy.

Desassemble do `WYD.exe` (7662) fechou a dúvida da senha errada: o cliente **não compara
`0x0117` (`_MSG_NewAccountFail`)** em nenhum ponto; conhece 87 opcodes do nosso catálogo
(inventário em `docs/cliente/mensagens-cliente-7662.md`) e desenha texto livre por
`_MSG_MessagePanel` (`0x0101`). A recusa de senha passou a mandar também o
`MessagePanel` com "Senha ou conta incorreta" — o `*Fail` continua sendo o pacote de
contrato. Nome vermelho fechado no mesmo dia: `ChaosRate@15` é o **CP** codificado em `+75`, não
um stat; o cliente (`0x4e2393`) lê `pMob+0x11e`, limita a `10..150` e interpola a tabela
`0x6188a4` (vermelho → laranja → amarelo → verde → azul → branco), e inicializa mob com `0x4B`
(75 = CP 0) em `0x4d9ae2`. O servidor mandava 0 em `BaseScore`/`CurrentScore`/`CreateMob`; agora
manda `CP_NEUTRO = 0x4B`, com teste `personagem_nasce_com_cp_neutro_e_nao_com_o_lixo_do_template`.
Os demais valores da criação já estavam corretos no banco (83/46, 8/4/7/6, 5.000.000, pontos 0/0).

**CP persistido (2026-11-11):** coluna `characters.cp` (`-75..=75`, migration 0048), levada no
`LiveState`/save e convertida para o byte do fio (`cp + 75`) em `CreateMob`, `UpdateScore` e
`STRUCT_SELCHAR`. Regra pura em `crates/wyd-sim/src/formula/chaos.rs` (faixa, cor, penalidade de
kill, piso de PK). No abate PvP o autor perde 2 (vítima `0..24`) ou 3 (`>=25`) de CP, vítima já
listada não dá penalidade, e autor em `-75` não inicia PK; a vítima volta com HP cheio. PvP é
ligado pelo papel do canal (`Engine::set_pvp`).

**`/cp` no chat (2026-11-11):** `_MSG_MessageChat` (0x0333) passou a ser lido — 140 B, texto em
`+12` (sender do cliente `0x4695b3`) — e o comando responde no `_MSG_MessagePanel` com CP, faixa e
efeito (penalidade de 2 e piso de PK). Chat social no mesmo pacote: `session::ChatHub` guarda uma
fila por sessão em jogo, o falante entrega só para as sessões cuja vista inclui o próprio id de
cena (`HEADER.ID` do `_MSG_MessageChat`, que o cliente resolve no mob para desenhar
`nome : texto`) e cada conexão drena a fila na própria thread, com o `key_word` dela.

**Anti-flood e mute (2026-11-11):** o `ChatHub` limita **3 falas por 5 s** por sessão
(`/cp`, `/mute` e `/unmute` não consomem cota) e guarda por destinatário a lista de ignorados
(`/mute Nome`, `/unmute Nome`, `/mute` sem nome lista). Recusa do flood, confirmação do mute e
lista saem no `_MSG_MessagePanel`; a entrega pula quem silenciou o falante.
