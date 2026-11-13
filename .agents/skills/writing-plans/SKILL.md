---
name: writing-plans
description: Estrutura planos de execução de issues multi-etapa do WYD (decomposição em subtasks testáveis, ordem por dependência, arquivos e gates). Use antes de começar uma issue M/L ou quando o aceite ainda não estiver decomposto.
---

# Plano de execução de uma issue

Ciclo, branch, DoD e fechamento: `docs/rust/02-execucao.md`. Esta skill só define como
decompor o plano **dentro da própria issue**; não crie plano paralelo nem doc externo.

## Regras de decomposição

1. **Plano na issue.** Atualize `Subtasks`/`Passos` em `roadmap-rust/issues/`; nada de
   plano paralelo ou caminho fora da árvore.
2. **Uma subtask = um entregável testável.** Cabe em uma sessão, toca poucos arquivos e
   termina com gate verificável; se não termina sozinho, divida.
3. **Ordem por desbloqueio.** Primeiro contrato/fixture/migration/harness que remove
   bloqueio ou produz prova; depois implementação; por último docs/fechamento. Dependência
   externa leva o ID da issue dona.
4. **Arquivos e prova explícitos.** Liste criar/alterar por caminho real e o comando de
   verificação (`cargo test -p …`, `make check`, `make portal-test`). Sem “testar depois”.
5. **Sem placeholder.** “Implementar/ajustar/ver depois” não é subtask: diga arquivo,
   comportamento e prova. O aceite é a soma das subtasks.

## Formato mínimo

```markdown
## Subtasks
- [ ] <entregável 1> — arquivos: `...`; prova: `<comando>`; depende de: —
- [ ] <entregável 2> — arquivos: `...`; prova: `<comando>`; depende de: <ID>
```
