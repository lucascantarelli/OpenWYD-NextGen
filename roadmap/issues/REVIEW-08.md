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

## Analisar tudo

- Atualmente estamos implementando um portal administrativo para facilitar a gestão do sistema, incluindo recursos de configuração avançada para drops, mobs, quests, eventos, etc., para que possamos oferecer uma experiência mais completa e satisfatória para os administradores do servidor, além de otimizar o desempenho e garantir maior segurança na plataforma.
- devemos expandir o portal, alem de recursos de configuração avançada para drops, mobs, quests, eventos, etc., devemos incluir recursos de monitoramento de desempenho, análise de logs, gerenciamento de usuários, controle de permissões, relatórios e dashboards, para que possamos oferecer uma experiência mais completa e satisfatória para os administradores do servidor, além de otimizar o desempenho e garantir maior segurança na plataforma.
- na expansão, devemos ter tb o site do servidor, com informações, notícias, eventos, rankings, fóruns, suporte, etc., para que possamos oferecer uma experiência mais completa e satisfatória para os usuários finais, além de otimizar o desempenho e garantir maior segurança na plataforma.
- no portal devemos implementar a loja para clientes, com recursos de gerenciamento de produtos, pedidos, pagamentos, promoções e cupons, para que possamos oferecer uma experiência mais completa e satisfatória para os usuários finais, além de otimizar o desempenho e garantir maior segurança na plataforma.
- creio que o sistema implementado no jogo se chama premium mail, faça pesquisas sobre sites de wyd para pegar referencais a sites de servidores, partes para usuários, lojas, fóruns, suporte, etc., para que possamos oferecer uma experiência mais completa e satisfatória para os usuários finais, além de otimizar o desempenho e garantir maior segurança na plataforma.
- devemos ter um portal completo, para administradores e usuários, com recursos de monitoramento de desempenho, análise de logs, gerenciamento de usuários, controle de permissões, relatórios e dashboards, informações, notícias, eventos, rankings, fóruns, suporte, loja para clientes, com recursos de gerenciamento de produtos, pedidos, pagamentos, promoções e cupons, para que possamos oferecer uma experiência mais completa e satisfatória para os administradores do servidor e usuários finais, além de otimizar o desempenho e garantir maior segurança na plataforma.
- devemos ter a area administrativa do portal, devidamente segura, via papeis, pesquisar boas praticas para esse tipo de portal, o que deve ser seguro, quais as boas praticas, organização, padronização, etc., para que possamos oferecer uma experiência mais completa e satisfatória para os administradores do servidor, além de otimizar o desempenho e garantir maior segurança na plataforma.
- implemente melhorias em todo o sistema do portal.
- faça a separação de funcionalidades da parte de usuários/site/publico, da area administradiva, separe as funcionalidades, exponha de forma segura, usando as roles, para que possamos oferecer uma experiência mais completa e satisfatória para os administradores do servidor e usuários finais, além de otimizar o desempenho e garantir maior segurança na plataforma.
- da parte administratira, tente deixar sempre as modificações, informações, mais user friendly, com recursos ricos, cores, imagens, links, etc., para que possamos oferecer uma experiência mais completa e satisfatória para os administradores do servidor, além de otimizar o desempenho e garantir maior segurança na plataforma.
- usar as imagens reais dos itens dentro do painel administrativo, pesquise as imagens reais dos itens, para que possamos oferecer uma experiência mais completa e satisfatória para os administradores do servidor, além de otimizar o desempenho e garantir maior segurança na plataforma.
- dentro de vendedores, monstros, quests, eventos, etc., devemos ter imagens reais, com recursos ricos, cores, imagens, links, etc., para que possamos oferecer uma experiência mais completa e satisfatória para os administradores do servidor, além de otimizar o desempenho e garantir maior segurança na plataforma.
- nos npcs que contem itens como vendedores, monstros, tente montar um painel de edição dos itens maais parecido possivel com uma bolsa real, com imagens reais, com recursos ricos, cores, imagens, links, etc., para que possamos oferecer uma experiência mais completa e satisfatória para os administradores do servidor, além de otimizar o desempenho e garantir maior segurança na plataforma.
- nos itens/efeitos/ temos alguns tipos de informações que são ids, como adicionais, tente trazer essas informações mais claras possiveis, evitar ids, usar nomes que levem a uma melhor compreensão, com imagens reais, com recursos ricos, cores, imagens, links, etc., para que possamos oferecer uma experiência mais completa e satisfatória para os administradores do servidor, além de otimizar o desempenho e garantir maior segurança na plataforma.
- Aprimorar a identidade visual do projeto, incluindo logotipo, cores, tipografia, ícones e elementos gráficos, para que possamos oferecer uma experiência mais completa e satisfatória para os usuários finais, além de otimizar o desempenho e garantir maior segurança na plataforma.
- Aprimorar o design da interface do usuário, incluindo layout, navegação, responsividade e acessibilidade, para que possamos oferecer uma experiência mais completa e satisfatória para os usuários finais, além de otimizar o desempenho e garantir maior segurança na plataforma.
- Definir palheta de cores, tipografia, ícones e elementos gráficos consistentes para todo o projeto, para que possamos oferecer uma experiência mais completa e satisfatória para os usuários finais, além de otimizar o desempenho e garantir maior segurança na plataforma.
- Modernizar completamente a ui, usando templates modernos e atuais como referencia, para que possamos oferecer uma experiência mais completa e satisfatória para os usuários finais, além de otimizar o desempenho e garantir maior segurança na plataforma.
- faça pesquisas na internet por boas praticas de ui/ux, design de interface, design de interação, design visual, design responsivo, design acessível, design inclusivo, design emocional, design persuasivo, design de gamificação, design de microinterações, design de animações, design de tipografia, design de cores, design de ícones, design de imagens, design de vídeos, design de áudio, design de realidade aumentada, design de realidade virtual, design de inteligência artificial, para que possamos oferecer uma experiência mais completa e satisfatória para os usuários finais, além de otimizar o desempenho e garantir maior segurança na plataforma.
- atualize/crie as skills do agente para que possamos oferecer uma experiência mais completa e satisfatória para os usuários finais, além de otimizar o desempenho e garantir maior segurança na plataforma.
- implemente um layout moderno, organizado, profisional, completo, reutilizavel, responsivo, acessível, inclusivo, emocional, persuasivo, gamificado, com microinterações, animações, tipografia, cores, ícones, imagens, recursos ricos, para que possamos oferecer uma experiência mais completa e satisfatória para os usuários finais, além de otimizar o desempenho e garantir maior segurança na plataforma.
- organize sidebars, menus, botões, links, formulários, inputs, selects, checkboxes, radios, tables, grids, cards, modals, popups, tooltips, accordions, tabs, sliders, carousels, galleries, charts, graphs, maps, timelines, calendars, notifications, alerts, badges, tags, avatars, progress bars, loaders, spinners e outros elementos de interface de forma coerente e padronizada para que possamos oferecer uma experiência mais completa e satisfatória para os usuários finais.
- padronize os elementos de interface, incluindo cores, tipografia, ícones, espaçamento, alinhamento e estilo visual, para que possamos oferecer uma experiência mais completa e satisfatória para os usuários finais.


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
