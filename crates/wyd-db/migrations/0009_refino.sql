-- 0009_refino: taxas de refino (+0..+15) e as 4 gemas de troca de efeito.
-- Convenção: -- migrate:up / -- migrate:down (docs/arquitetura/modelo-banco.md §4).
-- A REGRA (escala 40 × (nível − 9)) vive em `crates/wyd-sim/src/formula/refine.rs` e em
-- `crates/wyd-core` (autoridade). Aqui fica só o DADO, para portal, auditoria e seed.
-- Números e fonte: docs/wyd/sistemas/refino.md §2–§3.

-- migrate:up
CREATE TABLE refine_rate (
    step       SMALLINT PRIMARY KEY CHECK (step BETWEEN 1 AND 15),
    rate_bps   INTEGER NOT NULL CHECK (rate_bps BETWEEN 0 AND 10000),
    -- 'dust' = refino com pó; 'compose' = composição (+10, Anct, +12..+15)
    kind       TEXT NOT NULL CHECK (kind IN ('dust', 'compose')),
    content_source TEXT NOT NULL,
    confidence SMALLINT NOT NULL DEFAULT 90 CHECK (confidence BETWEEN 0 AND 100),
    notes      TEXT NOT NULL DEFAULT ''
);

COMMENT ON TABLE refine_rate IS
    'Taxa de sucesso por passo em basis points (9000 = 90%): dado, não fórmula.';
COMMENT ON COLUMN refine_rate.kind IS
    'dust: pó de Ori/Lac (+0..+9). compose: +10, Ancient e +12..+15 (custos em insumo).';

CREATE TABLE refine_gem_effect (
    item_index      INTEGER PRIMARY KEY REFERENCES item_catalog (item_index) ON DELETE CASCADE,
    effect_code     TEXT NOT NULL CHECK (effect_code IN ('drop', 'pierce', 'exp', 'absorb')),
    flat_value      SMALLINT NOT NULL DEFAULT 0 CHECK (flat_value >= 0),
    per_level_value SMALLINT NOT NULL DEFAULT 0 CHECK (per_level_value >= 0),
    min_refine      SMALLINT NOT NULL DEFAULT 10 CHECK (min_refine >= 0),
    max_refine      SMALLINT NOT NULL DEFAULT 15 CHECK (max_refine <= 15),
    allows_ancient  BOOLEAN NOT NULL DEFAULT true,
    content_source  TEXT NOT NULL,
    confidence      SMALLINT NOT NULL DEFAULT 90 CHECK (confidence BETWEEN 0 AND 100),
    notes           TEXT NOT NULL DEFAULT '',
    CHECK (flat_value > 0 OR per_level_value > 0)
);

COMMENT ON TABLE refine_gem_effect IS
    'Gemas de troca de efeito (item_catalog 3386..3389): item Anct ou +10..+15.';
COMMENT ON COLUMN refine_gem_effect.per_level_value IS
    'Valor somado por nível acima de +9 (40 → +10 … 240 → +15).';

-- SancRate do legado: sucesso do refino (%) por pó e índice atual de refino.
-- Fonte: origin/main:tmserver/content/Common/Settings/SancRate.txt
-- sha256 80e90f4a205c2323e5c68545e3c427c1c608665ff13372e59a58e7fc83dd718b
-- O índice é o refino atual (0 = +0); os tetos vêm de RefineRate.txt (PO_MAX 6, PL_MAX 9,
-- HARD_CAP 11). A série de PO para em 5, PL em 9 e só Âmago chega a 11 — ausência é dado.
CREATE TABLE sanc_rate (
    powder         TEXT NOT NULL CHECK (powder IN ('PO', 'PL', 'AMAGO')),
    refine_index   SMALLINT NOT NULL CHECK (refine_index BETWEEN 0 AND 15),
    percent        SMALLINT NOT NULL CHECK (percent BETWEEN 0 AND 100),
    content_source TEXT NOT NULL,
    confidence     SMALLINT NOT NULL DEFAULT 95 CHECK (confidence BETWEEN 0 AND 100),
    notes          TEXT NOT NULL DEFAULT '',
    PRIMARY KEY (powder, refine_index)
);

COMMENT ON TABLE sanc_rate IS
    'SancRate.txt do legado: sucesso do refino (%) por pó e índice atual de refino.';
COMMENT ON COLUMN sanc_rate.refine_index IS
    'Refino atual do item (0 = +0). PO cobre 0..5, PL 0..9, AMAGO 0..11 (teto de RefineRate.txt).';

-- Combinar/compor: TODO número que o handler lê (taxa, bônus, penalidade, custo) vem
-- daqui. O core (`crates/wyd-core/src/systems/compose.rs`) só carrega ids de item e o passo
-- `GEM_LEVEL`; trocar uma célula muda a economia sem recompilar.
-- Fonte: origin/main:tmserver/content/Common/Settings/CompRate.txt (+ `g_pOdinRate` do
-- código vivo, que diverge do arquivo — a nota de cada linha diz qual dos dois vale).
CREATE TABLE combine_value (
    family         TEXT NOT NULL CHECK (family <> ''),
    key            TEXT NOT NULL CHECK (key <> ''),
    value          INTEGER NOT NULL CHECK (value >= 0),
    -- 'percent' = pontos percentuais de sucesso; 'coin' = moedas cobradas do personagem.
    unit           TEXT NOT NULL CHECK (unit IN ('percent', 'coin')),
    content_source TEXT NOT NULL,
    confidence     SMALLINT NOT NULL DEFAULT 90 CHECK (confidence BETWEEN 0 AND 100),
    notes          TEXT NOT NULL DEFAULT '',
    PRIMARY KEY (family, key)
);

COMMENT ON TABLE combine_value IS
    'Composição: taxas/bônus/penalidades/custos lidos pelos handlers (Aylin, Compositor, Odin, pó).';
COMMENT ON COLUMN combine_value.key IS
    'Chave estável por família (BASE, CHANCE_BASE, COST, ITEM_+7, PLUS12_BASE, PLUS12_REF_n…).';

-- migrate:down
DROP TABLE combine_value;
DROP TABLE sanc_rate;
DROP TABLE refine_gem_effect;
DROP TABLE refine_rate;
