# REVIEW-10 — Limpeza final, arquivos de IA e padrões de processo

**Fase:** revisão · **Depende de:** REVIEW-02..09 · **Esforço:** M · **Estado:** ⏳ proposta

## Objetivo

Fechar a faxina: uma única fonte para skills/instruções de IA, `scripts/` só com o que roda, nenhum
arquivo órfão, e as convenções de trabalho (DoD, ADR, commit, revisão) escritas onde o agente e a
pessoa vão ler.

## Contexto/evidência (05/11/2026)

- **Skills duplicadas**: `.agents/skills/` tem 22 diretórios (incluindo genéricas como
  `documentation-writer`, `frontend-design`, `ui-ux-practices`, `writing-plans`,
  `github-actions-docs`, `skill-creator`, `finishing-a-development-branch` e 6 `openspec-*`) e existe
  também `.claude/` com `commands/`, `settings.json`, `skills/` — dois espelhos que divergem.
- `openspec/` continua no repositório (`changes/`, `config.yaml`, `specs/`) sem referência no código.
- `project-analizer/` **já foi removido** (commit `f39aed3`); `docs/agents/` (3 relatórios do legado)
  segue como arquivo histórico.
- `scripts/`: 9 arquivos (`db-audit.sql`, `db-backup.sh`, `db-baseline.sh`, `e2e-worldevents.sh`,
  `gen-certs.sh`, `preflight.sh`, `run-local.sh`, `upload-item-icons-storage-manager.sh`) + o
  `disk-report.ps1` criado na limpeza de disco.
- `_archived/` das migrations foi removido; o histórico do squash vive no git.
- Convenções existem espalhadas (`AGENTS.md` §8, `qualidade-e-processo.md` §4) mas não há um
  documento único de Definition of Done.

## Analisar tudo
- Notei que temo diversos arquivos de IA espalhados pelo projeto, devemos analizar tudo relacionado a IA, e ver se podemos melhorar, otimizar, atualizar, migrar para novas tecnologias e frameworks modernos, ou até mesmo remover futuramente, caso não seja mais necessário.
- Devemos analizar todas as skills de IA, sugerir melhorias, otimizações, atualizações, migrações para novas tecnologias e frameworks modernos, ou até mesmo remoção futuramente, caso não seja mais necessário, criar novas skills de IA, caso necessário, para melhorar a experiência do usuário e otimizar o desempenho do sistema.
- Procurar por informações duplicadas, inconsistentes ou desatualizadas relacionadas a IA, e sugerir melhorias, otimizações, atualizações, migrações para novas tecnologias e frameworks modernos, ou até mesmo remoção futuramente, caso não seja mais necessário.
- vamos analizar completamente todos os arquivos relacionados a IA, vamos mover tudo para uma pasta em rodamap, chamada ia_legacy, para que possamos ter um código mais limpo, organizado e de fácil manutenção, além de otimizar o desempenho e garantir maior segurança na plataforma.
- recriar todos os arquivos de IA, atualizados, otimizados, migrados para novas tecnologias e frameworks modernos, para que possamos ter um código mais limpo, organizado e de fácil manutenção, além de otimizar o desempenho e garantir maior segurança na plataforma, pesquise por arquivos relacionados soltos dentro de todos os diretórios do projeto, analise se ainda precisam ser usados, existe um plugin antigo "project-analizer" que analisa o projeto, remova, não vamos precisar mais dele.
- Mapear o que pode ser removido, migrado, atualizado, otimizado, refatorado, melhorado, etc. para que possamos ter um código mais limpo, organizado e de fácil manutenção, além de otimizar o desempenho e garantir maior segurança na plataforma.
- analizar arquivos que podem ser reorganizador, renomeados, atualizados, migrados, otimizados, refatorados, melhorados, etc. para que possamos ter um código mais limpo, organizado e de fácil manutenção, além de otimizar o desempenho e garantir maior segurança na plataforma.
- Remover arquivos que não estão sendo usados, duplicados, obsoletos, etc. para que possamos ter um código mais limpo, organizado e de fácil manutenção, além de otimizar o desempenho e garantir maior segurança na plataforma.
- analisar a pasta scripts, deixar apenas o que realmente será usado, remover o que já foi migrado, está sem uso.
- Todos os artefatos e achados da análise, implementações, correções, limpeza, atualizações, melhorias, otimizações, refatorações, migrações, etc. devem ser documentados e organizados em um roadmap detalhado, incluindo issues detalhadas para cada tarefa a ser realizada, para que possamos ter um controle mais eficiente sobre o andamento do projeto e garantir que todas as funcionalidades sejam implementadas corretamente.
- atualmente temos a pasta roadmap e roadmap/issues, onde planejamos todas as issues de forma detalhada, mas precisamos melhorar a organização, detalhamento e priorização das issues, para que possamos ter um controle mais eficiente sobre o andamento do projeto e garantir que todas as funcionalidades sejam implementadas corretamente.
- Todas as issues devem ser detalhadas com códigos, arquivos, testes, documentos, referencias as skills de IA para uso do agente, e demais informações relevantes para a implementação correta da funcionalidade, para que possamos ter um controle mais eficiente sobre o andamento do projeto e garantir que todas as funcionalidades sejam implementadas corretamente.
- Mapear todos os achados da análise, alinhado com as issues atuais, vamos reorganizar a ordem de execução das issues, existem issues pendentes de testes no cli, vamos dá prioridade a exxas issues que faltam testes reais, para já finalizarmos algumas de forma rápida.

