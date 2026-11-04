-- Teleport routes as moderator-editable config (issue 42).
--
-- The table is an OVERLAY over the compiled teleportTable in
-- tmserver/internal/world/teleport.go:
--
--   * a row with enabled = true  replaces the compiled route at that origin tile
--   * a row with enabled = false removes it (the tile stops teleporting)
--   * an origin with NO row keeps the compiled route
--
-- Nothing has to be seeded for the server to keep working; `import-teleports`
-- (tmserver/cmd/importteleports) materializes the 23 compiled routes when the
-- moderator wants them listed/editable in the portal.
--
-- Coordinates are stored as the ROUNDED origin tile (x &^ 3), which is exactly
-- the key teleportDest looks up — so src_x/src_y must be multiples of 4. The
-- web-api normalizes a non-multiple origin and reports it as a warning instead
-- of silently changing the gameplay tile.
CREATE TABLE teleport_route (
    id          BIGSERIAL PRIMARY KEY,
    src_x       INTEGER NOT NULL CHECK (src_x >= 0 AND src_x < 4096 AND src_x % 4 = 0),
    src_y       INTEGER NOT NULL CHECK (src_y >= 0 AND src_y < 4096 AND src_y % 4 = 0),
    dst_x       INTEGER NOT NULL CHECK (dst_x >= 0 AND dst_x < 4096),
    dst_y       INTEGER NOT NULL CHECK (dst_y >= 0 AND dst_y < 4096),
    -- Gold charged on arrival. 0 is the free hub (Noatum) and every field/dungeon
    -- route; only the three city → Noatum routes charge (700).
    cost        INTEGER NOT NULL DEFAULT 0 CHECK (cost >= 0),
    -- Same gates teleportDest applies today: 'fairy' = Vale needs item 3916 in
    -- Equip[13]; 'kefra' = Deserto → Kefra needs the Kefra defeated.
    requirement TEXT    NOT NULL DEFAULT 'none' CHECK (requirement IN ('none', 'fairy', 'kefra')),
    enabled     BOOLEAN NOT NULL DEFAULT TRUE,
    note        TEXT    NOT NULL DEFAULT '',
    updated_by  BIGINT,
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT teleport_route_src_key UNIQUE (src_x, src_y)
);

COMMENT ON TABLE teleport_route IS
    'Moderator-editable teleport overlay (issue 42); rows override world.teleportTable by origin tile.';
