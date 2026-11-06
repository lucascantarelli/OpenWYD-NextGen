# Estado atual do projeto — inventário e respostas

Data: **2026-11-04**. Base: árvore de trabalho em `D:\dolorestec\wyd2\OpenWYD` (commit `114b4f3`),
stack local no ar (containers `openwyd-*`), Postgres com 46 tabelas e migração mais recente
`0042_account_admin`.

## 1. O que é o projeto (confirmando o apontamento 1)

**Sim**: é um servidor WYD reescrito em Go a partir de um legado C++ (`Source/`), mantendo o cliente
Windows fechado como borda. Ordem de grandeza do repositório:

| Pasta | Arquivos | Papel |
|-------|---------:|-------|
| `tmserver/` | 352 | servidor do jogo (CPSock :8281, status :80), dono do estado do mundo em uma goroutine |
| `internal/` | 182 | store pgx, migrações, domínio, argon2id, RNG, catálogos |
| `portal/` | 14.511 (inclui `node_modules`) | Next.js 15 como BFF puro sobre o web-api |
| `webserver/` | 91 | web-api gRPC (:7600) — contas, catálogos, admin |
| `dbserver/` | 23 | persistência gRPC (:7514) |
| `binserver/` | 5 | billing gRPC (:3000), allow-all por padrão |
| `api/` | 9 | contratos `db`, `bin`, `web` (+ gerados) |
| `logpipe/` | 2 | sidecar de logs (socket Docker read-only) |
| `tmserver/content/` | 2.138 | conteúdo do jogo (1.991 templates de NPC, CSVs, mapas binários) |
| `Source/` | 198 | C++ legado: 99 `.cpp`, 44 `.h`, 12 projetos |
| `docs/` | 104 | engenharia reversa, integrações, relatórios antigos |
| `roadmap/` | 53 | 48 issues + índice |
| `.agents/` | 41 | skills de operação/desenvolvimento |
| `openspec/` | 69 | 5 propostas + 3 specs (ver §7) |
| `project-analizer/` | 5 | plugin Claude Code que gerou `docs/agents/` (ver §8) |

O estado funcional está em `roadmap/index.md`: **31 de 48 issues fechadas**, 11 delas com ressalva
de evidência in-game.

## 2. Respostas diretas (apontamentos 1–10)

