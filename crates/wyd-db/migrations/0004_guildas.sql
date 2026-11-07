-- 0004_guildas: clãs e membros (um clã por personagem).
-- Convenção: -- migrate:up / -- migrate:down (ver docs/arquitetura/modelo-banco.md §4).

-- migrate:up
CREATE TABLE guilds (
    id          BIGSERIAL PRIMARY KEY,
    name        TEXT NOT NULL,
    master_character_id BIGINT REFERENCES characters (id) DEFERRABLE INITIALLY DEFERRED,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT guilds_name_nonempty CHECK (char_length(name) > 0)
);
CREATE UNIQUE INDEX guilds_name_unique ON guilds (name);

CREATE TABLE guild_members (
    guild_id     BIGINT NOT NULL REFERENCES guilds (id) ON DELETE CASCADE,
    character_id BIGINT NOT NULL REFERENCES characters (id) ON DELETE CASCADE,
    rank         SMALLINT NOT NULL DEFAULT 0,
    joined_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (guild_id, character_id),
    CONSTRAINT guild_members_rank CHECK (rank BETWEEN 0 AND 9)
);
CREATE UNIQUE INDEX guild_members_character_unique ON guild_members (character_id);

-- migrate:down
DROP INDEX guild_members_character_unique;
DROP TABLE guild_members;
DROP INDEX guilds_name_unique;
DROP TABLE guilds;
