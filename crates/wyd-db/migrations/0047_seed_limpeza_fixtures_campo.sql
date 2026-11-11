-- 0047_limpeza_fixtures_campo: remove as fixtures antigas do Campo de Treinamento.
--
-- As linhas `teste-campo-*` (ids 6201..6208, templates `Guarda_Carga`/guardas) foram
-- aplicadas à mão em bancos de desenvolvimento antes do seed definitivo (0046) e deixavam
-- **duas** instâncias de cada NPC do Campo na cena. O `down` restaura exatamente o estado
-- anterior (ADR-0013: seed reverte para o estado herdado, não apaga conteúdo do snapshot).
--
-- Nada de template/blob aqui: só linhas de posicionamento e os pesos de drop da fixture.

-- migrate:up
DELETE FROM npc_definition WHERE id BETWEEN 6201 AND 6208 AND slug LIKE 'teste-campo-%';
DELETE FROM drop_rate_override WHERE content_source = 'fixture-campo-teste';

-- migrate:down
INSERT INTO npc_definition
    (id, slug, template_name, display_name, enabled, map_id, pos_x, pos_y, merchant, origin)
VALUES
    (6201, 'teste-campo-treinador-1', 'Guarda_Carga', 'PrimeiroTreinador', true, 0, 2108, 2046,  36, 'custom'),
    (6202, 'teste-campo-treinador-2', 'Guarda_Carga', 'SegundoTreinador',  true, 0, 2112, 2046,  40, 'custom'),
    (6203, 'teste-campo-treinador-3', 'Guarda_Carga', 'TerceiroTreinador', true, 0, 2116, 2046,  41, 'custom'),
    (6204, 'teste-campo-treinador-4', 'Guarda_Carga', 'ChefeTreinador',    true, 0, 2120, 2046, 100, 'custom'),
    (6205, 'teste-campo-gremlin', 'GremlinDecoy', 'GremlinDecoy', true, 0, 2115, 2042, 0, 'custom'),
    (6206, 'teste-campo-grill',   'GiantCrill',   'GiantCrill',   true, 0, 2120, 2048, 0, 'custom'),
    (6207, 'teste-campo-sniper',  'OrcSniper',    'OrcSniper',    true, 0, 2125, 2052, 0, 'custom'),
    (6208, 'teste-campo-boss',    'OrcWarrior',   'OrcWarrior',   true, 0, 2130, 2055, 0, 'custom')
ON CONFLICT (id) DO NOTHING;

INSERT INTO drop_rate_override
    (template_name, item_index, weight, min_qty, max_qty, content_source, confidence)
VALUES
    ('GremlinDecoy', 451, 1000000, 1, 1, 'fixture-campo-teste', 100),
    ('GiantCrill',   452, 1000000, 1, 1, 'fixture-campo-teste', 100),
    ('OrcSniper',    453, 1000000, 1, 1, 'fixture-campo-teste', 100),
    ('OrcWarrior',   524, 1000000, 1, 1, 'fixture-campo-teste', 100)
ON CONFLICT (template_name, item_index) DO NOTHING;
