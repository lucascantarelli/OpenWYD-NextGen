-- 0003_inventario: itens de personagem e baú da conta (normalizado, anti-dupe).
-- Convenção: -- migrate:up / -- migrate:down (ver docs/arquitetura/modelo-banco.md §4).
--
-- Refino, Ancient e gema são estado DA LINHA de item, não do catálogo: por isso nascem aqui,
-- na criação da tabela, em vez de chegarem por `ALTER TABLE` depois. Regra e números:
-- docs/wyd/sistemas/refino.md §2–§3; espelho de `Item` em
-- crates/wyd-core/src/ecs/components.rs (autoridade do jogo).

-- migrate:up
CREATE TABLE character_items (
    character_id BIGINT NOT NULL REFERENCES characters (id) ON DELETE CASCADE,
    slot         SMALLINT NOT NULL,
    item_id      INTEGER NOT NULL,
    qty          INTEGER NOT NULL DEFAULT 1,
    flags        INTEGER NOT NULL DEFAULT 0,
    refine       SMALLINT NOT NULL DEFAULT 0,
    ancient      BOOLEAN NOT NULL DEFAULT false,
    gem_effect   SMALLINT NOT NULL DEFAULT 0,
    gem_value    SMALLINT NOT NULL DEFAULT 0,
    PRIMARY KEY (character_id, slot),
    CONSTRAINT character_items_slot CHECK (slot BETWEEN 0 AND 127),
    CONSTRAINT character_items_qty CHECK (qty >= 1),
    CONSTRAINT character_items_refine CHECK (refine BETWEEN 0 AND 15),
    CONSTRAINT character_items_gem_effect CHECK (gem_effect BETWEEN 0 AND 4),
    CONSTRAINT character_items_gem_value CHECK (gem_value >= 0),
    -- Efeito e valor andam juntos: um sem o outro é linha inconsistente.
    CONSTRAINT character_items_gem_pair CHECK ((gem_effect = 0) = (gem_value = 0)),
    -- Mesma regra do core: gema em item comum exige +10; Anct é a exceção.
    CONSTRAINT character_items_gem_refine CHECK (gem_effect = 0 OR ancient OR refine >= 10)
);

CREATE TABLE vault_items (
    account_id  BIGINT NOT NULL REFERENCES accounts (id) ON DELETE CASCADE,
    slot        SMALLINT NOT NULL,
    item_id     INTEGER NOT NULL,
    qty         INTEGER NOT NULL DEFAULT 1,
    flags       INTEGER NOT NULL DEFAULT 0,
    refine      SMALLINT NOT NULL DEFAULT 0,
    ancient     BOOLEAN NOT NULL DEFAULT false,
    gem_effect  SMALLINT NOT NULL DEFAULT 0,
    gem_value   SMALLINT NOT NULL DEFAULT 0,
    PRIMARY KEY (account_id, slot),
    CONSTRAINT vault_items_slot CHECK (slot BETWEEN 0 AND 127),
    CONSTRAINT vault_items_qty CHECK (qty >= 1),
    CONSTRAINT vault_items_refine CHECK (refine BETWEEN 0 AND 15),
    CONSTRAINT vault_items_gem_effect CHECK (gem_effect BETWEEN 0 AND 4),
    CONSTRAINT vault_items_gem_value CHECK (gem_value >= 0),
    CONSTRAINT vault_items_gem_pair CHECK ((gem_effect = 0) = (gem_value = 0)),
    CONSTRAINT vault_items_gem_refine CHECK (gem_effect = 0 OR ancient OR refine >= 10)
);

-- migrate:down
DROP TABLE vault_items;
DROP TABLE character_items;
