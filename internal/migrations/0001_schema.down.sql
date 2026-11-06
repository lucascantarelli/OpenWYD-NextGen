-- 0001_schema (down) — DESTRUTIVO. O baseline é o esquema inteiro: desfazê-lo
-- apaga todos os dados, inclusive conta e personagem. Só use num banco
-- descartável; em produção a recuperação é restaurar o dump, não este arquivo.
--
-- `DROP SCHEMA ... CASCADE` leva junto tabelas, índices, sequências, tipos
-- compostos e funções do schema — não sobra objeto órfão. O schema é recriado
-- vazio; as roles/grants voltam pelo serviço `db-roles` (DB-01).
--
-- Prova automatizada: ./scripts/db-down-check.sh (aplica o up, roda este down e
-- consulta pg_class/pg_type para confirmar que o schema voltou ao estado vazio).
DROP SCHEMA public CASCADE;
CREATE SCHEMA public;
