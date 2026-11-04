# Issue 47 — UI sem IDs crus e padrão de UI/UX

**Fase:** 6 · **Depende de:** 20, 43 · **Bloqueia:** 39 · **Esforço:** M · **Estado:** ✅ (nomes no lugar de IDs, escala `--space-*`, foco, alvo mínimo, contraste medido e retry em todas as telas com erro; ressalva: nenhuma inspeção visual em navegador nesta rodada porque o Docker não subiu)

## Objetivo

Nenhuma tela do portal mostra um identificador interno quando existe um nome humano, e o portal
passa a seguir **um** padrão de tokens, estados de tela e acessibilidade — em vez de cada tela
inventar espaçamento, rótulo e tratamento de erro.

## Contexto / evidência

- Duas skills novas foram adicionadas a `.agents/skills/`:
  - `frontend-design` — direção visual intencional, tipografia, e uma lista explícita de "tells" de
    página gerada (cadeia de metadados com `·`, rótulo em caixa alta, mono para etiqueta pequena,
    card com o mesmo raio e a mesma sombra em tudo). **Vale para este projeto.**
  - `ui-ux-practices` — escrita para **outro** produto (Tauri 2, `pnpm`, `packages/app/ui`,
    `gp100-core`, `invoke`). Os caminhos, gates e armadilhas **não** se aplicam aqui; valem apenas os
    princípios: escala de espaço, contraste AA, foco visível, alvo mínimo, estados explícitos.
- A UI atual mostra IDs em pelo menos quatro lugares: `#item_index` no detalhe do personagem,
  `#n` nas skills aprendidas, `eff:valor` cru nos adds, `#índice` na tabela de itens estáticos do
  atlas e `#índice nome` no editor de loja de NPC.
- O nome já existe e é barato de resolver: `webserver/internal/itemcatalog` lê
  `Release/Common/ItemList.csv` no boot (3.220 itens, com `display_name` e `icon_url` do pack).

## Regras adotadas (valem para toda tela nova)

1. **Nome antes de ID.** Lista e detalhe mostram o nome do catálogo; o índice continua existindo como
   link (`/itens/<index>`) e como `title`/coluna secundária "código" — nunca como rótulo principal.
   Fallback `#1234` só quando a linha realmente não está no catálogo.
2. **Tokens de espaço** base 4 na escala de Fibonacci: 4 · 8 · 12 · 20 · 32 · 52 · 84
   (`--space-1`…`--space-7`). Nada de `10`, `14`, `18` soltos em padding/margin/gap novos.
3. **Acessibilidade**: contraste AA (4,5:1 texto / 3:1 UI), `:focus-visible` de 2px com 3:1 contra o
   entorno, alvo clicável ≥ 32×32 CSS px, e nenhum estado indicado **só** por cor.
4. **Estados explícitos**: `loading` / `empty` / `error` / `ready` por superfície. Erro sempre com uma
   ação de recuperação ("tentar de novo", "voltar à busca"), nunca só a mensagem.
5. **Copy**: português, sentence case, o botão diz o resultado ("Conceder item" → "Item concedido"),
   o mesmo nome do começo ao fim do fluxo.
6. **Um acento por tela.** O dourado (`--accent`) marca a ação primária e o foco; o resto fica neutro.
   Metadado em cadeia (`a · b · c`) só para dado curto e homogêneo, nunca como decoração de cabeçalho.
7. **A UI não fala protocolo.** Nada de nome de campo do wire, código de enum ou hex na tela; o BFF
   entrega tipos já nomeados.

## Feito (nesta rodada)

- `web.proto`: `AdminCharacterItem` ganhou `name` (10) e `icon_url` (11).
- `webserver/internal/characteradmin`: `SetItemCatalog` + `Detail.Items` resolvem índice → nome/ícone
  só para os itens que o personagem realmente tem (não manda o catálogo inteiro por gRPC).
- `webserver/cmd/webserver`: liga o catálogo escaneado no serviço de personagem.
- Portal: `itemLabel()` em `lib/character.ts`; equipamento e inventário mostram **ícone + nome**
  (índice no `title`, link para `/itens/<index>`).

## Pendente

- ~~Skills aprendidas continuam `#n`~~ — resolvido: `webserver/internal/skillcatalog` lê o
  `SkillData.csv` (151 linhas, Latin-1) e o `GetCharacterAdminResponse.skill_names` nomeia o mask
  aprendido e a barra. A tela completa de skills (edição/overlay) continua na 46.
