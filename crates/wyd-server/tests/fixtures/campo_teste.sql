-- Fixtures de TESTE do Campo de Treinamento (FASE-03-18/15-4).
--
-- Regras:
--  * nenhum blob novo: todas as linhas reusam `npc_template` já presente no snapshot
--    (`Guarda_Carga` para os treinadores, `GremlinDecoy`/`GiantCrill`/`OrcSniper`/
--    `OrcWarrior` para os guardiões e o chefe);
--  * ids 6201..6209 ficam acima do maior id do conteúdo (6100) e os slugs começam com
--    `teste-campo-`, então `campo_teste_limpar.sql` remove tudo sem tocar conteúdo;
--  * os drops são `drop_rate_override` com peso 1.000.000 (o único peso da tabela do
--    template ⇒ taxa efetiva 100%), `content_source = 'fixture-campo-teste'`.
--
-- Aplicar (banco de teste/dev; nunca em produção):
--   docker compose exec -T db psql -U admin -d wyd < crates/wyd-server/tests/fixtures/campo_teste.sql
-- Limpar:
--   docker compose exec -T db psql -U admin -d wyd < crates/wyd-server/tests/fixtures/campo_teste_limpar.sql

BEGIN;

-- Treinadores: merchants da sequência A (36/40/41/100) ao redor do spawn do Campo.
INSERT INTO npc_definition
    (id, slug, template_name, display_name, enabled, map_id, pos_x, pos_y, merchant, origin)
VALUES
    (6201, 'teste-campo-treinador-1', 'Guarda_Carga', 'PrimeiroTreinador', true, 0, 2108, 2046,  36, 'custom'),
    (6202, 'teste-campo-treinador-2', 'Guarda_Carga', 'SegundoTreinador',  true, 0, 2112, 2046,  40, 'custom'),
    (6203, 'teste-campo-treinador-3', 'Guarda_Carga', 'TerceiroTreinador', true, 0, 2116, 2046,  41, 'custom'),
    (6204, 'teste-campo-treinador-4', 'Guarda_Carga', 'ChefeTreinador',    true, 0, 2120, 2046, 100, 'custom'),
    -- Guardiões e chefe: os templates reais do Campo já estão no snapshot.
    (6205, 'teste-campo-gremlin', 'GremlinDecoy', 'GremlinDecoy', true, 0, 2115, 2042, 0, 'custom'),
    (6206, 'teste-campo-grill',   'GiantCrill',   'GiantCrill',   true, 0, 2120, 2048, 0, 'custom'),
    (6207, 'teste-campo-sniper',  'OrcSniper',    'OrcSniper',    true, 0, 2125, 2052, 0, 'custom'),
    (6208, 'teste-campo-boss',    'OrcWarrior',   'OrcWarrior',   true, 0, 2130, 2055, 0, 'custom')
ON CONFLICT (id) DO NOTHING;

-- Drops 100%: cada guardião entrega a prova da sua etapa; o chefe entrega o emblema.
INSERT INTO drop_rate_override
    (template_name, item_index, weight, min_qty, max_qty, content_source, confidence)
VALUES
    ('GremlinDecoy', 451, 1000000, 1, 1, 'fixture-campo-teste', 100),
    ('GiantCrill',   452, 1000000, 1, 1, 'fixture-campo-teste', 100),
    ('OrcSniper',    453, 1000000, 1, 1, 'fixture-campo-teste', 100),
    ('OrcWarrior',   524, 1000000, 1, 1, 'fixture-campo-teste', 100)
ON CONFLICT (template_name, item_index) DO NOTHING;

COMMIT;
