-- 0013_quests: quest, passo e recompensa (cadeia principal, classe, diária e evento).
-- Domínio: docs/wyd/sistemas/quests.md. Texto de diálogo do 7662 é escasso e não vira
-- tabela de fala aqui; o que existe é objetivo, alvo e recompensa.

-- migrate:up
CREATE TABLE quest (
    code        TEXT PRIMARY KEY CHECK (code ~ '^[a-z0-9_]+$'),
    name        TEXT NOT NULL CHECK (name <> ''),
    kind        TEXT NOT NULL CHECK (kind IN ('main', 'class', 'daily', 'event', 'sub')),
    min_level   SMALLINT NOT NULL DEFAULT 1 CHECK (min_level BETWEEN 1 AND 400),
    class_code  TEXT REFERENCES game_class (code),
    repeatable  BOOLEAN NOT NULL DEFAULT false,
    cooldown_seconds INTEGER NOT NULL DEFAULT 0 CHECK (cooldown_seconds >= 0),
    requires    TEXT[] NOT NULL DEFAULT '{}',
    description TEXT NOT NULL DEFAULT '',
    content_source TEXT NOT NULL CHECK (content_source <> ''),
    confidence  SMALLINT NOT NULL DEFAULT 80 CHECK (confidence BETWEEN 0 AND 100)
);

CREATE INDEX quest_kind_level_idx ON quest (kind, min_level);

CREATE TABLE quest_step (
    quest_code  TEXT NOT NULL REFERENCES quest (code) ON DELETE CASCADE,
    step        SMALLINT NOT NULL CHECK (step > 0),
    objective   TEXT NOT NULL CHECK (objective IN (
        'kill', 'collect', 'talk', 'reach', 'refine', 'compose', 'use_item', 'escort'
    )),
    target_id   INTEGER,
    count       INTEGER NOT NULL DEFAULT 1 CHECK (count > 0),
    map_id      SMALLINT REFERENCES map_definition (map_id),
    pos_x       SMALLINT CHECK (pos_x IS NULL OR pos_x BETWEEN 0 AND 4095),
    pos_y       SMALLINT CHECK (pos_y IS NULL OR pos_y BETWEEN 0 AND 4095),
    description TEXT NOT NULL DEFAULT '',
    PRIMARY KEY (quest_code, step),
    CONSTRAINT quest_step_reach_needs_position CHECK (
        objective <> 'reach' OR (map_id IS NOT NULL AND pos_x IS NOT NULL AND pos_y IS NOT NULL)
    )
);

CREATE TABLE quest_reward (
    quest_code TEXT NOT NULL REFERENCES quest (code) ON DELETE CASCADE,
    kind       TEXT NOT NULL CHECK (kind IN ('item', 'gold', 'exp', 'fame', 'skill_point')),
    item_index INTEGER REFERENCES item_catalog (item_index),
    amount     BIGINT NOT NULL DEFAULT 1 CHECK (amount > 0),
    PRIMARY KEY (quest_code, kind, amount, item_index),
    CONSTRAINT quest_reward_item_needs_index CHECK (kind <> 'item' OR item_index IS NOT NULL)
);

-- Progresso por personagem: a web nunca escreve aqui (só o servidor drena e grava).
CREATE TABLE character_quest (
    character_id BIGINT NOT NULL REFERENCES characters (id) ON DELETE CASCADE,
    quest_code   TEXT NOT NULL REFERENCES quest (code) ON DELETE CASCADE,
    step         SMALLINT NOT NULL DEFAULT 1 CHECK (step > 0),
    progress     INTEGER NOT NULL DEFAULT 0 CHECK (progress >= 0),
    completed_at TIMESTAMPTZ,
    started_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (character_id, quest_code)
);

-- migrate:down
DROP TABLE character_quest;
DROP TABLE quest_reward;
DROP TABLE quest_step;
DROP TABLE quest;
