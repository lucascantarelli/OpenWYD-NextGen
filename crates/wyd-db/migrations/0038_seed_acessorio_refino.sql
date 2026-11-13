-- 0038_seed_acessorio_refino: regras publicadas de refino de acessório.
--
-- Fonte principal (aberta): WYD Global — Bahamut Accessories Guide,
-- https://wydglobal.raidhut.com/pt-br/3390 (também em /en/3391).
-- Fonte de contexto (aberta): central Kersef, artigo "Sistema de Refinação +0 a +11",
-- https://feedback.kersef.com/pt-br/knowledge-bases/21/articles/21072 — no sistema original
-- acessórios não podem ser combinados/refinados; o refino de acessório é sistema posterior.
--
-- O que o guia publica:
--   brinco   +0..+9 40% por passo; +9..+11 5%; falha mantém o refino e dá 10% de Bahamut Tear;
--            +11..+15 no NPC Torvald: 20/15/10/5%; custo 10kk + cristal + poeira.
--   colar/cinto, receita 1 Lac: +0..+6 60%; +6..+11 30%; falha volta a +0.
--   colar/cinto, receita 100 Lac: +0..+5 75%; +5..+7 50%; +7..+10 25%; +10..+11 10%;
--            falha mantém o refino.
-- Amuleto, ankh e pedra espiritual não têm regra publicada na era oficial nem no guia
-- aberto do Bahamut: ficam sem linha (o portal mostra "sem regra") em vez de número inventado.

-- migrate:up
-- Brinco Bahamut — receita de 1 Poeira de Lactolerium + Bahamut Crystal + 10kk.
INSERT INTO accessory_refine_rule
    (kind, recipe, step_from, step_to, rate_bps, fail_mode, fail_bonus_bps,
     gold_cost, material_item, material_name, material_qty, content_source, confidence, notes)
