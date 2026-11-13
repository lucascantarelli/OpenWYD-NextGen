# Roadmap Rust — índice executável

> **Estado:** 2026-11-13 · **35 issues ativas** + 15 descritores de fase. 58 issues concluídas estão em [`changelog.md`](changelog.md).
>
> **Auditoria de DevOps/CI (2026-11-12):** dez issues novas (`DEVOPS-01`…`DEVOPS-10`) abertas pela
> análise de pipeline, imagens, compose, testes e monitoramento; contrato em
> [`docs/rust/10-devops-ci.md`](../docs/rust/10-devops-ci.md). Já fechadas: `DEVOPS-01`
> (segredos fora do contexto de build, quatro imagens com base fixada) e `DEVOPS-02`
> (Alertmanager com entrega de notificação provada). Ganho rápido sem bloqueio externo na fila:
> `DEVOPS-04` (paridade gate local × CI). **Template de CI completo em 2026-11-13** — cinco
> workflows, duas actions compostas, `make check` em paridade com o job `gate` — e o runner
> `self-hosted,wyd-local` registrado e o deploy rodando nele: `DEVOPS-06` **fechou** em 2026-11-13
> com o run real das quatro imagens (build, labels, smoke, Trivy e SBOM); as sete restantes estão 🚧,
> com o que cada job já cobra e o que falta em `10-devops-ci.md` §1.1.
>
> **IDs canônicos preservados.** A numeração de fila `#01..#27` é nova e estável; os IDs
> `CATEGORIA-NN` continuam os mesmos para não reutilizar IDs já aposentados no changelog
> (ex.: `GAME-01` já é o tick concluído) nem quebrar as centenas de referências em docs, testes
> e comentários. O mapeamento histórico→canônico segue no fim deste índice.
>
> **Regra de ouro:** puxar primeiro os itens de esforço **S/M** e sem dependência externa;
> tarefas **L** só entram depois de decompostas em subtasks no próprio arquivo. Item bloqueado
> por captura, 2FA, patch ou decisão não entra na fila curta.

## Fila curta — ganhos rápidos sem bloqueio externo

| # | ID | Issue | Esforço | Próximo passo |
|---|---|---|---|---|
| 01 | `QA-11` | [Bots jogáveis e carga](issues/QA-11-bots-carga.md) | M | Rodar 2 bots em loopback e a carga de 200 com P99/rejeições. |
| 11 | `GAME-22` | [Árvore de EXP por flags](issues/GAME-22-exp-por-flags.md) | M | Fechar a árvore com o default inalterado e provar por flags. |
| 25 | `PROC-01` | [Organização da `docs/`, dedup e MkDocs](issues/PROC-01-organizacao-docs-mkdocs.md) | M | Fechar a dedup da documentação e publicar o MkDocs. |

> `CLIENT-06`, `CLIENT-10`, `BUG-01`, `BUG-02`, `GAME-17`, `GAME-18`, `WEB-07`, `WEB-08`,
> `WEB-09`, `WEB-10`, `WEB-11`, `CLIENT-11` e `GAME-13` têm resíduo bloqueado por captura/VM,
> 2FA, patch, decisão de contrato ou layout UNVERIFIED; não são “ganho rápido” até o gate cair.

## Resíduos de prova (não são ganhos rápidos)

| Issue | Status | O que já está provado | O que falta | Dependência |
|---|---|---|---|---|
| `BUG-01` | ⬜ | Recusas com motivo no log, testes de fio e entrada com `ClientID`/`UpdateScore`/cena. | Matriz mensagem × exibe/não-exibe e reprodução de cada recusa no 7662. | Cliente/VM (`CLIENT-06`). |
| `BUG-02` | 🟢 | Código, testes de fio e `clippy`; seleção antes do PIN devolve o frame. | Prova viva com `WYD_LIVE_SKIP_PIN=1` e entrada no cliente 7662. | Cliente/VM (`CLIENT-06`). |
| `GAME-17` | 🚧 | Faixa de `ClientID`, `UpdateScore`, `CreateType` e rajada real de `CreateMob`. | Render da entrada no cliente 7662, sem “Falha ao conectar”. | Cliente/VM (`CLIENT-06`). |
| `GAME-18` | 🟡 | 10 NPCs no seed `0046`, Sephira +9 no `0049`, 4 etapas, portões/chaves, loot no inventário, guardiões/boss no pipeline real de morte/loot (`integracao_campo_drops`) e `ReqLvl` do catálogo aplicado. | Buff do `Cabuncle`, portão visível e prova visual no 7662. | Captura no cliente/VM. |
| `GAME-11` | 🚧 | Depósito no inventário no abate e regra pura da caixa (30 s/3 min). | Handler `CreateItem`/`GetItem`/`DecayItem` e faixa de id do chão. | Captura do layout de item no chão (`CLIENT-06`). |
| `GAME-05` | 🚧 | Física, dano, skills e inventário autoritativos; recusa de pacote forjado. | `req_str/int/dex/con`, layout de rede do inventário e baú (`vault_items`). | `CLIENT-06` para o layout. |
| `CLIENT-10` | ⛔ | Layout candidato de 156 B + verificador `trade-layout.mjs` + runbook. | Frame real `MSG_Trade` do 7662. | Captura do usuário/VM. |

