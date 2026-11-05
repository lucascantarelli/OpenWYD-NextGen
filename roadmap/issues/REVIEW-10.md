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
