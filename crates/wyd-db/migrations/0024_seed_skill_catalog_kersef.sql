-- 0024_seed_skill_catalog_kersef: 3 skills publicadas pela Kersef que o seed do cliente não tem.
-- Fonte de cada linha: central Kersef (9ª/10ª skill por linhagem e lista da classe BeastMaster),
-- a mesma que alimenta `docs/wyd/classes.md` §2 e a curadoria de `0020_seed_skill_lineage.sql`.
-- O que NÃO é confirmado: o `skill_index` do cliente (o seed de 0018 veio do `SkillData.bin`/
-- `ItemList.bin`, que não trazem esses nomes). Por isso o índice é **provisório** — primeiro
-- slot livre da faixa 0..247 — e está declarado em `notes`, junto dos parâmetros pendentes.
-- Regra do repo: UNVERIFIED não vira gate (AGENTS.md invariante 5); quem for usar em combate
-- precisa do trace do cliente antes.

-- migrate:up
INSERT INTO skill_catalog
    (skill_index, name, class_code, lineage_code, skill_role, primary_system,
     classification_source, confidence, notes)
VALUES
    (103, 'Roubar_Dinheiro', 'huntress', 'trade', 'damage', 'combate', 'kersef:10th', 85,
     'Nome e linhagem publicados (Kersef, 10ª skill da linhagem Troca). Índice 103 provisório: slot não confirmado no cliente; mana/delay/efeito pendentes de trace — não usar como gate.'),
    (104, 'Evocar_Lobo_Selvagem', 'beastmaster', 'evocation', 'summon', 'combate',
     'kersef:class-beastmaster', 80,
     'Nome e linhagem publicados (lista da classe BeastMaster, linhagem Evocação). Índice 104 provisório: slot não confirmado no cliente; parâmetros pendentes de trace — não usar como gate.'),
    (105, 'Metamorfose_Superior', 'beastmaster', 'nature', 'buff', 'combate',
     'kersef:class-beastmaster', 70,
     'Nome e linhagem publicados (lista da classe BeastMaster, linhagem Natureza). Índice 105 provisório: slot não confirmado no cliente; parâmetros pendentes de trace — não usar como gate.')
ON CONFLICT (skill_index) DO NOTHING;

INSERT INTO content_version (id, version, note)
VALUES (1, 7, 'catálogo de skills: 3 nomes da Kersef que o seed do cliente não trazia')
ON CONFLICT (id) DO UPDATE SET version = GREATEST(content_version.version, 7), note = EXCLUDED.note;

-- migrate:down
DELETE FROM skill_catalog WHERE skill_index IN (103, 104, 105);
