-- 0015_arena: Arena Real e Battle Royale (modelo B: 2 equipes por rounds).
-- Regras completas: docs/wyd/sistemas/battle-royale.md. Nada aqui é escrito pela web.

-- migrate:up
CREATE TABLE arena (
    code        TEXT PRIMARY KEY CHECK (code ~ '^[a-z0-9_]+$'),
    name        TEXT NOT NULL CHECK (name <> ''),
    kind        TEXT NOT NULL CHECK (kind IN ('royal', 'battle_royale')),
    team_size   SMALLINT NOT NULL DEFAULT 16 CHECK (team_size BETWEEN 1 AND 64),
    rounds      SMALLINT NOT NULL DEFAULT 3 CHECK (rounds BETWEEN 1 AND 9),
    round_ms    INTEGER NOT NULL DEFAULT 300000 CHECK (round_ms > 0),
    respawns    SMALLINT NOT NULL DEFAULT 1 CHECK (respawns BETWEEN 0 AND 9),
    min_level   SMALLINT NOT NULL DEFAULT 200 CHECK (min_level BETWEEN 1 AND 400),
    no_drop     BOOLEAN NOT NULL DEFAULT true,
    enabled     BOOLEAN NOT NULL DEFAULT false,
    content_source TEXT NOT NULL DEFAULT 'docs/wyd/sistemas/battle-royale.md'
);

CREATE TABLE arena_match (
    id          BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    arena_code  TEXT NOT NULL REFERENCES arena (code) ON DELETE CASCADE,
    started_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    ended_at    TIMESTAMPTZ,
    winner_team SMALLINT CHECK (winner_team IN (1, 2)),
    CONSTRAINT arena_match_ends_after_start CHECK (ended_at IS NULL OR ended_at >= started_at)
);

CREATE INDEX arena_match_arena_started_idx ON arena_match (arena_code, started_at DESC);

CREATE TABLE arena_round (
    match_id    BIGINT NOT NULL REFERENCES arena_match (id) ON DELETE CASCADE,
    round_no    SMALLINT NOT NULL CHECK (round_no BETWEEN 1 AND 9),
    started_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    ended_at    TIMESTAMPTZ,
    winner_team SMALLINT CHECK (winner_team IN (1, 2)),
    is_tiebreak BOOLEAN NOT NULL DEFAULT false,
    PRIMARY KEY (match_id, round_no),
    CONSTRAINT arena_round_ends_after_start CHECK (ended_at IS NULL OR ended_at >= started_at)
);

-- Placar por jogador/round. `kills`/`assists`/`damage` nascem do combate validado no servidor.
CREATE TABLE arena_score (
    match_id     BIGINT NOT NULL,
    round_no     SMALLINT NOT NULL,
    character_id BIGINT NOT NULL REFERENCES characters (id) ON DELETE CASCADE,
    team         SMALLINT NOT NULL CHECK (team IN (1, 2)),
    kills        SMALLINT NOT NULL DEFAULT 0 CHECK (kills >= 0),
    assists      SMALLINT NOT NULL DEFAULT 0 CHECK (assists >= 0),
    damage       BIGINT NOT NULL DEFAULT 0 CHECK (damage >= 0),
    deaths       SMALLINT NOT NULL DEFAULT 0 CHECK (deaths >= 0),
    PRIMARY KEY (match_id, round_no, character_id),
    FOREIGN KEY (match_id, round_no) REFERENCES arena_round (match_id, round_no) ON DELETE CASCADE
);

CREATE INDEX arena_score_character_idx ON arena_score (character_id);

-- migrate:down
DROP TABLE arena_score;
DROP TABLE arena_round;
DROP TABLE arena_match;
DROP TABLE arena;
