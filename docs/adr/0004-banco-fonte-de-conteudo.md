# ADR 0004 — Banco como fonte de catálogo e spawn

**Data:** 2026-11 · **Estado:** aceita

## Contexto

Catálogo de item, skill, template de NPC e spawn viviam em CSV/TXT dentro de `Release/`, versionados
junto com o binário. Editar exigia mexer em arquivo e reiniciar; duas fontes (arquivo e banco)
conviveram e produziram duplicação real — 20 linhas de seed curado ficaram sombreadas por blocos
importados e duplicaram os mercadores de Armia.

## Decisão

`item_catalog`, `skill_catalog`, `npc_template`, `npc_definition` e `world_item_seed` vêm do Postgres
pelo gRPC do dbserver, e a ausência deles **recusa o boot**. Os arquivos saíram do repositório; o
`importcontent` continua existindo só para reprocessar uma árvore externa (backup).

## Consequências

- Uma fonte só: o que o portal edita é o que o servidor carrega.
- O servidor não sobe "pela metade" com catálogo vazio silencioso.
- Testes que liam CSV agora pulam com motivo declarado e o caminho do banco tem teste próprio
  (`scripts/check-skips.sh` cobra o motivo).
