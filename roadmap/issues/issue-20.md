# Issue 20 — Shell da UI: layout, navegação e componentes base

**Fase:** 6 · **Depende de:** 19 · **Bloqueia:** 21–39 (todas as telas) · **Esforço:** M
· **Estado:** ✅ concluída (2026-11-03)

## Objetivo

Toda tela de admin nasce dentro de um shell consistente — menu lateral com as seções, cabeçalho com
conta/papel/sair, e uma biblioteca de tabela/formulário/dialog/combobox/toast reaproveitável.

## Contexto / evidência

- São ~18 telas até a issue 39; sem componentes base cada uma reinventa tabela e formulário.
- O domínio tem particularidades que a UI precisa refletir: `coin` e `EXP` são inteiros grandes do
  jogo (formatar pt-BR), NPC tem `map_id` com nome de cidade (`ListMapZones` → Armia/Azran/Erion/
  Nippleheim/Noatum), item é `item_index` + nome do catálogo, e `AdminResult` tem 5 valores.
- A paleta do jogo é escura; um tema claro/escuro alternável evita fadiga em sessões longas.

## Passos

1. Definir tokens de design em `src/app/globals.css` (CSS vars): `--bg`, `--surface`, `--border`,
   `--text`, `--muted`, `--accent`, `--danger`, `--ok`, raio 8px, densidade "compacta" (tabela com
   32px de linha). Sem framework de UI: CSS Modules + vars.
2. `src/app/(admin)/layout.tsx`: sidebar fixa (240px) com seções e ícones, header com
   `nome da conta • papel` + botão Sair, `<main>` com `max-width: 1400px`.
3. Registrar o menu em `src/components/nav.ts` (fonte única):
   ```ts
   export const NAV = [
     { group: "Conteúdo", items: [
       { href: "/itens",    label: "Catálogo de itens", icon: "package" },
       { href: "/mobs",     label: "Mobs e stats",      icon: "swords" },
       { href: "/npcs",     label: "NPCs e lojas",      icon: "store" },
       { href: "/drops",    label: "Drops",             icon: "percent" },
     ]},
     { group: "Mundo", items: [
       { href: "/mapa",     label: "Mapa de colisão",   icon: "map" },
       { href: "/eventos",  label: "Eventos globais",   icon: "sparkles" },
       { href: "/config",   label: "Rates e flags",     icon: "sliders" },
     ]},
     { group: "Operação", items: [
       { href: "/contas",      label: "Contas e papéis", icon: "users" },
       { href: "/operacao",    label: "Ao vivo",         icon: "activity" },
       { href: "/faturamento", label: "Faturamento",     icon: "receipt" },
     ]},
   ] as const;
   ```
   Rotas ainda não implementadas aparecem com badge "em breve" (não link morto).
4. Componentes em `src/components/`:
   - `DataTable.tsx`: colunas tipadas, ordenação client-side, busca, paginação (50/100/200),
     empty state, linha clicável.
   - `Field.tsx` + `Form.tsx`: label, hint, erro por campo (zod), botão salvar com estado pendente.
   - `Dialog.tsx`: `<dialog>` nativo, foco preso, `Esc` fecha, `aria-modal`.
   - `Combobox.tsx`: busca assíncrona com debounce 200ms, teclado (↑↓/Enter), usado para item e
     template de mob (3.2k e 620 opções).
   - `Toast.tsx`: provider + `useToast()`, 4s, `role="status"`.
   - `Badge.tsx`, `Tabs.tsx`, `Stat.tsx` (número grande + rótulo).
5. Utilitários em `src/lib/format.ts`: `fmtCoin(n)` (`1.234.567`), `fmtExp(n)` (com sufixo `k`/`M`
   para tabela), `fmtDate(iso)` em `America/Sao_Paulo`.
6. `src/app/(admin)/page.tsx`: dashboard com 4 `Stat` (itens no catálogo, templates de mob, NPCs
   definidos, contas) ligados a RPCs que já existem (22–23) — se ainda não existirem, placeholders
   "—" com TODO.
