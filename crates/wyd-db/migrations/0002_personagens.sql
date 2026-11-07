-- 0002_personagens: personagens com soft-delete (nome/slot liberados ao apagar).
-- Convenção: -- migrate:up / -- migrate:down (ver docs/arquitetura/modelo-banco.md §4).

-- migrate:up
CREATE TABLE characters (
    id          BIGSERIAL PRIMARY KEY,
    account_id  BIGINT NOT NULL REFERENCES accounts (id) ON DELETE CASCADE,
    slot        SMALLINT NOT NULL,
    name        TEXT NOT NULL,
    class       SMALLINT NOT NULL DEFAULT 0,
    level       INTEGER NOT NULL DEFAULT 1,
    exp         BIGINT NOT NULL DEFAULT 0,
    hp          INTEGER NOT NULL DEFAULT 1,
    mp          INTEGER NOT NULL DEFAULT 0,
    pos_x       INTEGER NOT NULL DEFAULT 0,
    pos_y       INTEGER NOT NULL DEFAULT 0,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at  TIMESTAMPTZ,
    CONSTRAINT characters_slot CHECK (slot BETWEEN 0 AND 3),
    CONSTRAINT characters_name_nonempty CHECK (char_length(name) > 0),
    CONSTRAINT characters_level CHECK (level >= 1),
    CONSTRAINT characters_exp CHECK (exp >= 0),
    CONSTRAINT characters_hp CHECK (hp >= 0),
    CONSTRAINT characters_mp CHECK (mp >= 0)
);
CREATE INDEX characters_account_idx ON characters (account_id);
CREATE UNIQUE INDEX characters_name_unique ON characters (name) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX characters_account_slot_unique
    ON characters (account_id, slot) WHERE deleted_at IS NULL;

-- migrate:down
DROP INDEX characters_account_slot_unique;
DROP INDEX characters_name_unique;
DROP INDEX characters_account_idx;
DROP TABLE characters;
