-- 0027_seed_combine: taxas, bônus, penalidades e custos da composição (+10..+15).
--
-- Regra: o handler não carrega número de economia. `crates/wyd-core/src/systems/compose.rs`
-- lê cada célula por (family, key) via `ComposeRules`; aqui ficam só os valores.
--
-- Fontes primárias:
--   origin/main:tmserver/content/Common/Settings/CompRate.txt  (arquivo de conteúdo)
--   origin/main:tmserver/internal/combine/{combine,combine_odin}.go  (código vivo)
--   origin/main:tmserver/internal/handler/combine*.go
-- Onde o código vivo diverge do arquivo, a nota diz qual número o servidor realmente usa.
-- Dívida registrada em `docs/wyd/sistemas/refino-auditoria.md` §3.

-- migrate:up
INSERT INTO combine_value (family, key, value, unit, content_source, confidence, notes) VALUES
    -- Aylin: taxa = BASE + 4 × CHANCE_BASE = 41%; custo cobrado antes do sorteio.
    ('Aylin', 'BASE',        1, 'percent', 'legacy:CompRate.txt (Ailyn Base)', 95,
     'termo fixo da taxa; o 4× vem do handler (4 pares de jóias), não da tabela'),
    ('Aylin', 'CHANCE_BASE', 10, 'percent', 'legacy:CompRate.txt (Ailyn ChanceBase 10)', 95,
     'taxa efetiva 1 + 4×10 = 41%'),
    ('Aylin', 'COST', 50000000, 'coin', 'legacy:handler/combine.go (50_000_000)', 95,
     'débito antes do sorteio: falha perde as jóias, a pedra e o dinheiro'),

    -- Compositor (Anct): base 1% + bônus pelo refino de cada sacrifício.
    ('Compositor', 'BASE',   1, 'percent', 'legacy:CompRate.txt (Compositor Base)', 90,
     'arquivo não publica a base; 1% é o valor lido pelo código vivo'),
    ('Compositor', 'ITEM_+7', 2, 'percent', 'legacy:CompRate.txt (Item_+7 2)', 95,
     'bônus por sacrifício em +7'),
    ('Compositor', 'ITEM_+8', 4, 'percent', 'legacy:CompRate.txt (Item_+8 4)', 95,
     'bônus por sacrifício em +8'),
    ('Compositor', 'ITEM_+9', 10, 'percent', 'legacy:CompRate.txt (Item_+9 10)', 95,
     'bônus por sacrifício em +9'),

    -- Odin: uma taxa por receita numerada (ids 0..11).
    ('Odin', 'SETS', 100, 'percent', 'legacy:g_pOdinRate[0]', 95,
     'fallback do GetMatchCombineOdin; soma rand()%5 no handler (0..4)'),
    ('Odin', 'CELESTIAL', 35, 'percent', 'legacy:g_pOdinRate[1]', 95,
     'código vivo usa 35; CompRate.txt diz 5 — prevalece o código'),
    ('Odin', 'PLUS12_BASE', 100, 'percent', 'legacy:g_pOdinRate[2] + Plus12Rate', 95,
     'base do ramo +12+; penalidade por nível e bônus de runa entram por baixo'),
    ('Odin', 'PISTA', 40, 'percent', 'legacy:g_pOdinRate[3]', 95,
     'Pista de Runas; CompRate.txt não lista esta receita'),
    ('Odin', 'DESTRAVE_LV40', 100, 'percent', 'legacy:g_pOdinRate[4]', 95,
     'destrave Lv40 celestial'),
    ('Odin', 'PEDRA_DA_FURIA', 100, 'percent', 'legacy:g_pOdinRate[5]', 95,
     'Pedra da Fúria'),
    ('Odin', 'SECRETA_AGUA', 100, 'percent', 'legacy:g_pOdinRate[6]', 95,
     'Pedra Secreta da Água'),
    ('Odin', 'SECRETA_TERRA', 100, 'percent', 'legacy:g_pOdinRate[7]', 95,
     'Pedra Secreta da Terra'),
    ('Odin', 'SECRETA_SOL', 100, 'percent', 'legacy:g_pOdinRate[8]', 95,
     'Pedra Secreta do Sol'),
    ('Odin', 'SECRETA_VENTO', 100, 'percent', 'legacy:g_pOdinRate[9]', 95,
     'Pedra Secreta do Vento'),
    ('Odin', 'SEMENTE_CRISTAL', 100, 'percent', 'legacy:g_pOdinRate[10]', 95,
     'Semente de Cristal'),
    ('Odin', 'CAPA_CELESTIAL', 100, 'percent', 'legacy:g_pOdinRate[11]', 95,
     'Capa Celestial'),

    -- Ramo +12+: bônus de runa/refinação e penalidade pelo refino do alvo.
    ('Odin', 'PLUS12_RUNE', 1, 'percent', 'legacy:Plus12Rate (runas 5334..5337)', 95,
     'por pedra Secreta presente (até 4)'),
    ('Odin', 'PLUS12_REF_0',  2, 'percent', 'legacy:CompRate.txt (Ref_0 2)', 95,
     'bônus da Refinação Abençoada pelo refino dela'),
    ('Odin', 'PLUS12_REF_1',  3, 'percent', 'legacy:CompRate.txt (Ref_1 3)', 95, ''),
    ('Odin', 'PLUS12_REF_2',  4, 'percent', 'legacy:CompRate.txt (Ref_2 4)', 95, ''),
    ('Odin', 'PLUS12_REF_3',  5, 'percent', 'legacy:CompRate.txt (Ref_3 5)', 95, ''),
    ('Odin', 'PLUS12_REF_4',  6, 'percent', 'legacy:CompRate.txt (Ref_4 6)', 95, ''),
    ('Odin', 'PLUS12_REF_5',  7, 'percent', 'legacy:CompRate.txt (Ref_5 7)', 95, ''),
    ('Odin', 'PLUS12_REF_6',  8, 'percent', 'legacy:CompRate.txt (Ref_6 8)', 95, ''),
    ('Odin', 'PLUS12_REF_7',  9, 'percent', 'legacy:CompRate.txt (Ref_7 9)', 95, ''),
    ('Odin', 'PLUS12_REF_8', 12, 'percent', 'legacy:CompRate.txt (Ref_8 12)', 95, ''),
    ('Odin', 'PLUS12_REF_9', 15, 'percent', 'legacy:CompRate.txt (Ref_9 15)', 95, ''),
    ('Odin', 'PLUS12_MINUS_12', 2, 'percent', 'legacy:CompRate.txt (Minus_12 2)', 95,
     'penalidade por alvo +12'),
    ('Odin', 'PLUS12_MINUS_13', 3, 'percent', 'legacy:CompRate.txt (Minus_13 3)', 95,
     'penalidade por alvo +13'),
    ('Odin', 'PLUS12_MINUS_14', 4, 'percent', 'legacy:CompRate.txt (Minus_14 4)', 95,
     'penalidade por alvo +14'),
    ('Odin', 'PLUS12_MINUS_15', 5, 'percent', 'legacy:CompRate.txt (Minus_15 5)', 95,
     'penalidade por alvo +15 (alvo +15 é recusado antes: não há +16)'),

    -- Passo de pó +10 → +11 (o único com pó na composição).
    ('Powder', 'GEM_LEVEL', 15, 'percent', 'legacy:RefineRate.txt (Lac) + handler/refine.go', 95,
     'Poeira de Lactolerium (413) em +10; a taxa do arquivo SancRate não vale aqui'),
    ('Powder', 'AMAGO', 100, 'percent', 'legacy:handler/refine.go (Lactolerium_100)', 95,
     'item 4141 força a linha do Âmago e o passo sobe sempre')
ON CONFLICT (family, key) DO NOTHING;

INSERT INTO content_version (id, version, note)
VALUES (1, 9, 'Composição: 36 valores de Aylin/Compositor/Odin/pó em combine_value')
ON CONFLICT (id) DO UPDATE
    SET version = GREATEST(content_version.version, 9), note = EXCLUDED.note;

-- migrate:down
-- Tabela nasceu vazia (`0009`); reverter o seed devolve o estado vazio, não apaga schema.
DELETE FROM combine_value;
