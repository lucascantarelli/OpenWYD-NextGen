# REVIEW-03 — Testes: arquitetura, inflação, skips e e2e de navegador

**Fase:** revisão · **Depende de:** REVIEW-01 · **Esforço:** L · **Estado:** ⏳ proposta

## Objetivo

Uma suíte em que "verde" significa algo: unidade/integração/e2e separados por pasta e por comando,
sem testes mortos, sem `t.Skip` escondendo regressão, com fixtures compartilhadas e um e2e real de
navegador para o portal.

## Contexto/evidência (05/11/2026)

| Achado | Número | Leitura |
|---|---:|---|
| `_test.go` | 262 de 569 `.go` | volume bom, concentrado em `handler`/`world` |
| `t.Skip` | 43 | parte legítima (conteúdo proprietário ausente), parte pode mascarar regressão |
| Integração atrás de build tag `integration` | fora do CI | store/gRPC sem rede de segurança no gate |
| Portal | só `vitest` (`portal/package.json`) | sem Playwright/Cypress; nenhum job e2e no `ci.yml` |
| Páginas do portal | 30 `page.tsx` | cobertura de teste rasa nas páginas |

O dossiê (`docs/analysis/qualidade-e-processo.md` §3) já define a pirâmide alvo; o que falta é o
mecanismo: onde cada tipo de teste mora, como roda e o que reprova o PR.

## Passos

1. **Inventário**: `grep -rn "t.Skip" --include=*_test.go`, classificar cada um em *conteúdo ausente*,
   *feature não implementada* (deve ser `TODO` + issue, não skip) ou *morto* (apagar).
2. **Layout por tipo**: `tmserver/internal/handler/*_test.go` continua unidade; integração vai para
   `*_integration_test.go` com tag `integration`; e2e em `tests/e2e/`. Documentar em
   `docs/development-guidelines/testing.md`.
3. **Fixtures centralizadas**: um pacote `internal/testutil` (world mínimo, conta, personagem, item)
   usado por handler/world; hoje cada teste monta o seu (não há duplicação de helper de mundo — só 2
   definições —, mas há setup repetido por cenário).
4. **Playwright no portal**: `portal/e2e/` com `playwright.config.ts`, `webServer` apontando para o
   compose, e 5 fluxos mínimos: login, dashboard, lista de itens, edição de NPC, 403 por papel.
5. **CI**: job `integration` com `services: postgres`, job `e2e` noturno (ou por label), e publicar
   cobertura com piso por pacote crítico (`world`, `handler`, `store`, `protocol`).
6. **Remover obsoletos**: testes que só verificam legado removido (CSV/NPCGener) — os que sobraram
   devem testar o caminho do banco.

## Arquivos

- `docs/development-guidelines/testing.md` (novo)
- `internal/testutil/` (novo)
- `portal/e2e/**`, `portal/playwright.config.ts`, `portal/package.json`
- `.github/workflows/ci.yml` (+ `e2e.yml`)
- `tmserver/**/*_integration_test.go`

## Testes/verificação

```bash
go test ./...                                  # unidade
go test -tags=integration ./internal/... ./dbserver/...   # com Postgres
cd portal && npm test && npx playwright test   # e2e de navegador
```

## Critérios de aceite

- [ ] Nenhum `t.Skip` sem justificativa no próprio arquivo e sem issue associada quando for feature.
- [ ] Integração roda no CI com Postgres de serviço e migração em base vazia.
- [ ] Playwright com ≥5 fluxos reais verdes, incluindo um 403 por papel.
- [ ] Cobertura publicada com piso por pacote crítico.
- [ ] Testes que dependiam de `ItemList.csv`/`NPCGener.txt` migrados para o caminho do banco ou
      removidos com justificativa.

## Riscos/rollback

Risco: e2e de navegador virar flaky e ser desligado na primeira falha — o que devolve o problema.
Mitigação: poucos fluxos, `webServer` controlado pelo config, e retry só para falha de rede. Rollback:
o job e2e é opcional por label até estabilizar.

## Issues que este review abre

- `TEST-01` triagem dos 43 `t.Skip`
- `TEST-02` `internal/testutil` + migração dos setups
- `TEST-03` Playwright no portal (5 fluxos)
- `TEST-04` job `integration` com Postgres e migração em base vazia
- `TEST-05` cobertura com piso por pacote