| # | Pergunta | Resposta curta | Onde aprofundar |
|---|----------|----------------|-----------------|
| 1 | É melhoria de legado C++ portado para stack moderna? | Sim. Servidor em Go 1.26 + Postgres; C++ ficou como referência de comportamento, não como runtime. | §1, [migracao-legado.md](migracao-legado.md) |
| 2 | Ainda há dependências do legado? Dá para portar tudo? | **Nenhuma dependência de runtime**. O que resta são *referências* (`Source/`) e 3 pontos que exigem o cliente: protocolo CPSock, layouts de save binários e hooks da `ClientPatch`. Portar tudo é viável; **substituir o cliente** é o único limite duro. | [migracao-legado.md](migracao-legado.md) §2–§4 |
| 3 | Dá para migrar toda persistência para banco e apagar os textos? | **Quase tudo**: catálogos, NPCs, drops, rates e configs já têm tabela/overlay. **Não** devem ir para o banco: mapas binários (`HeightMap.dat`, `AttributeMap.dat`, 17 MiB), `itemicon.bin` (índice de atlas) e os assets do cliente. | [persistencia-conteudo.md](persistencia-conteudo.md) |
| 4 | A documentação basta para entender o projeto? | Para **engenharia reversa**, sim (58/58 handlers, protocolo, formatos, regras). Para **produto/operação**, não: falta visão de conjunto, catálogo de sistemas de jogo, ADRs, guia de contribuição e um índice único. Este dossiê + a issue 54 fecham a lacuna. | §9 |
| 5 | Para que serve `openspec/`? | É o fluxo **spec-driven** (proposta → design → tasks → apply/archive) de 5 mudanças antigas (`issue-309/316/321/327/344`) e 3 specs (`celestial-evolution`, `hunting-scrolls`, `arch-crystal-quest`). Foi usado antes do roadmap atual e **não é referenciado por nenhum código**; só as skills `openspec-*` o conhecem. Recomendação: manter como arquivo histórico e decidir entre adotá-lo (mudanças grandes) ou arquivá-lo. | [qualidade-e-processo.md](qualidade-e-processo.md) §6 |
| 6 | Para que serve `project-analizer/`? | Plugin de Claude Code (3 agentes: arquitetura, componente, dependências) cujo produto está em `docs/agents/` (relatórios de 2026-06-19). Não roda em CI nem é usado pelo servidor. | [qualidade-e-processo.md](qualidade-e-processo.md) §6 |
| 7 | `tmserver/content/` é lido pelo servidor? Dá para trocar por banco? | Sim, lido no boot. `ItemList.csv` aparece em 18 pontos do Go, `NPCGener.txt` em 13, e ainda `SkillData.csv`, `InitItem.csv`, `Regions.txt`, `Guard.txt`, `gameconfig.txt`, `CompRate/SancRate/QuestsRate`, `CastleQuest.txt`, `HeightMap/AttributeMap.dat`, `itemicon.bin`. Migração para o banco é possível para os **textuais**; ver §3.5 de [persistencia-conteudo.md](persistencia-conteudo.md). | [persistencia-conteudo.md](persistencia-conteudo.md) |
| 8 | `scripts/` tem algo que não cabe no portal? E o Makefile? | `scripts/` tem 5 itens: `preflight.sh`, `run-local.sh`, `gen-certs.sh`, `e2e-worldevents.sh`, `upload-item-icons-storage-manager.sh`. **Nada ali é exclusivo de portal**: preflight/run-local/gen-certs são operação de máquina (faz sentido manter CLI), e o upload de ícones já tem fluxo web. Makefile é atalho de desenvolvimento — não deve virar botão do portal. | [qualidade-e-processo.md](qualidade-e-processo.md) §5 |
| 9 | `serverlist bin/` serve para quê? Dá para migrar? | Serve para **gerar o `serverlist.bin`** que o cliente usa para descobrir IP/porta (`serverlist editor.exe` + 4 pares de `.bin` gerados com IPs privados). É 100% migrável para o portal (gerar/baixar o bin por conta), e a pasta pode sair do git — hoje ela é **versionada por engano** (viola `AGENTS.md`). | [cliente-launcher.md](cliente-launcher.md) §5 |
| 10 | `Source/` é o legado? Dá para se livrar? O que falta migrar? | Sim. Já portado: `TMSrv` (87 arquivos), `DBSrv` (12), `BISrv` (7, reescrito do zero), `CPSock`/`Basedef`/`CItem`/`CMob`/`CUser`/`CCastleZakum`/`CWarTower` e os `_MSG_*`. Falta portar: ferramentas Windows (`DropTool`, `ExpTool`, `NPTool`, `EDITAPPMOB`, `EDITAPPSHOP`, `SearchPass`, `ZerarSkill`, `AttributeMap_Editor`) — todas já substituídas por telas do portal — e o `ClientPatch_v7662`, que é do cliente. | [migracao-legado.md](migracao-legado.md) §1 |

## 3. As três perguntas adicionais

### 3.1 Dá para remover todo o legado? Quais limites?

Sim, com **três limites que não são código**:

1. **O cliente é fechado.** Enquanto o alvo for `WYD.exe` 7.64/7662, o `tmserver` tem de falar CPSock,
   respeitar `STRUCT_*` por offset e mandar os pacotes S→C que o cliente desenha. Isso é
   responsabilidade do servidor Go, não do C++.
2. **Os dados de conteúdo são binários com layout de struct C.** O banco pode guardar os campos; o
   *codec* continua existindo no Go (já existe: `tmserver/internal/protocol`).
3. **Ferramentas de authoring** só existem em C++/Windows; o substituto é o portal — o que já aconteceu
   para NPCs, drops, itens, preços, teleportes, rates e eventos.

Depois de migrar as ferramentas (issue 56) e o conteúdo textual (issue 50), `Source/` pode virar
**arquivo histórico read-only** (ou sair do checkout padrão por sparse-checkout).

### 3.2 Relatório profissional e estratégia

Está em [qualidade-e-processo.md](qualidade-e-processo.md): arquitetura (o que está sólido e o que é
dívida), pirâmide de testes, CI/CD, observabilidade, segurança, documentação e governança de IA,
com marcos de 30/60/90 dias.

### 3.3 Boas práticas, patterns e metodologia ágil

Também em [qualidade-e-processo.md](qualidade-e-processo.md) §2–§4: o que já existe (`internal/store`,
`domain`, `secret`, gRPC tipado, `adminCall`, overlays auditados), o que refatorar (wiring de
`cmd/*/main.go`, catálogos lidos no boot, duplicação de validação Go/TS) e como introduzir
TDD/BDD/CI sem parar a entrega.

## 4. O que está sólido (não mexer sem motivo)

