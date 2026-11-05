# Roadmap — Dolorestec WYD2

Plano de execução do servidor local do WYD2 (servidor Go + portal administrativo em `portal/`),
usando o cliente modificado em `client/`, incluindo a camada de agente de IA que administra e
evolui o servidor.

Este índice é a **fonte da verdade do progresso**. Cada issue em [`issues/`](issues/) é uma unidade
de trabalho executável por um agente ou por uma pessoa, com passos, arquivos, testes e resultado
esperado. Ao concluir uma issue, atualize a linha dela aqui e o `docs/migration/SESSION-PRIMER.md`
quando o comportamento observável mudar.

## Como ler uma issue

Cada arquivo segue o mesmo formato:

```
# Issue NN — título
Fase · Depende de · Bloqueia · Esforço (S/M/L)
Objetivo            — o resultado verificável, em uma frase
Contexto/evidência  — por que isso é necessário, com caminhos de arquivo
Passos              — sequência executável
Arquivos            — o que é criado/alterado
Testes/verificação  — comando + saída esperada
Critérios de aceite — checklist objetivo
Riscos/rollback     — o que pode dar errado e como voltar
```

Regra de ouro: **nenhuma issue é "concluída" sem evidência de execução** (log, teste verde, saída de
comando). "Escrevi o código" não fecha issue.

## Estado atual (2026-11-04)

| Item | Estado | Evidência |
|------|--------|-----------|
| Working tree completo (`Release/`, `scripts/`, `internal/`, `api/`, `dbserver/`, `binserver/`, `webserver/`) | ✅ restaurado nesta sessão | `git sparse-checkout disable`; `Release/` com 67 MB |
| Árvore de conteúdo íntegra | ✅ verificado | `ItemList.csv`, `SkillData.csv`, `npc/` 1.991 templates, `HeightMap.dat` 16 MiB, `AttributeMap.dat` 1 MiB, `BaseMob/{TK,FM,BM,HT}` 816 B |
| Toolchain Go no host | ❌ ausente | `go: command not found`; caminho é container `golang:1.26-alpine` |
| Docker / Compose | ✅ presente | Docker 29.9.2 daemon `29.9.2`, 8 CPUs / 16 GB; Compose v5.6.1 |
| Build da stack | ✅ 5 imagens construídas | build **serializado** por serviço (`db`, `dbserver`, `binserver`, `webserver`, `tmserver`), todos exit 0; o build paralelo era o que derrubava o daemon |
| Stack no ar | ✅ `db`, `binserver`, `dbserver`, `webserver`, `tmserver` Up | `tmserver listening addr=:8281`, status `:80` → HTTP 200; `NPCs spawned mobs=11930`, `walkability grid baked dim=4096` |
| Cliente apontando para o servidor local | ⏳ pendente | `serverlist.bin` atual aponta para canais remotos; `.remote_serverlist.bin` é uma página HTML de erro |
| Camada de agente de IA (AGENTS.md + skills) | ✅ criada nesta sessão | `AGENTS.md`, `.agents/skills/wyd-*` |

## Dossiê de análise (2026-11-04)

Antes de escolher a próxima frente, leia o dossiê: ele responde "o que é o projeto, o que do legado
ainda existe, o que pode ir para o banco, o que dá para fazer no cliente" com evidência do
repositório e pesquisa externa.

| Documento | Para quê |
|-----------|----------|
| [`docs/analysis/README.md`](../docs/analysis/README.md) | índice e resumo executivo |
| [`docs/analysis/estado-atual.md`](../docs/analysis/estado-atual.md) | inventário, respostas 1–10 e as 3 perguntas adicionais, dívidas |
| [`docs/analysis/migracao-legado.md`](../docs/analysis/migracao-legado.md) | mapa C++ → Go, o que falta, limites |
| [`docs/analysis/persistencia-conteudo.md`](../docs/analysis/persistencia-conteudo.md) | arquivos × banco, o que migrar e o que não |
| [`docs/analysis/cliente-launcher.md`](../docs/analysis/cliente-launcher.md) | cliente 7.64/12000, modernização e launcher |
| [`docs/analysis/qa-ingame.md`](../docs/analysis/qa-ingame.md) | fila de testes reais e sistema de feedback |
| [`docs/analysis/qualidade-e-processo.md`](../docs/analysis/qualidade-e-processo.md) | plano de engenharia e marcos |
| [`PLANO-2026-11.md`](PLANO-2026-11.md) | ordem de execução (fecha o que está pronto primeiro) |

## Revisão de engenharia (REVIEW-01..10)

O dossiê de 04/11 respondeu "o que é o projeto"; a série `REVIEW-*` transforma cada frente em
trabalho executável, **em sequência**, e cada uma abre issues filhas (`REFACTOR-`, `TEST-`, `CI-`,
`DB-`, `DOC-`, `PORTAL-`, `OBS-`, `CLEAN-`) com critério de aceite próprio. A ordem começa pelo que
fecha issue rápido (teste real no cliente) e termina na faxina que depende de todo o resto.

| Ordem | Issue | Entrega | Esforço |
|---:|---|---|---|
| 1 | [REVIEW-01](issues/REVIEW-01.md) | Triagem da fila in-game: fechar/reabrir as 🟡 com evidência | M |
| 2 | [REVIEW-02](issues/REVIEW-02.md) | Código Go: docstrings/comentários em PT, refatorações, `newApp` | L |
| 3 | [REVIEW-03](issues/REVIEW-03.md) | Testes: arquitetura, triagem de `t.Skip`, Playwright, fixtures | L |
| 4 | [REVIEW-04](issues/REVIEW-04.md) | CI/CD: workflows reutilizáveis, banco no pipeline, SBOM, `make ci` | M |
| 5 | [REVIEW-05](issues/REVIEW-05.md) | Banco: roles, restore testado, naming, índices de auditoria | M |
| 6 | [REVIEW-06](issues/REVIEW-06.md) | Documentação: mkdocs, índice de sistemas, ADRs, `doc_legacy` | L |
| 7 | [REVIEW-07](issues/REVIEW-07.md) | Legado C++ e ClientPatch_v7662: classificar e decidir | M |
| 8 | [REVIEW-08](issues/REVIEW-08.md) | Portal: público × admin, loja, fóruns, identidade visual | L |
| 9 | [REVIEW-09](issues/REVIEW-09.md) | Observabilidade: `/metrics`, `trace_id`, alertas, segredos | M |
| 10 | [REVIEW-10](issues/REVIEW-10.md) | Limpeza final: IA/skills, `openspec`, `scripts/`, DoD | M |

Regra: uma REVIEW só começa quando a anterior tem evidência registrada — cada uma abre issues
filhas que podem ser executadas em paralelo **dentro** dela, nunca fora de ordem.

## Fases e issues

### Fase 0 — Fundação: ambiente, conteúdo e banco
| # | Issue | Estado | Esforço |
|---|-------|--------|---------|
| 01 | [Restaurar e validar a árvore de trabalho](issues/issue-01.md) | ✅ | S |
| 02 | [Provisionar ambiente de execução (Go/Docker/portas)](issues/issue-02.md) | ✅ (`scripts/preflight.sh`: checkout, `Release/`, daemon/compose, portas e Go; exit 1 com 11 `[FAIL]` numa árvore incompleta e exit 0 aqui) | S |
| 03 | [Subir Postgres, migrações e contas seed](issues/issue-03.md) | ✅ | S |
| 04 | [Subir e validar o tmServer com conteúdo](issues/issue-04.md) | ✅ | M |

