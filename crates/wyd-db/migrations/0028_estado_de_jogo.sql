-- 0028_estado_de_jogo: o estado de jogo que faltava reidratar — buffs, preço da capa de reino,
-- recompensa diária, relações de clã e dono da GTorre.
-- Decisão de escopo (quais tabelas legadas são jogo e quais são portal) em
-- `docs/arquitetura/modelo-banco.md` §8; origem e consumidores do legado citados lá.
-- Exceção consciente à ordem "DDL antes de seed" (§6.5): o bloco 0017–0027 é de seed e estas
-- tabelas não são usadas por nenhum seed anterior (provado por `tests/catalogo.rs`).

-- migrate:up
-- Buffs/debuffs ativos por personagem: legado `affect` (`affect[char][32]`, savefmt.go offset 4572).
-- `type`/`value`/`time` mantêm a semântica do legado sem inventar faixa (UNVERIFIED).
CREATE TABLE character_affects (
    character_id BIGINT NOT NULL REFERENCES characters (id) ON DELETE CASCADE,
    type         SMALLINT NOT NULL,
    value        SMALLINT NOT NULL,
    level        INTEGER NOT NULL DEFAULT 0 CHECK (level >= 0),
    time         BIGINT NOT NULL DEFAULT 0 CHECK (time >= 0),
    PRIMARY KEY (character_id, type)
);

COMMENT ON TABLE character_affects IS
    'Buff/debuff persistido por personagem (legado affect); um por type, reidratado no login.';

-- Preço da capa de reino: legado `sapphire_balance` — apesar do nome, as colunas são preço
-- (`hekalotia_cost`/`akelonia_cost`, CHECK 4..32) e `version` é a revisão otimista usada por
-- `PurchaseKingdomCape(expectedRevision)`.
CREATE TABLE kingdom_cape_price (
    id             SMALLINT PRIMARY KEY DEFAULT 1,
    version        BIGINT NOT NULL DEFAULT 1 CHECK (version >= 1),
    hekalotia_cost INTEGER NOT NULL DEFAULT 8 CHECK (hekalotia_cost BETWEEN 4 AND 32),
    akelonia_cost  INTEGER NOT NULL DEFAULT 8 CHECK (akelonia_cost BETWEEN 4 AND 32),
    CONSTRAINT kingdom_cape_price_singleton CHECK (id = 1)
);

COMMENT ON TABLE kingdom_cape_price IS
    'Linha única: preço da capa por reino + versão para concorrência otimista (legado sapphire_balance).';

-- Recompensa diária: legado `daily_reward_claim` garante um resgate por conta por dia UTC
-- (unique_violation 23505 era o sinal de "já resgatou"); o item entregue vai por delivery_queue.
CREATE TABLE daily_reward_claim (
    id             BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    account_id     BIGINT NOT NULL REFERENCES accounts (id) ON DELETE CASCADE,
    reward_item_id BIGINT,
    claim_date     DATE NOT NULL,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT daily_reward_claim_once_per_day UNIQUE (account_id, claim_date)
);

COMMENT ON TABLE daily_reward_claim IS
    'Um resgate por conta e dia UTC (idempotência); reward_item_id referencia daily_reward_item por convenção.';

-- Relações entre clãs: legado `guild_relation`, `kind` 1 e 2 (aliado/hostil) com CHECK de
-- auto-relação proibida.
CREATE TABLE guild_relations (
    guild_id        BIGINT NOT NULL REFERENCES guilds (id) ON DELETE CASCADE,
    target_guild_id BIGINT NOT NULL REFERENCES guilds (id) ON DELETE CASCADE,
    kind            SMALLINT NOT NULL,
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (guild_id, target_guild_id),
    CONSTRAINT guild_relations_kind CHECK (kind IN (1, 2)),
    CONSTRAINT guild_relations_distinct CHECK (guild_id <> target_guild_id)
);

CREATE INDEX guild_relations_target_idx ON guild_relations (target_guild_id);

COMMENT ON TABLE guild_relations IS
    'Aliança/hostilidade entre clãs (legado guild_relation, kind 1/2), sem auto-relação.';

-- Dono da GTorre: legado `guild_tower_state`, linha única; `0` significa sem dono (por isso
-- não há FK — o sentinela não é um clã).
CREATE TABLE guild_tower (
    id              SMALLINT PRIMARY KEY DEFAULT 1,
    owner_guild     BIGINT NOT NULL DEFAULT 0 CHECK (owner_guild >= 0),
    updated_at_unix BIGINT NOT NULL DEFAULT 0 CHECK (updated_at_unix >= 0),
    CONSTRAINT guild_tower_singleton CHECK (id = 1)
);

COMMENT ON TABLE guild_tower IS
    'Linha única: clã dono da GTorre (0 = sem dono) e instante da última troca (legado guild_tower_state).';

-- migrate:down
DROP TABLE guild_tower;
DROP INDEX guild_relations_target_idx;
DROP TABLE guild_relations;
DROP TABLE daily_reward_claim;
DROP TABLE kingdom_cape_price;
DROP TABLE character_affects;
