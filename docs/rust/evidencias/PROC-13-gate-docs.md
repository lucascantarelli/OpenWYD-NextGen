# Evidência — PROC-13: gate de consistência de docs

> **Issue:** `PROC-13` — Gate de consistência entre roadmap, progresso, checklist e código, concluída.
> **Registro canônico:** [`roadmap-rust/changelog.md`](../../../roadmap-rust/changelog.md).
> **Data da coleta:** 2026-11-12.
> **Objetivo:** colocar no `make check` a validação de links relativos de `README.md`, `AGENTS.md`,
> `docs/` e `roadmap-rust/`, além da contagem/nomes das crates, fechando o residual que mantinha a issue aberta.

## Implementação

| Arquivo | Mudança |
|---|---|
| `tools/dev/check-docs-consistency.mjs` | Novo gate sem dependências: membros de `Cargo.toml` × diretórios `crates/<nome>/Cargo.toml`; frase de contagem e árvore do `README.md`; tabela de crates do `AGENTS.md`; links relativos de `README.md`, `AGENTS.md`, `docs/` e `roadmap-rust/`; resumo de issues/migrations. Falha com exit `1` e lista de divergências. |
| `Makefile` | Novo alvo `docs-check`, incluído em `check` (junto de `migrations-check` e `issue-status-check`). |
| `.github/workflows/rust.yml` | Executa o gate no job Rust e inclui `README.md`, `AGENTS.md`, `docs/**` e `roadmap-rust/**` nos gatilhos. |
| `README.md` | Corrigida a árvore (linha quebrada de `wyd-db`/`wyd-admin`), adicionado `wyd-bots` e corrigida a contagem para **9 crates**. |
| `AGENTS.md` | Adicionada a linha `crates/wyd-bots/`; a tabela passou a refletir as 9 crates do workspace. |
| `tools/dev/README.md` | Documenta o novo script e o alvo. |
| `docs/rust/02-execucao.md` | Gate único descrito com migrations, issues e docs. |
| `roadmap-rust/index.md`, `docs/rust/03-progresso.md` | PROC-13 removida da fila ativa e status/contagens alinhados (39 ativas, 43 concluídas). |

## Verificação

```text
$ node tools/dev/check-docs-consistency.mjs
docs: 39 issues, 9 crates, 10 membros, 48 migrations, 182 docs, 259 links — consistente
exit 0
```

O workspace tem 10 membros: 9 em `crates/` + `apps/launcher-ui`; a documentação usa a
contagem de **crates** (9), como a frase do README e a tabela do AGENTS.

Prova negativa — `README.md` alterado temporariamente de `compila as 9 crates` para
`compila as 8 crates`:

```text
docs: 1 divergência(s) | 9 crates | 182 docs | 259 links
  ✗ README.md: diz 8 crates, workspace tem 9
exit 1
```

Restaurado para 9: `exit 0`. Nenhuma alteração temporária permaneceu.

```text
$ make check
fmt --check ........................ exit 0
clippy --workspace --all-targets ... exit 0
build --workspace .................. exit 0
test --workspace ................... exit 0
migrations-check ................... 48 migrations, 86 tabelas, 159 INSERTs, ok
issue-status-check ................. 39/39 issues ativas, 0 cobertura incompleta
docs-check ......................... 39 issues, 9 crates, 10 membros, 48 migrations, 182 docs, 259 links
tools-test ......................... 7/7
MAKE_EXIT:0
```

A execução completa voltou a apresentar, uma vez, a falha intermitente já conhecida do teste
`sistema_telemetry` (`ConnectionReset` no cliente HTTP local); o teste passou **3/3** isolado e a
execução final do `make check` passou. Não é uma regressão do gate documental.

## Resultado

O residual da PROC-13 está fechado: link relativo quebrado em `README.md`, `AGENTS.md`, `docs/`
ou `roadmap-rust/`, e crate fora do número/nomes documentados, falham o `make check`; a árvore
consistente termina com exit `0`. A varredura cobre 182 arquivos Markdown e 259 links relativos.
