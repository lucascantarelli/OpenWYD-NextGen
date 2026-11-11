# Issue PORTAL-04 — Catálogo de itens + economia do servidor

**Tipo:** Código (portal+gateway) · **Esforço:** M · **Depende de:** ROAD-01, FASE-07-5 ✅ · **Bloqueia:** PORTAL-03

## Objetivo

Expandir catálogo (descrição, atributos, requisitos, filtros/busca/ordenação, validação, auditoria, relatórios de popularidade/vendas) e economia (transações, saldo, relatórios, alertas, integração com financeiro/monitoramento).

## Contexto / evidência

- Hoje: `/itens` somente leitura (`item_catalog/list_items`), `/economia` de fonte real (`city_tax_ledger` + `delivery_queue`); leitura de conteúdo já tem contrato (`npc/mob/effect/drop/teleport` — FASE-07-5).
- `prompt.md` pede: catálogo com tudo + personalização + integração com entregas/financeiro; economia com relatórios/alertas/auditoria e boas práticas; visuais interativos; sem número inventado.

## Passos

1. Catálogo: filtros (nome, classe, raridade, nível), ficha com atributos/requisitos/origem, validação contra `item_catalog`, trilha de alteração, relatórios (popularidade, vendas, drops).
2. Economia: séries de `city_tax_ledger` + `cash_ledger` + `delivery_queue` (quando PORTAL-03 existir) com gráficos, alertas (inflação, dreno, outlier) e exportação.
3. Integração: item do catálogo → premium mail (PORTAL-03) com origem preservada; preço de loja via `game_setting` (nunca constante).
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
- [ ] Preços/atributos/requisitos na ficha — BLOQUEADO: `npc_admin/list_item_prices` etc. declarados sem handler (501); sem número inventado

## Critérios de aceite

- [ ] Nenhum item sem fonte; nenhum preço sem `game_setting`.
- [ ] Relatórios reproduzíveis do banco (queries nomeadas).

## Riscos / rollback

- Risco: expor dado sensível de economia → mitigação: agregados por papel, sem linha por jogador para moderador.
- Rollback: voltar a somente-leitura.