## Passos

1. **Uma fonte de IA**: manter `.agents/skills` como canônico e gerar `.claude/skills` (symlink ou
   script de sincronização) — ou remover `.claude/skills` e deixar só `settings.json`/`commands`.
2. **Deduplicar skills genéricas**: uma por assunto; as do projeto (`wyd-*`) ficam.
3. **Decidir openspec**: adotar de verdade (com `config.yaml` preenchido) ou arquivar em
   `docs/archive/openspec/` e remover as 6 skills. Sem meio-termo.
4. **`ia_legacy`**: mover para `roadmap/ia_legacy/` o que for histórico (ex.: `docs/agents/`,
   relatórios antigos) com README dizendo o que substituiu.
5. **`scripts/`**: para cada script, provar uso (`grep -rn "scripts/<nome>" .`) ou remover; alinhar
   com os alvos do `Makefile` (21 alvos, alguns podem ser atalho de script).
6. **Definition of Done única** em `docs/development-guidelines/processo.md`: código + teste + doc +
   evidência + rollback, revisão <400 linhas, issue = um resultado verificável, ADR para decisão cara.
7. **Regra de comentário**: nada de número de issue/roadmap no código (ver REVIEW-02) — o histórico
   fica no git e no roadmap.

## Arquivos

- `.agents/skills/**`, `.claude/**`, `openspec/**` (ou `docs/archive/openspec/**`)
- `roadmap/ia_legacy/**` (novo)
- `scripts/**`
- `docs/development-guidelines/processo.md` (novo)
- `AGENTS.md` §7/§8

## Testes/verificação

```bash
ls .agents/skills | wc -l && ls .claude/skills 2>/dev/null | wc -l   # esperado: igual ou ausente
for s in scripts/*; do grep -rq "${s#scripts/}" --include=*.md --include=*.sh --include=Makefile . || echo "orfa: $s"; done
```

## Critérios de aceite

- [ ] Uma única cópia de cada skill; `.claude` sem espelho divergente.
- [ ] `openspec` adotado ou arquivado (decisão registrada em ADR).
- [ ] `scripts/` sem arquivo órfão; cada um citado em doc, Makefile ou CI.
- [ ] `docs/development-guidelines/processo.md` com a DoD e referência no `AGENTS.md`.
- [ ] `roadmap/ia_legacy/` com README de substituição.

## Riscos/rollback

Risco: remover skill que o agente usa na prática. Mitigação: antes de remover, `grep` em
`AGENTS.md`, `CLAUDE.md` e nas issues; o que for usado continua. Rollback: movimentação em commits
por grupo (skills, openspec, scripts).

## Issues que este review abre

- `CLEAN-01` consolidar `.agents` × `.claude`
- `CLEAN-02` decidir openspec (adotar ou arquivar)
- `CLEAN-03` `ia_legacy` + `docs/agents` histórico
- `CLEAN-04` auditoria de `scripts/` e alvos do `Makefile`
- `CLEAN-05` Definition of Done única
