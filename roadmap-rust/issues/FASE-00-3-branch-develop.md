# Issue 00-3 — Branch `develop` como principal de desenvolvimento

**Fase:** 0 · **Depende de:** — · **Bloqueia:** FASE-01 a FASE-15 · **Esforço:** S

## Objetivo

Instituir `develop` como branch principal de desenvolvimento, com fluxo
issue→branch→PR→`develop` documentado e exigido.

## Contexto / evidência

`start-rust.md`: "a branch develop deve ser a branch principal de
desenvolvimento". Hoje o trabalho está em staging não committado; sem política
de branch, execuções futuras divergem.

## Passos

1. Verificar branches existentes: `git branch -a`.
2. Criar/atualizar `develop` a partir do estado atual e documentar o fluxo
   (uma issue = uma branch `fase/NN-descricao` → PR → `develop`).
3. Registrar a política em `docs/rust/02-execucao.md` (criada na FASE-02-1) ou
   nesta issue até lá.
4. Proteger `develop` contra push direto (configuração do remoto, pelo dono).

## Arquivos

- Criados: nenhum (política + branch).
- Alterados: docs de processo (`docs/rust/02-execucao.md` na FASE-02-1).
- Proibido tocar: código; histórico existente (sem rebase/force-push).

## Testes / verificação

```bash
git branch -a
```

Saída esperada: `develop` existe como branch de desenvolvimento; fluxo
documentado.

## Skills Relacionadas do freebuff

- .agents/skills/writing-plans - documentar o fluxo.
- .agents/skills/wyd-rust-toolchain - operações git seguras.

## Documentação de referencia

- start-rust.md - branch develop como principal.
- roadmap-rust/issues/FASE-02.md - ritmo de execução.

## Subtasks

- [ ] Codificar issue
- [ ] Testar issue
- [ ] Revisar issue
- [ ] Documentar issue
- [ ] Versionar issue, mergear branch e fechar issue
- [ ] Limpar branch (se aplicável)

## Critérios de aceite

- [ ] `develop` existe; fluxo issue→branch→PR→develop documentado.
- [ ] Sem force-push no histórico.

## Riscos / rollback

- Risco: trabalho em branch errada → mitigação: agente sempre confirma a branch
  (`git branch --show-current`) antes de committar.
- Rollback: política é documentação; `git checkout -- docs/`.