### Fase 1 — Cliente modificado e login
| # | Issue | Estado | Esforço |
|---|-------|--------|---------|
| 05 | [Configurar o cliente para o servidor local](issues/issue-05.md) | ✅ (04/11: status + CPSock + `0x020d`; bin usado é o de 02/11, sem `.bak`) | M |
| 06 | [Alinhar versão de protocolo e autenticar](issues/issue-06.md) | ✅ (04/11: cliente envia **12000**; `account login: OK … chars=2`) | S |
| 07 | [Login ponta-a-ponta (headless + gráfico)](issues/issue-07.md) | ✅ (04/11: entrou no mundo, andou, abriu loja e comprou; headless PASS) | M |

### Fase 2 — Personagens e persistência
| # | Issue | Estado | Esforço |
|---|-------|--------|---------|
| 08 | [Criação e seleção de personagem com equip inicial](issues/issue-08.md) | ⏳ | M |
| 09 | [Persistência de personagem, cargo e contas/papéis](issues/issue-09.md) | ⏳ | M |

### Fase 3 — Funcionalidades do jogo
| # | Issue | Estado | Esforço |
|---|-------|--------|---------|
| 10 | [Movimento, visão, cidades e teleporte](issues/issue-10.md) | ⏳ | M |
| 11 | [Combate PvE, IA de mob, drop e EXP](issues/issue-11.md) | ⏳ | L |
| 12 | [Comércio: loja NPC, banco/cargo, trade e autotrade](issues/issue-12.md) | ⏳ | L |
| 13 | [Quests, refino/combine e progressão](issues/issue-13.md) | ⏳ | L |

### Fase 4 — Agente de IA e automação
| # | Issue | Estado | Esforço |
|---|-------|--------|---------|
| 14 | [Camada de agente: AGENTS.md, skills e disciplina de execução](issues/issue-14.md) | ✅ (v1) | M |
| 15 | [Ferramentas de administração e economia para o agente](issues/issue-15.md) | ⏳ | M |
| 16 | [Bots de teste e harness de carga](issues/issue-16.md) | ⏳ | L |

### Fase 5 — Operação e evolução
| # | Issue | Estado | Esforço |
|---|-------|--------|---------|
| 17 | [Observabilidade, backup, CI e hardening](issues/issue-17.md) | ⏳ | M |

### Fase 6 — Portal web de administração (UI completa)

| # | Issue | Estado | Esforço |
|---|-------|--------|---------|
| 18 | [Portal Next.js: bootstrap do BFF e cliente gRPC](issues/issue-18.md) | ✅ | S |
| 19 | [Login web, sessão httpOnly e guarda de papel](issues/issue-19.md) | ✅ | S |
| 20 | [Shell da UI: layout, navegação e componentes base](issues/issue-20.md) | ✅ | M |
| 21 | [Camada de admin no BFF: resultado→HTTP e auditoria](issues/issue-21.md) | ✅ | M |
| 22 | [Catálogo de itens com ícones (leitura)](issues/issue-22.md) | ✅ | M |
| 23 | [Editor de NPCs e mercadores](issues/issue-23.md) | ✅ (UI 03/11; NPC do portal visível e comprável no jogo 04/11) | L |
| 24 | [Preços globais de itens](issues/issue-24.md) | 🟡 (UI ✅, in-game ⏳) | S |
| 25 | [Editor de stats e equip de mobs](issues/issue-25.md) | 🟡 (lista, stats com read-through/salvar/reverter, badge/filtro de override e agora o editor de `Equip[]` com picker de item ✅ ao vivo — `SetMobTemplateEquip(slot 5, item 412)` devolve `equip=[[5,412]]`; falta só a evidência in-game) | L |
| 26 | [Relatórios de drop: item ↔ mob](issues/issue-26.md) | ✅ (duas visões, CSV e cache; validado ao vivo: 1.349 templates, 13.813 slots, consistência item↔mob, player 403) | M |
| 27 | [Editor do mapa de colisão (AttributeMap)](issues/issue-27.md) | ✅ (info + histograma + transformação + download de 1 MiB com sha conferido; `/mapa` 200 e rota de download 200) | M |
| 28 | [Eventos globais do mundo](issues/issue-28.md) | ✅ (`/eventos` com estado ativo/esgotado/incompleto, item por catálogo, divisor em "1 em cada N", gate de EXP dobrada e "desligar tudo"; salvar relê o servidor e o tmserver recarregou em ~7 s sem restart) | S |
| 29 | [Administração da loja de donate](issues/issue-29.md) | ✅ (`/loja`: CRUD com preço em donate, efeitos e `expires_days` do proto, habilitar/desabilitar, apagar exigindo o título digitado e crédito manual com motivo + confirmação dupla, tudo auditado) | M |
| 30 | [Administração da recompensa diária](issues/issue-30.md) | ✅ (`/diaria` com CRUD completo, item por catálogo, efeitos/`expires_days` fiéis ao proto, habilitar/duplicar/apagar com confirmação; resgate provado duas vezes → `ALREADY_CLAIMED` e `delivery_queue` `daily_reward:1`) | S |
| 31 | [Faturamento e relatórios de donate](issues/issue-31.md) | ✅ (`/faturamento` só `admin`: janela 7/30/90/personalizada em Brasília, receita em R$ separada dos créditos de donate, pendentes em cartão próprio ("não é receita"), série em barras por dia/semana/mês, pedidos com CSV, top compradores com LTV, extrato da carteira e filtro por conta; 6 KPIs conferidos contra SQL, `moderator` → `/sem-permissao`) | M |
| 32 | [Contas, papéis e banimento (backend novo)](issues/issue-32.md) | ✅ (04/11: backend, portal e recusa **vista no cliente** — “Conta bloqueada Grade X…” após corrigir o formato do aviso; migração `0042_account_admin` com `account_admin_audit`, `AccountAdminService` (list/keyset, get+auditoria, set role/block, create) e `/contas` com filtros, detalhe e motivo obrigatório; regras no store dentro da transação — sem auto-alteração, última admin ativa protegida, moderador só bloqueia jogador; e2e contra o web-api ✅ e `VerifyCredentials` vê o bloqueio. Falta só a recusa no login do cliente) | L |
| 33 | [Catálogo de itens editável (overlay de ItemList.csv)](issues/issue-33.md) | ⏳ | L |
| 34 | [Drops editáveis: taxa por slot e Carry[] do mob](issues/issue-34.md) | 🟡 (taxa por slot e `Carry[]` ✅ ponta a ponta: migrações 0040/0041, `DropAdminService` com os dois editores — **`/drops/editar`** e **`/drops/carry`** —, overlays no boot com log `templates=1`/`applied=1`, `Set/Restore` validados e auditoria; falta só a evidência in-game de frequência) | L |
| 35 | [Curva de EXP e rebalanceamento em massa](issues/issue-35.md) | ⏳ | M |
| 36 | [Rates editáveis e visão das flags](issues/issue-36.md) | 🟡 (overlay + `/config` + read-through + aba de flags ✅ validados ao vivo: 16 flags gravadas no boot e tabela em `/config`; falta só evidência in-game de refino/combine) | M |
| 37 | [Spawns e localização de mobs](issues/issue-37.md) | ⏳ | L |
| 38 | [Operação ao vivo: online, kick, teleport e aviso](issues/issue-38.md) | ⏳ | L |
| 39 | [QA final do portal: E2E, segurança e fechamento](issues/issue-39.md) | ⏳ | M |
| 40 | [Logs dos containers na UI (streaming)](issues/issue-40.md) | ✅ (`/logs` em SSE por serviço + "Tudo", agora **funcionando no container**: sidecar `logpipe` lê o socket do Docker read-only e serve `/status`/`/logs`; o portal autentica e repassa. 401 sem sessão, 403 `player`, 404 fora da whitelist, `portal` incluído, pausa/filtro/limpar/baixar) | M |
| 41 | [Atlas do mundo: zonas, coordenadas e portais](issues/issue-41.md) | ✅ | M |
| 42 | [Rotas de teleporte editáveis](issues/issue-42.md) | ✅ | L |
| 43 | [Inventário e status de personagem](issues/issue-43.md) | 🟡 leitura+fila | L |
| 44 | [Adds e efeitos de item (base, instância, refino, combine)](issues/issue-44.md) | 🟡 (leitura ✅ + overlay ✅ + read-through ✅; refino/combine ⏳) | L |
| 45 | [Guildas, castelo e Kefra](issues/issue-45.md) | 🟡 (leitura ✅, taxa ✅, dono de cidade/torre ✅, Kefra ✅, escritas de guilda/membro/relação ✅; falta a janela de horários, que é da 36) | L |
| 46 | [Catálogo de skills](issues/issue-46.md) | 🟡 (nomes ✅) | M |
| 47 | [UI sem IDs crus e padrão de UI/UX](issues/issue-47.md) | ✅ (nomes no lugar de IDs, `--space-*`, foco, alvo 32px, contraste medido, retry global; `/config` e `/drops` respondendo 200 com sessão) | M |
| 48 | [Mapa de navegação: relevo, zoom e leitura de ponto](issues/issue-48.md) | 🟡 (relevo/zoom/leitura ✅) | M |