> Roteiro consolidado para as quatro provas visuais: [`docs/cliente/roteiro-prova-visual-7662.md`](../docs/cliente/roteiro-prova-visual-7662.md).
>
> **Tentativa de 2026-11-12:** o cliente 7662 abriu e desenhou o server select, mas descartou
> mouse e teclado sintéticos (`Fechar` não reagiu); sem login no canal e sem TCP em `8281`.
> O `Config.bin` foi corrigido do modo 8 (`1024×768×32`, cliente `1024×759`) para o modo 7
> (`800×600×32`, cliente exato `800×600`), mas o erro de render target e o input inerte
> permaneceram; modo 16 bits e GPU NVIDIA também não mudaram o quadro. `BUG-01`, `BUG-02` e
> `GAME-17` ficaram com veredito **B** na matriz, sem marcar aceite visual; `GAME-18` ficou
> **N/A** nesta sessão. Próximo passo: checklist humano do runbook §10.

## Fila de desbloqueio (quem trava o quê)

| Ordem | ID | Issue | Por que agora |
|---|---|---|---|
| 01 | `QA-11` | [Bots e carga](issues/QA-11-bots-carga.md) | Crate existe; falta prova de carga e relatório. |
| 02 | `CLIENT-06` | [RE dinâmica](issues/CLIENT-06-cliente-reverso-dinamico.md) | Captura de layout destrava trade (CLIENT-10; candidato 156 B × 154 B), pacotes S→C e provas visuais. |
| 03 | `BUG-02` | [PIN opcional](issues/BUG-02-entrada-no-mundo-pin-opcional.md) | Código/testes verdes; restam duas provas no cliente real. |
| 04 | `GAME-17` | [Entrada fiel](issues/GAME-17-entrada-no-mundo-fiel.md) | Só falta prova visual; destrava a experiência do cliente e o Campo. |

## Grafo de dependências das fases

Fonte única: o cabeçalho `Depende de`/`Bloqueia` de cada `roadmap-rust/issues/FASE-NN.md`.
A aresta `A → B` significa **B depende de A**; `FASE-02` é um portão de ritmo, não uma
dependência dura de todas as fases.

| Fase | Tema | Depende de | Bloqueia |
|---|---|---|---|
| `FASE-01` | Auditoria do roadmap e issues | — (raiz) | FASE-02 a FASE-15 |
| `FASE-02` | Execução controlada e ritmo | FASE-01 | FASE-03 a FASE-15 (ritmo) |
| `FASE-03` | Engine `wyd-core` (tick, ECS, grid) | FASE-01, FASE-08 | FASE-06, FASE-10 |
| `FASE-04` | Banco e persistência (`wyd-db`) | FASE-01 | FASE-07 |
| `FASE-05` | DevOps e CI/CD | FASE-01 | FASE-15 |
| `FASE-06` | Observabilidade e segurança | FASE-03 | FASE-12 |
| `FASE-07` | Portal web e ecossistema | FASE-04 | — |
| `FASE-08` | Protocolo, rede e launcher | FASE-01 | FASE-03 |
| `FASE-09` | IA, skills e ferramentas | FASE-01 | — (contínua) |
| `FASE-10` | Matemática e balanceamento | FASE-03, FASE-08 | FASE-15 |
| `FASE-11` | Documentação e padronização | FASE-01 | — (contínua) |
| `FASE-12` | Dashboards e visualização | FASE-06 | — |
| `FASE-13` | ADRs | FASE-01 | — (contínua) |
| `FASE-14` | Pesquisa técnica | — (contínua) | — (contínua) |
| `FASE-15` | Auditoria final e release | FASE-03 a FASE-08, FASE-10 | release |

