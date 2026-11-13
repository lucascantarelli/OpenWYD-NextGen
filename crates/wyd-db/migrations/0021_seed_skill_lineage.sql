-- 0021_seed_skill_lineage: classe e linhagem das skills com fonte oficial (docs/wyd/decisoes-servidor.md §6).
-- GERADO por tools/content/gen-skill-lineage.mjs — NÃO EDITAR À MÃO.
-- Só entram skills cuja linhagem está publicada (Kersef 9ª/10ª e a tabela de 12ª de
-- docs/wyd/classes.md §2). O resto continua
-- NULL com classification_source = 'unmapped': ausência de fonte não vira chute.
-- Cobertura: 69 de 151 skills (12 já mapeadas no seed +
-- 57 novas nesta migração; 6 linhas curadas repetiam o seed).
-- Down seletivo: as 6 linhas que a 0019 já classificava voltam ao estado da
-- 0019 (valores restaurados na própria migração, lidos do seed pelo gerador); as 57
-- classificadas pela primeira vez aqui voltam a 'unmapped'. Nada é apagado, então reverter só
-- esta migração é íntegro e o ciclo up→down→up reproduz o mesmo estado.

-- migrate:up
UPDATE skill_catalog AS s
SET class_code = v.class_code,
    lineage_code = v.lineage_code,
    skill_role = v.skill_role,
    classification_source = v.source,
    confidence = v.confidence,
    updated_at = now()
FROM (VALUES
    (224, 'beastmaster', 'elemental', 'buff', 'kersef:9th', 95),
    (228, 'beastmaster', 'evocation', 'passive', 'kersef:9th', 95),
    (232, 'beastmaster', 'nature', 'passive', 'kersef:9th', 95),
    (200, 'transknight', 'faith', 'buff', 'kersef:9th', 95),
    (204, 'transknight', 'trans', 'passive', 'kersef:9th', 95),
    (236, 'huntress', 'survival', 'passive', 'kersef:9th', 90),
    (240, 'huntress', 'trade', 'passive', 'kersef:9th', 95),
    (212, 'foema', 'white_magic', 'passive', 'kersef:9th', 95),
    (216, 'foema', 'black_magic', 'buff', 'kersef:9th', 95),
    (213, 'foema', 'white_magic', 'buff', 'kersef:10th', 95),
    (217, 'foema', 'black_magic', 'passive', 'kersef:10th', 95),
    (221, 'foema', 'special_magic', 'debuff', 'kersef:10th', 90),
    (201, 'transknight', 'faith', 'passive', 'kersef:10th', 95),
    (9, 'transknight', 'trans', 'passive', 'kersef:10th', 90),
    (209, 'transknight', 'magic_sword', 'passive', 'kersef:10th', 95),
    (225, 'beastmaster', 'elemental', 'buff', 'kersef:10th', 95),
    (229, 'beastmaster', 'evocation', 'summon', 'kersef:10th', 90),
    (233, 'beastmaster', 'nature', 'passive', 'kersef:10th', 90),
    (237, 'huntress', 'survival', 'passive', 'kersef:10th', 95),
    (27, 'foema', 'white_magic', 'heal', 'kersef:class-foema', 80),
    (29, 'foema', 'white_magic', 'heal', 'kersef:class-foema', 80),
    (31, 'foema', 'white_magic', 'heal', 'kersef:class-foema', 80),
    (43, 'foema', 'white_magic', 'buff', 'kersef:class-foema', 75),
    (32, 'foema', 'black_magic', 'damage', 'kersef:class-foema', 85),
    (33, 'foema', 'black_magic', 'damage', 'kersef:class-foema', 85),
    (34, 'foema', 'black_magic', 'damage', 'kersef:class-foema', 85),
    (36, 'foema', 'black_magic', 'damage', 'kersef:class-foema', 80),
    (39, 'foema', 'black_magic', 'damage', 'kersef:class-foema', 80),
    (47, 'foema', 'special_magic', 'debuff', 'kersef:class-foema', 85),
    (46, 'foema', 'special_magic', 'debuff', 'kersef:class-foema', 80),
    (220, 'foema', 'special_magic', 'debuff', 'wyd-international:9-12', 85),
    (7, 'transknight', 'faith', 'damage', 'kersef:class-transknight', 85),
    (1, 'transknight', 'faith', 'debuff', 'kersef:class-transknight', 80),
    (8, 'transknight', 'trans', 'damage', 'kersef:class-transknight', 85),
    (11, 'transknight', 'trans', 'damage', 'kersef:class-transknight', 85),
    (10, 'transknight', 'trans', 'damage', 'kersef:class-transknight', 80),
    (12, 'transknight', 'magic_sword', 'damage', 'kersef:class-transknight', 75),
    (23, 'transknight', 'magic_sword', 'damage', 'wyd-international:9-12', 85),
    (24, 'huntress', 'survival', 'damage', 'kersef:class-huntress', 80),
    (245, 'huntress', 'survival', 'damage', 'kersef:class-huntress', 80),
    (83, 'huntress', 'trade', 'craft', 'kersef:class-huntress', 80),
    (84, 'huntress', 'trade', 'craft', 'kersef:class-huntress', 75),
    (80, 'huntress', 'capture', 'damage', 'kersef:class-huntress', 75),
    (81, 'huntress', 'capture', 'debuff', 'kersef:class-huntress', 75),
    (48, 'beastmaster', 'elemental', 'summon', 'kersef:class-beastmaster', 75),
    (53, 'beastmaster', 'elemental', 'buff', 'kersef:class-beastmaster', 75),
    (56, 'beastmaster', 'evocation', 'summon', 'kersef:class-beastmaster', 80),
    (60, 'beastmaster', 'evocation', 'summon', 'kersef:class-beastmaster', 80),
    (59, 'beastmaster', 'evocation', 'summon', 'kersef:class-beastmaster', 80),
    (61, 'beastmaster', 'evocation', 'summon', 'kersef:class-beastmaster', 80),
    (62, 'beastmaster', 'evocation', 'summon', 'kersef:class-beastmaster', 80),
    (63, 'beastmaster', 'evocation', 'summon', 'kersef:class-beastmaster', 80),
    (208, 'transknight', 'magic_sword', 'passive', 'kersef:9th', 90),
    (203, 'transknight', 'faith', 'passive', 'kersef:12th', 90),
    (207, 'transknight', 'trans', 'passive', 'kersef:12th', 90),
    (211, 'transknight', 'magic_sword', 'passive', 'kersef:12th', 90),
    (215, 'foema', 'white_magic', 'passive', 'kersef:12th', 90),
    (219, 'foema', 'black_magic', 'passive', 'kersef:12th', 90),
    (223, 'foema', 'special_magic', 'buff', 'kersef:12th', 85),
    (227, 'beastmaster', 'elemental', 'passive', 'kersef:12th', 90),
    (231, 'beastmaster', 'evocation', 'passive', 'kersef:12th', 90),
    (239, 'huntress', 'survival', NULL, 'kersef:12th', 85),
    (243, 'huntress', 'trade', NULL, 'kersef:12th', 85)
) AS v (skill_index, class_code, lineage_code, skill_role, source, confidence)
WHERE s.skill_index = v.skill_index;

