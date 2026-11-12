-- 0044_seed_quest_campo: conteúdo do Campo de Treinamento (GAME-18).
-- As tabelas já existem desde `0013_quests` (`quest`, `quest_step`, `quest_reward`,
-- `character_quest`); aqui entra só o dado do legado, sem DDL.
--
-- Fonte: decisão A do dono (sequência fiel ao 7662), registrada em
-- `docs/wyd/pesquisa-campo-de-treinamento-2026-11-10.md` §2.6 e §3.
-- As quatro etapas são as entregas das provas 451/452/453/524 nos treinadores
-- (`_MSG_Quest.cpp`); a recompensa de cada uma é aplicada pelo servidor
-- (`gameplay::quest_engage`), que é a autoridade — a tabela é catálogo, não script.

-- migrate:up
INSERT INTO quest (
    code, name, kind, min_level, class_code, repeatable, cooldown_seconds,
    requires, description, content_source, confidence
) VALUES (
    'campo_de_treinamento',
    'Campo de Treinamento',
    'main',
    1,
    NULL,
    false,
    0,
    '{}',
    'Quatro treinadores, tres portoes e o Guerreiro Orc. Limite de nivel 1 a 35.',
    'docs/wyd/pesquisa-campo-de-treinamento-2026-11-10.md',
    95
)
ON CONFLICT (code) DO NOTHING;

INSERT INTO quest_step (quest_code, step, objective, target_id, count, description) VALUES
    ('campo_de_treinamento', 1, 'collect', 451, 1,
     'Entregar a Chave da Primeira Porta ao primeiro treinador'),
    ('campo_de_treinamento', 2, 'collect', 452, 1,
     'Entregar a Chave da Segunda Porta ao segundo treinador'),
    ('campo_de_treinamento', 3, 'collect', 453, 1,
     'Entregar a Chave da Ultima Porta ao terceiro treinador'),
    ('campo_de_treinamento', 4, 'collect', 524, 1,
     'Entregar o Emblema Orc ao Chefe de Treino')
ON CONFLICT (quest_code, step) DO NOTHING;

INSERT INTO quest_reward (quest_code, kind, item_index, amount) VALUES
    ('campo_de_treinamento', 'item', 682, 1),
    ('campo_de_treinamento', 'item', 481, 1),
    ('campo_de_treinamento', 'item', 652, 1)
ON CONFLICT DO NOTHING;

-- migrate:down
DELETE FROM quest WHERE code = 'campo_de_treinamento';
