# REVIEW-19 — Revisão permanente: IA, Skills, agentes e documentação

**Fase:** revisão contínua · **Depende de:** — · **Bloqueia:** — · **Esforço:** contínuo · **Estado:** 🔁 viva

## Objetivo

Ponto de controle dos arquivos que ensinam os agentes a trabalhar neste repositório (`AGENTS.md`,
`CLAUDE.md`, `.agents/skills/`, `development-guidelines/`) e da documentação que eles leem, para que
instrução e realidade não divirjam.

## Escopo

- `AGENTS.md`: invariantes, caminho felizes, árvore de conteúdo, verificação.
- Skills: `wyd-server-ops`, `wyd-database`, `wyd-client-connect`, `wyd-game-admin`, `wyd-economy`,
  `wyd-server-dev`, `wyd-testing-bots`, `wyd-web-admin-ui` + as de direção visual/UX.
- `development-guidelines/processo.md` como Definition of Done única.
- `docs/`: mkdocs, links, páginas de sistema, ADRs, `doc_legacy/`.
- Duplicidade de regra entre documentos e instrução conflitante.

## O que registrar a cada rodada

| Item | Como medir |
|---|---|
| Links de documentação | `bash scripts/check-docs-links.sh` |
| mkdocs estrito | `mkdocs build --strict` |
| Skills e tamanho | `ls .agents/skills/*/SKILL.md` e nº de linhas por skill |
| Regra duplicada | comparação manual `AGENTS.md` × `CLAUDE.md` × `processo.md` × skills |
| Fato envelhecido | toda afirmação de estado ("toolchain ausente", "sem CI") reconferida |
| Achados novos | uma linha por achado com arquivo:linha |

## Critérios de aceite (por rodada)

- [ ] Nenhuma skill ensina comando que não roda
- [ ] Nenhum documento afirma estado que a medição contradiz
- [ ] `processo.md` continua a única Definition of Done
- [ ] Toda skill cita arquivo existente (o teste de guias já reprova `.go` inexistente)

## Histórico

| Data | Achados | Destino |
|---|---|---|
| 06/11/2026 | `roadmap/index.md` afirma "toolchain Go no host ❌ ausente"; medido `go1.27.3 windows/amd64` | corrigido nesta auditoria |
| 06/11/2026 | `docs/sistemas/` cobre 12 sistemas, mas não existe página de matemática/balanceamento | MATH-01 |
| 06/11/2026 | skill `wyd-testing-bots` §4 descreve bot "próprio" sem apontar para o pacote existente | atualizada nesta auditoria |