VALUES
    ('earring', 'lac1', 0, 1, 4000, 'keep', 1000, 10000000, NULL, 'Bahamut Crystal + Poeira de Lactolerium', 1,
     'wydglobal/3390', 85, 'faixa +0..+9 publicada como 40%'),
    ('earring', 'lac1', 1, 2, 4000, 'keep', 1000, 10000000, NULL, 'Bahamut Crystal + Poeira de Lactolerium', 1,
     'wydglobal/3390', 85, 'faixa +0..+9 publicada como 40%'),
    ('earring', 'lac1', 2, 3, 4000, 'keep', 1000, 10000000, NULL, 'Bahamut Crystal + Poeira de Lactolerium', 1,
     'wydglobal/3390', 85, 'faixa +0..+9 publicada como 40%'),
    ('earring', 'lac1', 3, 4, 4000, 'keep', 1000, 10000000, NULL, 'Bahamut Crystal + Poeira de Lactolerium', 1,
     'wydglobal/3390', 85, 'faixa +0..+9 publicada como 40%'),
    ('earring', 'lac1', 4, 5, 4000, 'keep', 1000, 10000000, NULL, 'Bahamut Crystal + Poeira de Lactolerium', 1,
     'wydglobal/3390', 85, 'faixa +0..+9 publicada como 40%'),
    ('earring', 'lac1', 5, 6, 4000, 'keep', 1000, 10000000, NULL, 'Bahamut Crystal + Poeira de Lactolerium', 1,
     'wydglobal/3390', 85, 'faixa +0..+9 publicada como 40%'),
    ('earring', 'lac1', 6, 7, 4000, 'keep', 1000, 10000000, NULL, 'Bahamut Crystal + Poeira de Lactolerium', 1,
     'wydglobal/3390', 85, 'faixa +0..+9 publicada como 40%'),
    ('earring', 'lac1', 7, 8, 4000, 'keep', 1000, 10000000, NULL, 'Bahamut Crystal + Poeira de Lactolerium', 1,
     'wydglobal/3390', 85, 'faixa +0..+9 publicada como 40%'),
    ('earring', 'lac1', 8, 9, 4000, 'keep', 1000, 10000000, NULL, 'Bahamut Crystal + Poeira de Lactolerium', 1,
     'wydglobal/3390', 85, 'faixa +0..+9 publicada como 40%'),
    ('earring', 'lac1', 9, 10, 500, 'keep', 1000, 10000000, NULL, 'Bahamut Crystal + Poeira de Lactolerium', 1,
     'wydglobal/3390', 85, 'faixa +9..+11 publicada como 5%'),
    ('earring', 'lac1', 10, 11, 500, 'keep', 1000, 10000000, NULL, 'Bahamut Crystal + Poeira de Lactolerium', 1,
     'wydglobal/3390', 85, 'faixa +9..+11 publicada como 5%'),
    ('earring', 'torvald', 11, 12, 2000, 'keep', 1000, 10000000, NULL, 'Bahamut Crystal + Poeira de Lactolerium', 1,
     'wydglobal/3390', 85, 'NPC Torvald; falha devolve +11'),
    ('earring', 'torvald', 12, 13, 1500, 'keep', 1000, 10000000, NULL, 'Bahamut Crystal + Poeira de Lactolerium', 1,
     'wydglobal/3390', 85, 'NPC Torvald; falha devolve +11'),
    ('earring', 'torvald', 13, 14, 1000, 'keep', 1000, 10000000, NULL, 'Bahamut Crystal + Poeira de Lactolerium', 1,
     'wydglobal/3390', 85, 'NPC Torvald; falha devolve +11'),
    ('earring', 'torvald', 14, 15, 500, 'keep', 1000, 10000000, NULL, 'Bahamut Crystal + Poeira de Lactolerium', 1,
     'wydglobal/3390', 85, 'NPC Torvald; falha devolve +11'),

    -- Colar/cinto — receita de 1 Poeira de Lactolerium (falha volta a +0).
    ('necklace', 'lac1', 0, 1, 6000, 'reset', 0, 10000000, NULL, 'Poeira de Lactolerium', 1,
     'wydglobal/3390', 80, 'faixa +0..+6 60%'),
    ('necklace', 'lac1', 1, 2, 6000, 'reset', 0, 10000000, NULL, 'Poeira de Lactolerium', 1,
     'wydglobal/3390', 80, 'faixa +0..+6 60%'),
    ('necklace', 'lac1', 2, 3, 6000, 'reset', 0, 10000000, NULL, 'Poeira de Lactolerium', 1,
     'wydglobal/3390', 80, 'faixa +0..+6 60%'),
    ('necklace', 'lac1', 3, 4, 6000, 'reset', 0, 10000000, NULL, 'Poeira de Lactolerium', 1,
     'wydglobal/3390', 80, 'faixa +0..+6 60%'),
    ('necklace', 'lac1', 4, 5, 6000, 'reset', 0, 10000000, NULL, 'Poeira de Lactolerium', 1,
     'wydglobal/3390', 80, 'faixa +0..+6 60%'),
    ('necklace', 'lac1', 5, 6, 6000, 'reset', 0, 10000000, NULL, 'Poeira de Lactolerium', 1,
     'wydglobal/3390', 80, 'faixa +0..+6 60%'),
    ('necklace', 'lac1', 6, 7, 3000, 'reset', 0, 10000000, NULL, 'Poeira de Lactolerium', 1,
     'wydglobal/3390', 80, 'faixa +6..+11 30%'),
    ('necklace', 'lac1', 7, 8, 3000, 'reset', 0, 10000000, NULL, 'Poeira de Lactolerium', 1,
     'wydglobal/3390', 80, 'faixa +6..+11 30%'),
    ('necklace', 'lac1', 8, 9, 3000, 'reset', 0, 10000000, NULL, 'Poeira de Lactolerium', 1,
     'wydglobal/3390', 80, 'faixa +6..+11 30%'),
    ('necklace', 'lac1', 9, 10, 3000, 'reset', 0, 10000000, NULL, 'Poeira de Lactolerium', 1,
     'wydglobal/3390', 80, 'faixa +6..+11 30%'),
    ('necklace', 'lac1', 10, 11, 3000, 'reset', 0, 10000000, NULL, 'Poeira de Lactolerium', 1,
     'wydglobal/3390', 80, 'faixa +6..+11 30%'),

    -- Colar/cinto — receita de 100 Poeira de Lactolerium (falha mantém o refino).
    ('necklace', 'lac100', 0, 1, 7500, 'keep', 0, 10000000, NULL, 'Poeira de Lactolerium', 100,
     'wydglobal/3390', 80, 'faixa +0..+5 75%'),
    ('necklace', 'lac100', 1, 2, 7500, 'keep', 0, 10000000, NULL, 'Poeira de Lactolerium', 100,
     'wydglobal/3390', 80, 'faixa +0..+5 75%'),
    ('necklace', 'lac100', 2, 3, 7500, 'keep', 0, 10000000, NULL, 'Poeira de Lactolerium', 100,
     'wydglobal/3390', 80, 'faixa +0..+5 75%'),
    ('necklace', 'lac100', 3, 4, 7500, 'keep', 0, 10000000, NULL, 'Poeira de Lactolerium', 100,
     'wydglobal/3390', 80, 'faixa +0..+5 75%'),
    ('necklace', 'lac100', 4, 5, 7500, 'keep', 0, 10000000, NULL, 'Poeira de Lactolerium', 100,
     'wydglobal/3390', 80, 'faixa +0..+5 75%'),
    ('necklace', 'lac100', 5, 6, 5000, 'keep', 0, 10000000, NULL, 'Poeira de Lactolerium', 100,
     'wydglobal/3390', 80, 'faixa +5..+7 50%'),
    ('necklace', 'lac100', 6, 7, 5000, 'keep', 0, 10000000, NULL, 'Poeira de Lactolerium', 100,
     'wydglobal/3390', 80, 'faixa +5..+7 50%'),
    ('necklace', 'lac100', 7, 8, 2500, 'keep', 0, 10000000, NULL, 'Poeira de Lactolerium', 100,
     'wydglobal/3390', 80, 'faixa +7..+10 25%'),
    ('necklace', 'lac100', 8, 9, 2500, 'keep', 0, 10000000, NULL, 'Poeira de Lactolerium', 100,
     'wydglobal/3390', 80, 'faixa +7..+10 25%'),
    ('necklace', 'lac100', 9, 10, 2500, 'keep', 0, 10000000, NULL, 'Poeira de Lactolerium', 100,
     'wydglobal/3390', 80, 'faixa +7..+10 25%'),
    ('necklace', 'lac100', 10, 11, 1000, 'keep', 0, 10000000, NULL, 'Poeira de Lactolerium', 100,
     'wydglobal/3390', 80, 'faixa +10..+11 10%'),

    -- Cinto segue as mesmas duas receitas do colar no guia aberto.
    ('belt', 'lac1', 0, 1, 6000, 'reset', 0, 10000000, NULL, 'Poeira de Lactolerium', 1,
     'wydglobal/3390', 75, 'mesma receita do colar'),
    ('belt', 'lac1', 1, 2, 6000, 'reset', 0, 10000000, NULL, 'Poeira de Lactolerium', 1,
     'wydglobal/3390', 75, 'mesma receita do colar'),
    ('belt', 'lac1', 2, 3, 6000, 'reset', 0, 10000000, NULL, 'Poeira de Lactolerium', 1,
     'wydglobal/3390', 75, 'mesma receita do colar'),
    ('belt', 'lac1', 3, 4, 6000, 'reset', 0, 10000000, NULL, 'Poeira de Lactolerium', 1,
     'wydglobal/3390', 75, 'mesma receita do colar'),
    ('belt', 'lac1', 4, 5, 6000, 'reset', 0, 10000000, NULL, 'Poeira de Lactolerium', 1,
     'wydglobal/3390', 75, 'mesma receita do colar'),
    ('belt', 'lac1', 5, 6, 6000, 'reset', 0, 10000000, NULL, 'Poeira de Lactolerium', 1,
     'wydglobal/3390', 75, 'mesma receita do colar'),
    ('belt', 'lac1', 6, 7, 3000, 'reset', 0, 10000000, NULL, 'Poeira de Lactolerium', 1,
     'wydglobal/3390', 75, 'mesma receita do colar'),
    ('belt', 'lac1', 7, 8, 3000, 'reset', 0, 10000000, NULL, 'Poeira de Lactolerium', 1,
     'wydglobal/3390', 75, 'mesma receita do colar'),
    ('belt', 'lac1', 8, 9, 3000, 'reset', 0, 10000000, NULL, 'Poeira de Lactolerium', 1,
     'wydglobal/3390', 75, 'mesma receita do colar'),
    ('belt', 'lac1', 9, 10, 3000, 'reset', 0, 10000000, NULL, 'Poeira de Lactolerium', 1,
     'wydglobal/3390', 75, 'mesma receita do colar'),
    ('belt', 'lac1', 10, 11, 3000, 'reset', 0, 10000000, NULL, 'Poeira de Lactolerium', 1,
     'wydglobal/3390', 75, 'mesma receita do colar'),
    ('belt', 'lac100', 0, 1, 7500, 'keep', 0, 10000000, NULL, 'Poeira de Lactolerium', 100,
     'wydglobal/3390', 75, 'mesma receita do colar'),
    ('belt', 'lac100', 1, 2, 7500, 'keep', 0, 10000000, NULL, 'Poeira de Lactolerium', 100,
     'wydglobal/3390', 75, 'mesma receita do colar'),
    ('belt', 'lac100', 2, 3, 7500, 'keep', 0, 10000000, NULL, 'Poeira de Lactolerium', 100,
     'wydglobal/3390', 75, 'mesma receita do colar'),
    ('belt', 'lac100', 3, 4, 7500, 'keep', 0, 10000000, NULL, 'Poeira de Lactolerium', 100,
     'wydglobal/3390', 75, 'mesma receita do colar'),
    ('belt', 'lac100', 4, 5, 7500, 'keep', 0, 10000000, NULL, 'Poeira de Lactolerium', 100,
     'wydglobal/3390', 75, 'mesma receita do colar'),
    ('belt', 'lac100', 5, 6, 5000, 'keep', 0, 10000000, NULL, 'Poeira de Lactolerium', 100,
     'wydglobal/3390', 75, 'mesma receita do colar'),
    ('belt', 'lac100', 6, 7, 5000, 'keep', 0, 10000000, NULL, 'Poeira de Lactolerium', 100,
     'wydglobal/3390', 75, 'mesma receita do colar'),
    ('belt', 'lac100', 7, 8, 2500, 'keep', 0, 10000000, NULL, 'Poeira de Lactolerium', 100,
     'wydglobal/3390', 75, 'mesma receita do colar'),
    ('belt', 'lac100', 8, 9, 2500, 'keep', 0, 10000000, NULL, 'Poeira de Lactolerium', 100,
     'wydglobal/3390', 75, 'mesma receita do colar'),
    ('belt', 'lac100', 9, 10, 2500, 'keep', 0, 10000000, NULL, 'Poeira de Lactolerium', 100,
     'wydglobal/3390', 75, 'mesma receita do colar'),
    ('belt', 'lac100', 10, 11, 1000, 'keep', 0, 10000000, NULL, 'Poeira de Lactolerium', 100,
     'wydglobal/3390', 75, 'mesma receita do colar'),

    -- +11..+15 no Torvald também para colar e cinto (mesma escada do brinco).
    ('necklace', 'torvald', 11, 12, 2000, 'keep', 0, 10000000, NULL, 'Amunra Crystal', 1,
     'wydglobal/3390', 80, 'NPC Torvald; +11..+15 usa Amunra Crystal; falha devolve +11'),
    ('necklace', 'torvald', 12, 13, 1500, 'keep', 0, 10000000, NULL, 'Amunra Crystal', 1,
     'wydglobal/3390', 80, 'NPC Torvald; +11..+15 usa Amunra Crystal; falha devolve +11'),
    ('necklace', 'torvald', 13, 14, 1000, 'keep', 0, 10000000, NULL, 'Amunra Crystal', 1,
     'wydglobal/3390', 80, 'NPC Torvald; +11..+15 usa Amunra Crystal; falha devolve +11'),
    ('necklace', 'torvald', 14, 15, 500, 'keep', 0, 10000000, NULL, 'Amunra Crystal', 1,
     'wydglobal/3390', 80, 'NPC Torvald; +11..+15 usa Amunra Crystal; falha devolve +11'),
    ('belt', 'torvald', 11, 12, 2000, 'keep', 0, 10000000, NULL, 'Amunra Crystal', 1,
     'wydglobal/3390', 75, 'NPC Torvald; +11..+15 usa Amunra Crystal; falha devolve +11'),
    ('belt', 'torvald', 12, 13, 1500, 'keep', 0, 10000000, NULL, 'Amunra Crystal', 1,
     'wydglobal/3390', 75, 'NPC Torvald; +11..+15 usa Amunra Crystal; falha devolve +11'),
    ('belt', 'torvald', 13, 14, 1000, 'keep', 0, 10000000, NULL, 'Amunra Crystal', 1,
     'wydglobal/3390', 75, 'NPC Torvald; +11..+15 usa Amunra Crystal; falha devolve +11'),
    ('belt', 'torvald', 14, 15, 500, 'keep', 0, 10000000, NULL, 'Amunra Crystal', 1,
     'wydglobal/3390', 75, 'NPC Torvald; +11..+15 usa Amunra Crystal; falha devolve +11')
ON CONFLICT (kind, recipe, step_from) DO NOTHING;

-- migrate:down
DELETE FROM accessory_refine_rule;