- ~~Adds/efeitos (`eff:valor`)~~ — resolvido: `internal/itemeffects` carrega o vocabulário
  de `Source/Code/ItemEffect.h` (0–127) com rótulos em português para os códigos que aparecem como
  add (Dano, Defesa, Refino, Crítico…), e `add_labels` devolve só os códigos do save. Código fora do
  header cai em `efeito N`. A **edição** de adds continua na 44.
- ~~**Atlas**: coluna "Item" mostra o índice~~ — resolvido: a página junta `static_items` com o
  catálogo em cache e a tabela mostra o nome (índice no `title` e na busca).
- ~~**Editor de loja de NPC**: linha era `#índice nome`~~ — resolvido: o slot mostra o nome e o
  índice fica no `title`; o mesmo no "escolhido" do seletor.
- ~~**Teleportes**: origem/destino em coordenadas cruas~~ — resolvido: `lib/zone.ts` (`zoneAt`
  escolhe a **menor** caixa do atlas que contém o ponto, então cidade vence região) e `/teleportes`
  mostra "Armia → Montanha Gelada" com a coordenada como detalhe em `title`. Sem atlas gerado a
  tabela volta às coordenadas, sem quebrar.
- ~~Escala de espaço~~ — resolvido: `--space-1..7` (4 · 8 · 12 · 20 · 32 · 52 · 84) no `:root` e já
  aplicada nas superfícies compartilhadas (`ui-content`, `ui-filters`, `ui-tabs`, `ui-table-toolbar`,
  `ui-table-footer`, `ui-card`, `ui-select`) e nas telas novas (`/drops`). Falta migrar os valores
  soltos que sobraram nos componentes antigos.
- ~~Foco invisível em tabs e cabeçalhos ordenáveis~~ — resolvido: `all: unset` em `.ui-tab` e em
  `.ui-table th > button` apagava o `outline` da regra global de `:focus-visible`; os dois agora
  redeclaram o anel de 2px. Também subimos `.ui-chip` (26 → 32px) e `.ui-select` (28 → 32px) para o
  alvo mínimo de 32 px.
- **Contraste medido**, não estimado (WCAG 2.1, luminância relativa): texto 16,09:1 · `--muted`
  8,34/7,69/7,11:1 (bg/surface/surface2) · `--accent` 7,85/7,23/6,69:1 · `--accent-ink` sobre
  `--accent` 7,60:1 · `--ok` 7,14/6,61:1 · `--warn` 7,74/7,16:1 · `--danger` 5,34/4,94:1 — todos
  acima de 4,5:1. O achado real ficou nas **bordas**: `--border` (1,45:1) e `--border-strong`
  (1,87:1) são divisores decorativos, mas eram usados como borda de input/select/chip/botão, que é
  a única affordance que identifica o campo (WCAG 1.4.11 pede 3:1). Entrou `--border-control:
  #707b8d` (4,42:1 no bg, 3,77:1 no surface2) nesses quatro controles.
- **Recuperação de erro em todo lugar**: `components/RetryButton.tsx` (`router.refresh()` dentro de
  transição, com "recarregando…" enquanto espera) entrou em `/castelo`, `/guildas`, `/personagens`,
  `/npcs`, `/teleportes`, `/config` e `/drops`. O erro de formulário de `/precos` não tem retry de
  propósito: o próprio botão de salvar é a nova tentativa.

## Arquivos

- Alterados: `api/web/v1/web.proto`, `webserver/internal/characteradmin/service.go`,
  `webserver/internal/grpcsrv/characteradmin.go`, `webserver/cmd/webserver/main.go`,
  `portal/src/lib/character.ts`, `portal/src/app/(admin)/personagens/[id]/page.tsx`,
  `.agents/skills/wyd-web-admin-ui/SKILL.md` (§12).
- Proibido: resolver nome no browser (o catálogo tem 3,2 mil linhas — isso é trabalho do BFF), e
  importar `webserver/internal/*` de componente client.

## Testes / verificação

```bash
go build ./... && go test ./webserver/internal/characteradmin/ -v
cd portal && npx tsc --noEmit && npm run build
```

Evidência: `/personagens/1` mostra nome+ícone nos itens; item fora do catálogo cai em `#índice`.

## Critérios de aceite

- [x] Detalhe do personagem sem `#índice` como rótulo de item
- [x] Nenhuma tela lista `#id` quando o catálogo tem o nome (skills/adds/atlas/loja/teleportes)
- [x] Espaçamentos novos usando `--space-*`
- [x] Todo `error` de tela tem ação de recuperação (`RetryButton`)
- [x] Contraste medido nos pares novos (foco e alvo mínimo também resolvidos)
