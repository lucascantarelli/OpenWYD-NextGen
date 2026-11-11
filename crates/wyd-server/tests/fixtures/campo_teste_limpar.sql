-- Remove SOMENTE as fixtures de `campo_teste.sql` (slugs `teste-campo-*` e drops com
-- `content_source = 'fixture-campo-teste'`). Nenhum conteúdo do snapshot é tocado.
--
--   docker compose exec -T db psql -U admin -d wyd < crates/wyd-server/tests/fixtures/campo_teste_limpar.sql

BEGIN;

DELETE FROM npc_definition WHERE slug LIKE 'teste-campo-%';
DELETE FROM drop_rate_override WHERE content_source = 'fixture-campo-teste';

COMMIT;
