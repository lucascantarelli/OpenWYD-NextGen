-- 0011_rotas_mundo: mapas lógicos e rotas de teleporte (NPC, custo e requisito).
-- Fonte do domínio: docs/wyd/sistemas/mundo.md. Lista de teleportes do 7662 é UNVERIFIED;
-- a tabela existe para receber o dado quando ele for capturado, nunca para inventá-lo.

-- migrate:up
CREATE TABLE map_definition (
    map_id      SMALLINT PRIMARY KEY CHECK (map_id BETWEEN 0 AND 255),
    name        TEXT NOT NULL CHECK (name <> ''),
    region_id   SMALLINT REFERENCES region (id),
    is_pvp      BOOLEAN NOT NULL DEFAULT false,
    is_safe     BOOLEAN NOT NULL DEFAULT false,
    is_dungeon  BOOLEAN NOT NULL DEFAULT false,
    content_source TEXT NOT NULL DEFAULT 'content_source'
);

COMMENT ON TABLE map_definition IS
    'Mapa lógico: id do STRUCT_MOB/mapa, nome exibido e vínculo ao retângulo de região.';

CREATE TABLE teleport_route (
    id             INTEGER GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    from_map_id    SMALLINT NOT NULL REFERENCES map_definition (map_id),
    from_x         SMALLINT NOT NULL CHECK (from_x BETWEEN 0 AND 4095),
    from_y         SMALLINT NOT NULL CHECK (from_y BETWEEN 0 AND 4095),
    to_map_id      SMALLINT NOT NULL REFERENCES map_definition (map_id),
    to_x           SMALLINT NOT NULL CHECK (to_x BETWEEN 0 AND 4095),
    to_y           SMALLINT NOT NULL CHECK (to_y BETWEEN 0 AND 4095),
    npc_id         INTEGER REFERENCES npc_definition (id) ON DELETE SET NULL,
    cost_gold      INTEGER NOT NULL DEFAULT 0 CHECK (cost_gold >= 0),
    req_level      SMALLINT NOT NULL DEFAULT 1 CHECK (req_level >= 1),
    req_item_index INTEGER REFERENCES item_catalog (item_index),
    enabled        BOOLEAN NOT NULL DEFAULT true,
    -- Procedência obrigatória: rota sem fonte é chute e não entra.
    content_source TEXT NOT NULL CHECK (content_source <> ''),
    confidence     SMALLINT NOT NULL DEFAULT 80 CHECK (confidence BETWEEN 0 AND 100),
    UNIQUE (from_map_id, from_x, from_y, to_map_id, to_x, to_y)
);

CREATE INDEX teleport_route_npc_idx ON teleport_route (npc_id) WHERE enabled;

COMMENT ON COLUMN teleport_route.content_source IS
    'De onde veio a rota (npc, capture, doc). Sem fonte não existe linha.';

-- migrate:down
DROP TABLE teleport_route;
DROP TABLE map_definition;