```text
FASE-01 (raiz)
├── FASE-02 ──(ritmo)──> FASE-03..15
├── FASE-08 ──> FASE-03 ──> FASE-06 ──> FASE-12
│                          ├──> FASE-10 ──┐
│                          └──────────────┼──> FASE-15 (release)
├── FASE-04 ──> FASE-07 ──────────────────┤
├── FASE-05 ──────────────────────────────┤
├── FASE-09 / FASE-11 / FASE-13 (contínuas)
└── FASE-14 (contínua, sem aresta)
```

**Ordem topológica:** `01 → 02 → 08 → 04 → 05 → 09 → 11 → 13 → 03 → 06 → 10 → 07 → 12 → 15`;
`FASE-14` é contínua e pode rodar em qualquer ponto.

**Caminho crítico de release:** `FASE-01 → FASE-08 → FASE-03 → FASE-10 → FASE-15`.
É a maior cadeia de dependências duras (5 fases). `FASE-03 → FASE-06 → FASE-12` tem o mesmo
comprimento, mas termina no dashboard; `FASE-15` também espera `FASE-04`, `FASE-05`,
`FASE-06` e `FASE-07`, então atraso em qualquer um desses ramos atrasa o release.

**Caminho crítico de produto/portal:** `FASE-08 → FASE-03 → FASE-04 → FASE-07`.
A aresta `03→04` é a ordem de valor do §16 (contratos → engine → persistência → APIs), **não**
uma dependência declarada: `FASE-04` pode começar após `FASE-01`, mas o portal só fecha com o
contrato de rede e a engine estáveis.

Nenhuma fase fica fora do grafo: `FASE-01` é a raiz, `FASE-15` é o sorvedouro e `FASE-14` é
isolada por desenho contínuo. Não há fase órfã acidental.

## Issues ativas por categoria

> `#` é a posição na fila nova (01–27), não o ID. A ordem prioriza o que não depende de
> captura, 2FA, patch ou decisão pendente; os bloqueados aparecem no fim da fila.

### Bugs de fio (2)

| # | ID | Issue | Status | Esforço | Depende de |
|---:|---|---|---|---|---|
| 06 | `BUG-01` | [Mensagens do cliente ausentes + personagem não entra no mundo](issues/BUG-01-mensagens-char-mundo.md) | ⬜ | M | CLIENT-06 |
| 03 | `BUG-02` | [“Falha ao conectar” ao entrar no mundo: PIN opcional e seleção antes do PIN](issues/BUG-02-entrada-no-mundo-pin-opcional.md) | 🟢 | S | GAME-17 🚧 (mesma prova visual) |

### Jogo e gameplay (10)

| # | ID | Issue | Status | Esforço | Depende de |
|---:|---|---|---|---|---|
| 08 | `GAME-05` | [Autoridade total do servidor (física, dano, skills, inventário)](issues/GAME-05-autoridade-servidor.md) | 🚧 | M | GAME-02 ✅, GAME-03 ✅, GAME-04 ✅; layout CLIENT-06 |
| 13 | `GAME-11` | [Drop no inventário e caixa no chão com posse](issues/GAME-11-drop-inventario-e-caixa.md) | 🚧 | M | CLIENT-06 (id do item no chão); depósito já feito |
| 12 | `GAME-13` | [Familiares: slot, Nyerds, Imp, Griupan e Trajes](issues/GAME-13-familiares-slot-e-itens.md) | ⬜ | M | CLIENT-06 (slot Familiar) |
| 14 | `GAME-14` | [Fadas: bônus de drop/EXP, duração e auto-coleta](issues/GAME-14-fadas-bonus-e-auto-coleta.md) | ⬜ | M | GAME-11 |
| 04 | `GAME-17` | [Entrada no mundo fiel ao legado (id de jogador, `UpdateScore`, `CreateType`)](issues/GAME-17-entrada-no-mundo-fiel.md) | 🚧 | M | GAME-08 ✅, GAME-09 ✅; prova visual |
| 05 | `GAME-18` | [Campo de Treinamento: quest, portões, buff, boss e recompensas](issues/GAME-18-campo-de-treinamento.md) | 🟡 | L | GAME-17; prova visual |
| 09 | `GAME-20` | [Anomalia de velocidade (~200%) e hits duplicados](issues/GAME-20-velocidade.md) | ⬜ | M | GAME-19 ✅ (ordem provisória; trace em CLIENT-06) |
| 10 | `GAME-21` | [Laboratório `wyd-sim` (milhões de iterações)](issues/GAME-21-laboratorio.md) | ⬜ | M | GAME-19 ✅, GAME-20 |
| 11 | `GAME-22` | [Árvore de EXP por flags (default inalterado)](issues/GAME-22-exp-por-flags.md) | ⬜ | M | GAME-08 ✅ |
| 07 | `GAME-23` | [Bots autônomos do Campo de Treinamento (um por raça)](issues/GAME-23-bots-do-campo-de-treinamento.md) | 🟡 | M | QA-11 🟡, GAME-18 🟡 |

