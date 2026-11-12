# Índice de evidências técnicas

> Provas registradas no repositório que sustentam issues concluídas. Cada arquivo registra o
> que foi medido, como foi verificado e o que permanece como hipótese ou pendência. O status
> canônico das issues continua em [`roadmap-rust/index.md`](../../../roadmap-rust/index.md),
> o painel em [`docs/rust/03-progresso.md`](../03-progresso.md) e o fechamento em
> [`roadmap-rust/changelog.md`](../../../roadmap-rust/changelog.md).

| Evidência | Issue | Data | Resultado |
|---|---|---|---|
| [`AUDIT-IA-2026-11-12.md`](AUDIT-IA-2026-11-12.md) | `PROC-18` | 2026-11-12 | Auditoria das instruções de IA; skills reduzidas a 23 e referências/gates íntegros. |
| [`PROC-03-rust-analyzer.md`](PROC-03-rust-analyzer.md) | `PROC-03` | 2026-11-12 | Workspace analisado pelo rust-analyzer, VS Code em Rust e sem extensão Go. |
| [`PROC-05-auditoria-2026-11-12.md`](PROC-05-auditoria-2026-11-12.md) | `PROC-05` | 2026-11-12 | **36 issues do baseline:** 21 abertas, 8 em andamento, 2 parciais, 1 pronta, 1 bloqueada e 3 concluídas; 33 ativas + 3 fechadas, com próximo passo por linha. |
| [`PROC-11-revisao-fria-2026-11-12.md`](PROC-11-revisao-fria-2026-11-12.md) | `PROC-11` | 2026-11-12 | Árvore real, DAG e módulos confrontados com o documento normativo; divergências corrigidas. |
| [`PROC-13-gate-docs.md`](PROC-13-gate-docs.md) | `PROC-13` | 2026-11-12 | Gate de links e consistência de docs incorporado ao `make check`. |
| [`QA-08-prova-viva-2026-11-12.md`](QA-08-prova-viva-2026-11-12.md) | `QA-08` | 2026-11-12 | Dashboards com série real dos dois canais; 10 regras e 12 painéis. |
| [`WEB-02-painel-vivo.md`](WEB-02-painel-vivo.md) | `WEB-02` | 2026-11-12 | Quatro fluxos do painel do jogador provados no portal/gateway vivos. |
| [`WEB-04-e2e-2026-11-12.md`](WEB-04-e2e-2026-11-12.md) | `WEB-04` | 2026-11-12 | E2E Playwright: 6 specs, três execuções verdes. |

## Regras

- Evidência nova entra com issue, data, comando/medição e resultado; hipótese é marcada como
  hipótese, nunca como prova.
- Issue concluída sai de `roadmap-rust/issues/` e entra no changelog; esta tabela guarda o
  caminho da prova.
- Arquivo de evidência não substitui o critério de aceite da issue nem promove dependências
  externas (cliente 7662/VM) sem a prova correspondente.