### Fase 7 — Fechamento in-game e consolidação (nov/2026)

Ordem definida em [`PLANO-2026-11.md`](PLANO-2026-11.md): primeiro o que já está pronto e só espera
evidência humana, depois o risco estrutural, depois produto.

| # | Issue | Estado | Esforço |
|---|-------|--------|---------|
| 49 | [QA in-game em lote](issues/issue-49.md) — fechar as 13 issues 🟡 | ⏳ | M |
| 50 | [Conteúdo do servidor no banco](issues/issue-50.md) — CSV vira seed | ⏳ | L |
| 51 | [Launcher próprio](issues/issue-51.md) — patch, serverlist e login | ⏳ | L |
| 52 | [Cliente: offsets, assets e resolução](issues/issue-52.md) | ⏳ | L |
| 53 | [Feedback dos jogadores](issues/issue-53.md) — bug/sugestão no portal | ⏳ | M |
| 54 | [Hub de documentação, ADRs e runbooks](issues/issue-54.md) | ⏳ | M |
| 55 | [Qualidade: CI, banco efêmero e observabilidade](issues/issue-55.md) | ⏳ | L |
| 56 | [Limpeza do repositório e legado arquivado](issues/issue-56.md) | ⏳ | S |
| 57 | [NPCs: migrar todos os blocos e dropar `NPCGener.txt`](issues/issue-57.md) | ✅ | L |
| 58 | [Baseline do banco: uma inicialização completa, sem arquivos de conteúdo](issues/issue-58.md) | ✅ | L |
| 59 | [Configs de conteúdo restantes no banco (guard, level item, mob merc, game settings)](issues/issue-59.md) | ⏳ | M |
| 60 | [Mapas (`HeightMap`/`AttributeMap`): banco ou asset?](issues/issue-60.md) | ⏳ | M |
| 61 | [Remover os artefatos do DropTool (`ItemDropList`/`MobDropList`)](issues/issue-61.md) | ⏳ | S |
| 62 | [webserver: catálogos de conteúdo ainda vêm de arquivo](issues/issue-62.md) — resíduo da fase 5 | ⏳ | M |

Skill de execução: `.agents/skills/wyd-web-admin-ui/SKILL.md`. O portal mora em `portal/`, dentro do
repo, com contexto Docker próprio (`portal/Dockerfile`) para não inflar o contexto dos serviços Go.
As issues 18–31
consomem contratos que **já existem** no `webserver` (gRPC :7600); as 32–38 criam os contratos que
faltam (contas/papéis, catálogo, drops, EXP, rates, spawns, live ops); a 39 fecha com E2E e segurança; a 40 abre a observabilidade dentro do portal (logs ao vivo dos containers);
a 47 padroniza a UI (nome no lugar de ID, tokens de espaço e a11y) sobre `frontend-design` e
`ui-ux-practices`.

## Grafo de dependências

```
01 ──► 02 ──► 03 ──► 04 ──┬──► 05 ──► 06 ──► 07 ──┬──► 08 ──► 09
                          │                        │
                          └────────────────────────┴──► 10 ──► 11 ──► 12 ──► 13

14 (contínuo, não bloqueia)         15 ── depende de 03+04
16 ── depende de 07 (login headless funcionando)
17 ── depende de 04 (stack estável)
```

Caminho crítico para a meta principal ("servidor no ar + login com o cliente modificado"):
**01 → 02 → 03 → 04 → 05 → 06 → 07**. Nada mais deve ser iniciado antes de 07 fechar, exceto 14
(documentação de agente) e 15/16 em paralelo depois de 04.

Grafo da Fase 6:

```
18 ──► 19 ──► 20 ──► 21 ──┬──► 22 ──► 23 ──► 24
                          │           └────► 26 ──► 34
                          ├──► 25 ──► 26, 34, 35
                          ├──► 27 ──► 28 ──► 29 ──► 30, 31
                          ├──► 32 ──► 38
                          └──► 33, 36, 37
39 ── depende de 18–38 (fechamento)
40 ── depende de 20 (shell); independente de 21–38
41 (atlas) ── bloqueia 42 (teleportes) e 45 (guildas/castelo)
44 (adds/refino) ── depende de 25+33; 43 (personagens) ── depende de 32+38
```

## Metas do projeto e onde são verificadas

| Meta | Issue de fechamento | Prova |
|------|--------------------|-------|
| Servidor local com funcionalidades básicas | 04 | log `content loaded` + `NPCs spawned` + status HTTP |
| Login com o cliente modificado | 07 | cliente entra no mundo; log `0x020d`/`0x0213` |
| Criar e configurar personagens | 08 | 4 classes criadas com BaseMob e equip inicial |
| Batalha, comércio e quests testados | 11, 12, 13 | cenários da skill `wyd-testing-bots` §6 |
| Agente de IA para administração/manutenção | 14, 15 | skills + ferramentas + auditoria |
| Bots controlados por IA | 16 | harness + bot headless jogando o roteiro |

## Riscos estruturais

| Risco | Impacto | Mitigação |
|-------|---------|-----------|
| `Release/` é ignorado por padrão (`[Rr]elease/` no `.gitignore`) e já sumiu uma vez por sparse checkout | crítico — servidor sobe vazio | issue 01 + skill `wyd-server-ops` §1 |
| Cliente fechado com proteções (`JProtectDll.dll`, `jlock10*`) e launcher que sobrescreve `serverlist.bin` | alto — login impossível de reproduzir | issue 05, skill `wyd-client-connect` |
| Versão de protocolo divergente (7640 vs 12000) | resolvido | issue 06 **fechada em 04/11**: o cliente gráfico envia 12000 (login OK, sem mismatch) |
| Build Docker instável nesta máquina (daemon EOF) | médio — bloqueia 03/04 | issue 02 (diagnóstico de daemon/recursos) |
| Dados de conteúdo sem paridade com o cliente (`ItemList.csv` vs `ItemList.bin`) | médio — item invisível/nome errado | skill `wyd-economy` §8 |
| Migração de schema sem rebuild `--no-cache` | médio — "valor certo no banco, serviço antigo" | skill `wyd-server-dev` §4 |

