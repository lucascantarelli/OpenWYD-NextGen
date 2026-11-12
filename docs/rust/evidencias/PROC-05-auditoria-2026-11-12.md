# PROC-05 — Auditoria do estado real das 36 issues do baseline (2026-11-12)

> **Escopo desta auditoria:** as **36 issues do baseline** de `roadmap-rust/issues/` na
> abertura da rodada (os 15 descritores `FASE-NN.md` não são issues). Três foram concluídas
> e removidas; a reconciliação para as **33 ativas** está no §6. O histórico Go foi contado, mas não reauditado
> item a item; a limitação está declarada no §3. Nenhum status foi promovido sem evidência.
>
> **Regra:** “declarado” é o status do índice/arquivo; “auditado” é o veredito depois de
> confrontar código, teste, documento e execução. Onde não há prova, a linha diz **hipótese**
> e o que falta. Não há célula vazia nem “TBD”.

## 1. Como a auditoria foi feita

1. Listagem na abertura: `roadmap-rust/issues/` tinha **51 arquivos** = **36 issues canônicas**
   + **15 descritores `FASE-NN.md`**. Após os três fechamentos, restam 48 arquivos = 33 issues
   canônicas + 15 descritores.
2. Para cada issue: leitura do objetivo, critérios de aceite, subtasks, revisão de prova e
   links; confronto com `rg`/`cargo test`/`make check` e com os documentos citados.
3. Contagem de concluídas: **46** entradas em `roadmap-rust/changelog.md` (inclui a QA-08 e
   a WEB-04 mais recentes).
4. O histórico Go foi medido pelo inventário já publicado: **138 arquivos**, **62 issues
   abertas**, **54 em `done/`**, **13 em `test/`** (`docs/manual-técnico/auditoria-roadmap.md`
   §1). A leitura item a item de `done/` continua **hipótese**, como o próprio documento
   declara no §4/§6.
5. `make check` fechou verde antes desta tabela; a prova final está no §4.

## 2. Tabela das 36 issues do baseline

