-- 0046_seed_campo_treinamento: conteúdo do Campo de Treinamento no mapa (GAME-18).
--
-- Nenhum blob novo: todas as linhas apontam para `npc_template` já presente no snapshot
-- (1991 templates do legado). Coordenadas e funções vêm do guia oficial e do droplist
-- registrados em `docs/wyd/pesquisa-campo-de-treinamento-2026-11-11.md` §1–§2; os drops
-- saem de `ItemDropList.txt` (mesma tabela do legado). `origin = 'custom'` marca que a
-- linha é posicionamento nosso sobre template real, não linha do dump.
--
-- Ids 6301..6310 ficam acima do maior id do conteúdo (6100) para não colidir com o dump.

-- migrate:up
INSERT INTO npc_definition
    (id, slug, template_name, display_name, enabled, map_id, pos_x, pos_y, merchant, origin)
VALUES
    -- Treinadores: merchants da sequência A (36/40/41) e chefe (100 + grade 16 do Equip[0]).
    (6301, 'campo-treinador-1', 'FirstTrainer',  'Primeiro Treinador', true, 0, 2077, 2017,  36, 'custom'),
    (6302, 'campo-treinador-2', 'SecondTrainer', 'Segundo Treinador',  true, 0, 2144, 1989,  40, 'custom'),
    (6303, 'campo-treinador-3', 'ThirdTrainer',  'Terceiro Treinador', true, 0, 2082, 1966,  41, 'custom'),
    (6304, 'campo-chefe-treino', 'TrainerChief', 'Chefe de Treino',    true, 0, 2122, 2041, 100, 'custom'),
    -- Arma inicial e buff (merchant 0: o efeito é tratado pelo servidor, não é loja).
    (6305, 'campo-unicorn',      'Unicorn',      'Unicorn',            true, 0, 2130, 2040,   0, 'custom'),
    (6306, 'campo-cabuncle',     'Cabuncle',     'Cabuncle Wind',      true, 0, 2131, 2033,   0, 'custom'),
    -- Guardiões e boss: templates reais do Campo.
    (6307, 'campo-gremlin-decoy', 'GremlinDecoy', 'Gremlin Decoy',     true, 0, 2076, 2022,   0, 'custom'),
    (6308, 'campo-giant-crill',   'GiantCrill',   'Giant Crill',       true, 0, 2146, 1997,   0, 'custom'),
    (6309, 'campo-orc-sniper',    'OrcSniper',    'Orc Sniper',        true, 0, 2078, 1971,   0, 'custom'),
    (6310, 'campo-orc-warrior',   'OrcWarrior',   'Guerreiro Orc',     true, 0, 2135, 1948,   0, 'custom')
ON CONFLICT (id) DO NOTHING;

-- Drops das provas e do emblema (`ItemDropList.txt`): um único item na tabela do template,
-- então o peso 1.000.000 (taxa efetiva 100%) entrega a prova da etapa.
INSERT INTO drop_rate_override
    (template_name, item_index, weight, min_qty, max_qty, content_source, confidence)
VALUES
    ('GremlinDecoy', 451, 1000000, 1, 1, 'docs/wyd/pesquisa-campo-de-treinamento-2026-11-11.md', 90),
    ('GiantCrill',   452, 1000000, 1, 1, 'docs/wyd/pesquisa-campo-de-treinamento-2026-11-11.md', 90),
    ('OrcSniper',    453, 1000000, 1, 1, 'docs/wyd/pesquisa-campo-de-treinamento-2026-11-11.md', 90),
    ('OrcWarrior',   524, 1000000, 1, 1, 'docs/wyd/pesquisa-campo-de-treinamento-2026-11-11.md', 90)
ON CONFLICT (template_name, item_index) DO NOTHING;

-- migrate:down
DELETE FROM drop_rate_override
 WHERE content_source = 'docs/wyd/pesquisa-campo-de-treinamento-2026-11-11.md';
DELETE FROM npc_definition WHERE id BETWEEN 6301 AND 6310;