## Convenções

- Numeração sequencial; não reutilize número de issue fechada.
- Uma issue = um resultado verificável. Se virar duas coisas independentes, divida.
- Bloqueio real (ambiente, credencial, decisão humana) vira nota na issue com **o que foi tentado** e
  **o que falta** — não vira "feito".
- Todo comando citado deve rodar de `` salvo indicação contrária.

## Changelog

- **2026-11-04 (issue 57 fechada: o banco é a única fonte de NPCs)** — os 6.099 blocos de
  `NPCGener.txt` estão em `npc_definition` (o import deixou de filtrar merchant; o shop só é gravado
  para quem vende, porque o `Carry[]` de monstro é a tabela de drop). No boot o tmServer **registra um
  gerador por definição** (`npc generators registered generators=6099 slots=6099 event_owned=199
  training_grounds=104`) e materializa as entidades do mesmo snapshot (`npc config applied at boot
  version=17 npcs=5161`); 739 blocos com `max_num_mob=0` ou posição bloqueada não produzem entidade —
  o mesmo comportamento de antes, agora visível no log em vez de silencioso. Saíram: `spawnNPCs` (o
  parser do arquivo no tmServer), `skipMerchants`/`rawMerchant`, `Generator.DBManaged`,
  `DBManagedGeneratorCount`, a flag `-npc-editing`/`W2PP_NPC_EDITING` e o log `NPCs spawned`. O
  `NPCGener.txt` fica no repositório só como **insumo do importador** do dbserver. Evidência:
  `go build/vet` + testes de `tmserver/cmd/tmserver`, `handler`, `world`, `content` e
  `dbserver/cmd/dbserver` verdes; boot acima; `TestE2ESmokeLogin` aprovado com 6099 geradores no ar.

- **2026-11-04 (legado do seed curado removido)** — a
  duplicação de Armia veio de **duas fontes vivas**, não do arquivo: 20 linhas do seed curado (0006)
  sem `generator_index` sombreando blocos já importados. A migração **0043** apaga essas linhas
  (irreversível de propósito — o bloco importado é o dono), `store.SeedNPCDefinitions` mantém
  `retireShadowedDefinitions` como guarda em re-seed e a regra "migrar = remover o legado" entrou no
  `AGENTS.md` §8. A **issue 57** fechou a migração na sequência: os 6.099 blocos estão no banco, o
  tmServer registra os geradores e materializa as entidades a partir de `npc_definition`, e
  `skipMerchants`, o campo `DBManaged` (com `DBManagedGeneratorCount`) e a flag `-npc-editing`
  saíram. O `NPCGener.txt` continua no repositório apenas como **insumo do importador** do dbserver
  (o runtime não o lê).

- **2026-11-04 (issue 23 fechada + B18: editar 1 NPC duplicava a cidade)** — o `QA Ferreiro` criado
pelo editor apareceu em Armia com os três itens da loja e os três foram comprados no cliente: o
overlay de NPC funciona ponta a ponta. No mesmo teste apareceu o **B18** (mercadores de Armia em
dobro): 20 linhas legadas do seed pré-0018 (`generator_index` NULL) ficaram sombreadas pelos blocos
NPCGener na mesma posição/template e as duas eram materializadas. As sombras foram desativadas, o
seed passou a aposentá-las (`retireShadowedDefinitions`) e o reload virou **diff** (`managedNPC` +
`npcDefKey`), então um edit não remonta mais a cidade inteira —
`TestApplyNPCConfigKeepsUnchangedNPCs`.

- **2026-11-04 (issue 07 fechada — o cliente entrou no mundo)** — `test` logou (`chars=2`), selecionou o
`Cantarelli` 39 (`0x0213` slot 1 → `CNFCharacterLogin` corpo 1.820 B, spawn 2089,2106), **andou**
(`0x036C` com rota de 6 passos), abriu loja em três mercadores (`0x027B` → `MSG_ShopList`) e
**comprou** (`0x0379` → `buy ok item=5134 price=0`), saindo por `EOF`. Headless
`TestE2ESmokeLogin` PASS com versão 12000. Achado menor: `0x03AE` chegou com `routed=false` antes do
logout — provável pacote de quit do cliente, a mapear.

- **2026-11-04 (issue 32: recusa de conta bloqueada + bug das mensagens de login)** — com `test`
bloqueada no banco (`is_blocked=t`), o servidor recusa o login e fecha a conexão
(`TestLoginNoAccountAndBlocked` PASS). No caminho apareceu **o motivo de nenhuma mensagem de erro
aparecer no cliente**: `notify()` enviava `_MSG_MessageBoxOk` com um código numérico local de 4 bytes,
mas o struct legado é `{int;int;char String[96]}` (`Basedef.h:1534-1541`) — o cliente abria uma caixa
vazia. O legado manda **texto** via `SendClientMessage` (`_MSG_MessagePanel`, `HEADER.ID 0`,
`ProcessDBMessage.cpp:518-570`). As cinco recusas de login com texto no `Language.txt` (versão 36, 3
senhas 37, conta 131, bloqueada 132, senha 133) passaram a sair assim, com os testes exigindo
`MsgMessagePanel` + texto exato. **Fechada no cliente** em 19:51 e 19:52: cada tentativa gerou `0x0101` ID 0 com 140 B e a recusa
encerrou a conexão; a conta `test` foi liberada depois do teste.

- **2026-11-04 (issue 07 parcial: login e lista de personagens no cliente)** — a sessão gráfica autenticou
`test` e recebeu a lista com **2 personagens** (`account login: OK … chars=2`, frame `0x010a` de
1.996 B), mas a conexão caiu por `EOF` **7 s depois, sem pacote de seleção**: não houve entrada no
mundo. A issue continua 🟡 com os critérios marcados até onde a evidência alcança — headless e
seleção/entrada seguem pendentes.

- **2026-11-04 (issues 05 e 06 fechadas — o cliente conectou)** — o `WYD.exe` 7662 da pasta
`client/` consultou o status (`GET /serv00.htm` em `:80`) e abriu o CPSock em `:8281`:
`recv packet type=0x020d len=104` → `account login: relaying to dbServer account=test` →
`account login: OK id=1 role=admin chars=2`, **sem** rejeição de versão. Isso encerra a divergência
7640 × 12000: a checagem do `handler/login.go` é igualdade estrita e o servidor estava com
`-client-version 12000`, logo este build envia **12000** (o 7640 é o valor de `Basedef.h`). README,
`CLAUDE.md`, `docs/migration/README.md`, `protocol-spec` §4.1 e as skills `wyd-client-connect` /
`wyd-server-ops` foram corrigidos; `docs/analysis/qa-ingame.md` e `roteiros-qa.md` atualizados.
Ressalva: `client/serverlist.bin` não foi regerado (7.040 B, 02/11) e não há backup.

