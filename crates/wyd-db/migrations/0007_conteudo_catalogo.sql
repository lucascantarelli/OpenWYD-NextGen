-- 0007_conteudo_catalogo: conteúdo estático no banco (ADR-0009) — itens, skills e taxonomia.
-- Convenção: -- migrate:up / -- migrate:down (ver docs/arquitetura/modelo-banco.md §4).
-- Fonte canônica é ESTA tabela; nenhum CSV/TXT é lido em runtime (ADR-0009).

-- migrate:up
-- Chave de busca insensível a acento e caixa (ADR-0009 §sanitização): o valor gravado é
-- sempre NFC, o que se compara é a dobra. Não usamos `unaccent` — é STABLE, não entra em
-- índice nem em coluna gerada — nem `citext`, que ignora caixa mas não acento. `lower()`,
-- `translate()` e `regexp_replace()` são IMMUTABLE, então a dobra é determinística e
-- indexável em qualquer Postgres (sem extensão, sem superusuário).
CREATE FUNCTION content_fold(value TEXT) RETURNS TEXT
LANGUAGE sql IMMUTABLE STRICT PARALLEL SAFE
AS $$
    SELECT btrim(regexp_replace(
        translate(
            replace(replace(replace(lower(value),
                U&'\00E6', 'ae'), U&'\0153', 'oe'), U&'\00DF', 'ss'),
            U&'\00E1\00E0\00E2\00E3\00E4\00E5\00E9\00E8\00EA\00EB\00ED\00EC\00EE\00EF\00F3\00F2\00F4\00F5\00F6\00FA\00F9\00FB\00FC\00E7\00F1\00FD\00FF\00AA\00BA',
            'aaaaaaeeeeiiiiooooouuuucnyyao'
        ),
        '[^a-z0-9]+', ' ', 'g'
    ));
$$;

