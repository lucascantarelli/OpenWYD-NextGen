# REVIEW-08 — Portal: público × administrativo, loja, fóruns e identidade visual

**Fase:** revisão · **Depende de:** 47 · **Esforço:** L · **Estado:** ⏳ proposta

## Objetivo

Levar o portal de "painel administrativo em construção" para **duas superfícies separadas e seguras**
— site público do servidor (notícias, rankings, eventos, suporte, loja) e área administrativa por
papéis — com padrão visual único e dados ricos (imagem real de item, bolsa de NPC, efeitos sem ID).

## Contexto/evidência (05/11/2026)

- `portal/src/app` já tem os grupos `(admin)`, `(public)`, `api`, `sem-permissao` — a separação
  começou, mas o grupo público está raso (30 `page.tsx` no total, maioria admin).
- BFF correto: toda ação admin passa por `adminCall` (`portal/src/server/admin.ts`) com sessão,
  `moderator_id` da sessão, deadline, mapeamento `AdminResult` → HTTP e auditoria (`AGENTS.md` §3).
- Testes do portal: só `vitest`; sem e2e de navegador (REVIEW-03).
- Identidade visual existe em tokens (`frontend-design`/`ui-ux-practices` como skills), mas não há
  documento de marca (logo, paleta, tipografia, ícones) nem componente de "shell" público.
- Dados ricos: `item_catalog` já tem efeitos e nomes; falta a camada de apresentação (imagem real do
  item, nome de efeito em vez de ID, bolsa visual para NPC vendedor).

## Passos

1. **Contrato de superfícies**: rotas `(public)` só com dados públicos (rankings, notícias, eventos,
   status do servidor); qualquer leitura por personagem exige sessão de jogador; `(admin)` exige papel
   e nunca é alcançável por link público (middleware + teste de 403 — ver REVIEW-03).
2. **Papéis**: matriz papel × rota × ação (`moderator`, `admin`, `support`, leitor) em
   `docs/analysis/` + testes de autorização por rota; nenhum `moderator_id` vindo do corpo.
3. **Site público**: home com status/onlines, notícias, rankings (nível, guild, PvP), calendário de
   eventos, páginas de suporte/regras — conteúdo vindo do web-api, nunca hardcoded.
4. **Loja**: catálogo, carrinho, pedido, cupom/promoção e histórico; pagamento entra por provider
   escolhido com pesquisa de serviço (não inventar integração); entrega de item **só** por
   `delivery_queue` (`AGENTS.md` §3, invariante 7).
5. **Premium Mail / fóruns**: pesquisar referências de portais WYD antes de projetar (o usuário pediu
   referencial real); fórum pode começar como leitura + tópicos com moderação auditada.
6. **Visual**: documento de marca (paleta, tipografia, ícones, espaçamento), shell público com
   navegação e estados de erro/carregamento, e revisão de acessibilidade (foco, contraste — o
   `--border-control` documentado na skill já resolve o caso de borda de input).
7. **Dados ricos**: componente de item com imagem real (manifest de ícones já existe), painel de
   bolsa para NPC vendedor (grade + tooltip de efeito), e tradução de IDs de efeito/add via catálogo.

## Arquivos

- `portal/src/app/(public)/**` (novas rotas), `portal/src/components/site/**`
- `portal/src/components/item/**` (item com imagem, bolsa)
- `portal/src/server/admin.ts` (reforço de autorização), `portal/src/middleware.ts`
- `docs/analysis/portal-papeis.md` (novo), `docs/analysis/premium-mail.md` (novo)

## Testes/verificação

```bash
cd portal && npm run typecheck && npm test
curl -s -o /dev/null -w "%{http_code}" http://127.0.0.1:3000/admin/itens   # sem sessão: 302/403
```

## Critérios de aceite

- [ ] Nenhuma rota `(public)` lê dado de personagem sem sessão.
- [ ] Matriz de papéis documentada e coberta por teste de autorização.
- [ ] Loja entrega item exclusivamente por `delivery_queue`, com auditoria do pedido.
- [ ] Documento de marca + shell público aplicados nas rotas públicas.
- [ ] Item e NPC exibem imagem real e nome de efeito (sem ID cru na tela).

## Riscos/rollback

Risco: abrir superfície pública que expõe dado de personagem (o pior defeito possível aqui).
Mitigação: middleware default-deny + teste de 403 por rota no CI antes de qualquer tela nova.
Rollback: rotas públicas novas entram atrás de flag de ambiente.

## Issues que este review abre

- `PORTAL-01` matriz de papéis + middleware + testes
- `PORTAL-02` site público (home, rankings, notícias, eventos)
- `PORTAL-03` loja (catálogo → pedido → delivery_queue)
- `PORTAL-04` Premium Mail e fóruns
- `PORTAL-05` identidade visual e shell público
- `PORTAL-06` item com imagem real + bolsa de NPC
