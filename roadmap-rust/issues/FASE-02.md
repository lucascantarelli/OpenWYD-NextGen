# Issue 02 — Execução controlada e evolução sequencial

**Fase:** 2 · **Depende de:** FASE-01 · **Bloqueia:** FASE-03 a FASE-15 (ritmo) · **Esforço:** S

## Objetivo

Fixar o ritmo de execução (uma issue por vez, ciclo análise→design→código→teste→docs)
com revisão consolidada a cada 10 issues, documentado e exigível pelo agente.

## Contexto / evidência

Sem ritmo fixo, refatorações parciais convivem com código legado e o defeito nasce da
convivência (precedente real: mercadores duplicados de Armia quando duas fontes
viveram juntas — ver `AGENTS.md` §8). Ordem de dependências: `roadmap-rust/index.md`
(contratos/rede → engine → persistência → APIs/portal) e `pre-release-rust.md` FASE 2.

## Passos

1. Registrar o ciclo obrigatório em `docs/rust/02-execucao.md` (criar): análise,
   design, código Rust, teste, docs, atualização da issue.
2. Definir a regra de revisão a cada 10 issues (regressão, performance, integração).
3. Registrar a ordem de priorização (dependências → criticidade → risco → desbloqueio).
4. Rodar o gate: `make check`.

## Arquivos

- Criados: `docs/rust/02-execucao.md`.
- Alterados: `roadmap-rust/issues/FASE-02.md` (este).
- Proibido tocar: `crates/*/src/` (fase de processo, sem código de engine),
  `.deprecated-go/`, `client/`.

## Testes / verificação

```bash
make check
test -f docs/rust/02-execucao.md && echo RITMO-OK
```

Saída esperada: gates verdes e `RITMO-OK`.

## Skills Relacionadas do freebuff

- .agents/skills/writing-plans - desenhar o ciclo e os critérios de revisão.
- .agents/skills/wyd-rust-toolchain - executar `make check`.

## Documentação de referencia

- docs/rust/00-visao-estrutura.md - estrutura e toolchain.
- pre-release-rust.md (FASE 2) - execução unitária e revisões.
- roadmap-rust/index.md - ordem estratégica.

## Subtasks

- [ ] Codificar issue
- [ ] Testar issue
- [ ] Revisar issue
- [ ] Documentar issue
- [ ] Versionar issue, mergear branch e fechar issue
- [ ] Limpar branch (se aplicável)

## Critérios de aceite

- [ ] `docs/rust/02-execucao.md` existe com ciclo, revisão e priorização.
- [ ] `make check` verde; evidência anexada.

## Riscos / rollback

- Risco: processo ignorado na pressa → mitigação: `make check` como gate exigido
  em todo Critério de aceite das issues seguintes.
- Rollback: `git checkout -- docs/rust/02-execucao.md`.
