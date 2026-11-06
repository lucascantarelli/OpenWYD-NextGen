# REVIEW-01 — Triagem da fila in-game: fechar o que só falta teste real

> **Análise profunda 06/11/2026:** números medidos, correções das afirmações envelhecidas e
> referências externas em [`REVISAO-PROFUNDA-2026-11-06.md`](../REVISAO-PROFUNDA-2026-11-06.md) §4.

**Fase:** revisão · **Depende de:** 49 · **Esforço:** M · **Estado:** ⏳ proposta

## Objetivo

Reduzir o passivo de issues 🟡 transformando cada uma em **fechada com evidência** ou **reaberta com
bug reproduzido**, começando pelas que dependem apenas de um teste no cliente — é o caminho mais
curto para o roadmap voltar a refletir a realidade.

## Contexto/evidência

- O dossiê (`docs/analysis/qa-ingame.md`) lista a ordem de teste e o sistema de feedback; 11 issues
  estão 🟡 "falta evidência no cliente".
- `AGENTS.md` §5: login headless (`TestE2ESmokeLogin`) **sempre** antes de culpar código de jogo — é
  o que separa bug de servidor de bug de cliente/configuração.
- `docs/migration/ingame-bugs.md` já registra B12/B16 com golden case antes da correção; o padrão
  existe, falta aplicá-lo à fila inteira.
- Hoje não há coluna de "evidência" por issue: algumas 🟡 têm teste automatizado verde e continuam
  abertas por inércia.

## Passos

1. Gerar a lista real: `grep -l "🟡" roadmap/issues/issue-*.md` e cruzar com a tabela do
   `roadmap/index.md`.
2. Para cada issue 🟡, escrever no próprio arquivo um bloco **Evidência** com: o que foi testado,
   comando/roteiro, saída, e o que falta (nada / cliente / humano).
3. Rodar o que é automatizável: `TestE2ESmokeLogin` + os `e2e` de eventos
   (`scripts/e2e-worldevents.sh`), registrando o log.
4. O que depender do cliente vira item numerado no `qa-ingame.md` com passos reproduzíveis (personagem,
   mapa, nível, item) — não "testar quests".
5. Issue cujo comportamento está coberto por teste verde: fechar citando o teste e a saída.
6. Issue com defeito confirmado: abrir issue de bug **com o teste que falha primeiro** (ou golden file),
   e só então corrigir.

## Arquivos

- `roadmap/issues/issue-*.md` (bloco de evidência)
- `docs/analysis/qa-ingame.md` (ordem e passos)
- `docs/migration/ingame-bugs.md` (bug confirmado)
- `roadmap/index.md` (status)

## Testes/verificação

```bash
MSYS_NO_PATHCONV=1 docker run --rm --network openwyd_default -v "D:/dolorestec/wyd2/OpenWYD:/src" \
  -e W2PP_E2E_ADDR=tmserver:8281 -e W2PP_E2E_ACCOUNT=test -e W2PP_E2E_PASSWORD=test123 \
  -e W2PP_E2E_VERSION=12000 golang:1.26-alpine \
  sh -c "cd /src && go test -tags=e2e -run TestE2ESmokeLogin ./tmserver/internal/world/ -v"
```

## Critérios de aceite

- [ ] Zero issues 🟡 sem bloco **Evidência** preenchido.
- [ ] Cada issue fechada cita comando/teste + saída.
- [ ] Cada bug confirmado tem teste (ou golden case) que falha antes da correção.
- [ ] `roadmap/index.md` e `SESSION-PRIMER.md` atualizados na mesma entrega.

## Riscos/rollback

Risco: fechar issue por teste superficial (login passa, gameplay não). Mitigação: o bloco de evidência
exige o cenário concreto (personagem/mapa/item), não só o nome do teste. Rollback: reabrir a issue é
barato; o custo real é declarar pronto o que não está — por isso o critério é evidência, não opinião.

## Issues que este review abre

- `BUG-*` para cada defeito confirmado (com teste que falha primeiro)
- `QA-*` para roteiro que precisa do cliente e ainda não existe
