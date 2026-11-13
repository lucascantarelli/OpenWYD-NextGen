-- 0014_eventos: catálogo de eventos, agenda, estado do Kefra e do castelo (Noatun).
-- Regras oficiais e decisões nossas: docs/wyd/decisoes-servidor.md §4
-- (Kefra sábado; Noatun quinzenal em 3 rounds; agenda é nossa e vive em `event_schedule`).

-- migrate:up
CREATE TABLE world_event (
    code     TEXT PRIMARY KEY CHECK (code ~ '^[a-z0-9_]+$'),
    name     TEXT NOT NULL CHECK (name <> ''),
    kind     TEXT NOT NULL CHECK (kind IN (
        'kefra', 'noatun', 'city_war', 'rvr', 'tower', 'battle_royale', 'custom'
    )),
    enabled  BOOLEAN NOT NULL DEFAULT true,
    requires_city BOOLEAN NOT NULL DEFAULT false,
    description TEXT NOT NULL DEFAULT '',
    content_source TEXT NOT NULL DEFAULT 'docs/wyd/sistemas/eventos.md'
);

-- Agenda: dia da semana + hora local do servidor. `timezone` explícito evita o evento
-- "andar" quando o host muda de fuso (decisoes-servidor.md §4).
CREATE TABLE event_schedule (
    id            INTEGER GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    event_code    TEXT NOT NULL REFERENCES world_event (code) ON DELETE CASCADE,
    weekday       SMALLINT NOT NULL CHECK (weekday BETWEEN 0 AND 6),
    start_time    TIME NOT NULL,
    duration_minutes SMALLINT NOT NULL CHECK (duration_minutes > 0),
    timezone      TEXT NOT NULL DEFAULT 'America/Sao_Paulo',
    every_n_weeks SMALLINT NOT NULL DEFAULT 1 CHECK (every_n_weeks BETWEEN 1 AND 8),
    enabled       BOOLEAN NOT NULL DEFAULT true,
    UNIQUE (event_code, weekday, start_time)
);

COMMENT ON COLUMN event_schedule.every_n_weeks IS
    '1 = semanal; 2 = quinzenal (Noatun oficial).';

CREATE TABLE kefra_state (
    id            SMALLINT PRIMARY KEY CHECK (id = 1),
    last_kill_at  TIMESTAMPTZ,
    next_spawn_at TIMESTAMPTZ,
    kill_count    INTEGER NOT NULL DEFAULT 0 CHECK (kill_count >= 0),
    last_guild_id BIGINT REFERENCES guilds (id) ON DELETE SET NULL
);

CREATE TABLE castle_state (
    id            SMALLINT PRIMARY KEY CHECK (id = 1),
    owner_guild_id BIGINT REFERENCES guilds (id) ON DELETE SET NULL,
    city_code     TEXT REFERENCES city (code),
    vault_gold    BIGINT NOT NULL DEFAULT 0 CHECK (vault_gold >= 0),
    last_collected_at TIMESTAMPTZ,
    last_war_at   TIMESTAMPTZ
);

CREATE TABLE event_participation (
    id           BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    event_code   TEXT NOT NULL REFERENCES world_event (code) ON DELETE CASCADE,
    character_id BIGINT REFERENCES characters (id) ON DELETE SET NULL,
    guild_id     BIGINT REFERENCES guilds (id) ON DELETE SET NULL,
    result       TEXT NOT NULL CHECK (result IN ('win', 'lose', 'draw', 'participated')),
    score        BIGINT NOT NULL DEFAULT 0,
    at           TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX event_participation_event_at_idx ON event_participation (event_code, at);

-- migrate:down
DROP TABLE event_participation;
DROP TABLE castle_state;
DROP TABLE kefra_state;
DROP TABLE event_schedule;
DROP TABLE world_event;
