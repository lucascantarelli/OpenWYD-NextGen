# Issue 53 — Feedback dos jogadores (bug/sugestão) no portal

**Fase:** 7 · **Depende de:** 19, 21 · **Esforço:** M · **Estado:** ⏳ aberta

## Objetivo

Dar ao jogador um canal autenticado para reportar bug/sugestão e ao staff uma fila no portal que
classifica, responde e transforma o relato em issue do roadmap — com métrica de triagem.

## Contexto

- Hoje o único canal é conversa informal; nada vira registro. `docs/analysis/qa-ingame.md` §4 tem o
  desenho mínimo.
- O portal já tem sessão, papéis, tabela de auditoria e padrão `adminCall` — o módulo é aderente.
- O cliente não precisa mudar: a entrada é uma página web.

## Passos

1. Migração `player_report` (tipo, status, prioridade, `client_version`, `screenshot_url`, autor,
   timestamps, `handled_by`) + `player_report_audit` (mudança de status com motivo).
2. BFF: `FeedbackService` (criar/lista/detalhe/atualizar status), autorização por papel
   (jogador cria e vê os próprios; staff vê a fila).
3. Público: `/suporte` autenticado (fora de `(admin)`), formulário com validação e upload de print
   (reaproveitar o pipeline de storage dos ícones).
4. Staff: `/feedback` na área admin, filtros (tipo/status/prioridade/versão), detalhe, mudança de
   status com motivo, botão "virar issue" que gera `roadmap/issues/issue-NN.md` a partir do relato.
5. Métricas: tempo médio de triagem, taxa de confirmados, top versões com bug.

## Arquivos

- `internal/migrations/0043_player_report*.sql`, `internal/store/feedback.go`,
  `api/web/v1/web.proto` (+gerados), `webserver/internal/feedback/`, `webserver/internal/grpcsrv/`,
  `portal/src/app/(public)/suporte/`, `portal/src/app/(admin)/feedback/`,
  `portal/src/server/{feedback.ts,actions/feedback.ts}`, `portal/src/lib/feedback.ts` (+teste),
  `portal/src/components/nav.ts`, docs de integração.

## Verificação

```bash
go test ./webserver/internal/feedback/ ./internal/store/ 2>&1 | tail -5
cd portal && npx tsc --noEmit && npx vitest run && npx next build
# ponta a ponta: jogador cria, staff lista, muda status, gera issue
```

## Critérios de aceite

- [ ] Jogador autenticado cria relato e vê o próprio histórico; `player` não acessa a fila
- [ ] Staff classifica e muda status com motivo auditado
- [ ] "Virar issue" gera arquivo com texto do jogador e link do print
- [ ] Nada de senha/PIN/IP guardado; print passa por moderação
- [ ] Métricas de triagem visíveis no painel

## Riscos

- **Spam/abuso**: rate limit por conta, limite de abertos, bloqueio de conta refletido.
- **Print com dado sensível**: moderação antes de qualquer exposição pública.
