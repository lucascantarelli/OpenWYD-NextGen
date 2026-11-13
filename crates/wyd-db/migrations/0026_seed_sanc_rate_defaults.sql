-- 0026_seed_sanc_rate_defaults: células AUSENTES no SancRate.txt e que o loader do legado NÃO
-- deixa vazias — ele parte do default compilado e só sobrescreve os índices listados.
-- Fonte: origin/main:Basedef.cpp g_pSancRate (via tmserver/internal/content/rates.go,
-- `sancRateDefaults`), conferida em 2026-11-08.
--
-- Por que isto existe: `0025_seed_sanc_rate.sql` gravou as 28 linhas literais do arquivo e
-- tratou ausência como "sem dado". O legado não funciona assim: o arquivo é aplicado sobre
-- a tabela compilada, então um índice que o arquivo não lista vale o default (0), não
-- "inexistente". Sem estas 8 linhas, o caminho de pó em +5→+6 (Ori) e +9→+10 (Lac) ficaria
-- sem taxa, quando o valor real é `0 + piedade × peso`.
--
-- Defaults compilados (3 linhas × 12 índices), para referência:
--   PO    100 100 100 100 100 100 100   0   0   0  0 0
--   PL    100 100 100 100 100 100 100 100 100 100  0 0
--   Âmago 100 100 100 100 100 100 100 100 100 100 80 80
-- O arquivo sobrescreve PO 0..5, PL 0..9 e Âmago 0..11; sobram PO 6..11 e PL 10..11.

-- migrate:up
INSERT INTO sanc_rate (powder, refine_index, percent, content_source, confidence, notes) VALUES
    ('PO', 6,  0, 'legacy:Basedef.cpp g_pSancRate (default)', 95,
     'sem linha no SancRate.txt: default 0. +5 → +6 só sobe com piedade (peso 3).'),
    ('PO', 7,  0, 'legacy:Basedef.cpp g_pSancRate (default)', 95, 'fora do teto PO_MAX 6'),
    ('PO', 8,  0, 'legacy:Basedef.cpp g_pSancRate (default)', 95, 'fora do teto PO_MAX 6'),
    ('PO', 9,  0, 'legacy:Basedef.cpp g_pSancRate (default)', 95, 'fora do teto PO_MAX 6'),
    ('PO', 10, 0, 'legacy:Basedef.cpp g_pSancRate (default)', 95, 'fora do teto PO_MAX 6'),
    ('PO', 11, 0, 'legacy:Basedef.cpp g_pSancRate (default)', 95, 'fora do teto PO_MAX 6'),
    ('PL', 10, 0, 'legacy:Basedef.cpp g_pSancRate (default)', 95,
     'nível +9 é recusado por PL_MAX 9 antes do sorteio; célula existe no default'),
    ('PL', 11, 0, 'legacy:Basedef.cpp g_pSancRate (default)', 95,
     'HARD_CAP 11: o caminho de pó recusa antes de ler esta célula')
ON CONFLICT (powder, refine_index) DO NOTHING;

INSERT INTO content_version (id, version, note)
VALUES (1, 8, 'SancRate efetivo: 28 linhas do arquivo + 8 defaults compilados (36 células)')
ON CONFLICT (id) DO UPDATE
    SET version = GREATEST(content_version.version, 8), note = EXCLUDED.note;

-- migrate:down
DELETE FROM sanc_rate
WHERE (powder = 'PO' AND refine_index BETWEEN 6 AND 11)
   OR (powder = 'PL' AND refine_index BETWEEN 10 AND 11);
