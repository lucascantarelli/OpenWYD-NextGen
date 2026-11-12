---
name: writing-plans
description: Estrutura planos de execução de issues multi-etapa do WYD (decomposição em subtasks testáveis, ordem por dependência, arquivos e gates). Use antes de começar uma issue M/L ou quando o aceite ainda não estiver decomposto.
---

# Plano de execução de uma issue

Use antes de tocar código em issue **M/L** ou quando o aceite tiver vários entregáveis.
O ciclo normativo e o Definition of Done estão em `docs/rust/02-execucao.md`; esta skill
só define como decompor e registrar o plano **dentro da própria issue**.

## Regras

1. **Plano na issue.** Atualize a seção `Subtasks`/`Passos` do arquivo em
   `roadmap-rust/issues/`; não crie `docs/superpowers/`, planos paralelos ou caminhos fora da
   árvore do repositório.
2. **Uma subtask = um entregável testável.** Cada item deve caber em uma sessão, tocar poucos
   arquivos e terminar com um gate verificável. Se não termina sozinho, divida.
3. **Ordem por desbloqueio.** Primeiro o que remove bloqueio ou produz prova (contrato, fixture,
   migration, harness); depois a implementação; por último docs e fechamento. Marque dependências
   externas com o ID da issue dona.
4. **Arquivos explícitos.** Liste criar/alterar por caminho real e confira a árvore de
   `docs/arquitetura/estrutura-e-padroes.md`; arquivo novo fora dela exige ADR.
5. **Gate por subtask.** Defina o comando de verificação (`cargo test -p …`, `make check`,
   `make portal-test`, `make stack-config`) e o resultado esperado. Não use “testar depois”.
6. **Sem placeholders.** Nada de “implementar”, “ajustar” ou “ver depois” sem dizer o arquivo,
   o comportamento e a prova. O aceite da issue é a soma das subtasks, não uma promessa.

## Formato mínimo

```markdown
## Subtasks
- [ ] <entregável 1> — arquivos: `...`; prova: `<comando>`; depende de: —
- [ ] <entregável 2> — arquivos: `...`; prova: `<comando>`; depende de: <ID>
```

Ao concluir, marque os checkboxes, anexe a evidência e só então remova a issue para o
`roadmap-rust/changelog.md`.