-- migrate:down
UPDATE skill_catalog AS s
SET class_code = v.class_code,
    lineage_code = v.lineage_code,
    skill_role = v.skill_role,
    classification_source = v.source,
    confidence = v.confidence,
    updated_at = now()
FROM (VALUES
    (201, 'transknight', 'faith', 'passive', 'rule:name_confirmed', 80),
    (225, 'beastmaster', 'elemental', 'summon', 'rule:name_confirmed', 80),
    (8, 'transknight', 'trans', 'damage', 'rule:name_confirmed', 80),
    (11, 'transknight', 'trans', 'buff', 'rule:name_confirmed', 80),
    (203, 'transknight', 'magic_sword', 'debuff', 'rule:name_confirmed', 80),
    (223, 'foema', 'special_magic', 'passive', 'rule:name_confirmed', 80)
) AS v (skill_index, class_code, lineage_code, skill_role, source, confidence)
WHERE s.skill_index = v.skill_index;

UPDATE skill_catalog AS s
SET class_code = NULL,
    lineage_code = NULL,
    skill_role = NULL,
    classification_source = 'unmapped',
    confidence = NULL,
    updated_at = now()
WHERE s.skill_index IN (224, 228, 232, 200, 204, 236, 240, 212, 216, 213, 217, 221, 9, 209, 229, 233, 237, 27, 29, 31, 43, 32, 33, 34, 36, 39, 47, 46, 220, 7, 1, 10, 12, 23, 24, 245, 83, 84, 80, 81, 48, 53, 56, 60, 59, 61, 62, 63, 208, 207, 211, 215, 219, 227, 231, 239, 243);