CREATE TABLE content_version (
    id         SMALLINT PRIMARY KEY CHECK (id = 1),
    version    INTEGER NOT NULL CHECK (version >= 1),
    note       TEXT NOT NULL DEFAULT '',
    applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE game_system (
    code        TEXT PRIMARY KEY,
    label       TEXT NOT NULL,
    category    TEXT NOT NULL CHECK (category IN (
        'progressao', 'combate', 'economia', 'social', 'evento', 'mundo'
    )),
    description TEXT NOT NULL DEFAULT ''
);

CREATE TABLE game_class (
    code        TEXT PRIMARY KEY,
    label       TEXT NOT NULL,
    role        TEXT NOT NULL DEFAULT '',
    description TEXT NOT NULL DEFAULT ''
);

CREATE TABLE skill_lineage (
    code       TEXT PRIMARY KEY,
    class_code TEXT NOT NULL REFERENCES game_class (code),
    label      TEXT NOT NULL,
    focus      TEXT NOT NULL DEFAULT ''
);

CREATE TABLE item_kind (
    code        TEXT PRIMARY KEY,
    label       TEXT NOT NULL,
    slot        TEXT,
    description TEXT NOT NULL DEFAULT ''
);

-- Vocabulário EF_*: o nome é a chave (o código numérico é do ItemEffect.h e não foi
-- provado); `confirmed` separa o que a semântica já comprova do que é só símbolo visto.
CREATE TABLE item_effect (
    name        TEXT PRIMARY KEY,
    label       TEXT NOT NULL,
    category    TEXT NOT NULL CHECK (category IN ('score', 'restricao', 'economia', 'comportamento')),
    description TEXT NOT NULL DEFAULT '',
    confirmed   BOOLEAN NOT NULL DEFAULT false,
    first_seen  TEXT NOT NULL DEFAULT 'legacy item_catalog'
);

CREATE TABLE item_catalog (
    item_index   INTEGER PRIMARY KEY CHECK (item_index > 0),
    name         TEXT NOT NULL CHECK (name <> ''),
    -- Nome canônico (NFC) + chave de busca sem acento/caixa. O portal busca por aqui;
    -- nomes repetidos são legítimos (mesmo item em grades/eventos diferentes).
    name_fold    TEXT GENERATED ALWAYS AS (content_fold(name)) STORED,
    price        INTEGER NOT NULL DEFAULT 0 CHECK (price >= 0),
    grade        SMALLINT NOT NULL DEFAULT 0,
    unique_id    SMALLINT NOT NULL DEFAULT 0,
    extra        INTEGER NOT NULL DEFAULT 0,
    pos          SMALLINT NOT NULL DEFAULT 0,
    req_level    SMALLINT NOT NULL DEFAULT 0 CHECK (req_level >= 0),
    req_str      SMALLINT NOT NULL DEFAULT 0,
    req_int      SMALLINT NOT NULL DEFAULT 0,
    req_dex      SMALLINT NOT NULL DEFAULT 0,
    req_con      SMALLINT NOT NULL DEFAULT 0,
    volatile     SMALLINT,
    attack_range SMALLINT,
    effects      JSONB NOT NULL DEFAULT '[]'::jsonb
                 CHECK (jsonb_typeof(effects) = 'array'),
    effect_names TEXT[] NOT NULL DEFAULT '{}',
    fields       JSONB NOT NULL,
    -- Classificação de domínio (docs/wyd/itens.md). NULL = não sabemos; nunca chute.
    kind                  TEXT REFERENCES item_kind (code),
    primary_system        TEXT REFERENCES game_system (code),
    event_only            BOOLEAN,
    tradeable             BOOLEAN,
    stackable             BOOLEAN,
    class_mask            TEXT,
    classification_source TEXT NOT NULL DEFAULT 'legacy',
    confidence            SMALLINT CHECK (confidence BETWEEN 0 AND 100),
    notes                 TEXT NOT NULL DEFAULT '',
    sha256                TEXT NOT NULL DEFAULT '',
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX item_catalog_kind_idx ON item_catalog (kind);
CREATE INDEX item_catalog_system_idx ON item_catalog (primary_system);
CREATE INDEX item_catalog_name_lower_idx ON item_catalog (lower(name));
CREATE INDEX item_catalog_name_fold_idx ON item_catalog (name_fold);
CREATE INDEX item_catalog_effects_gin ON item_catalog USING GIN (effect_names);

CREATE TABLE skill_catalog (
    skill_index        INTEGER PRIMARY KEY CHECK (skill_index >= 0 AND skill_index < 248),
    name               TEXT NOT NULL DEFAULT '',
    skill_point        SMALLINT NOT NULL DEFAULT 0,
    target_type        SMALLINT NOT NULL DEFAULT 0,
    mana_spent         SMALLINT NOT NULL DEFAULT 0,
    delay_ticks        SMALLINT NOT NULL DEFAULT 0,
    range_tiles        SMALLINT NOT NULL DEFAULT 0,
    instance_type      SMALLINT NOT NULL DEFAULT 0,
    instance_value     SMALLINT NOT NULL DEFAULT 0,
    tick_type          SMALLINT NOT NULL DEFAULT 0,
    tick_value         SMALLINT NOT NULL DEFAULT 0,
    affect_type        SMALLINT NOT NULL DEFAULT 0,
    affect_value       SMALLINT NOT NULL DEFAULT 0,
    affect_time        SMALLINT NOT NULL DEFAULT 0,
    instance_attribute SMALLINT NOT NULL DEFAULT 0,
    tick_attribute     SMALLINT NOT NULL DEFAULT 0,
    aggressive         SMALLINT NOT NULL DEFAULT 0,
    max_target         SMALLINT NOT NULL DEFAULT 0,
    party              SMALLINT NOT NULL DEFAULT 0,
    affect_resist      SMALLINT NOT NULL DEFAULT 0,
    passive            SMALLINT NOT NULL DEFAULT 0,
    -- Classificação de domínio (docs/wyd/skills.md).
    class_code            TEXT REFERENCES game_class (code),
    lineage_code          TEXT REFERENCES skill_lineage (code),
    primary_system        TEXT REFERENCES game_system (code),
    skill_role            TEXT CHECK (skill_role IN (
        'damage', 'heal', 'buff', 'debuff', 'summon', 'mobility', 'passive', 'craft'
    )),
    classification_source TEXT NOT NULL DEFAULT 'legacy',
    confidence            SMALLINT CHECK (confidence BETWEEN 0 AND 100),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX skill_catalog_class_idx ON skill_catalog (class_code, lineage_code);
CREATE INDEX skill_catalog_name_lower_idx ON skill_catalog (lower(name));

COMMENT ON TABLE item_catalog IS
    'Catálogo de itens (ItemList) — base canônica no Postgres (ADR-0009); campos kind/system descrevem uso, não só número.';
COMMENT ON COLUMN item_catalog.effect_names IS
    'Nomes EF_* extraídos dos fields crus; consulta por efeito sem varrer JSON.';
COMMENT ON COLUMN item_catalog.class_mask IS
    'Quarto campo do ItemList (ex. 262.0.0.0.0): semântica UNVERIFIED, não usar como gate.';
COMMENT ON COLUMN item_catalog.classification_source IS
    'legacy | rule:name_v1 | unmapped — diz de onde veio kind/system, para auditoria.';

COMMENT ON TABLE skill_catalog IS
    'Catálogo de skills (SkillData) — comportamento no dado; classe/linhagem/papel são semântica classificada.';

INSERT INTO content_version (id, version, note)
VALUES (1, 1, 'catálogo de conteúdo no banco (ADR-0009)');

-- migrate:down
DROP TABLE skill_catalog;
DROP TABLE item_catalog;
DROP TABLE item_effect;
DROP TABLE item_kind;
DROP TABLE skill_lineage;
DROP TABLE game_class;
DROP TABLE game_system;
DROP TABLE content_version;
DROP FUNCTION content_fold(text);
