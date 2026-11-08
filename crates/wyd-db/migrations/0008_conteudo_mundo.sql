-- 0008_conteudo_mundo: mundo (NPCs, lojas, prêmios por nível, spawns, regiões) + proveniência.
-- Convenção: -- migrate:up / -- migrate:down (docs/arquitetura/modelo-banco.md §4).
-- Fonte canônica é o Postgres (ADR-0009); nada é lido de arquivo em runtime.
-- Sanitização aplicada ao insumo e mapa coluna↔insumo: modelo-banco.md §6.6.

-- migrate:up
-- Proveniência: cada insumo legado consumido por uma migração de conteúdo, com o sha256
-- do byte-a-byte que o gerador leu (ADR-0009). É auditoria, não dependência: apagar o
-- insumo não muda o banco.
CREATE TABLE content_source (
    path        TEXT PRIMARY KEY,
    sha256      TEXT NOT NULL CHECK (sha256 ~ '^[0-9a-f]{64}$'),
    size_bytes  INTEGER NOT NULL CHECK (size_bytes > 0),
    row_count   INTEGER NOT NULL CHECK (row_count >= 0),
    loaded_by   TEXT NOT NULL,
    transform   TEXT NOT NULL DEFAULT '',
    imported_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- STRUCT_MOB canônico (816 B) exatamente como o legado entregou. `source_size` preserva o
-- tamanho do arquivo de origem (756/816/920) para auditoria: o legado já converteu tudo
-- para 816 antes de gravar; `sha256` bate byte-a-byte com `blob` (conferido no gerador).
CREATE TABLE npc_template (
    template_name   TEXT PRIMARY KEY CHECK (template_name <> ''),
    file_name       TEXT NOT NULL CHECK (file_name <> ''),
    source_size     SMALLINT NOT NULL CHECK (source_size IN (756, 816, 920)),
    sha256          TEXT NOT NULL CHECK (sha256 ~ '^[0-9a-f]{64}$'),
    blob            BYTEA NOT NULL CHECK (octet_length(blob) = 816),
    -- 20 nomes vieram de arquivo coreano (CP949) decodificado como ISO-8859-1. A chave é
    -- preservada como está porque é referência de cliente e de `npc_definition`; a
    -- leitura em coreano vai em `name_recovered`. Ver modelo-banco §6.6.
    encoding_origin TEXT NOT NULL DEFAULT 'utf8'
                    CHECK (encoding_origin IN ('utf8', 'cp949-as-latin1')),
    name_recovered  TEXT,
    -- Nem todo nome recupera inteiro (8 dos 20 têm byte coreano perdido), então a regra
    -- é uma só: só existe `name_recovered` quando a origem declarada é o mojibake.
    CONSTRAINT npc_template_recovered_needs_mojibake CHECK (
        name_recovered IS NULL OR encoding_origin = 'cp949-as-latin1'
    )
);

CREATE INDEX npc_template_source_size_idx ON npc_template (source_size);

-- NPC/gerador de mundo (NPCGener). `origin = 'custom'` é edição do servidor e o import
-- não a sobrescreve; `generator_data` guarda os segmentos (seg_x/seg_y/seg_wait/
-- seg_range/die_action/fight_action) sem inventar colunas para um layout UNVERIFIED.
CREATE TABLE npc_definition (
    id                INTEGER PRIMARY KEY CHECK (id > 0),
    slug              TEXT NOT NULL UNIQUE CHECK (slug <> ''),
    template_name     TEXT NOT NULL REFERENCES npc_template (template_name),
    display_name      TEXT NOT NULL DEFAULT '',
    enabled           BOOLEAN NOT NULL DEFAULT true,
    map_id            SMALLINT NOT NULL DEFAULT 0 CHECK (map_id BETWEEN 0 AND 255),
    pos_x             SMALLINT NOT NULL DEFAULT 0 CHECK (pos_x BETWEEN 0 AND 4095),
    pos_y             SMALLINT NOT NULL DEFAULT 0 CHECK (pos_y BETWEEN 0 AND 4095),
    route_type        SMALLINT NOT NULL DEFAULT 0 CHECK (route_type BETWEEN 0 AND 7),
    merchant          SMALLINT NOT NULL DEFAULT 0 CHECK (merchant >= 0),
    origin            TEXT NOT NULL DEFAULT 'content' CHECK (origin IN ('content', 'custom')),
    generator_index   INTEGER CHECK (generator_index >= 0),
    follower_template TEXT NOT NULL DEFAULT '',
    minute_generate   INTEGER NOT NULL DEFAULT 0,
    min_group         SMALLINT NOT NULL DEFAULT 0 CHECK (min_group >= 0),
    max_group         SMALLINT NOT NULL DEFAULT 0 CHECK (max_group >= 0),
    max_num_mob       SMALLINT NOT NULL DEFAULT 1,
    formation         SMALLINT NOT NULL DEFAULT 0 CHECK (formation BETWEEN 0 AND 7),
    generator_data    JSONB NOT NULL DEFAULT '{}'::jsonb
                      CHECK (jsonb_typeof(generator_data) = 'object')
);

CREATE INDEX npc_definition_map_idx ON npc_definition (map_id, pos_x, pos_y);
CREATE INDEX npc_definition_template_idx ON npc_definition (template_name);
CREATE INDEX npc_definition_generator_idx ON npc_definition (generator_index);
-- Busca do portal por loja: 549 dos 6.100 NPCs são comerciantes.
CREATE INDEX npc_definition_merchant_idx ON npc_definition (merchant) WHERE merchant > 0;

-- Estoque da loja: 27 slots fixos por NPC (0..26), sem duplicata (conferido no gerador).
CREATE TABLE npc_shop_item (
    npc_id     INTEGER NOT NULL REFERENCES npc_definition (id) ON DELETE CASCADE,
    slot       SMALLINT NOT NULL CHECK (slot BETWEEN 0 AND 26),
    item_index INTEGER NOT NULL,
    eff1       SMALLINT NOT NULL DEFAULT 0,
    effv1      SMALLINT NOT NULL DEFAULT 0,
    eff2       SMALLINT NOT NULL DEFAULT 0,
    effv2      SMALLINT NOT NULL DEFAULT 0,
    eff3       SMALLINT NOT NULL DEFAULT 0,
    effv3      SMALLINT NOT NULL DEFAULT 0,
    quantity   SMALLINT NOT NULL DEFAULT 1 CHECK (quantity BETWEEN 1 AND 255),
    PRIMARY KEY (npc_id, slot)
);

CREATE INDEX npc_shop_item_item_idx ON npc_shop_item (item_index);

-- LevelItem.txt: "# SEQUENCIA, LEVEL, CLASSE, TYPE, ITEM" + 6 extras.
-- TYPE (documentado no próprio arquivo): -1 entrega independente da classe, 0 independente
-- dos pontos, 1 mais pontos em STR, 2 em INT, 3 em DEX. CLASSE: 0=TK, 1=FM, 2=BM, 3=HT.
-- O arquivo tem 318 linhas e apenas 161 linhas de conteúdo: o bloco inteiro aparece
-- repetido. `ordinal` é a posição na lista já deduplicada; `source_seq` é o id do arquivo
-- (dois `seq` têm duas linhas distintas — dado do legado, preservado).
CREATE TABLE level_item (
    ordinal     SMALLINT PRIMARY KEY CHECK (ordinal > 0),
    source_seq  SMALLINT NOT NULL CHECK (source_seq >= 0),
    source_line INTEGER NOT NULL CHECK (source_line > 0),
    level       SMALLINT NOT NULL CHECK (level BETWEEN 1 AND 255),
    class_id    SMALLINT NOT NULL CHECK (class_id BETWEEN 0 AND 3),
    type_id     SMALLINT NOT NULL CHECK (type_id BETWEEN -1 AND 3),
    item_index  INTEGER NOT NULL CHECK (item_index > 0),
    extras      JSONB NOT NULL DEFAULT '[]'::jsonb CHECK (jsonb_typeof(extras) = 'array')
);

CREATE INDEX level_item_level_class_idx ON level_item (level, class_id);
CREATE INDEX level_item_item_idx ON level_item (item_index);

-- Guard.txt: "<guard_index>,<npc_index>" + N blocos "[8 números]".
-- O cabeçalho do arquivo descreve 13 campos e os blocos têm 8: o layout interno é
-- UNVERIFIED, então o bloco é preservado na ordem original (`values`), sem inventar nomes.
CREATE TABLE guard_spawn (
    guard_index SMALLINT PRIMARY KEY CHECK (guard_index >= 0),
    npc_id      INTEGER NOT NULL REFERENCES npc_definition (id),
    source_line INTEGER NOT NULL
);

CREATE TABLE guard_spawn_reward (
    guard_index SMALLINT NOT NULL REFERENCES guard_spawn (guard_index) ON DELETE CASCADE,
    req_slot    SMALLINT NOT NULL CHECK (req_slot BETWEEN 0 AND 7),
    values      SMALLINT[] NOT NULL CHECK (array_length(values, 1) = 8),
    PRIMARY KEY (guard_index, req_slot)
);

-- MobMerc.txt: loja de mercador com renovação periódica (1 bloco no insumo atual).
CREATE TABLE mob_merc (
    generator_index INTEGER PRIMARY KEY,
    template_name   TEXT NOT NULL DEFAULT '',
    renew_seconds   INTEGER NOT NULL CHECK (renew_seconds >= 0),
    reborn_seconds  INTEGER NOT NULL CHECK (reborn_seconds >= 0),
    items           JSONB NOT NULL DEFAULT '[]'::jsonb CHECK (jsonb_typeof(items) = 'array')
);

-- gameconfig.txt: par chave/valor por seção. `evstart` tem 4 números, então o valor fica
-- em `value_text` (verdade do arquivo) e `value_int` só existe quando é um inteiro só.
CREATE TABLE game_setting (
    section     TEXT NOT NULL CHECK (section <> ''),
    key         TEXT NOT NULL CHECK (key <> ''),
    value_int   BIGINT,
    value_text  TEXT NOT NULL DEFAULT '',
    source_line INTEGER NOT NULL,
    PRIMARY KEY (section, key)
);

-- Seções matriciais do gameconfig (Item Drop Bonus 4x16, Treasure 8x2): valores na ordem
-- do arquivo, sem semântica inventada.
CREATE TABLE game_matrix (
    section     TEXT NOT NULL CHECK (section <> ''),
    block_index SMALLINT NOT NULL CHECK (block_index >= 0),
    line_index  SMALLINT NOT NULL CHECK (line_index >= 0),
    values      SMALLINT[] NOT NULL CHECK (array_length(values, 1) > 0),
    source_line INTEGER NOT NULL,
    PRIMARY KEY (section, block_index, line_index)
);

-- Regions.txt: "<x1>,<y1>,<x2>,<y2> = Nome". O nome NÃO é único: Erion e Armia aparecem
-- duas vezes com retângulos diferentes (por isso id sintético + chave natural completa).
CREATE TABLE region (
    id          SMALLINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name        TEXT NOT NULL CHECK (name <> ''),
    name_fold   TEXT GENERATED ALWAYS AS (content_fold(name)) STORED,
    -- Sem teto: o insumo tem uma região com 6841, acima do espaço 0..4095 do cliente.
    -- O teto seria uma invenção nossa; quem valida contra o mapa é o loader (FASE-03-4).
    x1          INTEGER NOT NULL CHECK (x1 >= 0),
    y1          INTEGER NOT NULL CHECK (y1 >= 0),
    x2          INTEGER NOT NULL CHECK (x2 >= 0),
    y2          INTEGER NOT NULL CHECK (y2 >= 0),
    source_line INTEGER NOT NULL,
    CONSTRAINT region_order CHECK (x1 <= x2 AND y1 <= y2),
    CONSTRAINT region_unique_rect UNIQUE (name, x1, y1, x2, y2)
);

CREATE INDEX region_name_fold_idx ON region (name_fold);

CREATE TABLE world_item_seed (
    id         INTEGER PRIMARY KEY CHECK (id > 0),
    item_index INTEGER NOT NULL CHECK (item_index > 0),
    pos_x      SMALLINT NOT NULL CHECK (pos_x >= 0),
    pos_y      SMALLINT NOT NULL CHECK (pos_y >= 0),
    rotate     SMALLINT NOT NULL DEFAULT 0,
    enabled    BOOLEAN NOT NULL DEFAULT true
);

CREATE TABLE guild_zone (
    clan            SMALLINT NOT NULL,
    zone            SMALLINT NOT NULL,
    victory         SMALLINT NOT NULL DEFAULT 0,
    city_tax        SMALLINT NOT NULL DEFAULT 0,
    tax_vault       SMALLINT NOT NULL DEFAULT 0,
    charge_guild    SMALLINT NOT NULL DEFAULT 0,
    challenge_guild SMALLINT NOT NULL DEFAULT 0,
    challenge_money SMALLINT NOT NULL DEFAULT 0,
    PRIMARY KEY (clan, zone)
);

COMMENT ON TABLE content_source IS
    'Proveniência do conteúdo importado: path/sha256/tamanho/linhas do insumo legado.';
COMMENT ON TABLE npc_definition IS
    'NPC/gerador no mundo (NPCGener): posição, template e grupo. origin=custom não é sobrescrito pelo import.';
COMMENT ON COLUMN npc_definition.merchant IS
    'Campo Merchant do NPCGener (0 = não comerciante). A semântica exata acima de 0 é UNVERIFIED.';
COMMENT ON COLUMN npc_shop_item.item_index IS
    'Referencia item_catalog por convenção, SEM FK: 22 itens vendidos não existem no catálogo legado e um deles é negativo (-25426).';
COMMENT ON COLUMN npc_shop_item.eff1 IS
    'eff1..eff3/effv1..effv3 são os efeitos aplicados na compra; código numérico EF_* é UNVERIFIED, não vira gate.';
COMMENT ON COLUMN level_item.type_id IS
    'TYPE do LevelItem.txt: -1 independente de classe, 0 independente de pontos, 1 STR, 2 INT, 3 DEX (documentado no insumo).';
COMMENT ON TABLE guard_spawn IS
    'Guard.txt: NPC guardião que exige itens. O bloco de 8 números por exigência é preservado cru (UNVERIFIED).';
COMMENT ON TABLE game_setting IS
    'Pares chave/valor do gameconfig.txt; a chave é única dentro da seção.';
COMMENT ON TABLE game_matrix IS
    'Matrizes do gameconfig (Item Drop Bonus, Treasure) na ordem do arquivo.';
COMMENT ON TABLE region IS
    'Regiões de PvP/siege do Regions.txt: nome repetido com retângulos diferentes é legítimo.';
COMMENT ON COLUMN region.name_fold IS
    'Chave de busca sem acento/caixa (content_fold) para o portal não depender de digitação exata.';

-- migrate:down
DROP TABLE guild_zone;
DROP TABLE world_item_seed;
DROP TABLE region;
DROP TABLE game_matrix;
DROP TABLE game_setting;
DROP TABLE mob_merc;
DROP TABLE guard_spawn_reward;
DROP TABLE guard_spawn;
DROP TABLE level_item;
DROP TABLE npc_shop_item;
DROP TABLE npc_definition;
DROP TABLE npc_template;
DROP TABLE content_source;
