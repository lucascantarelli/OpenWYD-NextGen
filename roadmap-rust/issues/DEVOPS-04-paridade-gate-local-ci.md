# DEVOPS-04 — Paridade gate local × CI: fim do verde falso

> **ID canônico:** `DEVOPS-04`
> **ID histórico:** `DEVOPS-04`
> **Estado 2026-11-13:** 🚧 · ver `roadmap-rust/index.md`.
> **Run real:** [43354688714](https://github.com/lucascantarelli/wyd/actions/runs/43354688714).

**Tipo:** DevOps · **Esforço:** S · **Depende de:** PROC-13 ✅, QA-02 ✅ · **Bloqueia:** DEVOPS-05

## Objetivo

Fazer o CI executar **tudo** o que `make check` executa, e o `Makefile` ser honesto sobre o que
ele cobre — de modo que "CI verde" e "gate verde" passem a significar a mesma coisa. Inclui
consertar a rotina de fechamento (`make local-rebuild`) que hoje falha por número fixo de schema.

## Contexto / evidência

- `make check` = `fmt clippy build test migrations-check issue-status-check docs-check
  skills-check tools-test`. O workflow roda apenas `check-docs-consistency.mjs` (job `rust`).
  Faltam no CI: `migrations-check`, `issue-status-check`, `skills-check`, `tools-test`.
- Nenhum dos dois roda `portal-test` (`tsc --noEmit` + `vitest`) nem `monitoring-rules`.
- `.PHONY` não cobre `clean-target`, `clean-tmp`, `tools-test`, `tools-inventario`,
  `tools-zip-check`, `client-zip`.
- `make help` imprime 4 linhas fixas, ignorando os `##` presentes em todos os targets.
- `make fmt` = `cargo fmt --check`; CI = `cargo fmt --all --check`.
- `make local-rebuild` exige `"schema":49`, mas `SCHEMA_VERSION = 50` (50 migrations): a rotina
  oficial de fechamento reprova sem defeito real. `docs/rust/03-progresso.md` ainda registra
  `schema:36` na mesma seção.
- Análise completa em `docs/rust/10-devops-ci.md` §2/§4 (achados B1, B9, C1–C3).
- **Medido em 2026-11-12 (mesma stack, sem mudar código):** `make check` **exit 0** — 811 testes
  (`cargo test --workspace`), `check-migrations` 50/86/160, 35 issues, 318 links, 18 skills,
  `tools-test` 7/7. Fora do `check` e **verdes agora**: `make portal-test` exit 0 (`tsc --noEmit`
  + 29 arquivos vitest) e `make portal-build` exit 0 (34 páginas estáticas) — dá para ligá-los ao
  gate/CI sem dívida pré-existente. `make monitoring-rules` também é verde e segue fora do
  `check`. `make bench` (`cargo bench --workspace`) é **instável por ruído de host**: mesmo
  binário, 4 medições, p99 de IA 10,9 % → 3,1 % → 4,9 % → 1,0 % do orçamento (teto 10 %) e ECS
  68 % → 37 % → 27 % (teto 50 %); os gates determinísticos passam (`bench-build` exit 0,
  `bench-gate` exit 0 com 3 cenários dentro do limite). Conclusão para o passo 3: manter `bench`
  fora do `check`, gatear por `bench-gate` e não por p99 medido em máquina com Docker rodando.

## Run real (2026-11-13)

- Run [43354688714](https://github.com/lucascantarelli/wyd/actions/runs/43354688714): os **sete** jobs
  verdes — `gate` (fmt, clippy `-D warnings`, build, test, migrations, issue-status, docs, skills,
  versão única e tools-test), `portal` (tsc + vitest + next build), `sistema` (Postgres real),
  `monitoramento`, `compose`, `benches` e `actionlint`. A paridade deixou de ser promessa: é o log.
- Falta para fechar: provar o **negativo** — injetar uma divergência de status/link/skill e ver o CI
  reprovar, depois reconciliar para verde.

## Progresso (2026-11-13)

- Paridade implementada: o job `gate` do `.github/workflows/rust.yml` roda a mesma lista do
  `make check` (`fmt --all --check`, `clippy -D warnings`, build, test, migrations, issues, docs,
  skills, `versao-check`, `tools-test`) **e** o `portal` cobre `portal-test`/`portal-build`.
  `make ci` passou a espelhar de verdade: `test-all bench-build bench-gate audit deny
  monitoring-rules stack-config compose-hardening portal-build`.
- `Makefile`: `.PHONY` completo, `help` derivado dos comentários `##`, `make local-rebuild` deriva
  `SCHEMA_VERSION` de `crates/wyd-db/src/lib.rs` (fim do `"schema":49` fixo contra 50).
  Evidência: `make help` lista os 30+ alvos, `make check` exit 0.
- Ainda falta o que só um run real prova: injetar uma divergência de status/link/skill e ver o CI
  reprovar, depois reconciliar para verde; e reexecutar `make local-rebuild` após a mudança do
  schema.

## Passos

1. Workflow: no job `rust`, rodar `node tools/content/check-migrations.mjs`,
   `node tools/content/check-issue-status.mjs`, `node tools/dev/check-skills.mjs` e
   `node --test apps/tools/test/` (Node 22 já disponível no runner).
2. `make check`: acrescentar `portal-test`. Manter `bench-gate` fora do `check` (é medição) e
   dentro de `make ci`.
3. `make ci`: passar a incluir `bench-gate`, `monitoring-rules`, `portal-test` e `stack-config`,
   ou renomear para `ci-rust` e criar `ci` completo — decidir e refletir no comentário do arquivo.
4. `make fmt`: `cargo fmt --all --check`, igual ao CI.
5. `local-rebuild`: derivar o schema esperado de `SCHEMA_VERSION` (ou comparar com o `/health`
   real) em vez de `grep '"schema":49'`; o `netstat` deixa de ser `if` silencioso — ausência da
   ferramenta vira falha explícita ou uso de `ss`/`Get-NetTCPConnection`.
6. `.PHONY` completo; `help` gerado por `grep -E '^[a-z][a-z0-9-]*:.*## '` com formatação.
7. Docs: `docs/rust/10-devops-ci.md` §2 (tabela de paridade), `docs/rust/03-progresso.md`
   (`schema:36` → valor real), `AGENTS.md` §3 se o conjunto de comandos mudar.

## Arquivos

- Alterados: `.github/workflows/rust.yml`, `Makefile`, `docs/rust/10-devops-ci.md`,
  `docs/rust/03-progresso.md`, `docs/rust/02-execucao.md` §1 (se o gate mudar).
- Proibido: afrouxar qualquer verificação para o CI ficar verde.

## Testes / verificação

```bash
make check                 # deve executar os nove + portal-test
make help                  # lista todos os targets com descrição
make local-rebuild         # passa a comparar o schema real
node tools/content/check-issue-status.mjs
```

Saída esperada: `make check` exit 0 com as dez verificações; `local-rebuild` verde com
`SCHEMA_VERSION = 50`; uma divergência injetada de propósito em `index.md` **falha** o CI.

## Skills Relacionadas do freebuff

- .agents/skills/wyd-rust-toolchain — gate, CI e toolchain.
- .agents/skills/wyd-server-ops — prova de fechamento da stack.

## Documentação de referencia

- `docs/rust/10-devops-ci.md` §2 e §4.
- `docs/rust/02-execucao.md` §1 e §5.
- `docs/rust/03-progresso.md` ("Rotina de fechamento").

## Critérios de aceite

- [x] CI executa as mesmas verificações de `make check` (+ `portal-test`).
- [ ] Uma divergência de status/link/skill reprova o CI, provado com injeção temporária.
- [ ] `make local-rebuild` sem número fixo de schema; `null`/`netstat` falha de forma ruidosa.
- [x] `.PHONY` completo e `make help` derivado dos comentários.

## Riscos / rollback

- Risco: gates novos reprovarem PRs antigos por dívida existente → mitigação: rodar os quatro
  scripts localmente antes de ligar no CI e corrigir o que aparecer no mesmo commit.
- Rollback: reverter o workflow; nenhum efeito em runtime.
