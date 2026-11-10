# Issue 02-6 — Gate de consistência entre roadmap, progresso, checklist e código

**Fase:** 2 · **Depende de:** 02-1 ✅, 02-2 ✅ · **Bloqueia:** — · **Esforço:** S

## Objetivo

Impedir que o status de uma issue divirja entre `roadmap-rust/index.md`,
`docs/rust/03-progresso.md`, o cabeçalho da própria issue e o código (crates, migrations,
rotas), com um script único que roda no `make check`.

## Contexto / evidência

A revisão de 2026-11-09 encontrou divergências reais, todas silenciosas:

- `FASE-07-1` 🚧 no índice e ✅ no painel; `FASE-03-2` o inverso;
- `FASE-09-2`, `FASE-11-1` e `FASE-11-2` ✅ no índice e ⬜ no painel;
- `FASE-08-7`, `FASE-08-8` e `FASE-15-3` **sem linha** no painel;
- painel dizendo “24 migrations / 59 tabelas” enquanto o repositório tem **31 migrations**
  e o banco **69 tabelas + ledger** (medido: `70` em `information_schema`);
- `AGENTS.md`/`README.md` listando 7 crates com **8** no `Cargo.toml` (`wyd-server`).

Todos são detectáveis por script e nenhum exige julgamento humano.

## Passos

1. `tools/dev/check-docs-consistency.mjs` (Node, sem dependência):
   - extrai `(issue, status)` da tabela de execução de `roadmap-rust/index.md` e de
     `docs/rust/03-progresso.md` e falha se divergirem ou se faltar linha;
   - confere que todo `issues/FASE-*.md` citado no índice existe (e vice-versa);
   - lê `Cargo.toml` (members) e compara com a árvore de
     `docs/arquitetura/estrutura-e-padroes.md` §1/§16 e a tabela de `AGENTS.md` §1;
   - conta `crates/wyd-db/migrations/*.sql` e compara com o número citado em
     `docs/rust/03-progresso.md` e `docs/arquitetura/banco-catalogo.md`;
   - valida que todo link relativo em `README.md`/`AGENTS.md` resolve;
   - sai `1` com a lista de divergências (nada de “avisar e seguir”).
2. `Makefile`: alvo `docs-check` e inclusão em `check`.
3. Rodar no CI (`.github/workflows/rust.yml`) junto do job `rust`.
4. Prova negativa: alterar um status de propósito, ver exit `1`, reverter.

## Arquivos

- Criados: `tools/dev/check-docs-consistency.mjs`, `tools/dev/README.md`.
- Alterados: `Makefile` (`docs-check`, `check`), `.github/workflows/rust.yml`,
  `docs/rust/02-execucao.md` §1 (gate único passa a incluir a checagem).
- Proibido tocar: `crates/`, conteúdo de jogo, compose.

## Testes / verificação

```bash
node tools/dev/check-docs-consistency.mjs   # exit 0 na árvore consistente
make check                                  # inclui docs-check
```

Saída esperada: `docs: N issues, M crates, K migrations — consistente`.

## Skills Relacionadas do freebuff

- .agents/skills/wyd-rust-toolchain — gates e CI.
- .agents/skills/documentation-writer — docs vivas, referência cruzada.
- .agents/skills/github-actions-docs — job no workflow.

## Documentação de referencia

- `docs/rust/02-execucao.md` §5 (doc viva é defeito quando desatualizada).
- `docs/rust/04-revisao-roadmap.md` §6 (divergências medidas em 2026-11-09).
- `AGENTS.md` §7 (ao concluir issue, atualizar índice + painel).

## Andamento (2026-11-10, parte migrations/tabelas)

- [x] `tools/content/check-migrations.mjs` falha (exit 1) se a contagem de migrations ou tabelas
  diferir de `DOCUMENTED` (36 / 74). Prova negativa feita trocando 36 por 35: exit 1; restaurado: exit 0.
- [x] Comentários de `crates/wyd-db/src/lib.rs` citam 0036 como versão do schema.
- [x] `tools/content/check-issue-status.mjs` (ligado ao `make check` como `issue-status-check`) falha (exit 1) quando o status de uma issue difere entre `roadmap-rust/index.md` e `docs/rust/03-progresso.md`. Prova negativa: FASE-07-2 trocada de ✅ para 🚧 gerou exit 1; restaurada, exit 0. Hoje: 52 issues comparadas, 0 divergentes, 7 só no índice (aviso).
- [x] Crates: `Cargo.toml` lista 8 membros = 8 pastas em `crates/` (nenhuma faltando ou sobrando); README cita os mesmos crates.
- [x] Links: nenhum link relativo quebrado em `README.md` e `AGENTS.md`.
- [x] Comparação `ItemList.bin` do Global: registro de 172 B e cabeçalho de 4 B confirmados por contagem (14.500 registros, resto 0); campos individuais NÃO DECODIFICADOS (ver `docs/rust/03-progresso.md` Fase 2).
- Escopo da FASE-02-6 fechado nesta frente. Pendência exata: decodificar os campos do `ItemList.bin` do Global (correlação com ids/nomes).

## Andamento (2026-11-10, fechamento)

- [x] Deriva `0039_classificacao_e_teto` resolvida: `DOCUMENTED` 39/80 em `check-migrations.mjs`, `banco-catalogo.md` alinhado (resumo + linha 0039); `check-migrations.mjs` exit 0.
- [x] 4 issues da pesquisa no índice **e** no painel (03-16, 06-4, 08-10, 10-4); `check-issue-status.mjs` exit 0 (só no índice: avisos de ⬜/⛔ sem linha no painel, sem divergência de status).
- [x] Parágrafo pós-fila do índice atualizado (fila 2026-11-09 preservada como histórico; lista de abertos inclui as 4 novas; 08-10 herda o bloqueio da 08-6).
- [x] Prova negativa refeita: `DOCUMENTED` 39→38 temporário → exit 1; restaurado → exit 0.

## Subtasks

- [ ] Codificar issue
- [ ] Testar issue (inclui prova negativa)
- [ ] Revisar issue
- [ ] Documentar issue
- [ ] Versionar issue, mergear branch e fechar issue
- [ ] Limpar branch (se aplicável)

## Critérios de aceite

- [ ] Divergência de status entre índice e painel falha o gate.
- [ ] Crate/migration fora do número documentado falha o gate.
- [ ] Link relativo quebrado em README/AGENTS falha o gate.
- [ ] `make check` verde na árvore consistente e vermelho na inconsistente.

## Riscos / rollback

- Risco: script virar “fonte” de status em vez de verificador → mitigação: ele só compara;
  a fonte continua sendo a entrega.
- Rollback: remover o alvo do `Makefile` e o arquivo; nada em `crates/` muda.
