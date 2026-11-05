# openspec arquivado (05/11/2026)

Este diretório guarda o fluxo `openspec/` que existia na raiz do repositório: 5 mudanças pendentes
(`issue-309`, `issue-316`, `issue-321`, `issue-327`, `issue-344`), as specs e o histórico em
`changes/archive/`.

**Por que foi arquivado:** nenhum arquivo do repositório referenciava o openspec (zero ocorrências em
`.go`), nenhuma das 5 mudanças avançou desde 2026-09 e o planejamento real vive em
`roadmap/index.md` + `roadmap/issues/`. Manter duas fontes de plano era o padrão que o `AGENTS.md` §8
proíbe: migração sem remoção deixa o defeito nascer da convivência.

**O que substituiu cada parte:**

| Antes (openspec) | Agora |
|---|---|
| `changes/<issue-NNN>/proposal.md` | a issue correspondente em `roadmap/issues/` |
| `changes/<issue-NNN>/design.md` | seção "Contexto/evidência" + "Riscos/rollback" da issue |
| `changes/<issue-NNN>/tasks.md` | checklist "Critérios de aceite" da issue |
| `specs/` (requisitos por área) | `docs/migration/` (protocolo, formatos, regras) e os testes que os fixam |
| skills `openspec-*` | removidas; o fluxo é proposta → issue no roadmap |

Para consultar o histórico: `git log -- docs/archive/openspec`.