| ID | Declarado | Auditado | Evidência real ou hipótese declarada | Próximo passo |
|---|---|---|---|---|
| `BUG-01` | ⬜ | ⬜ aberta | Recusas com motivo no log e testes de fio (`login`, `selchar`, `cena_mobs`) passam; a matriz mensagem × exibe e a reprodução no 7662 não existem. | Provar visualmente com conta admin; `CLIENT-06`. |
| `BUG-02` | 🟢 | 🟢 pronta para validar | Código e testes de fio provam PIN opcional/seleção antes do PIN; falta `WYD_LIVE_SKIP_PIN=1` com conta real e entrada no 7662. | Uma prova viva + uma visual; depois fechar. |
| `GAME-05` | 🚧 | 🚧 em andamento | Física/dano/skills/inventário autoritativos com 10+9 testes de núcleo; o fio de mover/equipar/vender e o layout de rede do inventário não existem. | Fechar o eixo de fio do inventário; `CLIENT-06` para layout. |
| `GAME-11` | ⬜ | ⬜ aberta | Loot no inventário está provado (`campo_vivo`), mas `CreateItem`/`GetItem`/`CNFGetItem`/`DecayItem` não têm handler nem call site de caixa no chão. | Implementar caixa/pickup após o rito da tabela; offsets em `docs/cliente/pacotes.md`. |
| `GAME-12` | ⬜ | ⬜ aberta | `view item_stack_cap` e `max_stack` existem; a aglutinação item a item no fio não. | Implementar com testes de pilha e catálogo. |
| `GAME-13` | ⬜ | ⬜ aberta | Existe documentação de familiares e slots; não há handler, entidade nem teste de Nyerds/Imp/Griupan/Trajes. | Implementar o domínio antes do fio. |
| `GAME-14` | ⬜ | ⬜ aberta | Fadas/bônus estão documentados; nenhuma regra de duração/auto-coleta no código. Depende de `GAME-11`. | Implementar depois da caixa/pickup. |
| `GAME-15` | ⬜ | ⬜ aberta | Catálogo tem baldes/teto; a classificação e correções de teto por item não foram implementadas. Depende de `GAME-12`. | Fechar junto de `GAME-12`. |
| `GAME-17` | 🚧 | 🚧 em andamento | Faixa de `ClientID`, `UpdateScore`, `CreateType` e rajada de `CreateMob` provados (`cena_mobs`, `world`); falta o render no 7662 e o “Falha ao conectar”. | Prova visual; `CLIENT-06`. |
| `GAME-18` | 🟡 | 🟡 parcial | Seed `0046` com 10 NPCs, 4 etapas, portões/chaves, loot no inventário e `GremlinDecoy`/451 no combate real. Faltam os outros guardiões/boss, Sephira +9, buff do `Cabuncle`, portão visível e prova visual. | Fluxo completo dos guardiões + visual; `CLIENT-06`. |
| `GAME-19` | 🚧 | 🚧 em andamento | Só `damage_after_absorb` está provado; `damage_chain`/`STAGE_ORDER` são hipótese (D-02…D-04). B13–B17 sem affect no servidor. | Goldens L3 + LCG; depois affects. |
| `GAME-20` | ⬜ | ⬜ aberta | A anomalia ~200% e hits duplicados não têm reprodução no Rust; `formula/speed.rs` está reservado. | Reproduzir com GAME-19 antes de codar. |
| `GAME-21` | ⬜ | ⬜ aberta | `wyd-sim` tem fórmulas, mas não o laboratório de milhões de iterações. | Montar harness com seed e relatório. |
| `GAME-22` | ⬜ | ⬜ aberta | A árvore de EXP por flags (`DOUBLEMODE`, `KefraLive`, party, ±15%) não existe; o default atual é `grant_exp` + multiplicador. | Implementar flags `off` por default com teste de regressão. |
| `GAME-23` | 🟡 | 🟡 parcial | `wyd-bots` tem 4 raças e campanha de 4 etapas (3 testes), mas a caixa/pickup e a campanha autônoma completa não estão provadas. | Depende de GAME-18/GAME-11. |
| `CLIENT-06` | 🚧 | 🚧 em andamento | RE estática entregue (6/6 âncoras, 3/3 patches); captura dinâmica/VM e goldens S→C não foram produzidos. | Executar na VM com o 7662; único bloqueio externo. |
| `CLIENT-08` | 🚧 | 🚧 em andamento | `apps/launcher-ui` tem shell Tauri e comandos; o fluxo completo da GUI e a prova de janela não estão fechados. | Testar `--check`/`jogar` no Windows; fechar com prova. |
| `CLIENT-10` | ⛔ | ⛔ bloqueada | `MSG_Trade` está UNVERIFIED e não há payload/handler; o bloqueio é de layout, não de esforço. | Depende de `CLIENT-06`; não iniciar antes. |
| `CLIENT-11` | ⬜ | ⬜ aberta | Launcher tem patch/manifest/updater, mas a integração completa com portal/autoupdate e a prova de ponta a ponta não existem. | Depende de `CLIENT-08`/`PROC-24`. |
| `WEB-01` | 🚧 | 🚧 em andamento | 44 métodos do portal têm handler e `admin.pending = 0`; o resíduo é a rota legada `set_reward_item_enabled` em `501`. | Decidir remover ou criar coluna em migration nova. |
| `WEB-07` | ⬜ | ⬜ aberta | Monitoramento base existe (QA-08), mas logs ao vivo/servidor/DB/alertas customizados da tela não. | Implementar sobre WEB-01. |
| `WEB-08` | ⬜ | ⬜ aberta | Admin tem autorização/auditoria; área pública × administrativa e ciclo completo de contas/segurança não foram entregues. | Implementar sobre WEB-03/WEB-01. |
| `WEB-09` | ⬜ | ⬜ aberta | Grants idempotentes e donate existem; Premium Mail, Cash e gestão financeira completa não. | Implementar após `WEB-06`/`PROC-24`. |
| `WEB-10` | ⬜ | ⬜ aberta | Catálogo e economia têm leitura; a tela completa de catálogo/economia do portal não. | Implementar sobre `WEB-05`. |
| `WEB-11` | ⬜ | ⬜ aberta | Configurações já são editáveis no portal (rates), mas “100% via portal, sem arquivo manual” não foi provado. | Auditar cada setting e remover configuração manual. |
| `QA-01` | ⬜ | ⬜ aberta | Bots de carga/população com LLM não existem; `wyd-bots` é campanha de QA, não população. | Depende de BUG-01 e PROC-24. |
| `QA-07` | ⬜ | ✅ fechada nesta rodada | Catálogo publicado em `docs/wyd/sistemas/ameacas.md`: B1–B17 um a um + 29 ameaças do §6 com os 10 campos; checklist §6 atualizado. | Concluído: issue removida, catálogo e changelog publicados. |
| `QA-09` | ⬜ | ⬜ aberta | Não há tarpaulin/miri nem suíte de concorrência; há testes pontuais de sessão/grants. | Adicionar cobertura e rodar em CI. |
| `QA-10` | ⬜ | ⬜ aberta | QA-08 fechou a prova de métricas; carga de rede, integridade de banco e build de produção continuam sem execução. | Depende de QA-09. |
| `QA-11` | ⬜ | ⬜ aberta | `wyd-bots` faz campanha de 4 etapas, mas não carga jogável/autônoma com métricas. | Depende de GAME-06/CLIENT-02/QA-05. |
| `PROC-01` | ⬜ | ⬜ aberta | `docs/` tem índice e links validados, mas dedup estrutural e MkDocs não foram concluídos. | Executar dedup + build MkDocs. |
| `PROC-02` | 🚧 | 🚧 em andamento | `go clean` executado; Go ainda no PATH e `GOPATH` de 200 MB dependem de ação elevada do dono. | Remover Go/GOPATH após autorização. |
| `PROC-05` | 🚧 | ✅ fechada nesta rodada | Esta tabela classifica as 36 issues; o histórico Go tem contagem e hipótese declaradas no §3. | Concluído: issue removida, esta evidência e o changelog publicados. |
| `PROC-11` | 🚧 | ✅ fechada nesta rodada | Revisão fria em `docs/rust/evidencias/PROC-11-revisao-fria-2026-11-12.md`; árvore executada publicada e divergências documentais corrigidas. | Concluído: issue removida, evidência e changelog publicados. |
| `PROC-23` | 🚧 | 🚧 em andamento | Notas datadas e decisão manter/descartar existem; faltam os números medidos das PoCs (SoA × hecs etc.). | Rodar as PoCs e publicar números. |
| `PROC-24` | ⬜ | ⬜ aberta | O índice tem fases/caminho crítico, mas a curva de crescimento e as entregas reais por ciclo não foram fechadas. | Depende de PROC-01. |

