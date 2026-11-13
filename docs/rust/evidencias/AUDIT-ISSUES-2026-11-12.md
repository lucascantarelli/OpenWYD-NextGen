# Auditoria de fechamento — 29 issues ativas (2026-11-12)

> Escopo: confrontar cada critério de aceite com código, testes e documentação existentes,
> não com o texto da própria issue. Resultado: **29 conferidas, 2 fechadas e 27 mantidas**.
> As duas concluídas saíram de `roadmap-rust/issues/` e estão no `roadmap-rust/changelog.md`.

## Resultado

| Decisão | Issues |
|---|---|
| Fechadas | `GAME-19`, `WEB-01` |
| Mantidas com resíduo provado | `BUG-01`, `BUG-02`, `CLIENT-06`, `CLIENT-08`, `CLIENT-10`, `CLIENT-11`, `GAME-05`, `GAME-11`, `GAME-13`, `GAME-14`, `GAME-17`, `GAME-18`, `GAME-20`, `GAME-21`, `GAME-22`, `GAME-23`, `PROC-01`, `PROC-24`, `QA-01`, `QA-09`, `QA-10`, `QA-11`, `WEB-07`, `WEB-08`, `WEB-09`, `WEB-10`, `WEB-11` |

## Matriz das 29

| ID | Status | Evidência real | Critério que mantém aberta |
|---|---|---|---|
| `BUG-01` | ⬜ | `cpsock::reply_recusa`, testes `login`/`selchar`/`cena_mobs`; `docs/cliente/mensagens-cliente-7662.md` | Matriz A1–A6 e cada recusa reproduzida no 7662 com print/log. |
| `BUG-02` | 🟢 | `authorize_pin` + `serve_selection`; `cargo test -p wyd-server --lib`; `clippy` | Prova viva com `WYD_LIVE_SKIP_PIN=1` e entrada no cliente real. |
| `CLIENT-06` | 🚧 | 5 goldens C→S, handshake ADR-0020 e `cena_mobs` vivo | Captura ao vivo na VM/cliente; S→C completo e causa do freeze. |
| `CLIENT-08` | 🚧 | `apps/launcher-ui`, E2E Tauri, `client.zip` e reparo por hash | Barra de patch e instalador NSIS. |
| `CLIENT-10` | ⛔ | `tools/client/trade-layout.mjs` e layout candidato de 156 B | Frame real de `MSG_Trade`; codec e swap atômico. |
| `CLIENT-11` | 🚧 | Núcleo do launcher, config assinada, notícias, telemetria e painel | Geração de patch no admin e autoupdate assinado do launcher. |
| `GAME-05` | 🚧 | `systems/{movement,combat,skill,inventory}`, ADR-0014/0015 e testes de pacote forjado | `req_str/int/dex/con`, layout de rede do inventário e baú. |
| `GAME-11` | 🚧 | `inventory::deposit`, `GroundLoot`/`LootAccess` e `integracao_campo_drops` | Handlers `CreateItem`/`GetItem`/`DecayItem` e faixa de id do chão. |
| `GAME-13` | ⬜ | Dossiê `docs/wyd/sistemas/familiares.md` | Slot no 7662, efeito Nyerds e recusas de equipamento. |
| `GAME-14` | ⬜ | Dossiê `docs/wyd/sistemas/fadas.md` e regra de drop existente | Bônus por fada, duração, auto-coleta e auto-venda. |
| `GAME-17` | 🚧 | Faixa `ClientID`/`MobID`, `UpdateScore`, `CreateType` e `cena_mobs` vivo | Render da entrada no 7662 sem “Falha ao conectar”. |
| `GAME-18` | 🟡 | `_MSG_Quest`, `character_quest`, portões, loot e `integracao_campo_drops` | Buff do `Cabuncle`, portão visível e prova visual. |
| `GAME-19` | ✅ | `formula/*`, `scenario/combat.rs`, exemplo numérico §3.2.1 e teste de cadeia | Fechada: ordem provisória explicitamente marcada; trace do legado fica em `CLIENT-06`. |
| `GAME-20` | ⬜ | `formula/speed.rs` marcado ⛔ | Causa raiz da velocidade ~200% e correção modelada. |
| `GAME-21` | ⬜ | Fórmulas puras e `determinism.rs` | Harness de milhões de iterações e relatório. |
| `GAME-22` | ⬜ | `grant_exp`/`exp_reward` atuais e teste de level-up | Flags por ajuste com default inalterado e testes isolados/combinados. |
| `GAME-23` | 🟡 | `crates/wyd-bots` com login, ação, combate, loja e campanha | Quatro bots em paralelo no roteiro completo + relatório por raça. |
| `PROC-01` | 🚧 | `mkdocs.yml`, `docs/index.md`, `09-duplicidades.md`, 18 skills | Fusões de refino/pesquisas datadas e `mkdocs build --strict` reexecutado. |
| `PROC-24` | 🚧 | Fila `#01..#27`, grafo, caminhos críticos e `check-issue-status` | Onda 1 demonstrável no 7662 e DoD explícito por onda. |
| `QA-01` | ⬜ | `crates/wyd-bots` (protocolo real) | População, LLM híbrida, carga e logs reproduzíveis. |
| `QA-09` | ⬜ | `cargo test --all-targets` existe no CI | Tarpaulin/miri e concorrência nos críticos com evidência. |
| `QA-10` | ⬜ | Baseline QA-04/QA-08 e build release existentes | Carga P99, banco do zero, imagem/smoke e relatório de release. |
| `QA-11` | 🟡 | `crates/wyd-bots` e `cargo tree` sem `wyd-core`/`wyd-db` | Política por seed, prova viva de 2 bots e carga de 200. |
| `WEB-01` | ✅ | 44 métodos com handler, `admin.pending=0`, contrato 12/12 + 14/14 | Fechada: rota legada removida do contrato (404) sem pendência. |
| `WEB-07` | 🚧 | Páginas de monitoramento e `/api/logs/export` | SSE, CPU/mem/rede, saúde do DB via gateway e alertas com acknowledge. |
| `WEB-08` | 🚧 | Contas, auditoria, home pública e área do jogador | 2FA/recuperação/bloqueio automático e catálogo público completo. |
| `WEB-09` | 🚧 | Correio staff, financeiro e replay idempotente | Inbox do jogador, extrato por comprador e remoção/redirect de `/entregas`. |
| `WEB-10` | 🚧 | Catálogo, ficha, preços por mercador, relatórios e alertas | Atributos/requisitos na ficha, fonte de todo item e queries nomeadas. |
| `WEB-11` | 🚧 | Abas por domínio, validação, auditoria e recarga viva | Rollback de revisão e inventário `setting` × constante. |

## Provas das duas conclusões

- `GAME-19`: `cargo test -p wyd-sim damage` → **13 passed**, incluindo
  `worked_example_matches_documented_chain` (250) e `stage_order_is_the_documented_one`.
  Documento: `docs/arquitetura/matematica-sim.md` §3.2.1.
- `WEB-01`: `cargo test -p wyd-admin --test contrato_operacao` → **12 passed**;
  `cargo test -p wyd-admin --test contrato_transporte` → **14 passed**. A rota
  `daily_reward_admin/set_reward_item_enabled` não está mais em `SERVICOS` e responde 404,
  sem `admin.pending`.

## Regras aplicadas

- Nenhuma issue foi fechada por “parece pronto”; só por critério de aceite demonstrável.
- Checklist marcado apenas com teste/doc existente; o que depende do 7662/VM continua aberto.
- As três skills genéricas aposentadas não voltaram a ser citadas; as referências válidas são
  `wyd-web-admin-ui`, `docs/index.md` e `docs/manual-técnico/skills.md`. O `skills-check` valida
  caminhos de skill existentes, frontmatter e recursos citados.