### Cliente, protocolo e launcher (4)

| # | ID | Issue | Status | Esforço | Depende de |
|---:|---|---|---|---|---|
| 02 | `CLIENT-06` | [Engenharia reversa dinâmica e correlação](issues/CLIENT-06-cliente-reverso-dinamico.md) | 🚧 | L | CLIENT-05 ✅, CLIENT-02 ✅ |
| 20 | `CLIENT-08` | [GUI do launcher em Tauri v2](issues/CLIENT-08-launcher-gui-tauri.md) | 🚧 | L | CLIENT-03 ✅, CLIENT-07 ✅ |
| 27 | `CLIENT-10` | [Trade atômica por par (layout candidato; captura pendente)](issues/CLIENT-10-trade-atomica.md) | ⛔ | M | CLIENT-06 (captura do layout) |
| 21 | `CLIENT-11` | [Launcher completo + integração com o portal + autoupdate](issues/CLIENT-11-portal-autoupdate.md) | 🚧 | L | CLIENT-03 ✅, CLIENT-08 🚧 |

### Portal e gateway (5)

| # | ID | Issue | Status | Esforço | Depende de |
|---:|---|---|---|---|---|
| 15 | `WEB-07` | [Monitoramento customizado: logs ao vivo, servidor, DB e alertas](issues/WEB-07-monitoramento-custom.md) | 🚧 | L | WEB-01 ✅ |
| 18 | `WEB-08` | [Contas, segurança, auditoria e área pública × administrativa](issues/WEB-08-contas-seguranca-auditoria.md) | 🚧 | L | WEB-03 ✅; 2FA/e-mail |
| 19 | `WEB-09` | [Entregas → Premium Mail + Cash + gestão financeira](issues/WEB-09-premium-cash.md) | 🚧 | L | WEB-04 ✅; inbox do jogador |
| 16 | `WEB-10` | [Catálogo de itens + economia do servidor](issues/WEB-10-catalogo-economia.md) | 🚧 | M | WEB-05 ✅; atributos no gateway |
| 17 | `WEB-11` | [Configurações 100% via portal (sem arquivo manual)](issues/WEB-11-configuracoes.md) | 🚧 | M | PROC-24; rollback/histórico |

### Qualidade, carga e segurança (4)

| # | ID | Issue | Status | Esforço | Depende de |
|---:|---|---|---|---|---|
| 22 | `QA-01` | [Bots população real + LLM + gestão de logs de teste](issues/QA-01-populacao-llm-carga.md) | ⬜ | L | BUG-01 |
| 23 | `QA-09` | [Suite completa + concorrência (tarpaulin/miri)](issues/QA-09-suite-concorrencia.md) | ⬜ | M | QA-02 ✅, GAME-21 |
| 24 | `QA-10` | [Carga de rede + integridade do banco + build de produção](issues/QA-10-carga-banco-release.md) | ⬜ | M | QA-09, QA-08 ✅ |
| 01 | `QA-11` | [Bots jogáveis e carga (`crates/wyd-bots`)](issues/QA-11-bots-carga.md) | 🟡 | M | GAME-06 ✅, CLIENT-02 ✅, QA-05 ✅ |

### Processo, documentação e arquitetura (2)