- **Dono único do estado do mundo** (uma goroutine, sem locks) — é o que impede duplicação de item.
- **Web nunca escreve estado vivo**: tudo passa por `delivery_queue` ou overlay com auditoria.
- **Segredos** em argon2id; portal nunca vê hash nem DSN.
- **Protocolo e formatos documentados com evidência** (`docs/migration/`, 58 contratos de handler).
- **Overlays auditados** para conteúdo (NPC, drop, carry, rate, item effect, teleporte, evento, conta).
- **Portal como BFF**: `adminCall` centraliza sessão, `moderator_id`, deadline e auditoria.

## 5. Dívidas técnicas priorizadas

| # | Dívida | Impacto | Ação |
|---|--------|---------|------|
| D1 | Conteúdo lido de CSV no boot, com overlay em paralelo | dois caminhos de verdade (arquivo e tabela), deploy exige volume `tmserver/content/` | issue 50 |
| D2 | `serverlist.bin` com IP privado **versionado** | viola `AGENTS.md`, vaza topologia | issue 56 (primeiro passo) |
| D3 | Documentação sem índice de produto e sem ADRs | onboarding lento, decisões se perdem | issues 54 + este dossiê |
| D4 | Testes de integração exigem banco real e apagam o schema (`resetTestSchema`) | ninguém roda; regressão de store passa batido | issue 55 |
| D5 | `cmd/*/main.go` faz wiring + regras de import/catálogo | difícil testar | issue 55 |
| D6 | Validação duplicada Go/TS (conta, motivo, item) | mensagens divergem | issue 55 |
| D7 | Sem métricas/tracing (só logs) | incidente sem diagnóstico | issue 55 |
| D8 | `openspec/` e `project-analizer/` sem uso claro | ruído e expectativa falsa | issue 56 |
| D9 | ~~Versão do cliente inconsistente na doc (7640 vs 12000)~~ | — | **resolvido em 04/11:** cliente envia 12000 (issue 06); doc corrigida |

## 6. Lacunas de documentação (apontamento 4)

Existe muito material **de baixo nível** e pouco **de conjunto**. Faltam:

1. **Visão de produto**: o que é o servidor hoje, o que o jogador consegue fazer, quais sistemas
   existem (documentado só em `docs/game.md`, parcial).
2. **Catálogo de sistemas de jogo** com regra de negócio, estado (implementado/parcial/ausente) e
   arquivo de referência — evolução, guerra, guild, quest, drop, refino, evento.
3. **ADRs**: por que Go, por que overlay em vez de reescrever CSV, por que `logpipe`, por que
   `delivery_queue`. As decisões estão espalhadas em comentários e no changelog.
4. **Guia de contribuição** (branch, commit, testes, revisão) — hoje só `AGENTS.md`.
5. **Runbooks**: incidente, restore de banco, rotação de segredo, rollback de migração.
6. **Índice único de documentação** — `docs/` tem 104 arquivos sem porta de entrada; `docs/agents/`
   tem relatórios de junho/2026 que ninguém sabe se ainda valem.

Este dossiê é o começo do item 1–2; a issue 54 fecha 3–6.

## 7. `openspec/` em detalhe

- `openspec/config.yaml`: schema `spec-driven`, **todo o resto é o template comentado** (nenhuma regra
  de projeto configurada).
- 5 mudanças: `issue-309`, `issue-316`, `issue-321`, `issue-327` (com README), `issue-344` — cada uma
  com `proposal.md`, `design.md`, `tasks.md` e `.openspec.yaml`.
- 3 specs: `celestial-evolution`, `hunting-scrolls`, `arch-crystal-quest` — exatamente os sistemas que
  aparecem no roadmap como 44/45 (evolução celestial, caça, quest do cristal).
- **Nenhum arquivo de código referencia `openspec/`.** Só as skills `openspec-*` sabem operá-lo.
- Conclusão: é um **experimento de processo** interrompido. Ou se adota para mudanças grandes
  (proposta com design e tasks antes do código) ou se arquiva em `docs/archive/`. Decidir na issue 56.

## 8. `project-analizer/` em detalhe

- Plugin de Claude Code (`agents/`, `commands/`, `.claude-plugin/plugin.json`, `USAGE.md`), instalado
  via marketplace externo (`devfullcycle`).
- Produto: `docs/agents/` — MANIFEST + relatórios de arquitetura, dependências e 9 componentes do
  **C++ legado**, datados de 2026-06-19. Serviram para o plano de migração.
- Não é dependência do build. Valor hoje: histórico. Se o time quiser repetir a análise, o certo é
  rodá-la contra o código Go (issue 55/56), não contra o legado.
