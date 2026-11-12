-- 0043 — piedade de refino por item (GAME-16): o contador que o legado guarda no item.
--
-- O legado empacota nível + piedade no par EF_SANC do próprio STRUCT_ITEM
-- (`cValue = nível + 10×piedade`, `Basedef.cpp:2282`); o esquema normalizado não tem
-- par de efeito por linha, então a piedade mora aqui, chaveada por (personagem, slot).
-- Tabela própria em vez de `ALTER TABLE character_items`: a série proíbe `ALTER` no `up`.
-- Linha ausente = piedade 0 (valor inicial explícito, nunca taxa inventada).
-- Slot fora do item some junto: `ON DELETE CASCADE` no personagem e poda por poda
-- (`replace_items` apaga as linhas de slots que o novo inventário não tem).

-- migrate:up
CREATE TABLE refine_pity (
    character_id BIGINT NOT NULL REFERENCES characters (id) ON DELETE CASCADE,
    slot         SMALLINT NOT NULL CHECK (slot BETWEEN 0 AND 127),
    pity         SMALLINT NOT NULL DEFAULT 0 CHECK (pity BETWEEN 0 AND 20),
    PRIMARY KEY (character_id, slot)
);

-- migrate:down
DROP TABLE refine_pity;