| # | ID | Issue | Status | Esforço | Depende de |
|---:|---|---|---|---|---|
| 25 | `PROC-01` | [Organização da `docs/`, dedup e MkDocs](issues/PROC-01-organizacao-docs-mkdocs.md) | 🚧 | M | PROC-13 ✅ |
| 26 | `PROC-24` | [Reorganização do roadmap: caminho crítico, curva de crescimento e entregas reais](issues/PROC-24-curva-crescimento.md) | 🚧 | M | PROC-01 |

### DevOps e CI (10)

> Origem: auditoria completa de `.github/workflows/rust.yml`, `Makefile`, `docker/Dockerfile.*`,
> `docker-compose*.yml` e `docker/monitoring/**` em 2026-11-12. Contrato e achados por severidade
> (P0/P1/P2) em [`docs/rust/10-devops-ci.md`](../docs/rust/10-devops-ci.md) §3; ordem recomendada
> em §10. Nada aqui depende de captura do cliente ou de VM.

| # | ID | Issue | Status | Esforço | Depende de |
|---:|---|---|---|---|---|
| 30 | `DEVOPS-03` | [Superfície de produção do compose: TLS, tag, privilégio e limites](issues/DEVOPS-03-superficie-producao-compose.md) | 🚧 | M | QA-03 ✅, WEB-01 ✅ |
| 31 | `DEVOPS-04` | [Paridade gate local × CI: fim do verde falso](issues/DEVOPS-04-paridade-gate-local-ci.md) | 🚧 | S | PROC-13 ✅, QA-02 ✅ |
| 32 | `DEVOPS-05` | [Cache, timeouts e custo do pipeline](issues/DEVOPS-05-cache-e-custo-ci.md) | 🚧 | S | DEVOPS-04 |
| 34 | `DEVOPS-07` | [Testes de banco e portal no CI (fim do skip silencioso)](issues/DEVOPS-07-banco-e-portal-no-ci.md) | 🚧 | M | DEVOPS-04 |
| 35 | `DEVOPS-08` | [Supply-chain contínua e cobertura medida](issues/DEVOPS-08-supply-chain-e-cobertura.md) | 🚧 | M | DEVOPS-07 |
| 36 | `DEVOPS-09` | [Runner auto-hospedado e deploy local após CI verde](issues/DEVOPS-09-runner-deploy-local.md) | 🚧 | M | DEVOPS-02, DEVOPS-04 |
| 37 | `DEVOPS-10` | [Versionamento de release e notas rastreáveis](issues/DEVOPS-10-versionamento-release.md) | 🚧 | S | DEVOPS-06 |
| 38 | `DEVOPS-11` | [Pipeline em estágios: grupos, matrizes e notificação](issues/DEVOPS-11-pipeline-em-estagios.md) | 🚧 | M | DEVOPS-04, DEVOPS-05 |

## Descritores de fase (contexto, não são issues)

Os arquivos `FASE-00.md` … `FASE-15.md` descrevem o escopo de cada fase e permanecem em `issues/` como referência.

## Convenções

- **Status:** ✅ concluída (só permanece aqui se o aceite ainda tem pendência) · 🚧 em andamento · ⬜ não iniciada · ⛔ bloqueada · 🟡 parcial · 🟢 pronta para validar.
- **Concluída de verdade:** critérios de aceite 100% marcados + evidência no arquivo. Ao fechar, remover o arquivo e registrar em `changelog.md`.
- **Skills:** cada issue referencia as skills de código, teste, docs e domínio na seção `Skills`; roteador em `.agents/skills/README.md`.

## Mapeamento histórico → canônico

O ID antigo não é usado como identificador de trabalho; serve apenas para localizar o registro histórico.

<details>
<summary>82 issues mapeadas</summary>

