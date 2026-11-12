# WEB-10 — Catálogo de itens + economia do servidor

> **ID canônico:** `WEB-10`
> **ID histórico:** `PORTAL-04`
> **Estado 2026-11-12:** 🚧 · ver `roadmap-rust/index.md`.

**Tipo:** Código (portal+gateway) · **Esforço:** M · **Depende de:** PROC-24, WEB-05 ✅ · **Bloqueia:** WEB-09

## Objetivo

Expandir catálogo (descrição, atributos, requisitos, filtros/busca/ordenação, validação, auditoria, relatórios de popularidade/vendas) e economia (transações, saldo, relatórios, alertas, integração com financeiro/monitoramento).

## Contexto / evidência

- Hoje: `/itens` com ficha, busca/ordenação e preço por mercador (`item_price`); `/economia` de fonte real (`city_tax_ledger` + `delivery_queue`), relatórios CSV e alertas; leitura de conteúdo já tem contrato (`npc/mob/effect/drop/teleport` — WEB-05).
- `prompt.md` pede: catálogo com tudo + personalização + integração com entregas/financeiro; economia com relatórios/alertas/auditoria e boas práticas; visuais interativos; sem número inventado.

## Passos

1. Catálogo: filtros (nome, classe, raridade, nível), ficha com atributos/requisitos/origem, validação contra `item_catalog`, trilha de alteração, relatórios (popularidade, vendas, drops).
2. Economia: séries de `city_tax_ledger` + `cash_ledger` + `delivery_queue` (quando WEB-09 existir) com gráficos, alertas (inflação, dreno, outlier) e exportação.
3. Integração: item do catálogo → premium mail (WEB-09) com origem preservada; preço de loja via `game_setting` (nunca constante).
4. Auditoria de catálogo/economia na mesma trilha (`audit_log`, ator real).

## Arquivos

- Criados: `wyd-portal/src/app/(admin)/itens/ficha-*`, `wyd-portal/src/app/(admin)/economia/relatorios*`, testes.
- Alterados: `wyd-portal/src/server/{catalog,economy}.ts`, gateway `item_catalog/*`, `economy_admin/*`, `docs/wyd/{itens,economia}.md`.
- Proibido: editar balanceamento pelo portal sem `game_setting` + auditoria.

## Testes / verificação

```bash
cd wyd-portal && npx tsc --noEmit && npx vitest run
cargo test -p wyd-admin --test contract
```

## Skills relacionadas

- frontend-design, ui-ux-practices.
- wyd-rust-db — catálogo como dado validado, não regra.
- wyd-rust-obs — alertas econômicos.

## Documentação de referência

- `docs/wyd/itens.md`, `docs/wyd/economia.md`, `docs/arquitetura/banco-catalogo.md`

## Subtasks

- [x] Ficha de item (`?item=`, link no índice) + auditoria da leitura via `adminCall`
- [x] Relatórios de economia (CSV) + alertas (ouro ≥ 1M, cidade zerada) + barras por cidade
- [x] Integração catálogo → correio (link da ficha para `/correio`; concessão com chave rastreável)
- [x] Preço de loja por (item, mercador): `npc_admin/list_item_prices` + `set_item_price` no gateway, tabela `item_price`, ficha com tabela de preços e editor em `/configuracoes?aba=precos`; sem preço = ausência, nunca zero
- [ ] Atributos e requisitos na ficha — BLOQUEADO: `item_catalog/list_items` não devolve essas colunas; sem número inventado

## Critérios de aceite

- [x] Nenhum preço inventado: preço só existe em `item_price`, com procedência (`portal#<conta>`) e auditoria.
- [ ] Nenhum item sem fonte no catálogo.
- [ ] Relatórios reproduzíveis do banco (queries nomeadas).

## Riscos / rollback

- Risco: expor dado sensível de economia → mitigação: agregados por papel, sem linha por jogador para moderador.
- Rollback: voltar a somente-leitura.

## Verificação on-line (2026-11-11)

- **Catálogo**: `/itens` abriu com 3.220 itens, busca e ordenação. Corrigido:
  cabeçalho "Índice" truncado virou "Item"; contadores duplicados (topo +
  rodapé da tabela) reduzidos a um.
- **Ficha + preços**: preço por (item, mercador) lido de `item_price` e editável
  em `/configuracoes?aba=precos`.
- **Economia**: `/economia` mostrou imposto por cidade (vazio) e concessões por
  tipo (1 pedido / 100 de ouro após a entrega de teste). Corrigido: dicas dos
  cards não citam mais `city_tax_ledger`/`delivery_queue`/`gold_amount`.
- **Atributos/requisitos na ficha**: continua BLOQUEADO porque
  `item_catalog/list_items` não devolve essas colunas; não há número inventado.
- **Dados do catálogo**: os nomes reais do seed são genéricos ("TransKnight",
  "Foema" repetidos por índice) — é conteúdo do catálogo, não defeito da tela.
