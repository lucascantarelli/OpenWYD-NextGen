# Issue 13-1 — Rito ADR + template fixo

**Fase:** 13 · **Depende de:** 02-1 · **Bloqueia:** 13-2 · **Esforço:** S

## Objetivo

Instituir o rito (toda mudança estrutural exige ADR prévia) com template
`0000` e numeração sem lacunas.

## Contexto / evidência

`pre-release-rust.md` FASE 13: ADR antes de refatorações de grande escala.
Canônico: `docs/arquitetura/adr/` (com `0001-hot-path-monolitico.md` proposto).

## Passos

1. Criar `docs/arquitetura/adr/0000-template.md` (contexto → decisão →
   consequências → alternativas).
2. Definir estados (proposta/aceita/substituída) e regra de numeração.
3. Registrar o rito em `docs/rust/02-execucao.md` (nenhuma issue estrutural
   fecha sem ADR datada de antes da implementação).
4. `make check` + `ls docs/arquitetura/adr/`.

## Arquivos

- Criados: `docs/arquitetura/adr/0000-template.md`.
- Alterados: `docs/rust/02-execucao.md` (rito), esta issue.
- Proibido tocar: `crates/*/src/`.

## Testes / verificação

```bash
make check && ls docs/arquitetura/adr/
```

Saída esperada: gates verdes; `0000-template.md` + `0001` sem lacuna.

## Skills Relacionadas do freebuff

- .agents/skills/software-architecture - rito e consequências.
- .agents/skills/writing-plans - template.

## Documentação de referencia

- pre-release-rust.md (FASE 13).
- docs/arquitetura/adr/0001-hot-path-monolitico.md - exemplo a conformar.

## Subtasks

- [ ] Codificar issue
- [ ] Testar issue
- [ ] Revisar issue
- [ ] Documentar issue
- [ ] Versionar issue, mergear branch e fechar issue
- [ ] Limpar branch (se aplicável)

## Critérios de aceite

- [ ] Template publicado; rito registrado no processo; numeração íntegra.

## Riscos / rollback

- Risco: ADR escrita depois do código → mitigação: aceite das estruturais
  exige ADR prévia.
- Rollback: `git checkout -- docs/arquitetura/adr/ docs/rust/02-execucao.md`.
