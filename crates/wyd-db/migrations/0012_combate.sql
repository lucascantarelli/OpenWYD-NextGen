-- 0012_combate: stats de monstro, equipamento do mob, drop e multiplicadores de conteúdo.
-- Drop usa PESO, não probabilidade absoluta: roll 1..1.000.000 por item, cada item com seu
-- roll (docs/wyd/sistemas/drops.md; `DropItem` é peso — decisoes-servidor.md §1).

-- migrate:up
CREATE TABLE mob_template_stat (
    template_name TEXT PRIMARY KEY REFERENCES npc_template (template_name) ON DELETE CASCADE,
    level         SMALLINT NOT NULL DEFAULT 1 CHECK (level BETWEEN 1 AND 400),
    max_hp        BIGINT NOT NULL DEFAULT 1 CHECK (max_hp > 0),
    attack        INTEGER NOT NULL DEFAULT 0 CHECK (attack >= 0),
    defense       INTEGER NOT NULL DEFAULT 0 CHECK (defense >= 0),
    absorb        INTEGER NOT NULL DEFAULT 0 CHECK (absorb >= 0),
    magic_attack  INTEGER NOT NULL DEFAULT 0 CHECK (magic_attack >= 0),
    magic_defense INTEGER NOT NULL DEFAULT 0 CHECK (magic_defense >= 0),
    exp_reward    BIGINT NOT NULL DEFAULT 0 CHECK (exp_reward >= 0),
    gold_min      BIGINT NOT NULL DEFAULT 0 CHECK (gold_min >= 0),
    gold_max      BIGINT NOT NULL DEFAULT 0 CHECK (gold_max >= gold_min),
    content_source TEXT NOT NULL DEFAULT 'content_source'
);

CREATE TABLE mob_template_equip (
    template_name TEXT NOT NULL REFERENCES npc_template (template_name) ON DELETE CASCADE,
    slot          SMALLINT NOT NULL CHECK (slot BETWEEN 0 AND 15),
    item_index    INTEGER NOT NULL REFERENCES item_catalog (item_index),
    PRIMARY KEY (template_name, slot)
);

-- Drop: peso relativo por (mob, item). `source_bps` documenta a taxa publicada quando há.
CREATE TABLE drop_rate_override (
    template_name TEXT NOT NULL REFERENCES npc_template (template_name) ON DELETE CASCADE,
    item_index    INTEGER NOT NULL REFERENCES item_catalog (item_index) ON DELETE CASCADE,
    weight        BIGINT NOT NULL CHECK (weight > 0),
    min_qty       SMALLINT NOT NULL DEFAULT 1 CHECK (min_qty > 0),
    max_qty       SMALLINT NOT NULL DEFAULT 1 CHECK (max_qty >= min_qty),
    content_source TEXT NOT NULL CHECK (content_source <> ''),
    confidence    SMALLINT NOT NULL DEFAULT 70 CHECK (confidence BETWEEN 0 AND 100),
    PRIMARY KEY (template_name, item_index)
);

COMMENT ON COLUMN drop_rate_override.weight IS
    'Peso comparado ao roll de 1..1.000.000; taxa efetiva = peso / soma dos pesos do mob.';

-- Multiplicadores globais (EXP, drop, ouro) — 10000 = 100%.
CREATE TABLE content_rate_override (
    kind        TEXT PRIMARY KEY CHECK (kind IN ('exp', 'drop', 'gold', 'skill_point')),
    rate_bps    INTEGER NOT NULL CHECK (rate_bps BETWEEN 0 AND 1000000),
    content_source TEXT NOT NULL DEFAULT 'game_setting',
    notes       TEXT NOT NULL DEFAULT ''
);

-- migrate:down
DROP TABLE content_rate_override;
DROP TABLE drop_rate_override;
DROP TABLE mob_template_equip;
DROP TABLE mob_template_stat;
