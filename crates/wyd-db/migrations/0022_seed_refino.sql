-- 0022_seed_refino: taxas de refino (9 passos de pó, +10 e +12..+15) e as 4 gemas de efeito.
-- GERADO à mão a partir de docs/wyd/sistemas/refino.md §2–§3 (fonte citada por linha).
-- Idempotente: rodar duas vezes não duplica nem altera valor.

-- migrate:up
INSERT INTO refine_rate (step, rate_bps, kind, content_source, confidence, notes) VALUES
    (1, 9000, 'dust', 'wyd-international:server-information', 95, '+0 → +1'),
    (2, 8500, 'dust', 'wyd-international:server-information', 95, '+1 → +2'),
    (3, 8000, 'dust', 'wyd-international:server-information', 95, '+2 → +3'),
    (4, 7500, 'dust', 'wyd-international:server-information', 95, '+3 → +4'),
    (5, 7000, 'dust', 'wyd-international:server-information', 95, '+4 → +5'),
    (6, 6500, 'dust', 'wyd-international:server-information', 95, '+5 → +6'),
    (7, 6000, 'dust', 'wyd-international:server-information', 95, '+6 → +7'),
    (8, 5500, 'dust', 'wyd-international:server-information', 95, '+7 → +8'),
    (9, 5000, 'dust', 'wyd-international:server-information', 95, '+8 → +9'),
    -- +10: composição (2 itens +9 iguais, 4 joias iguais, Pedra do Sábio, 50kk). O doc §2
    -- publica 30%; a faixa 30–40% é de outra configuração de servidor. Vale o 30% citado.
    (10, 3000, 'compose', 'docs/wyd/sistemas/refino.md §2', 85, '+9 → +10 (30% em composição)'),
    -- +11: NÃO é composição — é refino do item +10 com Poeira de Lactolerium (Lac).
    -- Wikis publicam 3% com PL normal (e 100% com "Lac 100%"); a central Kersef confirma
    -- o caminho ("refinar para +11 usando Lac Powder") sem publicar número. O SancRate do
    -- legado traz 5% na linha Âmago para o passo especial — outra configuração de servidor,
    -- registrada como divergência em vez de virar o valor do seed.
    (11, 300, 'dust', 'wikis WYD: PL +10 → +11', 85, '+10 → +11 (3% com Poeira de Lac; Lac 100% = 100%)'),
    (12, 400, 'compose', 'wyd-international:refining-12-15', 85, '+11 → +12 (4%)'),
    (13, 300, 'compose', 'wyd-international:refining-12-15', 85, '+12 → +13 (3%)'),
    (14, 200, 'compose', 'wyd-international:refining-12-15', 85, '+13 → +14 (2%)'),
    (15, 100, 'compose', 'wyd-international:refining-12-15', 85, '+14 → +15 (1%)')
ON CONFLICT (step) DO NOTHING;

INSERT INTO refine_gem_effect
    (item_index, effect_code, flat_value, per_level_value, min_refine, max_refine,
     allows_ancient, content_source, confidence, notes)
VALUES
    (3386, 'drop',   8, 0,  10, 15, true, 'kersef:premium-neil', 95,
     '+8% de drop; fixo, não escala com o refino.'),
    (3387, 'pierce', 0, 40, 10, 15, true, 'kersef:premium-neil', 95,
     'Perfuração: 40 por nível acima de +9 (40..240).'),
    (3388, 'exp',    2, 0,  10, 15, true, 'kersef:premium-neil', 95,
     '+2% de EXP; fixo, não escala com o refino.'),
    (3389, 'absorb', 0, 40, 10, 15, true, 'kersef:premium-neil', 95,
     'Absorção: 40 por nível acima de +9 (40..240).')
ON CONFLICT (item_index) DO NOTHING;

INSERT INTO content_version (id, version, note)
VALUES (1, 7, 'refino: 15 passos (9 de pó, +10, +11 e +12..+15) e gemas de efeito')
ON CONFLICT (id) DO UPDATE SET version = GREATEST(content_version.version, 5), note = EXCLUDED.note;

-- migrate:down
DELETE FROM refine_gem_effect;
DELETE FROM refine_rate;