| ID histórico | ID canônico | Onde está |
|---|---|---|
| `BUG-01` | `BUG-01` | [`BUG-01-mensagens-char-mundo.md`](issues/BUG-01-mensagens-char-mundo.md) |
| `BUG-02` | `BUG-02` | [`BUG-02-entrada-no-mundo-pin-opcional.md`](issues/BUG-02-entrada-no-mundo-pin-opcional.md) |
| `FASE-03-1` | `GAME-01` | [`changelog.md`](changelog.md) |
| `FASE-03-2` | `GAME-02` | [`changelog.md`](changelog.md) |
| `FASE-03-3` | `GAME-03` | [`changelog.md`](changelog.md) |
| `FASE-03-4` | `GAME-04` | [`changelog.md`](changelog.md) |
| `FASE-03-5` | `GAME-05` | [`GAME-05-autoridade-servidor.md`](issues/GAME-05-autoridade-servidor.md) |
| `FASE-03-6` | `GAME-06` | [`changelog.md`](changelog.md) |
| `FASE-03-7` | `GAME-07` | [`changelog.md`](changelog.md) |
| `FASE-03-8` | `GAME-08` | [`changelog.md`](changelog.md) |
| `FASE-03-9` | `GAME-09` | [`changelog.md`](changelog.md) |
| `FASE-03-10` | `GAME-10` | [`changelog.md`](changelog.md) |
| `FASE-03-11` | `GAME-11` | [`GAME-11-drop-inventario-e-caixa.md`](issues/GAME-11-drop-inventario-e-caixa.md) |
| `FASE-03-12` | `GAME-12` | [`changelog.md`](changelog.md) |
| `FASE-03-13` | `GAME-13` | [`GAME-13-familiares-slot-e-itens.md`](issues/GAME-13-familiares-slot-e-itens.md) |
| `FASE-03-14` | `GAME-14` | [`GAME-14-fadas-bonus-e-auto-coleta.md`](issues/GAME-14-fadas-bonus-e-auto-coleta.md) |
| `FASE-03-15` | `GAME-15` | [`changelog.md`](changelog.md) |
| `FASE-03-16` | `GAME-16` | [`changelog.md`](changelog.md) |
| `FASE-03-17` | `GAME-17` | [`GAME-17-entrada-no-mundo-fiel.md`](issues/GAME-17-entrada-no-mundo-fiel.md) |
| `FASE-03-18` | `GAME-18` | [`GAME-18-campo-de-treinamento.md`](issues/GAME-18-campo-de-treinamento.md) |
| `FASE-10-1` | `GAME-19` | [`changelog.md`](changelog.md) |
| `FASE-10-2` | `GAME-20` | [`GAME-20-velocidade.md`](issues/GAME-20-velocidade.md) |
| `FASE-10-3` | `GAME-21` | [`GAME-21-laboratorio.md`](issues/GAME-21-laboratorio.md) |
| `FASE-10-4` | `GAME-22` | [`GAME-22-exp-por-flags.md`](issues/GAME-22-exp-por-flags.md) |
| `FASE-15-4` | `GAME-23` | [`GAME-23-bots-do-campo-de-treinamento.md`](issues/GAME-23-bots-do-campo-de-treinamento.md) |
| `FASE-08-1` | `CLIENT-01` | [`changelog.md`](changelog.md) |
| `FASE-08-2` | `CLIENT-02` | [`changelog.md`](changelog.md) |
| `FASE-08-3` | `CLIENT-03` | [`changelog.md`](changelog.md) |
| `FASE-08-4` | `CLIENT-04` | [`changelog.md`](changelog.md) |
| `FASE-08-5` | `CLIENT-05` | [`changelog.md`](changelog.md) |
| `FASE-08-6` | `CLIENT-06` | [`CLIENT-06-cliente-reverso-dinamico.md`](issues/CLIENT-06-cliente-reverso-dinamico.md) |
| `FASE-08-7` | `CLIENT-07` | [`changelog.md`](changelog.md) |
| `FASE-08-8` | `CLIENT-08` | [`CLIENT-08-launcher-gui-tauri.md`](issues/CLIENT-08-launcher-gui-tauri.md) |
| `FASE-08-9` | `CLIENT-09` | [`changelog.md`](changelog.md) |
| `FASE-08-10` | `CLIENT-10` | [`CLIENT-10-trade-atomica.md`](issues/CLIENT-10-trade-atomica.md) |
| `LAUNCHER-01` | `CLIENT-11` | [`CLIENT-11-portal-autoupdate.md`](issues/CLIENT-11-portal-autoupdate.md) |
| `FASE-07-1` | `WEB-01` | [`changelog.md`](changelog.md) |
| `FASE-07-2` | `WEB-02` | [`changelog.md`](changelog.md) |
| `FASE-07-3` | `WEB-03` | [`changelog.md`](changelog.md) |
| `FASE-07-4` | `WEB-04` | [`changelog.md`](changelog.md) |
| `FASE-07-5` | `WEB-05` | [`changelog.md`](changelog.md) |
| `FASE-07-6` | `WEB-06` | [`changelog.md`](changelog.md) |
| `PORTAL-01` | `WEB-07` | [`WEB-07-monitoramento-custom.md`](issues/WEB-07-monitoramento-custom.md) |
| `PORTAL-02` | `WEB-08` | [`WEB-08-contas-seguranca-auditoria.md`](issues/WEB-08-contas-seguranca-auditoria.md) |
| `PORTAL-03` | `WEB-09` | [`WEB-09-premium-cash.md`](issues/WEB-09-premium-cash.md) |
| `PORTAL-04` | `WEB-10` | [`WEB-10-catalogo-economia.md`](issues/WEB-10-catalogo-economia.md) |
| `PORTAL-05` | `WEB-11` | [`WEB-11-configuracoes.md`](issues/WEB-11-configuracoes.md) |
| `BOTS-01` | `QA-01` | [`QA-01-populacao-llm-carga.md`](issues/QA-01-populacao-llm-carga.md) |
| `FASE-05-1` | `QA-02` | [`changelog.md`](changelog.md) |
| `FASE-05-2` | `QA-03` | [`changelog.md`](changelog.md) |
| `FASE-06-1` | `QA-04` | [`changelog.md`](changelog.md) |
| `FASE-06-2` | `QA-05` | [`changelog.md`](changelog.md) |
| `FASE-06-3` | `QA-06` | [`changelog.md`](changelog.md) |
| `FASE-06-4` | `QA-07` | [`changelog.md`](changelog.md) |
| `FASE-12-1` | `QA-08` | [`changelog.md`](changelog.md) |
| `FASE-15-1` | `QA-09` | [`QA-09-suite-concorrencia.md`](issues/QA-09-suite-concorrencia.md) |
| `FASE-15-2` | `QA-10` | [`QA-10-carga-banco-release.md`](issues/QA-10-carga-banco-release.md) |
| `FASE-15-3` | `QA-11` | [`QA-11-bots-carga.md`](issues/QA-11-bots-carga.md) |
| `DOC-01` | `PROC-01` | [`PROC-01-organizacao-docs-mkdocs.md`](issues/PROC-01-organizacao-docs-mkdocs.md) |
| `FASE-00-1` | `PROC-02` | [`changelog.md`](changelog.md) · [evidência](../docs/rust/evidencias/PROC-02-limpeza-ambiente-2026-11-12.md) |
| `FASE-00-2` | `PROC-03` | [`changelog.md`](changelog.md) |
| `FASE-00-3` | `PROC-04` | [`changelog.md`](changelog.md) |
| `FASE-01-1` | `PROC-05` | [`changelog.md`](changelog.md) · [evidência](../docs/rust/evidencias/PROC-05-auditoria-2026-11-12.md) |
| `FASE-01-2` | `PROC-06` | [`changelog.md`](changelog.md) |
| `FASE-01-3` | `PROC-07` | [`changelog.md`](changelog.md) |
| `FASE-02-1` | `PROC-08` | [`changelog.md`](changelog.md) |
| `FASE-02-2` | `PROC-09` | [`changelog.md`](changelog.md) |
| `FASE-02-3` | `PROC-10` | [`changelog.md`](changelog.md) |
| `FASE-02-4` | `PROC-11` | [`changelog.md`](changelog.md) |
| `FASE-02-5` | `PROC-12` | [`changelog.md`](changelog.md) |
| `FASE-02-6` | `PROC-13` | [`changelog.md`](changelog.md) |
| `FASE-04-1` | `PROC-14` | [`changelog.md`](changelog.md) |
| `FASE-04-2` | `PROC-15` | [`changelog.md`](changelog.md) |
| `FASE-04-3` | `PROC-16` | [`changelog.md`](changelog.md) |
| `FASE-09-1` | `PROC-17` | [`changelog.md`](changelog.md) |
| `FASE-09-2` | `PROC-18` | [`changelog.md`](changelog.md) |
| `FASE-11-1` | `PROC-19` | [`changelog.md`](changelog.md) |
| `FASE-11-2` | `PROC-20` | [`changelog.md`](changelog.md) |
| `FASE-13-1` | `PROC-21` | [`changelog.md`](changelog.md) |
| `FASE-13-2` | `PROC-22` | [`changelog.md`](changelog.md) |
| `FASE-14-1` | `PROC-23` | [`changelog.md`](changelog.md) |
| `ROAD-01` | `PROC-24` | [`PROC-24-curva-crescimento.md`](issues/PROC-24-curva-crescimento.md) |

</details>
