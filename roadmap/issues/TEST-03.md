# TEST-03 — Playwright no portal: 5 fluxos reais, incluindo 403 por papel

**Fase:** revisão · **Depende de:** TEST-02 · **Esforço:** L · **Estado:** ⏳ proposta

## Objetivo

O portal ter um e2e de navegador que prova login, navegação e autorização — hoje o único teste é
`vitest` de unidade, e nenhuma regressão de página (RSC, cookie, redirect) é pega antes do usuário.

## Contexto/evidência (05/11/2026)

```bash
ls portal/e2e 2>/dev/null | wc -l     # 0  (não existe)
ls portal/playwright.config.ts        # não existe
```

- `portal/package.json` roda `vitest` (unidade) + `tsc --noEmit`; não há job de e2e no CI.
- O portal tem 30 `page.tsx` e rotas de API (`/api/auth/login`, `/api/search/items`, `/api/mapa/download`).
- Evidência manual desta sessão mostra que o caminho funciona (login `test/test123` → 200
  `role=admin`; `/mobs`, `/npcs`, `/mapa`, `/personagens`, `/drops` → 200 com dados), mas nada disso é
  automatizado: uma quebra de sessão ou de middleware só apareceria em teste humano.

## Passos

1. `portal/playwright.config.ts` com `webServer` apontando para o compose já no ar (ou
   `npm run start` contra `out/`), `baseURL=http://127.0.0.1:3000`, um projeto Chromium.
2. Cinco fluxos mínimos e estáveis:
   - login com `test/test123` → painel visível, cookie `httpOnly` presente;
   - `/itens` lista com item real (nome, não índice cru);
   - `/npcs` abre um NPC e a loja mostra itens;
   - `/mapa` responde com histograma;
   - **403 por papel**: sessão de jogador (`player`) em rota admin → `/sem-permissao`, e anônimo →
     redirect para login.
3. Fixtures: uma conta `player` semeada pelo compose (ou `storageState` gerado no setup), sem
   credencial em código.
4. CI: job `e2e` noturno ou por label `e2e`, com artefato de trace/screenshot na falha.
5. Documentar em `docs/development-guidelines/testing.md` como rodar local.

## Arquivos

- `portal/playwright.config.ts`, `portal/e2e/**` (novo)
- `portal/package.json` (`test:e2e`), `.github/workflows/e2e.yml` (novo)
- `docs/development-guidelines/testing.md`

## Testes/verificação

```bash
cd portal && npx playwright test --project=chromium
# esperado: 5 passed; trace em test-results/ quando falhar
```

## Critérios de aceite

- [ ] 5 fluxos verdes contra a stack local subida pelo compose.
- [ ] Um fluxo prova 403/redirect por papel (não só "página abre").
- [ ] Nenhuma credencial literal no código do teste (setup via env/fixture).
- [ ] Job de CI publica trace na falha.

## Riscos/rollback

Risco: e2e flaky desligado na primeira falha. Mitigação: poucos fluxos, `webServer` controlado pelo
config, retry só para erro de rede, e asserção por texto estável (não por classe CSS). Rollback: o job
entra por label até estabilizar.
