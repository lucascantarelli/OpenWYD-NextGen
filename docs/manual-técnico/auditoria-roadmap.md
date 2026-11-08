# Auditoria do planejamento legado (Go) — inventário

> Resultado da FASE-01-1 (`roadmap-rust/issues/FASE-01-1-auditoria-estado.md`).
> Fonte auditada: `roadmap-rust/` (arquivo do projeto Go). Status
> declarado **não** é tratado como prova: onde não há evidência, o item é
> marcado **hipótese**.

## 1. Ambiente auditado

| Item | Antes | Agora |
|---|---|---|
| `roadmap/` na árvore de trabalho | ausente (movido para `arquivo Go removido (2026-11)` e o diretório ficou vazio) | restaurado de `HEAD` (`b58a607`) em `roadmap-rust/` |
| Arquivos de planejamento | 0 em disco | 138 |
| `roadmap/issues/` | — | 62 |
| `roadmap/done/` | — | 54 |
| `roadmap/test/` | — | 13 |

Restauração executada (sem tocar no índice do git, pois `arquivo Go removido (2026-11)` está
fora do versionamento):

```bash
git archive HEAD roadmap | tar -x -C arquivo-Go-removido
```

**Achado:** o planejamento Go existia apenas no histórico do git. O conteúdo
foi preservado (regra de preservação de conhecimento de `pre-release.md` §1.2),
mas a árvore de trabalho não o continha — qualquer auditoria anterior estava
cega para as 138 fontes.

## 2. Documentos de topo (referência de domínio)

- `AUDITORIA-PRE-RELEASE-2026-11-06.md`
- `PLANO-2026-11.md`
- `PRE-RELEASE-PROGRAM.md`
- `REVIEW-11-2026-11-06.md`
- `REVISAO-PROFUNDA-2026-11-06.md`
- `analise-tecnica.md`, `atlas-mundo.md`, `editabilidade.md`, `index.md`

## 3. Débito técnico medido

| Fonte | Marcadores (`TODO/FIXME/XXX/HACK`) |
|---|---|
| Go (`tmserver`, `internal`, `dbserver`, `binserver`, `webserver`) | 0 |
| C++ legado (`Source/`) | 16 |

**Hipótese:** a ausência de marcadores no Go não significa ausência de débito —
o código foi traduzido em massa (commit "Arquivos com comentários traduzidos
para PT-BR") e comentários originais podem ter sido perdidos na tradução.
Verificar amostralmente antes de concluir que não há dívida.

### 3.1 Classificação das 16 ocorrências (C++) — FASE-01-2

| Fonte | Teor | Destino | Onde resolve |
|---|---|---|---|
| `Basedef.h` (2×) | renomear/confirmar `MSG_CNFAccountLogin` | **portar** | FASE-08-1 (tabela de opcodes) |
| `Basedef.h` | "Use MSG_STANDARDPARM" | **portar** | FASE-08-1 |
| `Basedef.h` | "check, confirm structure" | **investigar** | FASE-08-1 (UNVERIFIED) |
| `DBSrv/CRanking.cpp` (3×) | leitura de arquivo inicial/ranking | **descartar** | substituído por `wyd-db` + ranking no portal (FASE-07-2) |
| `DBSrv/Server.cpp` (2×) | leitura de arquivo inicial | **descartar** | idem |
| `TMSrv/MobKilled.cpp` | renomear `UNK_s` | **investigar** | FASE-10-1 (equações/drop) |
| `TMSrv/ProcessDBMessage.cpp` (4×) | `break` vs `return` no fluxo DB | **investigar** | FASE-08-1 (`docs/cliente/pacotes.md + crates/wyd-net/tests/`) |
| `TMSrv/ProcessSecMinTimer.cpp` | timer periódico: manter? | **investigar** | FASE-03-1 (tick substitui timers) |
| `TMSrv/Server.cpp` | "TODO: EVERYTHING" | **investigar** | FASE-03-1 (boot/validação de conteúdo) |
| `TMSrv/_MSG_AcceptParty.cpp` | trecho marcado `HACK` | **portar com cautela** | FASE-08-1 + FASE-03-5 (autoridade) |

Regra aplicada: nada é portado cegamente do C++/Go; cada item foi destinado a
uma issue dona, com o critério `portar` (comportamento necessário),
`descartar` (substituído pela nova arquitetura) ou `investigar` (evidência
insuficiente — **hipótese** até prova em captura/código).

## 4. Classificação das 54 issues em `done/` — como verificar

Critério (sem confiar no status manual):

1. A issue cita arquivos/caminhos? Conferir existência e conteúdo atual.
2. A issue cita teste? Conferir se o teste existe e cobre o que ela afirma.
3. Sem arquivo nem teste localizável → **hipótese de incompleta**.

A verificação item a item está aberta (subtarefa desta issue). Amostra inicial
de `done/`: `BOT-01`, `BOT-02`, `BUILD-01`, `CANAL-01`, `CANAL-03`,
`CHANNEL-01`, `CHANNEL-02`, `CI-01`.

## 5. Duplicidades e destino

- Todo item de `roadmap/` é histórico: o planejamento vigente é `roadmap-rust/`.
- Nenhuma issue Go é copiada para `roadmap-rust/`; o que sobrevive é regra de
  domínio (fórmula, mensagem, protocolo), incorporada nas issues Rust
  correspondentes como referência — nunca como requisito paralelo.
- Issues Go ainda abertas (62) são candidatas a virar evidência de lacuna na
  fase Rust equivalente (FASE-10 matemática, FASE-06 mensagens/canais).

## 6. Limitações desta rodada

- Não foi lido o conteúdo integral das 138 fontes (leitura direcionada por
  necessidade da fase em execução).
- Evidência de implementação das 54 `done/` ainda não verificada uma a uma.
- Nenhuma pendência de classificação: as 16 ocorrências C++ têm issue dona (§3.1).