## 3. Histórico Go — o que está provado e o que é hipótese

| Grupo | Contagem | Estado da prova |
|---|---:|---|
| Issues abertas no planejamento Go | 62 | **Hipótese de escopo histórico**: nomes e arquivos contados no inventário; não foram reexecutadas. |
| Issues em `done/` | 54 | **Hipótese de conclusão**: o §4 do inventário declara que a leitura item a item não foi feita; amostra de 8 foi citada. |
| Issues em `test/` | 13 | **Hipótese de estado**: existem como arquivo; o resultado dos testes não foi reconstruído nesta rodada. |
| Total de arquivos | 138 | **Medido** por listagem do arquivo histórico. |

A auditoria **não** usa esses 138 arquivos para promover nenhuma issue Rust. Eles explicam a
origem do roadmap e permanecem como dívida documental; o critério de fechamento desta PROC-05
é a tabela das 36 issues Rust acima.

## 4. Gate e resultado

```text
# na abertura da auditoria
make check -> 0
issue-status: 36/36 issues ativas comparadas | cobertura incompleta: 0
docs: 36 issues, 9 crates, 10 membros, 48 migrations, 181 docs, 264 links
tools: 7/7

# depois de remover QA-07, PROC-05 e PROC-11
make check -> 0
issue-status: 33/33 issues ativas comparadas | cobertura incompleta: 0
docs: 36 issues, 9 crates, 10 membros, 48 migrations, 181 docs, 268 links
tools: 7/7
```

Após o fechamento de QA-07, PROC-05 e PROC-11, a contagem passa a **33 ativas / 49
concluídas**; o índice e o painel são a fonte final.

## 5. Conclusão

- **36/36** issues do baseline classificadas; nenhuma linha vazia.
- **33 abertas/em andamento/prontas/parciais/bloqueadas** e **3 fechadas nesta rodada**.
- **Nenhum** item dependente do 7662 foi promovido: BUG-01, BUG-02, GAME-17 e GAME-18
  continuam com prova visual pendente.
- O histórico Go está contado e marcado como hipótese onde não houve reexecução; não foi
  usado como prova de conclusão.

## 6. Reconciliação com o estado atual

O §2 classifica as 36 linhas do baseline. A tabela abaixo reconstitui a soma por **status
auditado** e separa as três issues concluídas; a soma fecha em 36 e as ativas em 33.

| Status auditado | Baseline (36) | Fechadas | Ativas (33) |
|---|---:|---:|---:|
| ⬜ aberta | 21 | 0 | 21 |
| 🚧 em andamento | 8 | 0 | 8 |
| 🟡 parcial | 2 | 0 | 2 |
| 🟢 pronta para validar | 1 | 0 | 1 |
| ⛔ bloqueada | 1 | 0 | 1 |
| ✅ concluída | 3 | 3 | 0 |
| **Total** | **36** | **3** | **33** |

As três concluídas são `QA-07`, `PROC-05` e `PROC-11`; nenhuma issue ativa foi promovida
sem evidência, e os resíduos visuais do 7662 continuam abertos.

**Revisão de checklists (2026-11-12):** depois desta fotografia, a revisão dos checklists
promoveu `CLIENT-11`, `WEB-08`, `WEB-09`, `WEB-10` e `WEB-11` de ⬜ para 🚧 e `GAME-15` de
⬜ para 🟡, com base nas subtasks e verificações já registradas nos próprios arquivos. O
baseline do §2 permanece o retrato da auditoria (21/8/2/1/1/3 = 36); o estado corrente é o do
índice e do painel: **15 ⬜ · 13 🚧 · 3 🟡 · 1 🟢 · 1 ⛔** entre as 33 ativas.