- **2026-11-04 (dossiê de análise + reordenação do plano)** — varredura do repositório e pesquisa
externa viraram `docs/analysis/` (7 documentos): inventário com números reais (198 arquivos de C++,
2.138 de conteúdo, ~91 mil linhas de Go, 46 tabelas, 49 migrações), mapa C++→Go com o que já foi
substituído pelas telas do portal e o que é limite duro (cliente fechado, offsets, assets), análise de
persistência (quais CSVs podem ir para o banco e por que os mapas binários não devem), cliente
DX9/32-bit com caminhos de modernização e desenho do launcher, fila de QA in-game e plano de
engenharia (ADRs, runbooks, CI com banco efêmero, métricas). Achados que mudam decisão: o
`serverlist.bin` está **versionado com IP privado**; `openspec/` tem 5 mudanças paradas e zero
referência no código; a doc diverge sobre a versão do protocolo (7640 × 12000); teste de store usa
`resetTestSchema` (apaga o schema) e por isso não roda. O plano de novembro (`roadmap/PLANO-2026-11.md`)
prioriza fechar as 11 issues 🟡 (mais 05/06) antes de abrir features novas, e cria as issues 49–56.

- **2026-11-04 (issue 32 — contas, papéis e banimento)** — `/contas` deixou de ser "em breve":
  listagem com cursor keyset (`apos`), filtro por prefixo de login/papel/estado e detalhe com a
  **auditoria** de cada alteração. Papel e bloqueio exigem motivo (≥ 10) e passam por
  `AccountAdminService` (novo, `web.proto` regenerado no container), com `moderator_id` vindo da
  sessão. As regras ficaram no store, **dentro da transação**: ator `admin` re-lido do banco, ninguém
  altera a si mesmo, a última conta admin ativa é protegida (`FOR UPDATE` nos admins ativos antes de
  contar) e `moderator` só bloqueia `player`. Migração `0042_account_admin` acrescenta
  `role_changed_at`/`role_changed_by`/`blocked_reason` e `account_admin_audit` (antes → depois, autor,
  motivo); o e2e (`-tags=e2e`) cria contas `zq*`, prova cada recusa e limpa o banco no fim. O
  `adminCall` passou a repassar o `message` de negócio do web-api ("esta é a última conta admin
  ativa — promova outra antes") em vez do genérico "dados inválidos". Falta apenas a evidência no
  login do cliente do jogo.

- **2026-11-04 (issues 02 · 29 · 40)**

  - **02 — `scripts/preflight.sh`.** Valida, antes do bring-up: raiz do repo (e sparse
    checkout), `Release/` (ItemList/SkillData/NPCGener/1.991 templates/4 BaseMob, mapas como
    warning), daemon Docker + compose v2/v1 + memória mínima, portas 80/8281/7600/3000
    (ocupada pela própria stack = OK) e Go do host contra 1.26, imprimindo o caminho oficial em
    container quando falta. `--quiet` mostra só FAIL/WARN; exit 1 se houver `[FAIL]`.
  - **29 — `/loja` (loja de donate).** Catálogo com item por nome/ícone, **preço em donate**
    (nunca coin), validade, adicionais e estado; diálogo de criação/edição com os três pares
    `eff`/`effv` e `expires_days`; habilitar/desabilitar; **apagar exige digitar o título** (o
    servidor confere) e crédito manual exige motivo (>= 10 chars) e confirmação explícita,
    devolvendo `new_balance`. `lib/donate.ts` valida título 3–60, preço 1..1.000.000, validade
    ≤ 3.650 e o `uint8` dos adicionais (14 testes). O backend só exige `item_index > 0` e
    `price > 0`: a action recusa item fora do catálogo, porque o jogador compraria slot vazio.
  - **40 — `/logs` (streaming).** `server/docker.ts` concentra o Docker CLI: whitelist fixa de
    5 serviços + alvo `all`, `spawn` sem shell, `tail` 1..2000, `since` só no formato `10m`,
    `redact` de ANSI/DSN/segredo e `compose ps` cacheado 5s. `/api/logs/[service]` responde
    `text/event-stream` (sessão obrigatória, `player` 403, serviço fora da whitelist 404) e mata
    o filho no abort; a tela tem abas, chips de status, pausa que não fecha o stream, filtro,
    limpar, baixar `.txt` e buffer de 2.000 linhas com flush a cada 100ms.

- **2026-11-04 (issue 30 — recompensa diária)** — `/diaria` gerencia o catálogo da recompensa:
  tabela com título, **item por nome e ícone**, validade (`permanente`/`N dias`), resumo dos
  adicionais e estado; diálogo de criação/edição com os três pares de efeito (`eff`/`effv`) vindos do
  vocabulário de `ItemEffect.h`, `expires_days` (0 = permanente, até 3.650) e `enabled`; ações de
  linha habilitar/desabilitar, **duplicar** (a cópia nasce desabilitada para não publicar por engano)
  e apagar com confirmação. `lib/daily.ts` valida título 3–60, descrição ≤ 280, efeito/valor no
  `uint8` da `STRUCT_ITEM` (8 testes); a action também recusa item fora do catálogo — o backend só
  exige `item_index > 0` e o jogador receberia slot vazio. O painel de "resgates de hoje" ficou para
  a issue 31: **não existe RPC** para contar claims e o portal não fala Postgres. Verificado ao vivo:
  oferta `id 1` (item 400, 7 dias) criada pela tela, `claim #1 = OK`/`claim #2 = ALREADY_CLAIMED` para
  a conta 1, `daily_reward_claim` com `2026-11-04` e `delivery_queue` `pending` com
  `source=daily_reward:1`; desabilitar e apagar refletiram no banco e na tabela. `tsc`, `vitest`
  (39) e `next build` verdes (`/diaria` 6,59 kB). Falta só ver o item no personagem após login.

- **2026-11-04 (issue 28 — eventos globais)** — `/eventos` controla o evento global de mundo:
  estado do servidor com badge (ativo · esgotado · config incompleta · desligado), item escolhido
  pelo catálogo (nome no lugar de índice), divisor lido como "1 em cada N kills elegíveis", faixa
  `start/current/end` com barra de progresso e "restam N drops", e os toggles de serial, anúncio,
  EXP dobrada e evento newbie. `lib/worldevent.ts` espelha `worldevent.validConfig` (10 testes em
  `worldevent.test.ts`), o que corrigiu a leitura de que zeros com o evento desligado seriam
  inválidos (os inputs agora só exigem mínimo 1 com o evento ligado). A Server Action recusa item
  fora do catálogo — o backend aceita qualquer índice ≤ 32767 e o jogador receberia slot vazio —
  e **relê** o config depois de salvar; EXP dobrada exige confirmação explícita e há
  "desligar tudo" num diálogo único. Verificado ao vivo: salvar `item 400 / divisor 5 / faixa 1..50`
  gravou a linha (`updated_by=1`), a auditoria `set_config` e **version 1**, e o tmserver logou
  `world event config reloaded version=1` ~7 s depois — sem restart, pelo poll de 15 s. `/eventos`
  200 com sessão e 307 sem; `tsc`, `vitest` (31) e `next build` verdes. Falta só a evidência in-game
  (EXP dobrada e notice no cliente), que depende do cliente do usuário.

- **2026-11-04 (reestruturação — portal e roadmap dentro do repo)** — o portal passou a viver em
  `portal/` e o planejamento em `roadmap/`, com o cliente do jogo em `client/` (ignorado no git). O portal entrou na infraestrutura: imagem `portal/Dockerfile` (Next standalone,
  contexto da raiz para levar `api/web/v1/web.proto`), serviço `portal` no `docker-compose.yaml`
  (`:3000`, `WEB_API_ADDR=webserver:7600`), alvos `make portal-*`, bootstrap do `run-local.sh` e job
  de CI (typecheck, testes, build e imagem). Os arquivos de IA da raiz foram consolidados em
  `AGENTS.md`/`CLAUDE.md` + `.agents/skills`.

- **2026-11-04 (issue 27 — mapa de colisão)** — `/mapa` lê o `AttributeMap.dat` do content tree e
  mostra sha256, histograma de 256 valores e a tabela de significados (valor, bit e o que faz). O
  formulário de transformação cobre as cinco operações do backend (incluindo a regra legada
  `value < 64 && value != 1 → |= 64`), retângulo **em coordenadas de mundo** com a célula resultante
  (`x/4`) ao lado e filtro por valor exato ou máscara. O download só habilita para os mesmos
  parâmetros já pré-visualizados; a rota `/api/mapa/download` refaz a transformação, confere o
  tamanho de 1.048.576 bytes e devolve `AttributeMap_New.dat` com o sha256 no header — o arquivo do
  servidor não é reescrito. Verificado ao vivo: página 200, download 200 com **7.148 células**
  alteradas e sha do arquivo igual ao header; `npm test` 21/21, `tsc` e `next build` 0. O clique do
  “pré-visualizar” não foi exercitado no navegador (login do painel no browser de teste não submeteu
  o formulário) — a mesma RPC foi exercitada pelo download.

- **2026-11-04 (issue 34, fatia 3 — `Carry[]` por template)** — a tabela de drop de cada mob virou
  editável: `/drops/carry` escolhe o template pela URL e edita as posições agrupadas pela raridade
  que **a posição** vale (“Sempre cai”, “Ultra raro”…). Salvar substitui a tabela inteira, então o
  vazio é uma decisão explícita; restaurar volta ao arquivo. Do browser vem só o índice do item —
  o web-api preserva os efeitos de quem não mudou e zera os de quem mudou, e recusa item fora do
  catálogo ou template inexistente. Migração 0041 (`mob_carry_override`/`mob_carry_audit`),
  `ListMobCarry/SetMobCarry/ResetMobCarry` e `-mob-carry-editing` no tmserver (OFF), com evidência
  de boot `templates=1` e das rejeições `INVALID`/`FORBIDDEN`.

- **2026-11-03 (issue 34, fatia 2 — `/drops/editar`)** — a taxa por slot ganhou editor no portal:
  simulador embutido (nível do mob e bônus do matador recalculam a chance de todas as linhas ao vivo),
  divisor editável com motivo auditado, restaurar por posição e badge de estado. **Nada de ID cru
  como rótulo**: a linha é "Equipamento comum", "Ouro e poção", "Ultra raro", "Sempre cai"… e o
  intervalo do `Carry[]` é posição secundária; os slots 8–11 aparecem como "fixo no código" e ficam
  desabilitados, porque `EffectiveDropRate` reescreve o resultado deles. `lib/droprate.ts` espelha a
  fórmula do Go com 5 casos testados; a tabela base entra por parâmetro (server e client são
  instâncias diferentes do módulo). `/drops` e `/drops/editar` verificados com sessão (200) e
  `next build` 0.

- **2026-11-03 (issue 34, fatia 1 — taxa de drop por slot)** — `DropRate[64]` (que era `init()` de
  `loot`) virou a tabela compartilhada `internal/droprate`, porque o `webserver` não pode importar
  `tmserver/internal/loot` (regra de `internal` do Go) e o editor precisa mostrar os mesmos valores
  base; `loot.DropRate` continua sendo a cópia mutável que o overlay reescreve. Migração
  **0040_drop_config** (`drop_rate_override` + `drop_rate_audit`), store com auditoria,
  `DropConfigService` (dbserver → tmServer) e `DropAdminService` (`ListDropRates`/`UpsertDropRate`/
  `ResetDropRate`) com autorização por papel. tmServer aplica no boot atrás de
  `-drop-config-editing` (OFF = tabela do código, byte a byte) e loga `drop rate overlay applied`.
  Verificado ao vivo: 64 slots, `slot 12 base=900 → 450 overridden`, slot 64 e valor 0 INVALID,
  `player` FORBIDDEN, reset OK e NOT_FOUND na repetição; boot one-off aplicou `applied=1 rows=1
  slot_12=450`. Faltam portal, `Carry[]` e evidência in-game.

- **2026-11-03 (issue 25, editor de `Equip[]`)** — grade dos 16 slots com picker de item do catálogo,
  diff por slot e `SetMobTemplateEquip`; trocar item zera os efeitos do slot (efeitos são da issue 44)
  e o botão só habilita com override de stats salvo (`NOT_FOUND` do web-api vira "salve os stats
  antes" na UI). `saveMobStat` parou de aceitar equip do browser e relê o equip do servidor, senão
  salvar stats apagaria uma edição de equip feita logo antes. Verificado ao vivo: slot 5/item 412
  persistido e lido, slot duplicado e slot 16 recusados, `player` FORBIDDEN, `/mobs/@@Gargula` 200.

- **2026-11-03 (issue 25, flag de override)** — `MobTemplateFile.overridden` no `web.proto`;
  `ListTemplates` resolve o flag numa consulta só (`ListMobTemplateStats`) e devolve uma **cópia** do
  snapshot escaneado no boot, então listar nunca deixa flag velha. No portal, `/mobs` ganhou coluna
  "Override" (badge "no banco"), contador no cabeçalho e o filtro "somente com override". Verificado
  ao vivo: com a base recém-migrada a listagem vem `[]`, o upsert de `@@Gargula` faz
  `overrides=[@@Gargula]`/`overridden=true` e o delete volta a `[]`/`false`; `/mobs` responde 200.
  Teste novo `TestListTemplatesMarksOverrides` cobre o flag e o caso de delete.

- **2026-11-03 (issue 25, fatia principal)** — Editor de stats de mob no portal: `/mobs` lista os
  **1.991 templates** do content tree (656 com merchant) e `/mobs/[template]` abre com **read-through**
  do arquivo (`overridden=false`, ex.: `@@Gargula` nível 399, HP 14000, EXP 2.990.849, 2 slots de
  equip), seis seções de campos com os mesmos intervalos da validação do web-api, diff da sessão,
  contador de campos alterados, salvar (`UpsertMobTemplateStat`, devolvendo o `Equip[]` carregado
  porque o upsert substitui o override inteiro) e "reverter ao arquivo". `player` recebe FORBIDDEN e
  template inexistente devolve NOT_FOUND. `tsc`, `npm test` (8/8), `next build` e `/mobs` (HTTP 200)
  verificados. Pendente: editor de `Equip[]`, badge/filtro de override (pede campo novo no proto) e
  evidência in-game.

- **2026-11-03 (Docker de volta — evidências ao vivo)** — O Docker Desktop voltou a subir; imagens de
  `dbserver`/`tmserver`/`webserver` reconstruídas e a stack inteira no ar (`db`, `binserver`,
  `dbserver`, `tmserver`, `webserver`). Migração `0039_server_instance_flags` aplicada e o tmserver
  gravou **16 flags** no boot (`captured_at=2026-11-03T18:11:47Z`), que `/config` mostra; `/drops`
  responde 200 com as duas abas e `ListMobDrops(mob_query="a")` devolve 1.349 templates / 13.813
  slots com consistência item→mob confirmada e `player` recebendo FORBIDDEN. O bloqueio de `C:`
  continua latente (6,7 GB livres; builds só com `GOTMPDIR`/`TMPDIR` no `D:`).

- **2026-11-03 (issue 47, fechamento parcial)** — Nome de lugar nos teleportes: `lib/zone.ts`
  resolve coordenada → cidade/região escolhendo a **menor** caixa do atlas, e `/teleportes` mostra
  "Armia → Montanha Gelada" com a coordenada no `title` (sem atlas, volta às coordenadas). Escala de
  espaço `--space-1..7` entrou no `:root` e nas superfícies compartilhadas + telas novas. Dois bugs de
  acessibilidade corrigidos: `all: unset` em `.ui-tab` e nos cabeçalhos ordenáveis apagava o anel de
  `:focus-visible` (agora redeclarado), e `.ui-chip`/`.ui-select` estavam abaixo do alvo mínimo de
  32 px. `tsc`, `npm test` (8/8) e `next build` verdes.
  Fechamento: contraste medido par a par (pior caso `--danger` no surface2 = 4,94:1, acima de 4,5) e
  `--border-control: #707b8d` (4,42:1 no bg, 3,77:1 no surface2) nas bordas de input/select/chip/botão,
  substituindo os divisores decorativos `--border`/`--border-strong` (1,45:1 / 1,87:1) que não
  satisfazem 1.4.11 para controle. `RetryButton` (`router.refresh()` em transição) levou
  "tentar de novo" a `/castelo`, `/guildas`, `/personagens`, `/npcs`, `/teleportes`, `/config` e
  `/drops`.

- **2026-11-03 (issue 26)** — Relatório de drops no portal: `/drops` com as duas visões do mesmo
  `Carry[]` (**Por item** e **Por mob**), busca server-side com debounce e mínimo de 2 caracteres
  (sem filtro o relatório devolve o catálogo inteiro de pares), toggle `include_zero_drop_items`,
  raridade derivada do **divisor** de `g_pDropRate` (≤1 sempre · ≤100 muito comum · ≤1000 comum ·
  ≤3000 incomum · resto raro) e export CSV das duas visões. O BFF cacheia por filtro (o catálogo é
  varrido uma vez no boot do web-api), audita como `drop.byItem`/`drop.byMob` e mantém a guarda de
  sessão do `adminCall`. `tsc --noEmit` e `next build` 0 com `/drops` em 4,1 kB. Pendente: evidência ao
  vivo e a conferência manual de 2 templates contra o binário — Docker Desktop não sobe nesta máquina.

- **2026-11-03 (issue 36, aba de flags)** — O tmserver passa a gravar no boot o snapshot das suas 16
  flags de processo (`server_instance_flags`, migration 0039, via `InstanceFlagsService`), com
  `restart_required` e o efeito de cada uma; `/config` mostra a tabela acima das taxas e avisa que
  nenhuma flag é recarregada em runtime. Falha ao gravar é warning, não impede o boot. Verificado por
  `go build`, `go test` (snapshot ausente → `available=false`; JSON → flag com `restart_required` e
  `captured_at`) e `tsc`; a validação ao vivo ficou bloqueada pelo Docker.

- **2026-11-03 (issue 43, parcial 2)** — Concessão de item por `delivery_queue` com auditoria
  (`character_admin_audit`, migration 0035): `GrantItem`/`ListCharacterGrants` no
  `CharacterAdminService`, `EnqueueItemGrant` (uma linha por unidade) e diálogo de concessão no
  detalhe do personagem. Verificado executando o store contra o Postgres real (2 linhas + auditoria,
  removidas depois). Pendente: gold/cash e edição de campo.
- **2026-11-03 (issue 43, parcial)** — Inspeção de personagem só-leitura: `CharacterAdminService`
  (`SearchCharacters` por nome/conta; `GetCharacter` com status, skills e inventário com os 3 pares de
  efeito), store `SearchCharacters`/`LoadCharacterByID`, e `/personagens` + `/personagens/[id]` com busca
  e tabelas de equipamento/cargo. A escrita (concessão, `SetField`) fica bloqueada por dois achados
  registrados na issue: `delivery_queue` só drena item (sem gold/cash) e não existe presença online.

- **2026-11-03 (issue 42)** — Rotas de teleporte editáveis: tabela `teleport_route` (migration 0034,
  origem = tile múltiplo de 4, custo, requisito, ativa, nota) como **overlay** da tabela compilada, que
  passou a morar em `internal/teleportdata` para o web-api listar/simular sem cópia. `TeleportAdminService`
  (ListRoutes mesclado, UpsertRoute com normalização + warning, SetRouteEnabled, DeleteRoute,
  SimulateRoute) e `TeleportConfigService` no dbserver para o tmserver, que publica o snapshot num
  `atomic.Pointer` e o re-lê a cada 15 s com `W2PP_TELEPORT_EDITING=true` (ordem do RNG e gate
  fada/Kefra preservados, com teste). `/teleportes` lista as 23 rotas, edita destino/custo/requisito,
  desativa, volta ao padrão e duplica inverso; `import-teleports` semeia as rotas compiladas.

- **2026-11-03 (issue 41)** — Atlas do mundo: `tmserver/cmd/worldatlas` gera
  `dist/world-atlas.json` (5 cidades com spawn/limite/arena, 52 regiões do `Regions.txt`, 23 rotas de
  teleporte, 49 itens fixos do `InitItem.csv`, 6.099 geradores de `NPCGener.txt`, hash das fontes) e
  `dist/world-map.png` (AttributeMap 512²), lendo as **mesmas** constantes do servidor via
  `world.AtlasCities()`/`world.AtlasTeleports()`. `/atlas` mostra o mapa com camadas ligáveis,
  tabelas por camada, "copiar coordenada", destaque de região e export CSV/JSON por camada (401 sem
  sessão). Sem os artefatos a tela ensina o comando do gerador.

- **2026-11-03 (análise de editabilidade + issues 41–46)** — Levantamento completo do que é editável no
  servidor em `roadmap/editabilidade.md` (conteúdo de `Release/`, tabelas de overlay, estado vivo e
  constantes em Go) e o catálogo de áreas do mundo em `roadmap/atlas-mundo.md` (5 cidades com
  spawn/limite/arena,  52 regiões nomeadas, 23 rotas de teleporte, 49 itens fixos). Abertas seis issues
  para as lacunas: **41** atlas do mundo com mapa e camadas (gera `world-atlas.json` a partir de código
  e conteúdo), **42** rotas de teleporte editáveis (overlay + `TeleportAdminService`, preservando o
  arredondamento `x &^ 3` e a ordem do RNG), **43** inventário/status de personagem (edição só offline,
  concessão por `delivery_queue`), **44** adds e efeitos de item + refino/combine, **45** guildas,
  castelo e Kefra, **46** catálogo de skills. O menu do portal já lista as seis telas como "em breve".

- **2026-11-03 (issue 24)** — Preços globais no portal: `/precos` lista os overrides de
  `item_price` (`ListItemPrices`) com nome, ícone, **preço base do `ItemList.csv`**, preço atual e Δ;
  "definir preço" usa Combobox do catálogo (o hint do picker já mostra a base) e "remover override"
  manda `-1` com confirmação inline. Preço é global, explicitado na tela e no diálogo. Base lida do
  CSV no host (6ª coluna, mesma de `content.ItemList.Prices()`) via `unstable_cache`. Verificado:
  `SetItemPrice(1234,1500000)` → linha no banco, `-1` → 0 linhas; player recebe 403. Pendências: o
  backend aceita `item_index` inexistente (não há 422), contagem de chamadas ao catálogo não medida e
  evidência in-game não feita.

- **2026-11-03 (issue 23)** — Editor de NPC/mercador: `/npcs` lista as 568 definições do overlay
  (busca por slug/nome/template, filtro "somente ocultos", badges de origem e status), `/npcs/novo` e
  `/npcs/[id]` com form completo (slug travado na edição, template por Combobox de 228 templates,
  mapa por zona, posição, rota, tipo de mercador, visibilidade) e a aba **Loja** com 27 slots em 3
  abas de 9, diff antes de salvar e envio do array completo. Ações de linha: ocultar/mostrar,
  duplicar (copia a loja) e apagar com confirmação por slug — `CONTENT_OWNED` vira 409 tratado com
  "ocultar em vez de apagar", não erro interno. `adminCall` passou a **auditar a tentativa negada**
  (`guardStaff`), corrigindo o 500 que uma conta `player` recebia. Falta a evidência in-game: exige
  `W2PP_NPC_EDITING=true` e restart do tmserver.

- **2026-11-03 (issue 22)** — Catálogo de itens no portal: `/itens` lista os 3.220 itens do
  `ItemList.csv` com ícone (ou fallback estável), busca com debounce de 150 ms, chips de slot
  multi-seleção, filtro de grau, paginação 100/200/500 e `/itens/[index]` com o detalhe completo
  (inclusive `slot_mask`/`mesh`/`texture`/`icon_key` crus). Catálogo servido por
  `unstable_cache` na tag `item-catalog`; estados distintos para "web-api fora" e "serviço no ar
  sem `-content`". Pack de ícones ainda não gerado: a tela roda em fallback.

- **2026-11-03 (issue 21)** — Camada de admin do BFF: `adminCall` passa a ser o caminho único para
  RPC admin — exige sessão de staff, injeta `moderator_id` **da sessão** (nunca do body), aplica
  deadline de 10s e mapeia `AdminResult` → HTTP (1→200, 2→403, 3→422, 4→404, **5→409**,
  0/desconhecido→500) com mensagem acionável; falha de transporte → 502 "serviço de administração
  indisponível". Toda ação registra auditoria (`accountId, role, action, target, result, status`) no
  log do processo e, com `AUDIT_LOG_DIR`, em `audit-YYYY-MM-DD.ndjson`. 8 testes de vitest cobrem
  o mapa; `npm test` entrou no portal.

- **2026-11-04 (painel + logs no container)** — `/painel` deixou de exibir cartões "em breve":
  lê itens do catálogo, templates de mob, NPCs, ofertas de donate e recompensas diárias, e (para
  `admin`) receita/compradores dos últimos 30 dias pelo `DonateRevenueAdminService`. `/logs` passou
  a funcionar no formato de produção: novo sidecar **`logpipe`** (Go stdlib) monta o socket do
  Docker **read-only** e serve `/status` + `/logs/<serviço>` na rede do compose, com `portal`
  incluído; o portal só autentica e repassa por SSE. O socket nunca é montado no portal.
- **2026-11-04 (issue 31)** — Faturamento: `/faturamento` (só `admin`) sobre o
  `DonateRevenueAdminService` — janela 7/30/90/personalizada interpretada em `America/Sao_Paulo`,
  receita em R$ separada dos créditos de donate, `pending_cents` em cartão próprio com o aviso de
  que não é receita, série em barras CSS por dia/semana/mês, pedidos com CSV client-side, top
  compradores com LTV e extrato da carteira. A busca de conta (`SearchAccounts`) virou `Combobox`
  reutilizável e substituiu o "id numérico" do crédito manual da issue 29. Conferido ao vivo no
  container: 6 KPIs batem com SQL, `moderator` cai em `/sem-permissao`.
- **2026-11-03 (issue 40 criada)** — Monitoramento de logs na UI: `/logs` com streaming SSE por
  serviço (`tmserver`, `dbserver`, `binserver`, `webserver`, `db`) via `docker compose logs -f`
  a partir do BFF, whitelist fixa de serviços, redação de DSN/segredo, cap de 2000 linhas, pausa/
  filtro/download e chips de `docker compose ps`. Depende só do shell (20); já aparece no menu como
  "em breve".
- **2026-11-03 (issue 20)** — Shell do portal: sidebar gerada por `components/nav.ts` (11 seções,
  não implementadas com badge "em breve"), header com conta/papel/Sair, e a biblioteca base —
  `DataTable` (busca, ordenação, 50/100/200), `Form`/`Field`, `Dialog` nativo, `Combobox` com
  debounce 200ms, `Toast` (`role="status"`), `Badge`, `Tabs`, `Stat`. `/painel` mostra 3.220 itens
  do catálogo real; `/api/search/items` exige sessão (401 sem cookie).
- **2026-11-03 (issue 19)** — Login web fechado: `VerifyCredentials` → cookie `wyd_session`
  HS256 (8h, `HttpOnly`, `SameSite=Lax`), guarda em `(admin)/layout.tsx` + `middleware.ts` nas 12
  rotas. `test`/`test123` → 200 + cookie e `/npcs` 200; `teste` (player) → `/sem-permissao`;
  senha errada 401, corpo vazio 422, cookie forjado 307 para `/login`, logout 204 e sessão morta.
- **2026-11-03 (issue 18)** — `portal/` criado: Next.js 15.5.28 + React 19, cliente gRPC via
  `@grpc/proto-loader` sobre `api/web/v1/web.proto`, `/api/health` devolvendo
  `{"ok":true,"items":3220,"catalogVersion":"acce2f7cf63fb856"}` (mesmo `items=3220` do boot do
  tmserver). `tsc --noEmit` e `next build` exit 0. Stack: `webserver` Up em `:7600`.
- **2026-11-03** — Fase 6 aberta: 22 issues (18–39) e a skill `wyd-web-admin-ui` definem o
  portal web de administração (Next.js como BFF do gRPC `webserver`), cobrindo telas sobre contratos
  existentes (NPC, mob, item, drop, mapa, eventos, loja, faturamento) e os contratos que faltam
  (contas/papéis, catálogo editável, drops, EXP, rates, spawns, live ops), até o QA final.
  Login do cliente validado nesta sessão (`test`/`Cantarelli`, `role=admin`).
- **2026-11-02 (sessão 2)** — Stack no ar: build serializado resolveu a queda do daemon; `db`/`binserver`/
  `dbserver`/`webserver`/`tmserver` Up; migrações até `0033_save_celestial`; contas `test` e `test2`
  semeadas (argon2id, `role=player`, 0 personagens). Boot: `items=3220 skills=151`, `grid dim=4096`,
  `classes=4`, `layouts.loaded=620` (0 rejeitados), `mobs=11930`, `merchant_blocks_skipped=0`,
  `world items=49`. Status `:80` → 200 (30 bytes) e TCP `:8281` aberto. **Login headless OK**
  (`CNFAccountLogin`, `body=1996`). Pendente: cliente gráfico (05/06/07) e criação de personagem (08).
- **2026-11-02** — Roadmap criado. Issue 01 concluída (restauração do working tree e validação da
  árvore de conteúdo). Issue 14 v1 concluída (AGENTS.md + 6 skills). Build Docker tentado: o daemon
  caiu durante o `COPY . .` e passou a responder `Docker Desktop is unable to start`; Docker Desktop
  foi reiniciado e continuou indisponível — 02/03/04 abertas e bloqueadas no host, não no código.
