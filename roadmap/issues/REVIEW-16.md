# REVIEW-16 — Revisão permanente: portal web e experiência

**Fase:** revisão contínua · **Depende de:** — · **Bloqueia:** — · **Esforço:** contínuo · **Estado:** 🔁 viva

## Objetivo

Ponto de controle do portal Next.js como BFF: separação público × administrativo, autorização,
auditoria, acessibilidade, identidade visual e prova automatizada dos fluxos críticos.

## Escopo

- Arquitetura: Server/Client Components, rotas, layouts, cache, `no-store` no que é pessoal.
- BFF: tudo passa por `adminCall`; nenhum page/action chama gRPC admin direto.
- Portal do jogador: cadastro/login, perfil, personagens, notícias, rankings, suporte, guias.
- Portal administrativo: contas, personagens, conteúdo, itens, eventos, logs, métricas, configuração.
- Identidade visual e acessibilidade: tokens, contraste medido, alvo de toque, foco, dark/light.
- Assets: nenhuma imagem aleatória apresentada como recurso oficial do jogo.
- Testes: unidade do BFF, autorização por rota e e2e de navegador.

## O que registrar a cada rodada

| Item | Como medir |
|---|---|
| Rotas cobertas pelo middleware | `npx vitest run middleware.test.ts` |
| Testes do portal | `find portal -name '*.test.ts*' -not -path '*/node_modules/*' \| wc -l` |
| e2e de navegador | presença de Playwright e nº de fluxos |
| Typecheck/build | `make portal-test` e `make portal-build` |
| Acessibilidade | contraste medido, navegação por teclado nas telas novas |
| Achados novos | uma linha por achado com rota e papel |

## Critérios de aceite (por rodada)

- [ ] `make portal-test` e `make portal-build` verdes
- [ ] Toda rota nova entra no `matcher` **e** no teste de autorização
- [ ] Toda ação admin com auditoria e `moderator_id` da sessão
- [ ] Nenhum dado pessoal em resposta cacheável

## Histórico

| Data | Testes do portal | Achados | Destino |
|---|---|---|---|
| 06/11/2026 | 15 arquivos de teste, 0 Playwright | e2e de navegador inexistente | TEST-03 (aberta) |
| 06/11/2026 | — | loja/donate existe no admin, mas portal do jogador (cadastro, perfil, ranking) não tem issue única | REVIEW-08 (aberta), issue 53 (feedback) |
