-- 0010_economia: cidades, preço de item, loja pessoal, imposto, donate e recompensa diária.
-- Regras em docs/wyd/economia.md e decisões em docs/wyd/decisoes-servidor.md §3.
-- Imposto: 10% por loja pessoal na cidade; 40% do acumulado das 3 cidades vai para Noatun.

-- migrate:up
-- Cidade jogável. `region_id` liga ao retângulo do Regions.txt (0008) sem duplicar coordenada.
CREATE TABLE city (
    code        TEXT PRIMARY KEY CHECK (code ~ '^[a-z_]+$'),
    name        TEXT NOT NULL CHECK (name <> ''),
    -- UNVERIFIED: o build 7662 não publica qual `map_id` corresponde a cada cidade; o campo
    -- fica nulo até a prova (trace do cliente), em vez de receber número inventado.
    map_id      SMALLINT CHECK (map_id BETWEEN 0 AND 255),
    region_id   SMALLINT REFERENCES region (id),
    is_capital  BOOLEAN NOT NULL DEFAULT false,
    tax_rate_bps INTEGER NOT NULL DEFAULT 1000 CHECK (tax_rate_bps BETWEEN 0 AND 10000),
    content_source TEXT NOT NULL DEFAULT 'docs/wyd/economia.md'
);

COMMENT ON COLUMN city.tax_rate_bps IS
    'Imposto da loja pessoal na cidade, em bps (1000 = 10%).';

-- Preço de NPC: separado do catálogo porque o mesmo item custa diferente por mercador.
CREATE TABLE item_price (
    item_index  INTEGER NOT NULL REFERENCES item_catalog (item_index) ON DELETE CASCADE,
    npc_id      INTEGER REFERENCES npc_definition (id) ON DELETE CASCADE,
    buy_price   BIGINT NOT NULL CHECK (buy_price >= 0),
    sell_price  BIGINT NOT NULL CHECK (sell_price >= 0),
    content_source TEXT NOT NULL DEFAULT 'npc_shop_item',
    PRIMARY KEY (item_index, npc_id)
);

-- Loja pessoal (auto-venda). `tax_paid` é o acumulado já recolhido, para conferência.
CREATE TABLE personal_shop (
    id            BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    character_id  BIGINT NOT NULL REFERENCES characters (id) ON DELETE CASCADE,
    city_code     TEXT NOT NULL REFERENCES city (code),
    name          TEXT NOT NULL CHECK (name <> ''),
    is_open       BOOLEAN NOT NULL DEFAULT true,
    tax_bps       INTEGER NOT NULL DEFAULT 1000 CHECK (tax_bps BETWEEN 0 AND 10000),
    tax_paid      BIGINT NOT NULL DEFAULT 0 CHECK (tax_paid >= 0),
    opened_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    closed_at     TIMESTAMPTZ,
    CONSTRAINT personal_shop_closed_after_open CHECK (
        closed_at IS NULL OR closed_at >= opened_at
    )
);

CREATE INDEX personal_shop_city_open_idx ON personal_shop (city_code) WHERE is_open;

-- Livro-caixa do imposto: cada venda gera linha; o fechamento soma por cidade.
CREATE TABLE city_tax_ledger (
    id           BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    city_code    TEXT NOT NULL REFERENCES city (code),
    shop_id      BIGINT REFERENCES personal_shop (id) ON DELETE SET NULL,
    amount       BIGINT NOT NULL CHECK (amount >= 0),
    reason       TEXT NOT NULL CHECK (reason IN ('shop_sale', 'npc_sale', 'portal', 'collection')),
    collected_by BIGINT REFERENCES characters (id) ON DELETE SET NULL,
    at           TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX city_tax_ledger_city_at_idx ON city_tax_ledger (city_code, at);

-- Loja premium: dinheiro real nunca entra aqui (só crédito já conciliado pelo portal).
CREATE TABLE donate_shop_item (
    id          INTEGER GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    item_index  INTEGER NOT NULL REFERENCES item_catalog (item_index),
    price_credits INTEGER NOT NULL CHECK (price_credits > 0),
    duration_days SMALLINT CHECK (duration_days IS NULL OR duration_days > 0),
    enabled     BOOLEAN NOT NULL DEFAULT true
);

-- Recompensa diária: `day` 1..7 ou NULL para "todo dia"; item e/ou ouro.
CREATE TABLE daily_reward_item (
    id         INTEGER GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    day        SMALLINT CHECK (day BETWEEN 1 AND 7),
    item_index INTEGER REFERENCES item_catalog (item_index),
    gold       BIGINT NOT NULL DEFAULT 0 CHECK (gold >= 0),
    qty        SMALLINT NOT NULL DEFAULT 1 CHECK (qty > 0),
    account_wide BOOLEAN NOT NULL DEFAULT true,
    CHECK (item_index IS NOT NULL OR gold > 0)
);

-- migrate:down
DROP TABLE daily_reward_item;
DROP TABLE donate_shop_item;
DROP TABLE city_tax_ledger;
DROP TABLE personal_shop;
DROP TABLE item_price;
DROP TABLE city;