7. Acessibilidade mínima: contraste AA nos tokens, `focus-visible` visível, `lang="pt-BR"` no root.

## Arquivos

- Criados: `portal/src/components/{DataTable,Field,Form,Dialog,Combobox,Toast,Badge,Tabs,Stat}.tsx`,
  `portal/src/components/nav.ts`, `portal/src/lib/format.ts`, `portal/src/app/globals.css`,
  `portal/src/app/(admin)/page.tsx`.
- Alterados: `portal/src/app/(admin)/layout.tsx`, `portal/src/app/layout.tsx`.
- Proibido: adicionar biblioteca de UI pesada (shadcn/MUI/AntD) sem decisão registrada nesta issue;
  fonte externa não licenciada.

## Testes / verificação

```bash
cd portal && npx tsc --noEmit && npm run lint && npm run build
```

No browser, em 1280×800 e 1024×768: sidebar não cobre conteúdo, tabela rola na horizontal, `Tab`
percorre menu→busca→tabela, `Esc` fecha dialog, toast anuncia em leitor de tela (`role="status"`).

## Critérios de aceite

- [x] Menu gerado por `nav.ts`; nenhuma rota duplicada hardcoded
- [ ] `DataTable` usada por pelo menos 2 telas depois desta issue, sem fork
- [ ] `Combobox` busca item por nome e por índice, e template de mob por nome
- [x] Números do jogo formatados pt-BR em todas as telas (3.220 no painel)
- [x] Contraste AA e foco visível; `lang="pt-BR"`
- [x] `build` verde e captura de tela anexada em `portal/docs/painel-issue20.png`

## Riscos / rollback

- **Risco:** componentes base virarem um framework interno. **Mitigação:** props explícitas, sem
  contexto global além do toast; qualquer tela que precise de mais usa composição local.
- **Risco:** densidade excessiva em telas pequenas. **Mitigação:** breakpoint único em 1024px
  (sidebar colapsa para ícones).
- **Rollback:** componentes são aditivos; reverter remove só as telas que os usam.

## Evidência de execução (2026-11-03)

```text
$ npx tsc --noEmit   # tsc_exit=0
$ npm run build      # build_exit=0
  ƒ /painel · ƒ /npcs · ƒ /api/search/items · ƒ /login · ƒ Middleware 39.6 kB

# login admin -> /painel 200
$ curl -s -b jar localhost:3000/painel | grep "Itens no catálogo\|3\.220"
Itens no catálogo   →   3.220        (pt-BR, do ItemCatalogService)
# /npcs 200 com "ui-sidebar" presente (shell aplicado)
# /api/search/items sem sessão        401
# /api/search/items?q=espada          200 + opções (busca por nome)
# /api/search/items?q=1               200 + opções (busca por índice)
```

- Criados: `components/{nav.ts,Icon,Sidebar,LogoutButton,DataTable,Form,Field,Dialog,Combobox,Toast,Badge,Tabs,Stat}.tsx`,
  `lib/format.ts`, `server/catalog.ts`, `api/search/items/route.ts`, `(admin)/painel/page.tsx`.
- **Desvio 1:** o dashboard vive em `(admin)/painel/page.tsx`, não em `(admin)/page.tsx` — a rota
  `/` já pertence a `app/page.tsx` (redirect por sessão) e duas páginas resolvendo para `/` quebram
  o build. `/` agora manda staff para `/painel`.
- **Desvio 2:** os estilos ficam num único `globals.css` com classes `ui-*`, em vez de CSS Modules
  por componente: são ~40 classes curtas e um único tema; a decisão de "sem framework de UI" foi
  mantida e nenhuma dependência foi adicionada.
- **Pendente:** o `Combobox` busca itens por nome e índice (`/api/search/items`), mas templates de
  mob ainda não têm rota — `MobTemplateAdminService` existe no proto (linha 481) e o endpoint entra
  na issue 25, junto da tela que usa. Critério correspondente segue desmarcado.
- **Pendente:** `DataTable` ainda não é usada por tela nenhuma; as duas primeiras são as issues 22
  (catálogo de itens) e 23 (NPCs). Critério correspondente segue desmarcado.
