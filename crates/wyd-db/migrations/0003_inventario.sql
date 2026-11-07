-- 0003_inventario: itens de personagem e baú da conta (normalizado, anti-dupe).
-- Convenção: -- migrate:up / -- migrate:down (ver docs/arquitetura/modelo-banco.md §4).

-- migrate:up
CREATE TABLE character_items (
    character_id BIGINT NOT NULL REFERENCES characters (id) ON DELETE CASCADE,
    slot        SMALLINT NOT NULL,
    item_id     INTEGER NOT NULL,
    qty         INTEGER NOT NULL DEFAULT 1,
    flags       INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (character_id, slot),
    CONSTRAINT character_items_slot CHECK (slot BETWEEN 0 AND 127),
    CONSTRAINT character_items_qty CHECK (qty >= 1)
);

CREATE TABLE vault_items (
    account_id  BIGINT NOT NULL REFERENCES accounts (id) ON DELETE CASCADE,
    slot        SMALLINT NOT NULL,
    item_id     INTEGER NOT NULL,
    qty         INTEGER NOT NULL DEFAULT 1,
    flags       INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (account_id, slot),
    CONSTRAINT vault_items_slot CHECK (slot BETWEEN 0 AND 127),
    CONSTRAINT vault_items_qty CHECK (qty >= 1)
);

-- migrate:down
DROP TABLE vault_items;
DROP TABLE character_items;
