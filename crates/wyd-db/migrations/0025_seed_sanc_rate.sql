-- 0025_seed_sanc_rate: 28 células de SancRate, valor literal do arquivo do legado.
-- Fonte: origin/main:tmserver/content/Common/Settings/SancRate.txt
-- sha256 80e90f4a205c2323e5c68545e3c427c1c608665ff13372e59a58e7fc83dd718b
--
-- PO 0..5 e PL 0..9 e Âmago 0..11 — exatamente as linhas do arquivo, sem linha inventada e
-- sem preencher índice que o arquivo não lista (o default do loader é o que vale fora disso).
-- A ausência de um índice é dado: `refine_anvil_rate` não interpola.

-- migrate:up
INSERT INTO sanc_rate (powder, refine_index, percent, content_source, confidence, notes) VALUES
    ('PO', 0, 100, 'legacy:SancRate.txt', 95, 'linha literal do arquivo'),
    ('PO', 1, 100, 'legacy:SancRate.txt', 95, 'linha literal do arquivo'),
    ('PO', 2, 100, 'legacy:SancRate.txt', 95, 'linha literal do arquivo'),
    ('PO', 3, 85,  'legacy:SancRate.txt', 95, 'linha literal do arquivo'),
    ('PO', 4, 70,  'legacy:SancRate.txt', 95, 'linha literal do arquivo'),
    ('PO', 5, 40,  'legacy:SancRate.txt', 95, 'PO_MAX 6: o arquivo para em +5'),
    ('PL', 0, 100, 'legacy:SancRate.txt', 95, 'linha literal do arquivo'),
    ('PL', 1, 100, 'legacy:SancRate.txt', 95, 'linha literal do arquivo'),
    ('PL', 2, 100, 'legacy:SancRate.txt', 95, 'linha literal do arquivo'),
    ('PL', 3, 100, 'legacy:SancRate.txt', 95, 'linha literal do arquivo'),
    ('PL', 4, 100, 'legacy:SancRate.txt', 95, 'linha literal do arquivo'),
    ('PL', 5, 100, 'legacy:SancRate.txt', 95, 'linha literal do arquivo'),
    ('PL', 6, 80,  'legacy:SancRate.txt', 95, 'linha literal do arquivo'),
    ('PL', 7, 70,  'legacy:SancRate.txt', 95, 'linha literal do arquivo'),
    ('PL', 8, 70,  'legacy:SancRate.txt', 95, 'linha literal do arquivo'),
    ('PL', 9, 10,  'legacy:SancRate.txt', 95, 'PL_MAX 9: o arquivo para em +9'),
    ('AMAGO', 0,  100, 'legacy:SancRate.txt', 95, 'linha literal do arquivo'),
    ('AMAGO', 1,  80,  'legacy:SancRate.txt', 95, 'linha literal do arquivo'),
    ('AMAGO', 2,  60,  'legacy:SancRate.txt', 95, 'linha literal do arquivo'),
    ('AMAGO', 3,  40,  'legacy:SancRate.txt', 95, 'linha literal do arquivo'),
    ('AMAGO', 4,  20,  'legacy:SancRate.txt', 95, 'linha literal do arquivo'),
    ('AMAGO', 5,  10,  'legacy:SancRate.txt', 95, 'linha literal do arquivo'),
    ('AMAGO', 6,  10,  'legacy:SancRate.txt', 95, 'linha literal do arquivo'),
    ('AMAGO', 7,  10,  'legacy:SancRate.txt', 95, 'linha literal do arquivo'),
    ('AMAGO', 8,  5,   'legacy:SancRate.txt', 95, 'linha literal do arquivo'),
    ('AMAGO', 9,  5,   'legacy:SancRate.txt', 95, 'linha literal do arquivo'),
    ('AMAGO', 10, 5,   'legacy:SancRate.txt', 95, '+10: mesma taxa do índice 11 no arquivo'),
    ('AMAGO', 11, 5,   'legacy:SancRate.txt', 95, 'HARD_CAP 11: último índice do arquivo')
ON CONFLICT (powder, refine_index) DO NOTHING;

INSERT INTO content_version (id, version, note)
VALUES (1, 6, 'SancRate do legado (28 células) + passo +10 → +11 do refino')
ON CONFLICT (id) DO UPDATE
    SET version = GREATEST(content_version.version, 6), note = EXCLUDED.note;

-- migrate:down
DELETE FROM sanc_rate;
